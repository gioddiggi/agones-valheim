package monitor

import (
	"context"
	"log/slog"

	agones "agones.dev/agones/sdks/go"
)

func Start(ctx context.Context) {
	slog.Info("Starting Valheim monitor...")

	_, err := agones.NewSDK()

	if err != nil {
		slog.Error("Error while creating Agones SDK", "error", err)
	} else {
		slog.Info("Successfully created Agones SDK")
	}
}
