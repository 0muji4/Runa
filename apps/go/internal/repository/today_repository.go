package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// quoteColumns / songColumns are the shared SELECT lists; keep their order in sync
// with scanQuote / scanSong.
const (
	quoteColumns = `id, date, body_text`
	songColumns  = `id, date, title, artist, artwork_url, audio_url`
)

// fkViolation is the Postgres SQLSTATE for a FK violation (a play against an
// unknown song id); mapped to ErrNotFound.
const fkViolation = "23503"

// TodayRepository is the pgx-backed implementation of TodayStore.
type TodayRepository struct {
	pool *pgxpool.Pool
}

// NewTodayRepository wraps a pgx pool. A nil pool (DB unreachable at boot) makes
// every method return ErrNoDatabase instead of panicking, so liveness still serves.
func NewTodayRepository(pool *pgxpool.Pool) *TodayRepository {
	return &TodayRepository{pool: pool}
}

var _ TodayStore = (*TodayRepository)(nil)

func scanQuote(r row) (Quote, error) {
	var q Quote
	if err := r.Scan(&q.ID, &q.Date, &q.BodyText); err != nil {
		return Quote{}, err
	}
	return q, nil
}

func scanSong(r row) (Song, error) {
	var s Song
	if err := r.Scan(&s.ID, &s.Date, &s.Title, &s.Artist, &s.ArtworkURL, &s.AudioURL); err != nil {
		return Song{}, err
	}
	return s, nil
}

func (r *TodayRepository) GetQuoteForDate(ctx context.Context, date time.Time) (Quote, error) {
	if r.pool == nil {
		return Quote{}, ErrNoDatabase
	}
	const q = `SELECT ` + quoteColumns + ` FROM daily_quotes WHERE date = $1`
	quote, err := scanQuote(r.pool.QueryRow(ctx, q, date))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Quote{}, ErrNotFound
		}
		return Quote{}, fmt.Errorf("get quote for date: %w", err)
	}
	return quote, nil
}

func (r *TodayRepository) GetSongForDate(ctx context.Context, date time.Time) (Song, error) {
	if r.pool == nil {
		return Song{}, ErrNoDatabase
	}
	const q = `SELECT ` + songColumns + ` FROM daily_songs WHERE date = $1`
	song, err := scanSong(r.pool.QueryRow(ctx, q, date))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Song{}, ErrNotFound
		}
		return Song{}, fmt.Errorf("get song for date: %w", err)
	}
	return song, nil
}

func (r *TodayRepository) ListSongs(ctx context.Context, p ListSongsParams) ([]Song, error) {
	if r.pool == nil {
		return nil, ErrNoDatabase
	}

	// Keyset ((date, id) < cursor), not OFFSET, so inserts between pages never
	// skip/dupe rows. The first page has no cursor.
	var (
		rows pgx.Rows
		err  error
	)
	if p.Cursor == nil {
		const q = `
			SELECT ` + songColumns + `
			FROM daily_songs
			ORDER BY date DESC, id DESC
			LIMIT $1`
		rows, err = r.pool.Query(ctx, q, p.Limit)
	} else {
		const q = `
			SELECT ` + songColumns + `
			FROM daily_songs
			WHERE (date, id) < ($1, $2)
			ORDER BY date DESC, id DESC
			LIMIT $3`
		rows, err = r.pool.Query(ctx, q, p.Cursor.Date, p.Cursor.ID, p.Limit)
	}
	if err != nil {
		return nil, fmt.Errorf("list songs: %w", err)
	}
	return collectSongs(rows)
}

func (r *TodayRepository) RecordPlay(ctx context.Context, userID, songID string, playedAt time.Time) error {
	if r.pool == nil {
		return ErrNoDatabase
	}
	const q = `INSERT INTO song_history (user_id, song_id, played_at) VALUES ($1, $2, $3)`
	if _, err := r.pool.Exec(ctx, q, userID, songID, playedAt); err != nil {
		// An unknown song id fails the daily_songs FK → ErrNotFound (404, not 500).
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == fkViolation {
			return ErrNotFound
		}
		return fmt.Errorf("record play: %w", err)
	}
	return nil
}

func (r *TodayRepository) InsertQuote(ctx context.Context, p InsertQuoteParams) (Quote, error) {
	if r.pool == nil {
		return Quote{}, ErrNoDatabase
	}
	// Upsert on the date unique index so re-seeding a day replaces its copy.
	const q = `
		INSERT INTO daily_quotes (date, body_text)
		VALUES ($1, $2)
		ON CONFLICT (date) DO UPDATE SET body_text = EXCLUDED.body_text
		RETURNING ` + quoteColumns

	quote, err := scanQuote(r.pool.QueryRow(ctx, q, p.Date, p.BodyText))
	if err != nil {
		return Quote{}, fmt.Errorf("insert quote: %w", err)
	}
	return quote, nil
}

func (r *TodayRepository) InsertSong(ctx context.Context, p InsertSongParams) (Song, error) {
	if r.pool == nil {
		return Song{}, ErrNoDatabase
	}
	const q = `
		INSERT INTO daily_songs (date, title, artist, artwork_url, audio_url)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (date) DO UPDATE
			SET title       = EXCLUDED.title,
			    artist      = EXCLUDED.artist,
			    artwork_url = EXCLUDED.artwork_url,
			    audio_url   = EXCLUDED.audio_url
		RETURNING ` + songColumns

	song, err := scanSong(r.pool.QueryRow(ctx, q, p.Date, p.Title, p.Artist, p.ArtworkURL, p.AudioURL))
	if err != nil {
		return Song{}, fmt.Errorf("insert song: %w", err)
	}
	return song, nil
}

// collectSongs returns a non-nil empty slice for zero rows, so JSON encodes "[]"
// rather than "null".
func collectSongs(rows pgx.Rows) ([]Song, error) {
	defer rows.Close()
	songs := make([]Song, 0)
	for rows.Next() {
		s, err := scanSong(rows)
		if err != nil {
			return nil, fmt.Errorf("scan song: %w", err)
		}
		songs = append(songs, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate songs: %w", err)
	}
	return songs, nil
}
