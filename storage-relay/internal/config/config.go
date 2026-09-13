// Package config loads voxa-storage-relay settings from the environment.
package config

import (
	"fmt"
	"os"
)

type Config struct {
	Addr          string
	DatabaseURL   string
	RedisURL      string // e.g. redis://nexus:6379/2
	RecordingsDir string // e.g. /data/voxa/recordings
}

func Load() (Config, error) {
	cfg := Config{
		Addr:          getEnv("ADDR", ":8100"),
		DatabaseURL:   os.Getenv("DATABASE_URL"),
		RedisURL:      os.Getenv("REDIS_URL"),
		RecordingsDir: getEnv("RECORDINGS_DIR", "/data/voxa/recordings"),
	}

	if cfg.DatabaseURL == "" {
		return cfg, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.RedisURL == "" {
		return cfg, fmt.Errorf("REDIS_URL is required")
	}
	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
