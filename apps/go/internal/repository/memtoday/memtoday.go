// Package memtoday is an in-memory implementation of repository.TodayStore. It
// backs the today unit/integration tests (and lets the API run without Postgres)
// so the suite stays green in CI, which has no database. It mirrors the pgx
// implementation's semantics: exact-date quote/song lookup, keyset archive
// pagination, an append-only play log, and date-keyed admin upserts.
package memtoday

import (
	"context"
	"crypto/rand"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/0muji4/Runa/apps/go/internal/repository"
)

// Store is a goroutine-safe, in-memory TodayStore.
type Store struct {
	mu      sync.Mutex
	quotes  map[string]repository.Quote // keyed by day (YYYY-MM-DD)
	songs   map[string]repository.Song  // keyed by song id
	history []play
}

type play struct {
	userID   string
	songID   string
	playedAt time.Time
}

// New returns an empty in-memory store.
func New() *Store {
	return &Store{
		quotes: make(map[string]repository.Quote),
		songs:  make(map[string]repository.Song),
	}
}

var _ repository.TodayStore = (*Store)(nil)

func (s *Store) GetQuoteForDate(_ context.Context, date time.Time) (repository.Quote, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if q, ok := s.quotes[dayKey(date)]; ok {
		return q, nil
	}
	return repository.Quote{}, repository.ErrNotFound
}

func (s *Store) GetSongForDate(_ context.Context, date time.Time) (repository.Song, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, song := range s.songs {
		if dayKey(song.Date) == dayKey(date) {
			return song, nil
		}
	}
	return repository.Song{}, repository.ErrNotFound
}

func (s *Store) ListSongs(_ context.Context, p repository.ListSongsParams) ([]repository.Song, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]repository.Song, 0, len(s.songs))
	for _, song := range s.songs {
		if p.Cursor != nil && !olderThanCursor(song, *p.Cursor) {
			continue
		}
		out = append(out, song)
	}
	// Newest first: date DESC, then id DESC as a stable tiebreak.
	sort.Slice(out, func(i, j int) bool {
		if !out[i].Date.Equal(out[j].Date) {
			return out[i].Date.After(out[j].Date)
		}
		return out[i].ID > out[j].ID
	})
	if p.Limit > 0 && len(out) > p.Limit {
		out = out[:p.Limit]
	}
	return out, nil
}

func (s *Store) RecordPlay(_ context.Context, userID, songID string, playedAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.songs[songID]; !ok {
		return repository.ErrNotFound
	}
	s.history = append(s.history, play{userID: userID, songID: songID, playedAt: playedAt})
	return nil
}

func (s *Store) InsertQuote(_ context.Context, p repository.InsertQuoteParams) (repository.Quote, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := dayKey(p.Date)
	q := repository.Quote{Date: p.Date, BodyText: p.BodyText}
	if existing, ok := s.quotes[key]; ok {
		q.ID = existing.ID // keep id on upsert (matches ON CONFLICT DO UPDATE)
	} else {
		q.ID = newID()
	}
	s.quotes[key] = q
	return q, nil
}

func (s *Store) InsertSong(_ context.Context, p repository.InsertSongParams) (repository.Song, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	song := repository.Song{
		Date:       p.Date,
		Title:      p.Title,
		Artist:     p.Artist,
		ArtworkURL: p.ArtworkURL,
		AudioURL:   p.AudioURL,
	}
	// Upsert keyed by day: replace an existing song for that date, keeping its id.
	for id, existing := range s.songs {
		if dayKey(existing.Date) == dayKey(p.Date) {
			song.ID = id
			s.songs[id] = song
			return song, nil
		}
	}
	song.ID = newID()
	s.songs[song.ID] = song
	return song, nil
}

// dayKey normalizes a timestamp to its UTC calendar day, so lookups match the
// DATE column's day-only semantics regardless of any time component.
func dayKey(t time.Time) string {
	return t.UTC().Format("2006-01-02")
}

// olderThanCursor reports whether song sorts after the cursor in (date DESC, id
// DESC) order, i.e. belongs on a later page.
func olderThanCursor(song repository.Song, c repository.SongCursor) bool {
	if dayKey(song.Date) != dayKey(c.Date) {
		return song.Date.Before(c.Date)
	}
	return song.ID < c.ID
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
