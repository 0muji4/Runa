package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var allEnvKeys = []string{
	"PORT",
	"DATABASE_URL",
	"LOG_LEVEL",
	"CORS_ALLOWED_ORIGINS",
	"APP_ENV",
	"JWT_SECRET",
	"ACCESS_TOKEN_TTL",
	"REFRESH_TOKEN_TTL",
	"APPLE_CLIENT_IDS",
	"GOOGLE_CLIENT_IDS",
	"ADMIN_API_TOKEN",
	"S3_ENDPOINT",
	"S3_PUBLIC_ENDPOINT",
	"S3_REGION",
	"S3_BUCKET",
	"S3_ACCESS_KEY",
	"S3_SECRET_KEY",
	"S3_USE_SSL",
	"GALLERY_UPLOAD_URL_TTL",
	"GALLERY_VIEW_URL_TTL",
	"GALLERY_MAX_UPLOAD_BYTES",
	"GALLERY_ALLOWED_CONTENT_TYPES",
}

// applyEnv sets every key in allEnvKeys to env[key], defaulting to "" which every
// helper treats as unset, so results are deterministic regardless of the shell.
func applyEnv(t *testing.T, env map[string]string) {
	t.Helper()
	for _, key := range allEnvKeys {
		t.Setenv(key, env[key])
	}
}

func defaultConfig() Config {
	return Config{
		Port:               "8080",
		DatabaseURL:        "postgres://runa:runa@localhost:5432/runa?sslmode=disable",
		LogLevel:           "info",
		CORSAllowedOrigins: []string{"*"},
		AppEnv:             "development",
		JWTSecret:          "dev-insecure-secret-change-me",
		AccessTokenTTL:     15 * time.Minute,
		RefreshTokenTTL:    720 * time.Hour,
		AppleClientIDs:     []string{},
		GoogleClientIDs:    []string{},
		AdminAPIToken:      "",
		S3Endpoint:         "",
		S3PublicEndpoint:   "",
		S3Region:           "us-east-1",
		S3Bucket:           "runa-gallery",
		S3AccessKey:        "",
		S3SecretKey:        "",
		S3UseSSL:           false,

		GalleryUploadURLTTL:        15 * time.Minute,
		GalleryViewURLTTL:          60 * time.Minute,
		GalleryMaxUploadBytes:      10 * 1024 * 1024,
		GalleryAllowedContentTypes: []string{"image/jpeg", "image/png", "image/webp", "image/heic"},
	}
}

func TestLoad(t *testing.T) {
	// 環境変数を扱うため並列化しない
	overridden := Config{
		Port:               "9090",
		DatabaseURL:        "postgres://custom:pass@db:5432/app?sslmode=require",
		LogLevel:           "debug",
		CORSAllowedOrigins: []string{"https://a.example", "https://b.example"},
		AppEnv:             "production",
		JWTSecret:          "super-secret",
		AccessTokenTTL:     30 * time.Minute,
		RefreshTokenTTL:    48 * time.Hour,
		AppleClientIDs:     []string{"com.example.app", "com.example.svc"},
		GoogleClientIDs:    []string{"google-client-id"},
		AdminAPIToken:      "admin-token",
		S3Endpoint:         "minio:9000",
		S3PublicEndpoint:   "localhost:9000",
		S3Region:           "ap-northeast-1",
		S3Bucket:           "my-bucket",
		S3AccessKey:        "AKIAEXAMPLE",
		S3SecretKey:        "secretexample",
		S3UseSSL:           true,

		GalleryUploadURLTTL:        5 * time.Minute,
		GalleryViewURLTTL:          2 * time.Hour,
		GalleryMaxUploadBytes:      20 * 1024 * 1024,
		GalleryAllowedContentTypes: []string{"image/jpeg", "image/gif"},
	}

	tests := []struct {
		name string
		env  map[string]string
		want Config
	}{
		{
			name: "環境変数が空ならデフォルト値",
			env:  map[string]string{},
			want: defaultConfig(),
		},
		{
			name: "全ての値を上書きしてパースする",
			env: map[string]string{
				"PORT":                          "9090",
				"DATABASE_URL":                  "postgres://custom:pass@db:5432/app?sslmode=require",
				"LOG_LEVEL":                     "debug",
				"CORS_ALLOWED_ORIGINS":          "https://a.example, https://b.example",
				"APP_ENV":                       "production",
				"JWT_SECRET":                    "super-secret",
				"ACCESS_TOKEN_TTL":              "30m",
				"REFRESH_TOKEN_TTL":             "48h",
				"APPLE_CLIENT_IDS":              "com.example.app, com.example.svc",
				"GOOGLE_CLIENT_IDS":             "google-client-id",
				"ADMIN_API_TOKEN":               "admin-token",
				"S3_ENDPOINT":                   "minio:9000",
				"S3_PUBLIC_ENDPOINT":            "localhost:9000",
				"S3_REGION":                     "ap-northeast-1",
				"S3_BUCKET":                     "my-bucket",
				"S3_ACCESS_KEY":                 "AKIAEXAMPLE",
				"S3_SECRET_KEY":                 "secretexample",
				"S3_USE_SSL":                    "true",
				"GALLERY_UPLOAD_URL_TTL":        "5m",
				"GALLERY_VIEW_URL_TTL":          "2h",
				"GALLERY_MAX_UPLOAD_BYTES":      "20971520",
				"GALLERY_ALLOWED_CONTENT_TYPES": "image/jpeg, image/gif",
			},
			want: overridden,
		},
		{
			name: "型が不正な値はデフォルトにフォールバックする",
			env: map[string]string{
				"S3_USE_SSL":               "notabool",
				"GALLERY_MAX_UPLOAD_BYTES": "not-a-number",
				"ACCESS_TOKEN_TTL":         "15minutes",
				"REFRESH_TOKEN_TTL":        "",
			},
			want: defaultConfig(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 環境変数を扱うため並列化しない
			applyEnv(t, tt.env)
			assert.Equal(t, tt.want, Load())
		})
	}
}

func TestGetenv(t *testing.T) {
	// 環境変数を扱うため並列化しない
	const key = "CONFIG_TEST_GETENV"

	tests := []struct {
		name     string
		set      bool
		value    string
		fallback string
		want     string
	}{
		{
			name:     "非空の値が設定されていれば値を返す",
			set:      true,
			value:    "hello",
			fallback: "fb",
			want:     "hello",
		},
		{
			name:     "空文字が設定されていればフォールバックする",
			set:      true,
			value:    "",
			fallback: "fb",
			want:     "fb",
		},
		{
			name:     "未設定ならフォールバックする",
			set:      false,
			value:    "",
			fallback: "fb",
			want:     "fb",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 環境変数を扱うため並列化しない
			if tt.set {
				t.Setenv(key, tt.value)
			}
			assert.Equal(t, tt.want, getenv(key, tt.fallback))
		})
	}
}

func TestGetbool(t *testing.T) {
	// 環境変数を扱うため並列化しない
	const key = "CONFIG_TEST_GETBOOL"

	tests := []struct {
		name     string
		set      bool
		value    string
		fallback bool
		want     bool
	}{
		{
			name:     "文字列trueはtrue",
			set:      true,
			value:    "true",
			fallback: false,
			want:     true,
		},
		{
			name:     "文字列falseはfalse",
			set:      true,
			value:    "false",
			fallback: true,
			want:     false,
		},
		{
			name:     "1はtrue",
			set:      true,
			value:    "1",
			fallback: false,
			want:     true,
		},
		{
			name:     "0はfalse",
			set:      true,
			value:    "0",
			fallback: true,
			want:     false,
		},
		{
			name:     "不正な値はフォールバックしてtrue",
			set:      true,
			value:    "notabool",
			fallback: true,
			want:     true,
		},
		{
			name:     "不正な値はフォールバックしてfalse",
			set:      true,
			value:    "xyz",
			fallback: false,
			want:     false,
		},
		{
			name:     "空文字はフォールバックする",
			set:      true,
			value:    "",
			fallback: true,
			want:     true,
		},
		{
			name:     "未設定ならフォールバックする",
			set:      false,
			value:    "",
			fallback: true,
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 環境変数を扱うため並列化しない
			if tt.set {
				t.Setenv(key, tt.value)
			}
			assert.Equal(t, tt.want, getbool(key, tt.fallback))
		})
	}
}

func TestGetint64(t *testing.T) {
	// 環境変数を扱うため並列化しない
	const key = "CONFIG_TEST_GETINT64"

	tests := []struct {
		name     string
		set      bool
		value    string
		fallback int64
		want     int64
	}{
		{
			name:     "正の整数",
			set:      true,
			value:    "42",
			fallback: 0,
			want:     42,
		},
		{
			name:     "負の整数",
			set:      true,
			value:    "-5",
			fallback: 0,
			want:     -5,
		},
		{
			name:     "ゼロ",
			set:      true,
			value:    "0",
			fallback: 99,
			want:     0,
		},
		{
			name:     "int64の最大値境界",
			set:      true,
			value:    "9223372036854775807",
			fallback: 0,
			want:     9223372036854775807,
		},
		{
			name:     "不正な値はフォールバックする",
			set:      true,
			value:    "abc",
			fallback: 7,
			want:     7,
		},
		{
			name:     "オーバーフローはフォールバックする",
			set:      true,
			value:    "9223372036854775808",
			fallback: 7,
			want:     7,
		},
		{
			name:     "空文字はフォールバックする",
			set:      true,
			value:    "",
			fallback: 7,
			want:     7,
		},
		{
			name:     "未設定ならフォールバックする",
			set:      false,
			value:    "",
			fallback: 7,
			want:     7,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 環境変数を扱うため並列化しない
			if tt.set {
				t.Setenv(key, tt.value)
			}
			assert.Equal(t, tt.want, getint64(key, tt.fallback))
		})
	}
}

func TestGetduration(t *testing.T) {
	// 環境変数を扱うため並列化しない
	const key = "CONFIG_TEST_GETDURATION"

	tests := []struct {
		name     string
		set      bool
		value    string
		fallback time.Duration
		want     time.Duration
	}{
		{
			name:     "分",
			set:      true,
			value:    "15m",
			fallback: time.Second,
			want:     15 * time.Minute,
		},
		{
			name:     "時間",
			set:      true,
			value:    "720h",
			fallback: time.Second,
			want:     720 * time.Hour,
		},
		{
			name:     "複合表記",
			set:      true,
			value:    "1h30m",
			fallback: time.Second,
			want:     90 * time.Minute,
		},
		{
			name:     "不正な文字列はフォールバックする",
			set:      true,
			value:    "abc",
			fallback: 5 * time.Minute,
			want:     5 * time.Minute,
		},
		{
			name:     "単位なしはフォールバックする",
			set:      true,
			value:    "15",
			fallback: 5 * time.Minute,
			want:     5 * time.Minute,
		},
		{
			name:     "空文字はフォールバックする",
			set:      true,
			value:    "",
			fallback: 5 * time.Minute,
			want:     5 * time.Minute,
		},
		{
			name:     "未設定ならフォールバックする",
			set:      false,
			value:    "",
			fallback: 5 * time.Minute,
			want:     5 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 環境変数を扱うため並列化しない
			if tt.set {
				t.Setenv(key, tt.value)
			}
			assert.Equal(t, tt.want, getduration(key, tt.fallback))
		})
	}
}

func TestSplitList(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		raw  string
		want []string
	}{
		{
			name: "カンマ区切り",
			raw:  "a,b,c",
			want: []string{"a", "b", "c"},
		},
		{
			name: "前後の空白を除去する",
			raw:  " a , b ",
			want: []string{"a", "b"},
		},
		{
			name: "空要素をスキップする",
			raw:  "a,,b",
			want: []string{"a", "b"},
		},
		{
			name: "単一の値",
			raw:  "single",
			want: []string{"single"},
		},
		{
			name: "空文字は空スライス",
			raw:  "",
			want: []string{},
		},
		{
			name: "区切りと空白のみは空スライス",
			raw:  " , , ",
			want: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, splitList(tt.raw))
		})
	}
}

func TestSplitListDefault(t *testing.T) {
	// 環境変数を扱うため並列化しない
	const key = "CONFIG_TEST_SPLITLISTDEFAULT"
	fallback := []string{"default-a", "default-b"}

	tests := []struct {
		name  string
		set   bool
		value string
		want  []string
	}{
		{
			name:  "設定値をパースする",
			set:   true,
			value: "x,y",
			want:  []string{"x", "y"},
		},
		{
			name:  "空白除去と空要素スキップ",
			set:   true,
			value: " x , , y ",
			want:  []string{"x", "y"},
		},
		{
			name:  "空文字はフォールバックする",
			set:   true,
			value: "",
			want:  fallback,
		},
		{
			name:  "区切りのみはフォールバックする",
			set:   true,
			value: " , ",
			want:  fallback,
		},
		{
			name:  "未設定ならフォールバックする",
			set:   false,
			value: "",
			want:  fallback,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 環境変数を扱うため並列化しない
			if tt.set {
				t.Setenv(key, tt.value)
			}
			assert.Equal(t, tt.want, splitListDefault(key, fallback))
		})
	}
}

func TestSplitOrigins(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		raw  string
		want []string
	}{
		{
			name: "カンマ区切り",
			raw:  "http://a, http://b",
			want: []string{"http://a", "http://b"},
		},
		{
			name: "単一のオリジン",
			raw:  "single",
			want: []string{"single"},
		},
		{
			name: "空要素をスキップする",
			raw:  "a,,b",
			want: []string{"a", "b"},
		},
		{
			name: "空文字はワイルドカード",
			raw:  "",
			want: []string{"*"},
		},
		{
			name: "区切りのみはワイルドカード",
			raw:  " , ",
			want: []string{"*"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, splitOrigins(tt.raw))
		})
	}
}
