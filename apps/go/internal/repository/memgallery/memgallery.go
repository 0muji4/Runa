// Package memgallery is an in-memory implementation of repository.GalleryStore.
// It backs the gallery unit/integration tests (and lets the API run without
// Postgres) so the suite stays green in CI, which has no database. It mirrors the
// pgx implementation's semantics: idempotent upsert by object_key, keyset
// pagination, ownership scoping and soft delete.
package memgallery

import (
	"context"
	"crypto/rand"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/0muji4/Runa/apps/go/internal/repository"
)

// Store is a goroutine-safe, in-memory GalleryStore.
type Store struct {
	mu      sync.Mutex
	images  map[string]repository.GalleryImage // keyed by image id
	now     func() time.Time
	lastNow time.Time
}

// New returns an empty in-memory store.
func New() *Store {
	return &Store{
		images: make(map[string]repository.GalleryImage),
		now:    time.Now,
	}
}

var _ repository.GalleryStore = (*Store)(nil)

func (s *Store) InsertImage(_ context.Context, p repository.InsertGalleryParams) (repository.GalleryImage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if existing, ok := s.findByObjectKey(p.ObjectKey); ok {
		// Upsert in place, keeping id/created_at and reviving (matches ON CONFLICT).
		existing.Width = p.Width
		existing.Height = p.Height
		existing.Theme = p.Theme
		existing.DeletedAt = nil
		s.images[existing.ID] = existing
		return existing, nil
	}

	img := repository.GalleryImage{
		ID:        newID(),
		UserID:    p.UserID,
		ObjectKey: p.ObjectKey,
		Width:     p.Width,
		Height:    p.Height,
		Theme:     p.Theme,
		CreatedAt: s.tick(),
	}
	s.images[img.ID] = img
	return img, nil
}

func (s *Store) ListImages(_ context.Context, p repository.ListGalleryParams) ([]repository.GalleryImage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var out []repository.GalleryImage
	for _, img := range s.images {
		if img.UserID != p.UserID || img.DeletedAt != nil {
			continue
		}
		if p.Cursor != nil && !olderThanCursor(img, *p.Cursor) {
			continue
		}
		out = append(out, img)
	}
	// Newest first: created_at DESC, then id DESC as a stable tiebreak.
	sort.Slice(out, func(i, j int) bool {
		if !out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].CreatedAt.After(out[j].CreatedAt)
		}
		return out[i].ID > out[j].ID
	})
	if p.Limit > 0 && len(out) > p.Limit {
		out = out[:p.Limit]
	}
	return out, nil
}

func (s *Store) GetImage(_ context.Context, userID, id string) (repository.GalleryImage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if img, ok := s.images[id]; ok && img.UserID == userID && img.DeletedAt == nil {
		return img, nil
	}
	return repository.GalleryImage{}, repository.ErrNotFound
}

func (s *Store) SoftDeleteImage(_ context.Context, userID, id string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	img, ok := s.images[id]
	if !ok || img.UserID != userID {
		return "", repository.ErrNotFound
	}
	if img.DeletedAt == nil { // idempotent: skip if already deleted
		t := s.tick()
		img.DeletedAt = &t
		s.images[id] = img
	}
	return img.ObjectKey, nil
}

func (s *Store) ListObjectKeys(_ context.Context, userID string) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Includes soft-deleted rows, mirroring the pgx query used by account deletion.
	keys := make([]string, 0)
	for _, img := range s.images {
		if img.UserID == userID {
			keys = append(keys, img.ObjectKey)
		}
	}
	return keys, nil
}

// findByObjectKey locates an image by object_key. Caller holds the lock.
func (s *Store) findByObjectKey(objectKey string) (repository.GalleryImage, bool) {
	for _, img := range s.images {
		if img.ObjectKey == objectKey {
			return img, true
		}
	}
	return repository.GalleryImage{}, false
}

// tick returns a strictly increasing timestamp so created_at values never tie,
// keeping keyset ordering deterministic in fast tests. Caller holds the lock.
func (s *Store) tick() time.Time {
	t := s.now()
	if !t.After(s.lastNow) {
		t = s.lastNow.Add(time.Nanosecond)
	}
	s.lastNow = t
	return t
}

// olderThanCursor reports whether img sorts after the cursor in (created_at DESC,
// id DESC) order, i.e. belongs on a later page.
func olderThanCursor(img repository.GalleryImage, c repository.GalleryCursor) bool {
	if !img.CreatedAt.Equal(c.CreatedAt) {
		return img.CreatedAt.Before(c.CreatedAt)
	}
	return img.ID < c.ID
}

// newID returns a random v4-style UUID string without pulling in a dependency
// (same helper shape as memdiary/memauth).
func newID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
