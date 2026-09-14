package repository

import (
	"context"
	"time"
)

// DiaryEntry is the persistence model for the diary_entries table.
type DiaryEntry struct {
	ID        string
	UserID    string
	BodyText  string
	Mood      *string
	ClientID  string
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}

// UpsertDiaryParams carries the fields for an idempotent create keyed by (UserID, ClientID).
type UpsertDiaryParams struct {
	UserID    string
	ClientID  string
	BodyText  string
	Mood      *string
	CreatedAt time.Time
}

// UpdateDiaryParams carries the fields PATCH replaces wholesale; Mood == nil clears the column.
type UpdateDiaryParams struct {
	BodyText string
	Mood     *string
}

// ListDiaryParams is a keyset page request: entries strictly older than the
// cursor, newest first, capped at Limit; a nil Cursor starts at the newest.
type ListDiaryParams struct {
	UserID string
	Limit  int
	Cursor *DiaryCursor
}

// DiaryCursor is the (created_at, id) of the last row of the previous page.
type DiaryCursor struct {
	CreatedAt time.Time
	ID        string
}

// DiaryStore is the data-access boundary for the diary feature; a query for
// another user's row returns ErrNotFound.
type DiaryStore interface {
	// UpsertEntry inserts a new entry or, when (user_id, client_id) already
	// exists, updates it in place; created reports which happened.
	UpsertEntry(ctx context.Context, p UpsertDiaryParams) (entry DiaryEntry, created bool, err error)

	// ListEntries returns one keyset page of non-deleted entries, newest first.
	ListEntries(ctx context.Context, p ListDiaryParams) ([]DiaryEntry, error)

	// GetEntry returns a single non-deleted entry owned by userID, or ErrNotFound.
	GetEntry(ctx context.Context, userID, id string) (DiaryEntry, error)

	// UpdateEntry replaces the body/mood of a non-deleted entry, bumping
	// updated_at; ErrNotFound when missing, deleted or another user's.
	UpdateEntry(ctx context.Context, userID, id string, p UpdateDiaryParams) (DiaryEntry, error)

	// SoftDeleteEntry sets deleted_at on an owned entry; idempotent for an
	// already-deleted own entry, ErrNotFound only when no such id belongs to the user.
	SoftDeleteEntry(ctx context.Context, userID, id string) error

	// ListChangedSince returns every entry (including tombstones) whose updated_at
	// is strictly after since, oldest change first. A zero since returns all.
	ListChangedSince(ctx context.Context, userID string, since time.Time) ([]DiaryEntry, error)

	// CountByLocalDate counts a user's non-deleted entries created in [lo, hi),
	// keyed by local date "YYYY-MM-DD" in loc; only dates with entries appear.
	CountByLocalDate(ctx context.Context, userID string, lo, hi time.Time, loc *time.Location) (map[string]int, error)
}
