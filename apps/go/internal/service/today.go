package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/0muji4/Runa/apps/go/internal/itunes"
	"github.com/0muji4/Runa/apps/go/internal/repository"
)

// Song archive pagination bounds: limit is clamped into [1, MaxSongLimit], absent → DefaultSongLimit.
const (
	DefaultSongLimit = 20
	MaxSongLimit     = 50
)

// Song metadata read after RefreshTTL is re-fetched in the background; a failed
// re-fetch is not retried before RefreshRetryAfter.
const (
	RefreshTTL        = 24 * time.Hour
	RefreshRetryAfter = 5 * time.Minute
	// refreshTimeout bounds one background batch (lookup, artwork checks, row updates).
	refreshTimeout = 30 * time.Second
)

// ErrSongNotFound means the song id is not a known curated song.
var ErrSongNotFound = errors.New("service: song not found")

// Registration failures the admin can act on.
var (
	ErrTrackNotFound  = errors.New("service: track not found in the Apple catalog")
	ErrTrackNoPreview = errors.New("service: track has no preview")
)

// ErrTrackLookupFailed wraps a transport/upstream failure talking to Apple during registration.
var ErrTrackLookupFailed = errors.New("service: track lookup failed")

// TrackLookup is the seam to the iTunes Search API.
type TrackLookup interface {
	Lookup(ctx context.Context, ids []int64) ([]itunes.Track, error)
}

// TodayContent is the home payload for a given day; Quote and Song are nil when
// the day has no curated entry. The moon phase is computed on the client.
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

// TodayService implements the today use cases over a TodayStore and a TrackLookup.
type TodayService struct {
	store      repository.TodayStore
	lookup     TrackLookup
	now        func() time.Time
	background func(func())
	logger     *slog.Logger

	// refreshing marks songs whose background re-fetch is in flight or recently
	// failed (song id → when it may next be attempted).
	mu         sync.Mutex
	refreshing map[string]time.Time
}

// TodayOption customises NewTodayService.
type TodayOption func(*TodayService)

// WithTodayBackgroundRunner overrides how the metadata re-fetch is run.
func WithTodayBackgroundRunner(run func(func())) TodayOption {
	return func(s *TodayService) { s.background = run }
}

// WithTodayLogger sets the logger for background re-fetch failures.
func WithTodayLogger(logger *slog.Logger) TodayOption {
	return func(s *TodayService) { s.logger = logger }
}

// NewTodayService constructs the service, defaulting now to time.Now.
func NewTodayService(store repository.TodayStore, lookup TrackLookup, now func() time.Time, opts ...TodayOption) *TodayService {
	if now == nil {
		now = time.Now
	}
	s := &TodayService{
		store:      store,
		lookup:     lookup,
		now:        now,
		background: func(f func()) { go f() },
		logger:     slog.Default(),
		refreshing: make(map[string]time.Time),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Today returns the curated quote and song for the exact day; a missing one is left nil.
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
		s.refreshStale([]repository.Song{song})
	} else if !errors.Is(err, repository.ErrNotFound) {
		return TodayContent{}, err
	}

	return content, nil
}

// Archive returns one page of the song archive up to and including until, newest first.
func (s *TodayService) Archive(ctx context.Context, until time.Time, limit int, cursor *repository.SongCursor) (SongPage, error) {
	limit = clampSongLimit(limit)
	songs, err := s.store.ListSongs(ctx, repository.ListSongsParams{
		Until:  until,
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
	s.refreshStale(page.Songs)
	return page, nil
}

// MarkPlayed records a play; playedAt defaults to the server clock when zero.
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

// CreateSong registers Apple's track for a day (admin), fetching its metadata first.
func (s *TodayService) CreateSong(ctx context.Context, date time.Time, trackID int64) (repository.Song, error) {
	tracks, err := s.lookup.Lookup(ctx, []int64{trackID})
	if err != nil {
		return repository.Song{}, fmt.Errorf("%w: %w", ErrTrackLookupFailed, err)
	}
	track, ok := findTrack(tracks, trackID)
	if !ok {
		return repository.Song{}, ErrTrackNotFound
	}
	if track.PreviewURL == "" {
		return repository.Song{}, ErrTrackNoPreview
	}
	return s.store.InsertSong(ctx, repository.InsertSongParams{
		Date:          date,
		ITunesTrackID: trackID,
		SongMetadata:  toMetadata(track, s.now()),
	})
}

// refreshStale re-fetches, in the background, songs resolved more than RefreshTTL ago.
func (s *TodayService) refreshStale(songs []repository.Song) {
	now := s.now()
	var stale []repository.Song

	s.mu.Lock()
	for _, song := range songs {
		if now.Sub(song.ResolvedAt) < RefreshTTL {
			continue
		}
		if next, busy := s.refreshing[song.ID]; busy && now.Before(next) {
			continue
		}
		// Claim the song until the attempt finishes or the retry window passes.
		s.refreshing[song.ID] = now.Add(RefreshRetryAfter)
		stale = append(stale, song)
	}
	s.mu.Unlock()

	if len(stale) == 0 {
		return
	}
	s.background(func() { s.refresh(stale) })
}

// refresh is the background half of refreshStale; a track that failed keeps its
// claim in refreshing, so it is not retried before the window.
func (s *TodayService) refresh(songs []repository.Song) {
	ctx, cancel := context.WithTimeout(context.Background(), refreshTimeout)
	defer cancel()

	ids := make([]int64, 0, len(songs))
	for _, song := range songs {
		ids = append(ids, song.ITunesTrackID)
	}
	tracks, err := s.lookup.Lookup(ctx, ids)
	if err != nil {
		s.logger.Warn("song metadata refresh failed", slog.Any("error", err), slog.Int("songs", len(songs)))
		return
	}

	resolvedAt := s.now()
	for _, song := range songs {
		track, ok := findTrack(tracks, song.ITunesTrackID)
		if !ok || track.PreviewURL == "" {
			s.logger.Warn("song metadata refresh: track unavailable", slog.String("song_id", song.ID), slog.Int64("itunes_track_id", song.ITunesTrackID))
			continue
		}
		meta := toMetadata(track, resolvedAt)
		if itunes.LargerArtworkURL(track.ArtworkURL) == song.ArtworkURL {
			// Same artwork but the 600px check did not pass this time: keep the
			// rendition already verified rather than downgrade to 100px.
			meta.ArtworkURL = song.ArtworkURL
		}
		if err := s.store.UpdateSongMetadata(ctx, song.ID, meta); err != nil {
			s.logger.Warn("song metadata refresh: update failed", slog.Any("error", err), slog.String("song_id", song.ID))
			continue
		}
		s.mu.Lock()
		delete(s.refreshing, song.ID)
		s.mu.Unlock()
	}
}

func findTrack(tracks []itunes.Track, trackID int64) (itunes.Track, bool) {
	for _, t := range tracks {
		if t.TrackID == trackID {
			return t, true
		}
	}
	return itunes.Track{}, false
}

func toMetadata(t itunes.Track, resolvedAt time.Time) repository.SongMetadata {
	return repository.SongMetadata{
		Title:      t.Title,
		Artist:     t.Artist,
		ArtworkURL: t.ArtworkURL,
		PreviewURL: t.PreviewURL,
		StoreURL:   t.StoreURL,
		ResolvedAt: resolvedAt,
	}
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
