package repository

import (
	"context"
	"time"
)

// Quote is the persistence model for the daily_quotes table: one line per calendar day.
type Quote struct {
	ID       string
	Date     time.Time
	BodyText string
}

// Song is the persistence model for the daily_songs table: one song per calendar
// day; SongMetadata was fetched from Apple at ResolvedAt.
type Song struct {
	ID            string
	Date          time.Time
	ITunesTrackID int64
	SongMetadata
}

// SongMetadata is what Runa keeps from an iTunes lookup.
type SongMetadata struct {
	Title      string
	Artist     string
	ArtworkURL string
	PreviewURL string
	StoreURL   string
	ResolvedAt time.Time
}

// InsertQuoteParams / InsertSongParams carry the fields for an admin upsert keyed by Date.
type InsertQuoteParams struct {
	Date     time.Time
	BodyText string
}

type InsertSongParams struct {
	Date          time.Time
	ITunesTrackID int64
	SongMetadata
}

// ListSongsParams is a keyset page request: songs dated Until or earlier and
// strictly older than the cursor, newest first, capped at Limit; a nil Cursor starts at the newest.
type ListSongsParams struct {
	Until  time.Time
	Limit  int
	Cursor *SongCursor
}

// SongCursor is the (date, id) of the last row of the previous page.
type SongCursor struct {
	Date time.Time
	ID   string
}

// TodayStore is the data-access boundary for the today feature; quotes and
// songs are global, only play history is scoped by user.
type TodayStore interface {
	// GetQuoteForDate returns the quote curated for the exact day, or ErrNotFound.
	GetQuoteForDate(ctx context.Context, date time.Time) (Quote, error)
	// GetSongForDate returns the song curated for the exact day, or ErrNotFound.
	GetSongForDate(ctx context.Context, date time.Time) (Song, error)

	// ListSongs returns one keyset page of the song archive, newest first.
	ListSongs(ctx context.Context, p ListSongsParams) ([]Song, error)

	// RecordPlay appends a play-history row; ErrNotFound when songID is not a known song.
	RecordPlay(ctx context.Context, userID, songID string, playedAt time.Time) error

	// InsertQuote / InsertSong upsert a day's curated entry; a repeated Date replaces the row.
	InsertQuote(ctx context.Context, p InsertQuoteParams) (Quote, error)
	InsertSong(ctx context.Context, p InsertSongParams) (Song, error)

	// UpdateSongMetadata replaces a song's fetched metadata; ErrNotFound when songID is unknown.
	UpdateSongMetadata(ctx context.Context, songID string, m SongMetadata) error
}
