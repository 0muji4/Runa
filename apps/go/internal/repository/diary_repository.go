package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// diaryColumns is the shared SELECT list; keep its order in sync with scanDiaryEntry.
const diaryColumns = `id, user_id, body_text, mood, client_id,
	created_at, updated_at, deleted_at`

// DiaryRepository is the pgx-backed implementation of DiaryStore.
type DiaryRepository struct {
	pool *pgxpool.Pool
}

// NewDiaryRepository wraps a pgx pool. A nil pool (DB unreachable at boot) makes
// every method return ErrNoDatabase instead of panicking, so liveness still serves.
func NewDiaryRepository(pool *pgxpool.Pool) *DiaryRepository {
	return &DiaryRepository{pool: pool}
}

var _ DiaryStore = (*DiaryRepository)(nil)

func scanDiaryEntry(r row) (DiaryEntry, error) {
	var e DiaryEntry
	if err := r.Scan(
		&e.ID, &e.UserID, &e.BodyText, &e.Mood, &e.ClientID,
		&e.CreatedAt, &e.UpdatedAt, &e.DeletedAt,
	); err != nil {
		return DiaryEntry{}, err
	}
	return e, nil
}

func (r *DiaryRepository) UpsertEntry(ctx context.Context, p UpsertDiaryParams) (DiaryEntry, bool, error) {
	if r.pool == nil {
		return DiaryEntry{}, false, ErrNoDatabase
	}

	// ON CONFLICT on (user_id, client_id) makes a retried offline create idempotent
	// (keeps id/created_at, takes the latest body/mood). (xmax = 0) = inserted, not
	// updated — so the handler answers 201 vs 200.
	const q = `
		INSERT INTO diary_entries (user_id, client_id, body_text, mood, created_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id, client_id) DO UPDATE
			SET body_text = EXCLUDED.body_text,
			    mood       = EXCLUDED.mood,
			    updated_at = now()
		RETURNING ` + diaryColumns + `, (xmax = 0) AS inserted`

	var e DiaryEntry
	var inserted bool
	err := r.pool.QueryRow(ctx, q, p.UserID, p.ClientID, p.BodyText, p.Mood, p.CreatedAt).Scan(
		&e.ID, &e.UserID, &e.BodyText, &e.Mood, &e.ClientID,
		&e.CreatedAt, &e.UpdatedAt, &e.DeletedAt, &inserted,
	)
	if err != nil {
		return DiaryEntry{}, false, fmt.Errorf("upsert diary entry: %w", err)
	}
	return e, inserted, nil
}

func (r *DiaryRepository) ListEntries(ctx context.Context, p ListDiaryParams) ([]DiaryEntry, error) {
	if r.pool == nil {
		return nil, ErrNoDatabase
	}

	// Keyset ((created_at, id) < cursor), not OFFSET, so inserts between pages never
	// skip/dupe rows. The first page has no cursor.
	var (
		rows pgx.Rows
		err  error
	)
	if p.Cursor == nil {
		const q = `
			SELECT ` + diaryColumns + `
			FROM diary_entries
			WHERE user_id = $1 AND deleted_at IS NULL
			ORDER BY created_at DESC, id DESC
			LIMIT $2`
		rows, err = r.pool.Query(ctx, q, p.UserID, p.Limit)
	} else {
		const q = `
			SELECT ` + diaryColumns + `
			FROM diary_entries
			WHERE user_id = $1 AND deleted_at IS NULL
			  AND (created_at, id) < ($2, $3)
			ORDER BY created_at DESC, id DESC
			LIMIT $4`
		rows, err = r.pool.Query(ctx, q, p.UserID, p.Cursor.CreatedAt, p.Cursor.ID, p.Limit)
	}
	if err != nil {
		return nil, fmt.Errorf("list diary entries: %w", err)
	}
	return collectDiaryEntries(rows)
}

func (r *DiaryRepository) GetEntry(ctx context.Context, userID, id string) (DiaryEntry, error) {
	if r.pool == nil {
		return DiaryEntry{}, ErrNoDatabase
	}
	const q = `
		SELECT ` + diaryColumns + `
		FROM diary_entries
		WHERE user_id = $1 AND id = $2 AND deleted_at IS NULL`

	e, err := scanDiaryEntry(r.pool.QueryRow(ctx, q, userID, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return DiaryEntry{}, ErrNotFound
		}
		return DiaryEntry{}, fmt.Errorf("get diary entry: %w", err)
	}
	return e, nil
}

func (r *DiaryRepository) UpdateEntry(ctx context.Context, userID, id string, p UpdateDiaryParams) (DiaryEntry, error) {
	if r.pool == nil {
		return DiaryEntry{}, ErrNoDatabase
	}
	const q = `
		UPDATE diary_entries
		SET body_text = $3, mood = $4, updated_at = now()
		WHERE user_id = $1 AND id = $2 AND deleted_at IS NULL
		RETURNING ` + diaryColumns

	e, err := scanDiaryEntry(r.pool.QueryRow(ctx, q, userID, id, p.BodyText, p.Mood))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return DiaryEntry{}, ErrNotFound
		}
		return DiaryEntry{}, fmt.Errorf("update diary entry: %w", err)
	}
	return e, nil
}

func (r *DiaryRepository) SoftDeleteEntry(ctx context.Context, userID, id string) error {
	if r.pool == nil {
		return ErrNoDatabase
	}
	// COALESCE makes delete idempotent (already-deleted row still matched, timestamps
	// kept). RowsAffected 0 ⇒ not the caller's id ⇒ ErrNotFound (404 that doesn't leak
	// whether it exists for someone else).
	const q = `
		UPDATE diary_entries
		SET deleted_at = COALESCE(deleted_at, now()),
		    updated_at = CASE WHEN deleted_at IS NULL THEN now() ELSE updated_at END
		WHERE user_id = $1 AND id = $2`

	tag, err := r.pool.Exec(ctx, q, userID, id)
	if err != nil {
		return fmt.Errorf("soft delete diary entry: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *DiaryRepository) ListChangedSince(ctx context.Context, userID string, since time.Time) ([]DiaryEntry, error) {
	if r.pool == nil {
		return nil, ErrNoDatabase
	}
	// Includes tombstones (deleted_at set) so other devices learn of deletions.
	// Oldest change first keeps the client's merge order stable.
	const q = `
		SELECT ` + diaryColumns + `
		FROM diary_entries
		WHERE user_id = $1 AND updated_at > $2
		ORDER BY updated_at ASC`

	rows, err := r.pool.Query(ctx, q, userID, since)
	if err != nil {
		return nil, fmt.Errorf("list diary changes: %w", err)
	}
	return collectDiaryEntries(rows)
}

func (r *DiaryRepository) CountByLocalDate(ctx context.Context, userID string, lo, hi time.Time, loc *time.Location) (map[string]int, error) {
	if r.pool == nil {
		return nil, ErrNoDatabase
	}
	// Group by local calendar date (shift created_at into loc before truncating) so it
	// matches the client's grouping. [lo, hi) rides the (user_id, created_at) index.
	const q = `
		SELECT to_char((created_at AT TIME ZONE $4)::date, 'YYYY-MM-DD') AS local_date, count(*)
		FROM diary_entries
		WHERE user_id = $1 AND deleted_at IS NULL AND created_at >= $2 AND created_at < $3
		GROUP BY local_date`

	rows, err := r.pool.Query(ctx, q, userID, lo, hi, loc.String())
	if err != nil {
		return nil, fmt.Errorf("count diary by local date: %w", err)
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var date string
		var count int
		if err := rows.Scan(&date, &count); err != nil {
			return nil, fmt.Errorf("scan diary day count: %w", err)
		}
		counts[date] = count
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate diary day counts: %w", err)
	}
	return counts, nil
}

// EntriesInRange returns non-deleted entries created in [lo, hi), for insights.
func (r *DiaryRepository) EntriesInRange(ctx context.Context, userID string, lo, hi time.Time) ([]DiaryEntry, error) {
	if r.pool == nil {
		return nil, ErrNoDatabase
	}
	const q = `
		SELECT ` + diaryColumns + `
		FROM diary_entries
		WHERE user_id = $1 AND deleted_at IS NULL AND created_at >= $2 AND created_at < $3
		ORDER BY created_at ASC`

	rows, err := r.pool.Query(ctx, q, userID, lo, hi)
	if err != nil {
		return nil, fmt.Errorf("list diary entries in range: %w", err)
	}
	return collectDiaryEntries(rows)
}

// collectDiaryEntries returns a non-nil empty slice for zero rows, so JSON encodes
// "[]" rather than "null".
func collectDiaryEntries(rows pgx.Rows) ([]DiaryEntry, error) {
	defer rows.Close()
	entries := make([]DiaryEntry, 0)
	for rows.Next() {
		e, err := scanDiaryEntry(rows)
		if err != nil {
			return nil, fmt.Errorf("scan diary entry: %w", err)
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate diary entries: %w", err)
	}
	return entries, nil
}
