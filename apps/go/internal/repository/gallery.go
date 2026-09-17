package repository

import (
	"context"
	"time"
)

// GalleryImage is the persistence model for the gallery_images table; the bytes
// live in object storage keyed by ObjectKey.
type GalleryImage struct {
	ID        string
	UserID    string
	ObjectKey string
	Width     int
	Height    int
	CreatedAt time.Time
	DeletedAt *time.Time
}

// InsertGalleryParams carries the fields for an idempotent registration keyed by ObjectKey.
type InsertGalleryParams struct {
	UserID    string
	ObjectKey string
	Width     int
	Height    int
}

// ListGalleryParams is a keyset page request: images strictly older than the
// cursor, newest first, capped at Limit; a nil Cursor starts at the newest.
type ListGalleryParams struct {
	UserID string
	Limit  int
	Cursor *GalleryCursor
}

// GalleryCursor is the (created_at, id) of the last row of the previous page.
type GalleryCursor struct {
	CreatedAt time.Time
	ID        string
}

// GalleryStore is the data-access boundary for the gallery feature; a query for
// another user's row returns ErrNotFound.
type GalleryStore interface {
	// InsertImage registers image metadata, upserting on object_key.
	InsertImage(ctx context.Context, p InsertGalleryParams) (GalleryImage, error)

	// ListImages returns one keyset page of non-deleted images, newest first.
	ListImages(ctx context.Context, p ListGalleryParams) ([]GalleryImage, error)

	// GetImage returns a single non-deleted image owned by userID, or ErrNotFound.
	GetImage(ctx context.Context, userID, id string) (GalleryImage, error)

	// SoftDeleteImage sets deleted_at on an owned image and returns its object_key;
	// idempotent for an already-deleted own image, ErrNotFound when not the caller's.
	SoftDeleteImage(ctx context.Context, userID, id string) (objectKey string, err error)

	// ListObjectKeys returns every object_key the user has, INCLUDING soft-deleted
	// rows, in unspecified order.
	ListObjectKeys(ctx context.Context, userID string) ([]string, error)
}
