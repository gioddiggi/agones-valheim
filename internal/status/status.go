package status

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gioddiggi/agones-valheim/pkg/config"
	"github.com/gioddiggi/agones-valheim/pkg/data"
)

// WaitReady blocks until Valheim reports a running server for the first time.
// Downloading and booting the server can take minutes, so there is no attempt
// limit or timeout: it only returns an error when ctx is cancelled.
func WaitReady(ctx context.Context, env *config.Env) error {
	slog.Info("Waiting for Valheim server to start...")
	return poll(ctx, env, 0)
}

// Check verifies that Valheim still reports a running server. It gives up
// after env.HealthCheckAttempts failed probes, after env.HealthCheckTimeout
// overall, or when ctx is cancelled.
func Check(ctx context.Context, env *config.Env) error {
	slog.Info("Running a status check against Valheim server...")

	ctx, cancel := context.WithTimeout(ctx, env.HealthCheckTimeout)
	defer cancel()

	err := poll(ctx, env, env.HealthCheckAttempts)
	if errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("status check timed out after %s: %w", env.HealthCheckTimeout, err)
	}
	return err
}

// poll probes the status endpoint every env.HealthCheckInterval until it
// reports a running server, ctx is done or maxAttempts probes have failed
// (0 means no limit).
func poll(ctx context.Context, env *config.Env, maxAttempts int) error {
	ticker := time.NewTicker(env.HealthCheckInterval)
	defer ticker.Stop()

	reason := ""
	for attempt := 1; ; attempt++ {
		err := probe(ctx, env.HealthCheckURL)

		if err == nil {
			slog.Info("Server status is: Running")
			return nil
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Startup repeats the same failure for minutes, log it once
		if err.Error() != reason {
			reason = err.Error()
			slog.Warn("Valheim server is not running", "attempt", attempt, "reason", reason)
		}
		if attempt == maxAttempts {
			return fmt.Errorf("status check failed after %d attempts: %w", attempt, err)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

// probe requests status.json once. Its "error" field is non-null while the
// server is not reachable.
func probe(ctx context.Context, url string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code %d", res.StatusCode)
	}

	var body data.ValheimStatusHttpResponse
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		return fmt.Errorf("decoding status: %w", err)
	}
	if body.Error != nil {
		return fmt.Errorf("server reported: %s", *body.Error)
	}
	return nil
}
