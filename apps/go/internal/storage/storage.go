// Package storage is the object-storage seam: presigned URLs in and out, never image bytes.
package storage

import (
	"context"
	"errors"
	"time"
)

// ErrObjectNotFound is returned by Stat when the key is absent.
var ErrObjectNotFound = errors.New("storage: object not found")

// ObjectInfo is the stored-object metadata the service verifies at registration time.
type ObjectInfo struct {
	Size        int64
	ContentType string
}

// ObjectStore is the object-storage boundary. Presign* are pure HMAC signing (no network
// I/O) against the client-reachable endpoint; the other methods hit the internal endpoint.
type ObjectStore interface {
	// EnsureBucket creates the bucket if it does not exist.
	EnsureBucket(ctx context.Context) error

	// PresignPut returns a URL the client PUTs raw bytes to. The signature does not
	// bind content-type or size, so the real object must be re-verified with Stat.
	PresignPut(ctx context.Context, key string, ttl time.Duration) (string, error)

	// PresignGet returns a time-limited URL the client GETs the object from.
	PresignGet(ctx context.Context, key string, ttl time.Duration) (string, error)

	// Stat returns the stored object's size/content-type, or ErrObjectNotFound.
	Stat(ctx context.Context, key string) (ObjectInfo, error)

	// Remove deletes the object; a missing object is not an error (idempotent).
	Remove(ctx context.Context, key string) error
}
