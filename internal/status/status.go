package status

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gioddiggi/agones-valheim/pkg/config"
	"github.com/gioddiggi/agones-valheim/pkg/data"
)

// WaitReady blocks until the Valheim status endpoint reports a running server
// for the first time, probing it every env.HealthCheckInterval. Startup has no
// attempt limit or timeout, since downloading and booting the server can take
// several minutes: it only gives up when ctx is cancelled (graceful shutdown),
// returning ctx.Err().
func WaitReady(ctx context.Context, env *config.Env) error {
	slog.Info("Waiting for Valheim server to start...")

	ticker := time.NewTicker(env.HealthCheckInterval)
	defer ticker.Stop()

	lastErr := ""
	for {
		err := probe(ctx, env.HealthCheckURL)
		if err == nil {
			slog.Info("Server status is: Running")
			return nil
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Log only when the reason changes, startup would repeat it for minutes
		if err.Error() != lastErr {
			lastErr = err.Error()
			slog.Info("Valheim server is not ready yet", "reason", err)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

// Check probes the Valheim status endpoint until it reports a running server,
// waiting env.HealthCheckInterval between two failed attempts.
// It gives up after env.HealthCheckAttempts failed attempts, after
// env.HealthCheckTimeout overall, or as soon as ctx is cancelled
// (graceful shutdown), in which case the returned error wraps ctx.Err().
func Check(ctx context.Context, env *config.Env) error {
	slog.Info("Running a status check against Valheim server...")

	timeout := env.HealthCheckTimeout
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	for attempt := 1; ; attempt++ {
		err := probe(timeoutCtx, env.HealthCheckURL)
		if err == nil {
			slog.Info("Server status is: Running")
			return nil
		}

		// A probe interrupted by shutdown or by the deadline is not a failed attempt
		if timeoutCtx.Err() != nil {
			return interrupted(ctx, timeout)
		}

		slog.Warn("Status check failed", "attempt", attempt, "attempts", env.HealthCheckAttempts, "error", err)

		if attempt >= env.HealthCheckAttempts {
			return fmt.Errorf("status check failed after %d attempts: %w", attempt, err)
		}

		select {
		case <-timeoutCtx.Done():
			return interrupted(ctx, timeout)
		case <-time.After(env.HealthCheckInterval):
		}
	}
}

// interrupted tells a graceful shutdown (parent context cancelled) apart from
// the health check running out of time.
func interrupted(ctx context.Context, timeout time.Duration) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("status check interrupted by shutdown: %w", err)
	}
	return fmt.Errorf("status check timed out after %s: %w", timeout, context.DeadlineExceeded)
}

// probe runs a single request against the status.json served by the Valheim
// container, which carries a non-null "error" while the server is not reachable.
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

	body := &data.ValheimStatusHttpResponse{}

	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		return fmt.Errorf("decoding status: %w", err)
	}
	if body.Error != nil {
		return fmt.Errorf("server reported: %s", *body.Error)
	}

	return nil
}
