// Package memobject is an in-memory storage.ObjectStore for tests.
package memobject

import (
	"context"
	"sync"
	"time"

	"github.com/0muji4/Runa/apps/go/internal/storage"
)

// Store is an in-memory ObjectStore. The zero value is not usable; call New.
type Store struct {
	mu      sync.Mutex
	objects map[string]storage.ObjectInfo
	removed []string
}

// New returns an empty in-memory object store.
func New() *Store {
	return &Store{objects: make(map[string]storage.ObjectInfo)}
}

var _ storage.ObjectStore = (*Store)(nil)

// EnsureBucket is a no-op.
func (s *Store) EnsureBucket(context.Context) error { return nil }

// PresignPut returns a deterministic URL for the upload of key.
func (s *Store) PresignPut(_ context.Context, key string, _ time.Duration) (string, error) {
	return "https://objects.test/" + key + "?op=put", nil
}

// PresignGet returns a deterministic URL for the download of key.
func (s *Store) PresignGet(_ context.Context, key string, _ time.Duration) (string, error) {
	return "https://objects.test/" + key + "?op=get", nil
}

// Stat returns the seeded object's metadata, or ErrObjectNotFound when Put was never called.
func (s *Store) Stat(_ context.Context, key string) (storage.ObjectInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if info, ok := s.objects[key]; ok {
		return info, nil
	}
	return storage.ObjectInfo{}, storage.ErrObjectNotFound
}

// Remove records the key and drops it; a missing key is not an error.
func (s *Store) Remove(_ context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.removed = append(s.removed, key)
	delete(s.objects, key)
	return nil
}

// Put seeds an object as if the client had uploaded it.
func (s *Store) Put(key string, info storage.ObjectInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.objects[key] = info
}

// Removed reports whether Remove was ever called for key.
func (s *Store) Removed(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, k := range s.removed {
		if k == key {
			return true
		}
	}
	return false
}

// RemovedKeys returns every key passed to Remove, in call order.
func (s *Store) RemovedKeys() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]string, len(s.removed))
	copy(out, s.removed)
	return out
}
