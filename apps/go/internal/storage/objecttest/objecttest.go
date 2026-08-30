// Package objecttest holds the contract test suite for storage.ObjectStore. It
// runs against both implementations: memobject in its own package, and a real
// MinIO from internal/storage's test binary.
package objecttest

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/0muji4/Runa/apps/go/internal/storage"
)

// NewStore builds an ObjectStore with its bucket already ensured.
type NewStore func(t *testing.T) storage.ObjectStore

// RunObjectStoreSuite exercises the ObjectStore contract.
func RunObjectStoreSuite(t *testing.T, newStore NewStore) {
	t.Run("PutStatRoundTrip", func(t *testing.T) {
		t.Parallel()
		s := newStore(t)
		ctx := t.Context()
		key := "gallery/user-1/round-trip"

		putObject(t, s, key, []byte("hello moon"), "image/jpeg")

		info, err := s.Stat(ctx, key)
		if err != nil {
			t.Fatalf("Stat(%q) error = %v, want nil", key, err)
		}
		// The service re-checks size and content type at registration time,
		// because the presigned PUT does not enforce either.
		if info.Size != int64(len("hello moon")) {
			t.Errorf("Stat(%q) size = %d, want %d", key, info.Size, len("hello moon"))
		}
		if info.ContentType != "image/jpeg" {
			t.Errorf("Stat(%q) content type = %q, want %q", key, info.ContentType, "image/jpeg")
		}
	})

	t.Run("StatMissingObject", func(t *testing.T) {
		t.Parallel()
		s := newStore(t)

		// The service maps this to a 400: metadata registered for bytes that were
		// never uploaded.
		if _, err := s.Stat(t.Context(), "gallery/user-1/never-uploaded"); !errors.Is(err, storage.ErrObjectNotFound) {
			t.Errorf("Stat(missing) error = %v, want %v", err, storage.ErrObjectNotFound)
		}
	})

	t.Run("RemoveIsIdempotent", func(t *testing.T) {
		t.Parallel()
		s := newStore(t)
		ctx := t.Context()
		key := "gallery/user-1/removable"

		putObject(t, s, key, []byte("bytes"), "image/png")
		if err := s.Remove(ctx, key); err != nil {
			t.Fatalf("Remove(%q) error = %v, want nil", key, err)
		}
		if _, err := s.Stat(ctx, key); !errors.Is(err, storage.ErrObjectNotFound) {
			t.Errorf("Stat() after Remove error = %v, want %v", err, storage.ErrObjectNotFound)
		}
		// Purging runs in the background and may be retried.
		if err := s.Remove(ctx, key); err != nil {
			t.Errorf("second Remove(%q) error = %v, want nil", key, err)
		}
		if err := s.Remove(ctx, "gallery/user-1/never-existed"); err != nil {
			t.Errorf("Remove(missing) error = %v, want nil", err)
		}
	})

	t.Run("EnsureBucketIsIdempotent", func(t *testing.T) {
		t.Parallel()
		s := newStore(t)

		// It runs at every boot.
		if err := s.EnsureBucket(t.Context()); err != nil {
			t.Errorf("EnsureBucket() on an existing bucket error = %v, want nil", err)
		}
	})

	t.Run("PresignedURLsAreWellFormed", func(t *testing.T) {
		t.Parallel()
		s := newStore(t)
		ctx := t.Context()
		key := "gallery/user-1/presigned"

		put, err := s.PresignPut(ctx, key, 15*time.Minute)
		if err != nil {
			t.Fatalf("PresignPut(%q) error = %v, want nil", key, err)
		}
		get, err := s.PresignGet(ctx, key, time.Hour)
		if err != nil {
			t.Fatalf("PresignGet(%q) error = %v, want nil", key, err)
		}
		for name, u := range map[string]string{"PresignPut": put, "PresignGet": get} {
			if u == "" {
				t.Errorf("%s(%q) returned an empty URL", name, key)
			}
		}
	})
}

// putObject stores bytes through whichever write path the backend offers.
func putObject(t *testing.T, s storage.ObjectStore, key string, body []byte, contentType string) {
	t.Helper()
	w, ok := s.(Writer)
	if !ok {
		t.Fatalf("%T does not implement objecttest.Writer; the suite cannot seed objects", s)
	}
	if err := w.PutForTest(t.Context(), key, body, contentType); err != nil {
		t.Fatalf("seeding object %q: %v", key, err)
	}
}

// Writer is the seam the suite uses to create objects. ObjectStore has no write
// method because production never uploads bytes: clients PUT to a presigned URL.
type Writer interface {
	PutForTest(ctx context.Context, key string, body []byte, contentType string) error
}
