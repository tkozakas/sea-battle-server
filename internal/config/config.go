package config

import (
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Port                int
	TurnTimeout         time.Duration
	ReconnectGrace      time.Duration
	RoomCleanupInterval time.Duration
	MaxRooms            int
	AllowedOrigins      []string
	LogLevel            string
	PingInterval        time.Duration
	WriteTimeout        time.Duration
	ShutdownTimeout     time.Duration
}

func Load() *Config {
	return &Config{
		Port:                getEnvInt("PORT", 8080),
		TurnTimeout:         getEnvDuration("TURN_TIMEOUT", 30*time.Second),
		ReconnectGrace:      getEnvDuration("RECONNECT_GRACE", 60*time.Second),
		RoomCleanupInterval: getEnvDuration("ROOM_CLEANUP_INTERVAL", 60*time.Second),
		MaxRooms:            getEnvInt("MAX_ROOMS", 1000),
		AllowedOrigins:      getEnvStringSlice("ALLOWED_ORIGINS", []string{"*"}),
		LogLevel:            getEnvString("LOG_LEVEL", "info"),
		PingInterval:        getEnvDuration("PING_INTERVAL", 15*time.Second),
		WriteTimeout:        getEnvDuration("WRITE_TIMEOUT", 5*time.Second),
		ShutdownTimeout:     getEnvDuration("SHUTDOWN_TIMEOUT", 15*time.Second),
	}
}

func getEnvInt(key string, defaultVal int) int {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}
	parsed, err := strconv.Atoi(v)
	if err != nil {
		slog.Warn("invalid env var value, using default", "key", key, "value", v, "default", defaultVal)
		return defaultVal
	}
	return parsed
}

func getEnvDuration(key string, defaultVal time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}
	parsed, err := time.ParseDuration(v)
	if err != nil {
		slog.Warn("invalid env var value, using default", "key", key, "value", v, "default", defaultVal)
		return defaultVal
	}
	return parsed
}

func getEnvString(key, defaultVal string) string {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}
	return v
}

func getEnvStringSlice(key string, defaultVal []string) []string {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}
	parts := strings.Split(v, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	if len(result) == 0 {
		return defaultVal
	}
	return result
}
