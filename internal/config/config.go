package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all runtime configuration, sourced from environment variables.
type Config struct {
	HTTPAddr        string
	ShutdownTimeout time.Duration

	DatabaseURL string

	KafkaBrokers []string
	KafkaTopic   string
	KafkaEnabled bool
}

// Load reads configuration from the environment, applying sensible defaults for
// local development. It only returns an error for values it cannot parse.
func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:        getenv("HTTP_ADDR", ":8080"),
		ShutdownTimeout: 10 * time.Second,
		DatabaseURL:     getenv("DATABASE_URL", "postgres://listings:listings@localhost:5432/listings?sslmode=disable"),
		KafkaBrokers:    splitAndTrim(getenv("KAFKA_BROKERS", "localhost:9092")),
		KafkaTopic:      getenv("KAFKA_TOPIC", "listings.events"),
	}

	enabled, err := strconv.ParseBool(getenv("KAFKA_ENABLED", "true"))
	if err != nil {
		return Config{}, fmt.Errorf("parse KAFKA_ENABLED: %w", err)
	}
	cfg.KafkaEnabled = enabled

	if d := os.Getenv("SHUTDOWN_TIMEOUT"); d != "" {
		parsed, err := time.ParseDuration(d)
		if err != nil {
			return Config{}, fmt.Errorf("parse SHUTDOWN_TIMEOUT: %w", err)
		}
		cfg.ShutdownTimeout = parsed
	}

	return cfg, nil
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// splitAndTrim splits a comma-separated list, trimming each item and dropping
// empties (e.g. "a, b ,,c" -> ["a","b","c"]).
func splitAndTrim(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}
