// Package storage is the object-storage seam for image features. The backend
// never streams image bytes itself: it issues short-lived presigned URLs so the
// client uploads/downloads directly against an S3-compatible store (MinIO in
// local dev). This ObjectStore interface is the reusable "type" every future
// image feature (avatars, diary attachments, ...) depends on, so the concrete
// backend (MinIO/S3) and its two-endpoint presign quirk stay in one place and
// tests can substitute an in-memory fake.
package storage

import (
	"context"
	"errors"
	"time"
)

// ErrObjectNotFound is returned by Stat/Remove when the key is absent. It lets
// the service map a missing upload to a 400 (client registered metadata for an
// object it never actually PUT) rather than a 500.
var ErrObjectNotFound = errors.New("storage: object not found")

// ObjectInfo is the subset of stored-object metadata the service verifies at
// registration time (the presigned PUT does not itself enforce size/type, so the
// real object is re-checked here).
type ObjectInfo struct {
	Size        int64
	ContentType string
}

// ObjectStore is the object-storage boundary. Presign* build a URL by pure HMAC
// signing (no network I/O), so they can be pointed at a client-reachable public
// endpoint even from inside a container that talks to the store on a different
// internal host. Stat/Remove/EnsureBucket perform real requests against the
// internal endpoint.
type ObjectStore interface {
	// EnsureBucket creates the gallery bucket if it does not exist. Best-effort
	// at boot; a failure here does not stop the server from serving liveness.
	EnsureBucket(ctx context.Context) error

	// PresignPut returns a URL the client PUTs raw bytes to. The signature binds
	// the method + key + expiry only (not content-type), so the real object is
	// re-verified with Stat at registration.
	PresignPut(ctx context.Context, key string, ttl time.Duration) (string, error)

	// PresignGet returns a time-limited URL the client GETs the object from.
	PresignGet(ctx context.Context, key string, ttl time.Duration) (string, error)

	// Stat returns the stored object's size/content-type, or ErrObjectNotFound.
	Stat(ctx context.Context, key string) (ObjectInfo, error)

	// Remove deletes the object. A missing object is not an error (idempotent).
	Remove(ctx context.Context, key string) error
}
