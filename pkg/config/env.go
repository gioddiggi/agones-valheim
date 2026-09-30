package config

import (
	"log/slog"
	"os"
	"strconv"
	"time"
)

type Env struct {
	// Consecutive failed probes after which the health check gives up
	HealthCheckAttempts int
	// Pause between two probes, and so between two Agones health pings while
	// the server is healthy. Keep it below the GameServer health.periodSeconds.
	HealthCheckInterval time.Duration
	// Overall time after which a health check round gives up
	HealthCheckTimeout time.Duration
	// status.json served by the Valheim container (STATUS_HTTP=true)
	HealthCheckURL string
}

func LoadEnv() *Env {
	return &Env{
		HealthCheckAttempts: getInt("HEALTH_CHECK_ATTEMPTS", 5),
		HealthCheckInterval: getDuration("HEALTH_CHECK_INTERVAL", 2*time.Second),
		HealthCheckTimeout:  getDuration("HEALTH_CHECK_TIMEOUT", 10*time.Second),
		HealthCheckURL:      getString("HEALTH_CHECK_URL", "http://localhost:80/status.json"),
	}
}

func getString(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		slog.Warn("Invalid value, using default", "key", key, "value", value, "default", fallback)
		return fallback
	}
	return parsed
}

// getDuration parses values such as "500ms", "2s" or "1m".
func getDuration(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := time.ParseDuration(value)
	if err != nil || parsed <= 0 {
		slog.Warn("Invalid value, using default", "key", key, "value", value, "default", fallback)
		return fallback
	}
	return parsed
}
