package storage

import (
	"strings"
	"testing"
	"time"
)

// baseConfig is the two-endpoint docker setup: the server reaches the store on
// Endpoint (unreachable from tests) while presigned URLs must target the
// client-reachable PublicEndpoint.
func baseConfig() Config {
	return Config{
		Endpoint:       "minio:9000",
		PublicEndpoint: "localhost:9000",
		Region:         "us-east-1",
		Bucket:         "runa-gallery",
		AccessKey:      "runa",
		SecretKey:      "runa-secret",
		UseSSL:         false,
	}
}

// assertURL checks a presigned URL for the substrings that must (and must not)
// appear in it, reporting the whole URL on failure so the mismatch is diagnosable
// without re-running.
func assertURL(t *testing.T, got string, wantContains, wantAbsent []string) {
	t.Helper()
	for _, want := range wantContains {
		if !strings.Contains(got, want) {
			t.Errorf("presigned URL = %q, want it to contain %q", got, want)
		}
	}
	for _, absent := range wantAbsent {
		if strings.Contains(got, absent) {
			t.Errorf("presigned URL = %q, want it NOT to contain %q", got, absent)
		}
	}
}

func TestNewMinioObjectStore(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		cfg        Config
		wantNil    bool
		wantBucket string
	}{
		{
			name:       "空のエンドポイントはストレージ無効",
			cfg:        Config{},
			wantNil:    true,
			wantBucket: "",
		},
		{
			name:       "エンドポイント指定でストアを構築する",
			cfg:        baseConfig(),
			wantNil:    false,
			wantBucket: "runa-gallery",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			store, err := NewMinioObjectStore(tt.cfg)
			if err != nil {
				t.Fatalf("NewMinioObjectStore(%+v) error = %v, want nil", tt.cfg, err)
			}
			if tt.wantNil {
				if store != nil {
					t.Errorf("NewMinioObjectStore(%+v) = %v, want nil", tt.cfg, store)
				}
				return
			}
			if store == nil {
				t.Fatalf("NewMinioObjectStore(%+v) = nil, want a store", tt.cfg)
			}
			if store.bucket != tt.wantBucket {
				t.Errorf("NewMinioObjectStore(%+v).bucket = %q, want %q",
					tt.cfg, store.bucket, tt.wantBucket)
			}
		})
	}
}

func TestMinioObjectStore_PresignPut(t *testing.T) {
	t.Parallel()

	fallback := baseConfig()
	fallback.PublicEndpoint = "" // must fall back to Endpoint

	tests := []struct {
		name         string
		cfg          Config
		key          string
		ttl          time.Duration
		wantContains []string
		wantAbsent   []string
	}{
		{
			name:         "内部でなく公開エンドポイントを使う",
			cfg:          baseConfig(),
			key:          "gallery/user-1/abc",
			ttl:          15 * time.Minute,
			wantContains: []string{"localhost:9000", "runa-gallery", "gallery/user-1/abc", "X-Amz-Signature="},
			wantAbsent:   []string{"minio:9000"},
		},
		{
			name:         "公開エンドポイント未指定は内部にフォールバックする",
			cfg:          fallback,
			key:          "gallery/user-1/abc",
			ttl:          15 * time.Minute,
			wantContains: []string{"minio:9000", "runa-gallery", "gallery/user-1/abc", "X-Amz-Signature="},
			wantAbsent:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			store, err := NewMinioObjectStore(tt.cfg)
			if err != nil {
				t.Fatalf("NewMinioObjectStore(%+v) error = %v, want nil", tt.cfg, err)
			}
			if store == nil {
				t.Fatalf("NewMinioObjectStore(%+v) = nil, want a store", tt.cfg)
			}

			got, err := store.PresignPut(t.Context(), tt.key, tt.ttl)
			if err != nil {
				t.Fatalf("PresignPut(%q, %s) error = %v, want nil", tt.key, tt.ttl, err)
			}
			assertURL(t, got, tt.wantContains, tt.wantAbsent)
		})
	}
}

func TestMinioObjectStore_PresignGet(t *testing.T) {
	t.Parallel()

	fallback := baseConfig()
	fallback.PublicEndpoint = "" // must fall back to Endpoint

	tests := []struct {
		name         string
		cfg          Config
		key          string
		ttl          time.Duration
		wantContains []string
		wantAbsent   []string
	}{
		{
			name:         "内部でなく公開エンドポイントを使う",
			cfg:          baseConfig(),
			key:          "gallery/user-1/abc",
			ttl:          time.Hour,
			wantContains: []string{"localhost:9000", "runa-gallery", "gallery/user-1/abc", "X-Amz-Signature="},
			wantAbsent:   []string{"minio:9000"},
		},
		{
			name:         "公開エンドポイント未指定は内部にフォールバックする",
			cfg:          fallback,
			key:          "gallery/user-1/abc",
			ttl:          time.Hour,
			wantContains: []string{"minio:9000", "runa-gallery", "gallery/user-1/abc", "X-Amz-Signature="},
			wantAbsent:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			store, err := NewMinioObjectStore(tt.cfg)
			if err != nil {
				t.Fatalf("NewMinioObjectStore(%+v) error = %v, want nil", tt.cfg, err)
			}
			if store == nil {
				t.Fatalf("NewMinioObjectStore(%+v) = nil, want a store", tt.cfg)
			}

			got, err := store.PresignGet(t.Context(), tt.key, tt.ttl)
			if err != nil {
				t.Fatalf("PresignGet(%q, %s) error = %v, want nil", tt.key, tt.ttl, err)
			}
			assertURL(t, got, tt.wantContains, tt.wantAbsent)
		})
	}
}
