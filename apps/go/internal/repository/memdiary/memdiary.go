// Package memdiary is an in-memory implementation of repository.DiaryStore. It
// backs the diary unit/integration tests (and lets the API run without Postgres)
// so the suite stays green in CI, which has no database. It mirrors the pgx
// implementation's semantics: idempotent upsert by (user_id, client_id), keyset
// pagination, ownership scoping, soft delete and the updated_at delta.
package memdiary

import (
	"context"
	"crypto/rand"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/0muji4/Runa/apps/go/internal/repository"
)

// Store is a goroutine-safe, in-memory DiaryStore.
type Store struct {
	mu      sync.Mutex
	entries map[string]repository.DiaryEntry // keyed by entry id
	now     func() time.Time
	lastNow time.Time
}

// New returns an empty in-memory store.
func New() *Store {
	return &Store{
		entries: make(map[string]repository.DiaryEntry),
		now:     time.Now,
	}
}

var _ repository.DiaryStore = (*Store)(nil)

func (s *Store) UpsertEntry(_ context.Context, p repository.UpsertDiaryParams) (repository.DiaryEntry, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if existing, ok := s.findByClient(p.UserID, p.ClientID); ok {
		// Update in place, keeping id/created_at (matches ON CONFLICT DO UPDATE).
		existing.BodyText = p.BodyText
		existing.Mood = clonePtr(p.Mood)
		existing.UpdatedAt = s.tick()
		s.entries[existing.ID] = existing
		return existing, false, nil
	}

	now := s.tick()
	e := repository.DiaryEntry{
		ID:        newID(),
		UserID:    p.UserID,
		BodyText:  p.BodyText,
		Mood:      clonePtr(p.Mood),
		ClientID:  p.ClientID,
		CreatedAt: p.CreatedAt,
		UpdatedAt: now,
	}
	s.entries[e.ID] = e
	return e, true, nil
}

func (s *Store) ListEntries(_ context.Context, p repository.ListDiaryParams) ([]repository.DiaryEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var out []repository.DiaryEntry
	for _, e := range s.entries {
		if e.UserID != p.UserID || e.DeletedAt != nil {
			continue
		}
		if p.Cursor != nil && !olderThanCursor(e, *p.Cursor) {
			continue
		}
		out = append(out, e)
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

func (s *Store) GetEntry(_ context.Context, userID, id string) (repository.DiaryEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if e, ok := s.entries[id]; ok && e.UserID == userID && e.DeletedAt == nil {
		return e, nil
	}
	return repository.DiaryEntry{}, repository.ErrNotFound
}

func (s *Store) UpdateEntry(_ context.Context, userID, id string, p repository.UpdateDiaryParams) (repository.DiaryEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.entries[id]
	if !ok || e.UserID != userID || e.DeletedAt != nil {
		return repository.DiaryEntry{}, repository.ErrNotFound
	}
	e.BodyText = p.BodyText
	e.Mood = clonePtr(p.Mood)
	e.UpdatedAt = s.tick()
	s.entries[id] = e
	return e, nil
}

func (s *Store) SoftDeleteEntry(_ context.Context, userID, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.entries[id]
	if !ok || e.UserID != userID {
		return repository.ErrNotFound
	}
	if e.DeletedAt == nil { // idempotent: skip if already deleted
		t := s.tick()
		e.DeletedAt = &t
		e.UpdatedAt = t
		s.entries[id] = e
	}
	return nil
}

func (s *Store) ListChangedSince(_ context.Context, userID string, since time.Time) ([]repository.DiaryEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var out []repository.DiaryEntry
	for _, e := range s.entries {
		if e.UserID == userID && e.UpdatedAt.After(since) {
			out = append(out, e)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].UpdatedAt.Before(out[j].UpdatedAt) })
	return out, nil
}

func (s *Store) CountByLocalDate(_ context.Context, userID string, lo, hi time.Time, loc *time.Location) (map[string]int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	counts := make(map[string]int)
	for _, e := range s.entries {
		if e.UserID != userID || e.DeletedAt != nil {
			continue
		}
		// [lo, hi) as instants: the month window in loc, so an entry counts iff its
		// local date falls in the month.
		if e.CreatedAt.Before(lo) || !e.CreatedAt.Before(hi) {
			continue
		}
		counts[e.CreatedAt.In(loc).Format("2006-01-02")]++
	}
	return counts, nil
}

// findByClient locates an entry by (userID, clientID). Caller holds the lock.
func (s *Store) findByClient(userID, clientID string) (repository.DiaryEntry, bool) {
	for _, e := range s.entries {
		if e.UserID == userID && e.ClientID == clientID {
			return e, true
		}
	}
	return repository.DiaryEntry{}, false
}

// tick returns a strictly increasing timestamp so updated_at values never tie,
// keeping the "updated_at > since" delta deterministic in fast tests. Caller
// holds the lock.
func (s *Store) tick() time.Time {
	t := s.now()
	if !t.After(s.lastNow) {
		t = s.lastNow.Add(time.Nanosecond)
	}
	s.lastNow = t
	return t
}

// olderThanCursor reports whether e sorts after the cursor in (created_at DESC,
// id DESC) order, i.e. belongs on a later page.
func olderThanCursor(e repository.DiaryEntry, c repository.DiaryCursor) bool {
	if !e.CreatedAt.Equal(c.CreatedAt) {
		return e.CreatedAt.Before(c.CreatedAt)
	}
	return e.ID < c.ID
}

func clonePtr(p *string) *string {
	if p == nil {
		return nil
	}
	v := *p
	return &v
}

// newID returns a random v4-style UUID string without pulling in a dependency
// (same helper shape as memauth).
func newID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
