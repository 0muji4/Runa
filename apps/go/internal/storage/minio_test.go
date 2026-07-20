package storage

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func assertURL(t *testing.T, got string, wantContains, wantAbsent []string) {
	t.Helper()
	for _, want := range wantContains {
		assert.Contains(t, got, want)
	}
	for _, absent := range wantAbsent {
		assert.NotContains(t, got, absent)
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
			require.NoError(t, err)
			if tt.wantNil {
				assert.Nil(t, store)
				return
			}
			require.NotNil(t, store)
			assert.Equal(t, tt.wantBucket, store.bucket)
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
			require.NoError(t, err)
			require.NotNil(t, store)

			got, err := store.PresignPut(context.Background(), tt.key, tt.ttl)
			require.NoError(t, err)
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
			require.NoError(t, err)
			require.NotNil(t, store)

			got, err := store.PresignGet(context.Background(), tt.key, tt.ttl)
			require.NoError(t, err)
			assertURL(t, got, tt.wantContains, tt.wantAbsent)
		})
	}
}
