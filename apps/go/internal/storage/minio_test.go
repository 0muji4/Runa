package storage

import (
	"context"
	"strings"
	"testing"
	"time"
)

// TestNewMinioObjectStoreDisabled confirms an empty endpoint disables storage so
// the server can boot with the gallery URL endpoints returning 503.
func TestNewMinioObjectStoreDisabled(t *testing.T) {
	store, err := NewMinioObjectStore(Config{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if store != nil {
		t.Fatalf("store = %v, want nil when endpoint is empty", store)
	}
}

// TestPresignUsesPublicEndpoint proves presigning is offline (no network) and that
// URLs are built with the client-reachable PUBLIC endpoint, not the internal one.
// This is the crux of the two-endpoint scheme: the server signs URLs for a host
// it may not itself be able to reach.
func TestPresignUsesPublicEndpoint(t *testing.T) {
	store, err := NewMinioObjectStore(Config{
		Endpoint:       "minio:9000",     // internal (unreachable from tests)
		PublicEndpoint: "localhost:9000", // client-reachable
		Region:         "us-east-1",
		Bucket:         "runa-gallery",
		AccessKey:      "runa",
		SecretKey:      "runa-secret",
		UseSSL:         false,
	})
	if err != nil {
		t.Fatalf("construct store: %v", err)
	}

	const key = "gallery/user-1/abc"
	ctx := context.Background()

	putURL, err := store.PresignPut(ctx, key, 15*time.Minute)
	if err != nil {
		t.Fatalf("presign put: %v", err)
	}
	getURL, err := store.PresignGet(ctx, key, time.Hour)
	if err != nil {
		t.Fatalf("presign get: %v", err)
	}

	for name, u := range map[string]string{"put": putURL, "get": getURL} {
		if !strings.Contains(u, "localhost:9000") {
			t.Errorf("%s URL %q does not use the public endpoint", name, u)
		}
		if strings.Contains(u, "minio:9000") {
			t.Errorf("%s URL %q leaked the internal endpoint", name, u)
		}
		if !strings.Contains(u, "runa-gallery") || !strings.Contains(u, "gallery/user-1/abc") {
			t.Errorf("%s URL %q missing bucket/key", name, u)
		}
		if !strings.Contains(u, "X-Amz-Signature=") {
			t.Errorf("%s URL %q is not a signed URL", name, u)
		}
	}
}
