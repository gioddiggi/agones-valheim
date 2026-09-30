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
	} else {
		slog.Info("Successfully connected to Agones SDK")
	}

	env := config.LoadEnv()

	ticker := time.NewTicker(env.HealthCheckInterval)
	defer ticker.Stop()

	for {
		err := status.Check(ctx, env)
		if ctx.Err() != nil {
			slog.Info("Shutting down Valheim monitor...")
			return
		}
		if err != nil {
			slog.Error("Valheim server is unhealthy", "error", err)
			os.Exit(1)
		}
		sdk.Health()

		select {
		case <-ctx.Done():
			slog.Info("Shutting down Valheim monitor...")
			return
		case <-ticker.C:
		}
	}
}
