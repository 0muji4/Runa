// Package memdevices is an in-memory implementation of repository.DeviceStore. It
// backs the devices unit/integration tests (and lets the API run without
// Postgres) so the suite stays green in CI, which has no database. It mirrors the
// pgx implementation's semantics: idempotent upsert by (user_id, push_token).
package memdevices

import (
	"context"
	"crypto/rand"
	"fmt"
	"sync"
	"time"

	"github.com/0muji4/Runa/apps/go/internal/repository"
)

// Store is a goroutine-safe, in-memory DeviceStore.
type Store struct {
	mu      sync.Mutex
	devices map[string]repository.Device // keyed by device id
	now     func() time.Time
	lastNow time.Time
}

// New returns an empty in-memory store.
func New() *Store {
	return &Store{
		devices: make(map[string]repository.Device),
		now:     time.Now,
	}
}

var _ repository.DeviceStore = (*Store)(nil)

func (s *Store) UpsertDevice(_ context.Context, p repository.UpsertDeviceParams) (repository.Device, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if existing, ok := s.findByToken(p.UserID, p.PushToken); ok {
		// Update in place, keeping id/created_at (matches ON CONFLICT DO UPDATE).
		existing.Platform = p.Platform
		existing.NotifyTime = p.NotifyTime
		existing.Enabled = p.Enabled
		existing.UpdatedAt = s.tick()
		s.devices[existing.ID] = existing
		return existing, nil
	}

	now := s.tick()
	d := repository.Device{
		ID:         newID(),
		UserID:     p.UserID,
		PushToken:  p.PushToken,
		Platform:   p.Platform,
		NotifyTime: p.NotifyTime,
		Enabled:    p.Enabled,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	s.devices[d.ID] = d
	return d, nil
}

// findByToken locates a device by (userID, pushToken). Caller holds the lock.
func (s *Store) findByToken(userID, pushToken string) (repository.Device, bool) {
	for _, d := range s.devices {
		if d.UserID == userID && d.PushToken == pushToken {
			return d, true
		}
	}
	return repository.Device{}, false
}

// tick returns a strictly increasing timestamp so updated_at values never tie in
// fast tests. Caller holds the lock.
func (s *Store) tick() time.Time {
	t := s.now()
	if !t.After(s.lastNow) {
		t = s.lastNow.Add(time.Nanosecond)
	}
	s.lastNow = t
	return t
}

// newID returns a random v4-style UUID string without pulling in a dependency
// (same helper shape as memdiary).
func newID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
