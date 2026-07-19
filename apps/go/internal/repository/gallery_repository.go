package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// galleryColumns is the SELECT list shared by every gallery query, kept in one
// place so the scan order in scanGalleryImage stays in sync.
const galleryColumns = `id, user_id, object_key, width, height, theme, created_at, deleted_at`

// GalleryRepository is the pgx-backed implementation of GalleryStore. Like the
// other repositories the pool may be nil when the DB is unreachable at boot;
// every method then returns ErrNoDatabase instead of panicking.
type GalleryRepository struct {
	pool *pgxpool.Pool
}

// NewGalleryRepository wraps a pgx pool.
func NewGalleryRepository(pool *pgxpool.Pool) *GalleryRepository {
	return &GalleryRepository{pool: pool}
}

var _ GalleryStore = (*GalleryRepository)(nil)

func scanGalleryImage(r row) (GalleryImage, error) {
	var img GalleryImage
	if err := r.Scan(
		&img.ID, &img.UserID, &img.ObjectKey, &img.Width, &img.Height,
		&img.Theme, &img.CreatedAt, &img.DeletedAt,
	); err != nil {
		return GalleryImage{}, err
	}
	return img, nil
}

func (r *GalleryRepository) InsertImage(ctx context.Context, p InsertGalleryParams) (GalleryImage, error) {
	if r.pool == nil {
		return GalleryImage{}, ErrNoDatabase
	}
	// ON CONFLICT upserts onto the unique object_key, so a retried offline
	// registration never duplicates. A conflicting row keeps its id/created_at
	// and revives (deleted_at → NULL) with the latest metadata.
	const q = `
		INSERT INTO gallery_images (user_id, object_key, width, height, theme)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (object_key) DO UPDATE
			SET width      = EXCLUDED.width,
			    height     = EXCLUDED.height,
			    theme      = EXCLUDED.theme,
			    deleted_at = NULL
		RETURNING ` + galleryColumns

	img, err := scanGalleryImage(r.pool.QueryRow(ctx, q, p.UserID, p.ObjectKey, p.Width, p.Height, p.Theme))
	if err != nil {
		return GalleryImage{}, fmt.Errorf("insert gallery image: %w", err)
	}
	return img, nil
}

func (r *GalleryRepository) ListImages(ctx context.Context, p ListGalleryParams) ([]GalleryImage, error) {
	if r.pool == nil {
		return nil, ErrNoDatabase
	}
	// Keyset (not OFFSET) so inserts between page fetches never shift rows.
	var (
		rows pgx.Rows
		err  error
	)
	if p.Cursor == nil {
		const q = `
			SELECT ` + galleryColumns + `
			FROM gallery_images
			WHERE user_id = $1 AND deleted_at IS NULL
			ORDER BY created_at DESC, id DESC
			LIMIT $2`
		rows, err = r.pool.Query(ctx, q, p.UserID, p.Limit)
	} else {
		const q = `
			SELECT ` + galleryColumns + `
			FROM gallery_images
			WHERE user_id = $1 AND deleted_at IS NULL
			  AND (created_at, id) < ($2, $3)
			ORDER BY created_at DESC, id DESC
			LIMIT $4`
		rows, err = r.pool.Query(ctx, q, p.UserID, p.Cursor.CreatedAt, p.Cursor.ID, p.Limit)
	}
	if err != nil {
		return nil, fmt.Errorf("list gallery images: %w", err)
	}
	return collectGalleryImages(rows)
}

func (r *GalleryRepository) GetImage(ctx context.Context, userID, id string) (GalleryImage, error) {
	if r.pool == nil {
		return GalleryImage{}, ErrNoDatabase
	}
	const q = `
		SELECT ` + galleryColumns + `
		FROM gallery_images
		WHERE user_id = $1 AND id = $2 AND deleted_at IS NULL`

	img, err := scanGalleryImage(r.pool.QueryRow(ctx, q, userID, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return GalleryImage{}, ErrNotFound
		}
		return GalleryImage{}, fmt.Errorf("get gallery image: %w", err)
	}
	return img, nil
}

func (r *GalleryRepository) SoftDeleteImage(ctx context.Context, userID, id string) (string, error) {
	if r.pool == nil {
		return "", ErrNoDatabase
	}
	// RETURNING object_key gives the caller the storage path to remove. COALESCE
	// keeps delete idempotent: an already-deleted own row still returns its key
	// (no ErrNotFound) but its deleted_at is untouched. No row → ErrNotFound.
	const q = `
		UPDATE gallery_images
		SET deleted_at = COALESCE(deleted_at, now())
		WHERE user_id = $1 AND id = $2
		RETURNING object_key`

	var objectKey string
	err := r.pool.QueryRow(ctx, q, userID, id).Scan(&objectKey)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", fmt.Errorf("soft delete gallery image: %w", err)
	}
	return objectKey, nil
}

func (r *GalleryRepository) ListObjectKeys(ctx context.Context, userID string) ([]string, error) {
	if r.pool == nil {
		return nil, ErrNoDatabase
	}
	// No deleted_at filter: account deletion purges every object the user ever
	// stored, including rows already soft-deleted (whose background object cleanup
	// may have failed).
	const q = `SELECT object_key FROM gallery_images WHERE user_id = $1`
	rows, err := r.pool.Query(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("list object keys: %w", err)
	}
	defer rows.Close()

	keys := make([]string, 0)
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, fmt.Errorf("scan object key: %w", err)
		}
		keys = append(keys, key)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate object keys: %w", err)
	}
	return keys, nil
}

// collectGalleryImages scans and closes a gallery result set. It returns a
// non-nil empty slice for zero rows so JSON encodes "[]" rather than "null".
func collectGalleryImages(rows pgx.Rows) ([]GalleryImage, error) {
	defer rows.Close()
	images := make([]GalleryImage, 0)
	for rows.Next() {
		img, err := scanGalleryImage(rows)
		if err != nil {
			return nil, fmt.Errorf("scan gallery image: %w", err)
		}
		images = append(images, img)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate gallery images: %w", err)
	}
	return images, nil
}
