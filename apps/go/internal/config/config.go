// Package config loads runtime configuration from environment variables.
package config

import (
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/0muji4/Runa/apps/go/internal/itunes"
)

// Config holds all runtime configuration for the API server.
type Config struct {
	Port               string
	DatabaseURL        string
	LogLevel           string
	CORSAllowedOrigins []string
	AppEnv             string

	// JWTSecret MUST be overridden in any non-development deployment.
	JWTSecret       string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
	// AppleClientIDs / GoogleClientIDs are the accepted `aud` sets; empty disables that sign-in.
	AppleClientIDs  []string
	GoogleClientIDs []string
	// AdminAPIToken gates the /admin seed endpoints; empty disables them entirely.
	AdminAPIToken string
	ITunesBaseURL string

	// S3Endpoint is the host the SERVER reaches; empty disables gallery storage (endpoints answer 503).
	S3Endpoint string
	// S3PublicEndpoint is the host the CLIENT reaches (presigned URLs use it); empty falls back to S3Endpoint.
	S3PublicEndpoint string
	// S3Region is ignored by MinIO but SigV4 requires one.
	S3Region    string
	S3Bucket    string
	S3AccessKey string
	S3SecretKey string
	S3UseSSL    bool

	GalleryUploadURLTTL        time.Duration
	GalleryViewURLTTL          time.Duration
	GalleryMaxUploadBytes      int64
	GalleryAllowedContentTypes []string
}

// Load reads configuration from the environment with local development defaults.
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
		ITunesBaseURL:      getenv("ITUNES_BASE_URL", itunes.DefaultBaseURL),

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

// splitListDefault returns fallback when unset/empty, unlike splitList (empty slice).
func splitListDefault(key string, fallback []string) []string {
	if list := splitList(getenv(key, "")); len(list) > 0 {
		return list
	}
	return fallback
}

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

// splitList returns an empty slice when unset, unlike splitOrigins ("*").
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

func getenv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

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
