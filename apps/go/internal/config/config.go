// Package config loads runtime configuration from environment variables.
//
// Why env-only: the backend runs in containers (docker compose / CI / prod),
// where 12-factor style env configuration is the least surprising and keeps
// secrets (DATABASE_URL) out of the source tree.
package config

import (
	"log/slog"
	"os"
	"strconv"
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
	// AdminAPIToken gates the curated seed endpoints (POST /admin/quotes,
	// /admin/songs) via the X-Admin-Token header. Empty (the default) DISABLES
	// the admin surface entirely — it must be set to seed content.
	AdminAPIToken string

	// S3Endpoint is the host the SERVER reaches the object store on (e.g.
	// "minio:9000" inside docker). Empty (the default) DISABLES the gallery
	// storage surface: the gallery endpoints then answer 503.
	S3Endpoint string
	// S3PublicEndpoint is the host the CLIENT reaches the store on; presigned
	// URLs are built with it. Empty falls back to S3Endpoint. In docker the two
	// differ (server: "minio:9000", client: "localhost:9000"; Android emulator:
	// "10.0.2.2:9000").
	S3PublicEndpoint string
	// S3Region is the signing region (MinIO ignores it but SigV4 requires one).
	S3Region string
	// S3Bucket is the bucket gallery objects live in.
	S3Bucket string
	// S3AccessKey / S3SecretKey are the static credentials for signing.
	S3AccessKey string
	S3SecretKey string
	// S3UseSSL toggles https for both real requests and presigned URLs.
	S3UseSSL bool

	// GalleryUploadURLTTL is how long a presigned PUT URL is valid.
	GalleryUploadURLTTL time.Duration
	// GalleryViewURLTTL is how long a presigned GET (view) URL is valid.
	GalleryViewURLTTL time.Duration
	// GalleryMaxUploadBytes is the hard size cap enforced when issuing an upload
	// URL and re-checked against the real object at registration.
	GalleryMaxUploadBytes int64
	// GalleryAllowedContentTypes is the image MIME allowlist.
	GalleryAllowedContentTypes []string
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
		AdminAPIToken:      getenv("ADMIN_API_TOKEN", ""),

		S3Endpoint:       getenv("S3_ENDPOINT", ""),
		S3PublicEndpoint: getenv("S3_PUBLIC_ENDPOINT", ""),
		S3Region:         getenv("S3_REGION", "us-east-1"),
		S3Bucket:         getenv("S3_BUCKET", "runa-gallery"),
		S3AccessKey:      getenv("S3_ACCESS_KEY", ""),
		S3SecretKey:      getenv("S3_SECRET_KEY", ""),
		S3UseSSL:         getbool("S3_USE_SSL", false),

		GalleryUploadURLTTL:        getduration("GALLERY_UPLOAD_URL_TTL", 15*time.Minute),
		GalleryViewURLTTL:          getduration("GALLERY_VIEW_URL_TTL", 60*time.Minute),
		GalleryMaxUploadBytes:      getint64("GALLERY_MAX_UPLOAD_BYTES", 10*1024*1024), // 10 MiB
		GalleryAllowedContentTypes: splitListDefault("GALLERY_ALLOWED_CONTENT_TYPES", []string{"image/jpeg", "image/png", "image/webp", "image/heic"}),
	}
}

// getbool parses a boolean env value (1/t/true/0/f/false), falling back on parse
// errors or when unset.
func getbool(key string, fallback bool) bool {
	raw, ok := os.LookupEnv(key)
	if !ok || raw == "" {
		return fallback
	}
	v, err := strconv.ParseBool(raw)
	if err != nil {
		slog.Warn("invalid bool, using default", slog.String("key", key), slog.String("value", raw))
		return fallback
	}
	return v
}

// getint64 parses an int64 env value, falling back on parse errors or when unset.
func getint64(key string, fallback int64) int64 {
	raw, ok := os.LookupEnv(key)
	if !ok || raw == "" {
		return fallback
	}
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		slog.Warn("invalid int, using default", slog.String("key", key), slog.String("value", raw))
		return fallback
	}
	return v
}

// splitListDefault parses a comma-separated env value, returning fallback when
// unset/empty (unlike splitList which returns an empty slice).
func splitListDefault(key string, fallback []string) []string {
	if list := splitList(getenv(key, "")); len(list) > 0 {
		return list
	}
	return fallback
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
