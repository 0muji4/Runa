// Package memobject is an in-memory storage.ObjectStore for tests. It signs
// deterministic presigned URLs with no network I/O and records removals, so a test
// can seed an object with Put, drive the service, and assert what was removed with
// Removed/RemovedKeys.
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

// Compile-time check that Store satisfies the boundary the service depends on.
var _ storage.ObjectStore = (*Store)(nil)

// EnsureBucket is a no-op; there is no bucket to create in memory.
func (s *Store) EnsureBucket(context.Context) error { return nil }

// PresignPut returns a deterministic, non-network URL for the upload of key.
func (s *Store) PresignPut(_ context.Context, key string, _ time.Duration) (string, error) {
	return "https://objects.test/" + key + "?op=put", nil
}

// PresignGet returns a deterministic, non-network URL for the download of key.
func (s *Store) PresignGet(_ context.Context, key string, _ time.Duration) (string, error) {
	return "https://objects.test/" + key + "?op=get", nil
}

// Stat returns the seeded object's metadata, or ErrObjectNotFound if the client
// never PUT it (mirrors the real store: a presigned PUT does not create the
// object, so registration re-verifies it here).
func (s *Store) Stat(_ context.Context, key string) (storage.ObjectInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if info, ok := s.objects[key]; ok {
		return info, nil
	}
	return storage.ObjectInfo{}, storage.ErrObjectNotFound
}

// Remove records the key and drops it. Removing a missing key is not an error
// (idempotent), matching the real store.
func (s *Store) Remove(_ context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.removed = append(s.removed, key)
	delete(s.objects, key)
	return nil
}

// Put seeds an object as if the client had uploaded it directly to storage.
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
