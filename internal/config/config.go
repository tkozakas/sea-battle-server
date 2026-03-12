package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port                int
	TurnTimeout         time.Duration
	ReconnectGrace      time.Duration
	RoomCleanupInterval time.Duration
	MaxRooms            int
	LogLevel            string
}

func Load() *Config {
	return &Config{
		Port:                getEnvInt("PORT", 8080),
		TurnTimeout:         getEnvDuration("TURN_TIMEOUT", 30*time.Second),
		ReconnectGrace:      getEnvDuration("RECONNECT_GRACE", 60*time.Second),
		RoomCleanupInterval: getEnvDuration("ROOM_CLEANUP_INTERVAL", 60*time.Second),
		MaxRooms:            getEnvInt("MAX_ROOMS", 1000),
		LogLevel:            getEnvString("LOG_LEVEL", "info"),
	}
}

func getEnvInt(key string, defaultVal int) int {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}
	parsed, err := strconv.Atoi(v)
	if err != nil {
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
