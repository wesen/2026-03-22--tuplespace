package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTPListenAddr string
	DatabaseURL    string
	CandidateLimit int
	ShutdownGrace  time.Duration
}

func DefaultsFromEnv() Config {
	return Config{
		HTTPListenAddr: defaultEnv("TUPLESPACE_HTTP_LISTEN_ADDR", ":8080"),
		DatabaseURL:    os.Getenv("TUPLESPACE_DATABASE_URL"),
		CandidateLimit: 64,
		ShutdownGrace:  10 * time.Second,
	}
}

func LoadFromEnv() (Config, error) {
	cfg := DefaultsFromEnv()
	if rawLimit := os.Getenv("TUPLESPACE_CANDIDATE_LIMIT"); rawLimit != "" {
		limit, err := strconv.Atoi(rawLimit)
		if err != nil {
			return Config{}, fmt.Errorf("parse TUPLESPACE_CANDIDATE_LIMIT: %w", err)
		}
		cfg.CandidateLimit = limit
	}

	if rawGrace := os.Getenv("TUPLESPACE_SHUTDOWN_GRACE"); rawGrace != "" {
		grace, err := time.ParseDuration(rawGrace)
		if err != nil {
			return Config{}, fmt.Errorf("parse TUPLESPACE_SHUTDOWN_GRACE: %w", err)
		}
		cfg.ShutdownGrace = grace
	}

	return cfg, Validate(cfg)
}

func Validate(cfg Config) error {
	if cfg.DatabaseURL == "" {
		return fmt.Errorf("TUPLESPACE_DATABASE_URL is required")
	}
	if cfg.CandidateLimit <= 0 {
		return fmt.Errorf("candidate limit must be > 0")
	}
	if cfg.ShutdownGrace <= 0 {
		return fmt.Errorf("shutdown grace must be > 0")
	}
	return nil
}

func defaultEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
