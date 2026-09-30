package cmd

import (
	"context"
	"os/signal"
	"syscall"

	"github.com/gioddiggi/agones-valheim/internal/monitor"
)

func Execute() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	monitor.Start(ctx)
}
