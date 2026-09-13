// Package config loads voxa-api settings from the environment.
package config

import (
	"fmt"
	"os"
	"time"
)

type Config struct {
	Addr            string
	DatabaseURL     string
	JWTSecret       string
	JWTTTL          time.Duration
	StorageRelayURL string // e.g. http://100.127.159.10:8100
}

func Load() (Config, error) {
	cfg := Config{
		Addr:            getEnv("ADDR", ":8000"),
		DatabaseURL:     os.Getenv("DATABASE_URL"),
		JWTSecret:       os.Getenv("JWT_SECRET"),
		JWTTTL:          30 * 24 * time.Hour,
		StorageRelayURL: os.Getenv("STORAGE_RELAY_URL"),
	}

	if cfg.DatabaseURL == "" {
		return cfg, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.JWTSecret == "" {
		return cfg, fmt.Errorf("JWT_SECRET is required")
	}
	if cfg.StorageRelayURL == "" {
		return cfg, fmt.Errorf("STORAGE_RELAY_URL is required")
	}
	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
