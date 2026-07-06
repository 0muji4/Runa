// Package config loads runtime configuration from environment variables.
//
// Why env-only: the backend runs in containers (docker compose / CI / prod),
// where 12-factor style env configuration is the least surprising and keeps
// secrets (DATABASE_URL) out of the source tree.
package config

import (
	"os"
	"strings"
)

// Config holds all runtime configuration for the API server.
type Config struct {
	// Port is the TCP port the HTTP server listens on.
	Port string
	// DatabaseURL is the pgx/libpq connection string for Postgres.
	DatabaseURL string
	// LogLevel is one of debug|info|warn|error (slog levels).
	LogLevel string
	// CORSAllowedOrigins is the list of origins allowed by the CORS middleware.
	CORSAllowedOrigins []string
	// AppEnv identifies the deployment environment (development|staging|production).
	AppEnv string
}

// Load reads configuration from the environment, applying sensible local
// defaults so the server boots without any env setup during development.
func Load() Config {
	return Config{
		Port:               getenv("PORT", "8080"),
		DatabaseURL:        getenv("DATABASE_URL", "postgres://runa:runa@localhost:5432/runa?sslmode=disable"),
		LogLevel:           getenv("LOG_LEVEL", "info"),
		CORSAllowedOrigins: splitOrigins(getenv("CORS_ALLOWED_ORIGINS", "*")),
		AppEnv:             getenv("APP_ENV", "development"),
	}
}

// getenv returns the environment value for key, or fallback when unset/empty.
func getenv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

// splitOrigins parses a comma-separated origin list into a trimmed slice.
func splitOrigins(raw string) []string {
	parts := strings.Split(raw, ",")
	origins := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			origins = append(origins, trimmed)
		}
	}
	if len(origins) == 0 {
		return []string{"*"}
	}
	return origins
}
