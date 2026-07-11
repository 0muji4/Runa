package service

import (
	"context"
	"errors"
	"time"

	"github.com/0muji4/Runa/apps/go/internal/repository"
)

// Song archive pagination bounds. The handler clamps a client-supplied limit
// into [1, MaxSongLimit] and defaults an absent one to DefaultSongLimit.
const (
	DefaultSongLimit = 20
	MaxSongLimit     = 50
)

// ErrSongNotFound means the song id is not a known curated song. Recording a
// play against it answers 404.
var ErrSongNotFound = errors.New("service: song not found")

// TodayContent is the home payload for a given day: the curated quote and song.
// Either may be nil when that day has no curated entry — the client still renders
// (the moon, computed client-side, and any cached copy). The moon phase is NOT
// returned here; it is computed on the client in shared code.
type TodayContent struct {
	Date  time.Time
	Quote *repository.Quote
	Song  *repository.Song
}

// SongPage is one keyset page of the archive. NextCursor is nil on the last page.
type SongPage struct {
	Songs      []repository.Song
	NextCursor *repository.SongCursor
}

// TodayService implements the today use cases over a TodayStore.
type TodayService struct {
	store repository.TodayStore
	now   func() time.Time
}

// NewTodayService constructs the service, defaulting now to time.Now.
func NewTodayService(store repository.TodayStore, now func() time.Time) *TodayService {
	if now == nil {
		now = time.Now
	}
	return &TodayService{store: store, now: now}
}

// Today returns the curated quote and song for the exact day. A missing quote or
// song is not an error: the field is left nil so the home screen renders with a
// null element rather than failing.
func (s *TodayService) Today(ctx context.Context, date time.Time) (TodayContent, error) {
	content := TodayContent{Date: date}

	quote, err := s.store.GetQuoteForDate(ctx, date)
	if err == nil {
		content.Quote = &quote
	} else if !errors.Is(err, repository.ErrNotFound) {
		return TodayContent{}, err
	}

	song, err := s.store.GetSongForDate(ctx, date)
	if err == nil {
		content.Song = &song
	} else if !errors.Is(err, repository.ErrNotFound) {
		return TodayContent{}, err
	}

	return content, nil
}

// Archive returns one page of the song archive, newest first. It over-fetches by
// one row to decide whether a next page exists without a second query.
func (s *TodayService) Archive(ctx context.Context, limit int, cursor *repository.SongCursor) (SongPage, error) {
	limit = clampSongLimit(limit)
	songs, err := s.store.ListSongs(ctx, repository.ListSongsParams{
		Limit:  limit + 1, // sentinel row reveals a further page
		Cursor: cursor,
	})
	if err != nil {
		return SongPage{}, err
	}

	page := SongPage{Songs: songs}
	if len(songs) > limit {
		page.Songs = songs[:limit]
		last := page.Songs[limit-1]
		page.NextCursor = &repository.SongCursor{Date: last.Date, ID: last.ID}
	}
	return page, nil
}

// MarkPlayed records a play. playedAt defaults to the server clock when zero. An
// unknown song id maps to ErrSongNotFound (a 404).
func (s *TodayService) MarkPlayed(ctx context.Context, userID, songID string, playedAt time.Time) error {
	if playedAt.IsZero() {
		playedAt = s.now()
	}
	err := s.store.RecordPlay(ctx, userID, songID, playedAt)
	if errors.Is(err, repository.ErrNotFound) {
		return ErrSongNotFound
	}
	return err
}

// CreateQuote upserts a day's curated quote (admin).
func (s *TodayService) CreateQuote(ctx context.Context, date time.Time, bodyText string) (repository.Quote, error) {
	return s.store.InsertQuote(ctx, repository.InsertQuoteParams{Date: date, BodyText: bodyText})
}

// CreateSong upserts a day's curated song (admin).
func (s *TodayService) CreateSong(ctx context.Context, p repository.InsertSongParams) (repository.Song, error) {
	return s.store.InsertSong(ctx, p)
}

func clampSongLimit(limit int) int {
	switch {
	case limit <= 0:
		return DefaultSongLimit
	case limit > MaxSongLimit:
		return MaxSongLimit
	default:
		return limit
	}
}
