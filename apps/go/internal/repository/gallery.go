package repository

import (
	"context"
	"time"
)

// GalleryImage is the persistence model for the gallery_images table (migration
// 0005). It holds image metadata only — the bytes live in object storage keyed
// by ObjectKey.
type GalleryImage struct {
	ID        string
	UserID    string
	ObjectKey string
	Width     int
	Height    int
	Theme     string
	CreatedAt time.Time
	DeletedAt *time.Time
}

// InsertGalleryParams carries the fields for an idempotent registration keyed by
// ObjectKey. ObjectKey is server-generated ("gallery/{user}/{uuid}") and unique,
// so a retried POST /gallery upserts the same row.
type InsertGalleryParams struct {
	UserID    string
	ObjectKey string
	Width     int
	Height    int
	Theme     string
}

// ListGalleryParams is a keyset page request: images strictly older than the
// (CreatedAt, ID) cursor, newest first, capped at Limit. A nil Cursor starts at
// the newest image.
type ListGalleryParams struct {
	UserID string
	Limit  int
	Cursor *GalleryCursor
}

// GalleryCursor is the opaque page boundary: the (created_at, id) of the last row
// of the previous page. The handler encodes/decodes it to a string.
type GalleryCursor struct {
	CreatedAt time.Time
	ID        string
}

// GalleryStore is the data-access boundary for the gallery feature. Like
// DiaryStore every method is user-scoped: a query for another user's row returns
// ErrNotFound so ownership is enforced at the data layer.
type GalleryStore interface {
	// InsertImage registers image metadata, upserting on object_key so a retried
	// offline registration never duplicates.
	InsertImage(ctx context.Context, p InsertGalleryParams) (GalleryImage, error)

	// ListImages returns one keyset page of non-deleted images, newest first.
	ListImages(ctx context.Context, p ListGalleryParams) ([]GalleryImage, error)

	// GetImage returns a single non-deleted image owned by userID, or ErrNotFound.
	GetImage(ctx context.Context, userID, id string) (GalleryImage, error)

	// SoftDeleteImage sets deleted_at on an owned image and returns its object_key
	// so the caller can remove the stored object. Returns ErrNotFound when the id
	// is not the caller's. Idempotent for an already-deleted own image.
	SoftDeleteImage(ctx context.Context, userID, id string) (objectKey string, err error)

	// ListObjectKeys returns every object_key the user has, INCLUDING
	// soft-deleted rows, so account deletion can purge every stored object the
	// user ever registered — not only the currently-visible ones. Ordering is
	// unspecified.
	ListObjectKeys(ctx context.Context, userID string) ([]string, error)
}
