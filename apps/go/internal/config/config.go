// Package config loads runtime configuration from environment variables.
//
// Why env-only: the backend runs in containers (docker compose / CI / prod),
// where 12-factor style env configuration is the least surprising and keeps
// secrets (DATABASE_URL) out of the source tree.
package config

import (
	"log/slog"
	"os"
	"strings"
	"time"
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

	// JWTSecret signs and verifies HS256 access tokens. MUST be overridden in
	// any non-development deployment.
	JWTSecret string
	// AccessTokenTTL is the lifetime of an access token (short-lived).
	AccessTokenTTL time.Duration
	// RefreshTokenTTL is the lifetime of a refresh token (long-lived).
	RefreshTokenTTL time.Duration
	// AppleClientIDs is the accepted `aud` set for Apple ID tokens (bundle IDs
	// and/or Service IDs). Empty disables Apple sign-in verification.
	AppleClientIDs []string
	// GoogleClientIDs is the accepted `aud` set for Google ID tokens (the OAuth
	// client IDs). Empty disables Google sign-in verification.
	GoogleClientIDs []string
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
		JWTSecret:          getenv("JWT_SECRET", "dev-insecure-secret-change-me"),
		AccessTokenTTL:     getduration("ACCESS_TOKEN_TTL", 15*time.Minute),
		RefreshTokenTTL:    getduration("REFRESH_TOKEN_TTL", 720*time.Hour), // 30 days
		AppleClientIDs:     splitList(getenv("APPLE_CLIENT_IDS", "")),
		GoogleClientIDs:    splitList(getenv("GOOGLE_CLIENT_IDS", "")),
	}
}

// getduration parses a Go duration (e.g. "15m", "720h") from the environment,
// falling back on parse errors or when unset.
func getduration(key string, fallback time.Duration) time.Duration {
	raw, ok := os.LookupEnv(key)
	if !ok || raw == "" {
		return fallback
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		slog.Warn("invalid duration, using default", slog.String("key", key), slog.String("value", raw))
		return fallback
	}
	return d
}

// splitList parses a comma-separated env value into a trimmed slice. Unlike
// splitOrigins it returns an empty slice (not "*") when unset.
func splitList(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
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
