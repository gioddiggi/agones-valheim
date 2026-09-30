package monitor

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	agones "agones.dev/agones/sdks/go"
	"github.com/gioddiggi/agones-valheim/internal/status"
	"github.com/gioddiggi/agones-valheim/pkg/config"
)

// Start marks the GameServer as Ready once Valheim is up, then sends Agones
// health pings for as long as it stays up. It only returns on failure or when
// ctx is cancelled.
func Start(ctx context.Context) error {
	slog.Info("Starting Valheim monitor...")

	sdk, err := agones.NewSDK()
	if err != nil {
		return fmt.Errorf("creating Agones SDK: %w", err)
	}
	slog.Info("Successfully connected to Agones SDK")

	env := config.LoadEnv()

	if err := status.WaitReady(ctx, env); err != nil {
		return err
	}
	if err := sdk.Ready(); err != nil {
		return fmt.Errorf("marking GameServer as Ready: %w", err)
	}
	slog.Info("GameServer marked as Ready")

	ticker := time.NewTicker(env.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}

		if err := status.Check(ctx, env); err != nil {
			return err
		}
		if err := sdk.Health(); err != nil {
			slog.Warn("Error while sending health ping", "error", err)
		}
	}
}
