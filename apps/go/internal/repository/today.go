package repository

import (
	"context"
	"time"
)

// Quote is the persistence model for the daily_quotes table (migration 0004):
// one curated poetic line per calendar day. Date is the day it is shown on.
type Quote struct {
	ID       string
	Date     time.Time
	BodyText string
}

// Song is the persistence model for the daily_songs table (migration 0004): one
// curated song per calendar day, with the artwork and audio the player needs.
type Song struct {
	ID         string
	Date       time.Time
	Title      string
	Artist     string
	ArtworkURL string
	AudioURL   string
}

// InsertQuoteParams / InsertSongParams carry the fields for an admin upsert keyed
// by Date (one entry per day). A repeated Date replaces that day's entry.
type InsertQuoteParams struct {
	Date     time.Time
	BodyText string
}

type InsertSongParams struct {
	Date       time.Time
	Title      string
	Artist     string
	ArtworkURL string
	AudioURL   string
}

// ListSongsParams is a keyset page request for the song archive: songs strictly
// older than the (Date, ID) cursor, newest first, capped at Limit. A nil Cursor
// starts at the newest song.
type ListSongsParams struct {
	Limit  int
	Cursor *SongCursor
}

// SongCursor is the opaque archive page boundary: the (date, id) of the last row
// of the previous page. The handler encodes/decodes it to a string.
type SongCursor struct {
	Date time.Time
	ID   string
}

// TodayStore is the data-access boundary for the today feature. The service
// depends on this interface so tests can substitute an in-memory fake. Quotes
// and songs are global (curated content); only play history is scoped by user.
type TodayStore interface {
	// GetQuoteForDate returns the quote curated for the exact day, or ErrNotFound.
	GetQuoteForDate(ctx context.Context, date time.Time) (Quote, error)
	// GetSongForDate returns the song curated for the exact day, or ErrNotFound.
	GetSongForDate(ctx context.Context, date time.Time) (Song, error)

	// ListSongs returns one keyset page of the song archive, newest first.
	ListSongs(ctx context.Context, p ListSongsParams) ([]Song, error)

	// RecordPlay appends a play-history row for (userID, songID). It returns
	// ErrNotFound when songID is not a known song (the FK insert would fail).
	RecordPlay(ctx context.Context, userID, songID string, playedAt time.Time) error

	// InsertQuote / InsertSong upsert a day's curated entry (admin). A repeated
	// Date replaces the existing row for that day.
	InsertQuote(ctx context.Context, p InsertQuoteParams) (Quote, error)
	InsertSong(ctx context.Context, p InsertSongParams) (Song, error)
}
