package config

import (
	"cmp"
	"log/slog"
	"os"
	"strconv"
	"time"
)

type Env struct {
	// Consecutive failed probes after which a running server is unhealthy
	HealthCheckAttempts int
	// Pause between two probes, and so between two Agones health pings.
	// Keep it below the GameServer health.periodSeconds.
	HealthCheckInterval time.Duration
	// Overall time after which a health check gives up
	HealthCheckTimeout time.Duration
	// status.json served by the Valheim container (STATUS_HTTP=true)
	HealthCheckURL string
}

func LoadEnv() *Env {
	return &Env{
		HealthCheckAttempts: positive("HEALTH_CHECK_ATTEMPTS", 5, strconv.Atoi),
		HealthCheckInterval: positive("HEALTH_CHECK_INTERVAL", 2*time.Second, time.ParseDuration),
		HealthCheckTimeout:  positive("HEALTH_CHECK_TIMEOUT", 10*time.Second, time.ParseDuration),
		HealthCheckURL:      cmp.Or(os.Getenv("HEALTH_CHECK_URL"), "http://localhost:80/status.json"),
	}
}

// positive reads key with parse, falling back when it is unset, invalid or
// not greater than zero.
func positive[T int | time.Duration](key string, fallback T, parse func(string) (T, error)) T {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := parse(value)
	if err != nil || parsed <= 0 {
		slog.Warn("Invalid value, using default", "key", key, "value", value, "default", fallback)
		return fallback
	}
	return parsed
}
