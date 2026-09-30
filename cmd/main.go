package cmd

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/gioddiggi/agones-valheim/internal/monitor"
)

func Execute() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Start only returns once the monitor has stopped: unless that was asked
	// for with a signal, it is a failure
	if err := monitor.Start(ctx); ctx.Err() == nil {
		slog.Error("Valheim monitor stopped", "error", err)
		os.Exit(1)
	}
	slog.Info("Shutting down Valheim monitor...")
}
