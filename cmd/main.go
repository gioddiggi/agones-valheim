package cmd

import (
	"context"

	"github.com/gioddiggi/agones-valheim/internal/monitor"
)

func Execute() {
	monitor.Start(context.Background())
}
