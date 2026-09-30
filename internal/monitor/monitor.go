package monitor

import (
	"context"
	"log/slog"
	"os"
	"time"

	agones "agones.dev/agones/sdks/go"
	"github.com/gioddiggi/agones-valheim/internal/status"
	"github.com/gioddiggi/agones-valheim/pkg/config"
)

func Start(ctx context.Context) {

	slog.Info("Starting Valheim monitor...")

	sdk, err := agones.NewSDK()

	if err != nil {
		slog.Error("Error while creating Agones SDK", "error", err)
		os.Exit(1)
	}
	slog.Info("Successfully connected to Agones SDK")

	env := config.LoadEnv()

	// Startup: the GameServer becomes Ready the first time Valheim reports a running server
	if err := status.WaitReady(ctx, env); err != nil {
		slog.Info("Shutting down Valheim monitor...")
		return
	}

	if err := sdk.Ready(); err != nil {
		slog.Error("Error while marking GameServer as Ready", "error", err)
		os.Exit(1)
	}
	slog.Info("GameServer marked as Ready")

	// Running: keep checking the server and ping Agones while it is healthy
	ticker := time.NewTicker(env.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("Shutting down Valheim monitor...")
			return
		case <-ticker.C:
		}

		err := status.Check(ctx, env)
		if ctx.Err() != nil {
			slog.Info("Shutting down Valheim monitor...")
			return
		}
		if err != nil {
			slog.Error("Valheim server is unhealthy", "error", err)
			os.Exit(1)
		}

		if err := sdk.Health(); err != nil {
			slog.Warn("Error while sending health ping", "error", err)
		}
	}
}
