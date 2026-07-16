package repository

import (
	"context"
	"time"
)

// DiaryEntry is the persistence model for the diary_entries table (migration
// 0003). Nullable columns are pointers so an absent value is distinguishable from
// a zero value.
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

// UpsertDiaryParams carries the fields for an idempotent create keyed by
// (UserID, ClientID). CreatedAt is client-supplied so an entry written offline
// keeps its original authored time.
type UpsertDiaryParams struct {
	UserID    string
	ClientID  string
	BodyText  string
	Mood      *string
	CreatedAt time.Time
}

// UpdateDiaryParams carries the fields PATCH replaces. Both body and mood are set
// wholesale; Mood == nil clears the column.
type UpdateDiaryParams struct {
	BodyText string
	Mood     *string
}

// ListDiaryParams is a keyset page request: entries strictly older than the
// (CreatedAt, ID) cursor, newest first, capped at Limit. A nil Cursor starts at
// the newest entry.
type ListDiaryParams struct {
	UserID string
	Limit  int
	Cursor *DiaryCursor
}

// DiaryCursor is the opaque page boundary: the (created_at, id) of the last row
// of the previous page. The handler encodes/decodes it to a string.
type DiaryCursor struct {
	CreatedAt time.Time
	ID        string
}

// DiaryStore is the data-access boundary for the diary feature. The service
// depends on this interface so tests can substitute an in-memory fake. Every
// method is scoped by user id: a query for another user's row returns ErrNotFound
// so ownership is enforced at the data layer, not just the handler.
type DiaryStore interface {
	// UpsertEntry inserts a new entry or, when (user_id, client_id) already
	// exists, updates it in place. created reports which happened so the handler
	// can answer 201 vs 200.
	UpsertEntry(ctx context.Context, p UpsertDiaryParams) (entry DiaryEntry, created bool, err error)

	// ListEntries returns one keyset page of non-deleted entries, newest first.
	ListEntries(ctx context.Context, p ListDiaryParams) ([]DiaryEntry, error)

	// GetEntry returns a single non-deleted entry owned by userID, or ErrNotFound.
	GetEntry(ctx context.Context, userID, id string) (DiaryEntry, error)

	// UpdateEntry replaces the body/mood of a non-deleted entry, bumping
	// updated_at. Returns ErrNotFound when the row is missing, deleted or owned
	// by another user.
	UpdateEntry(ctx context.Context, userID, id string, p UpdateDiaryParams) (DiaryEntry, error)

	// SoftDeleteEntry sets deleted_at on an owned entry. It is idempotent:
	// deleting an already-deleted own entry succeeds. Returns ErrNotFound only
	// when no such id belongs to the user.
	SoftDeleteEntry(ctx context.Context, userID, id string) error

	// ListChangedSince returns every entry (including tombstones) whose updated_at
	// is strictly after since, oldest change first. A zero since returns all.
	ListChangedSince(ctx context.Context, userID string, since time.Time) ([]DiaryEntry, error)

	// CountByLocalDate counts a user's non-deleted entries created in [lo, hi),
	// grouped by their local calendar date in loc. The key is the local date
	// "YYYY-MM-DD"; only dates with at least one entry appear. Backs the calendar
	// endpoint, whose grouping must match the client's local-date grouping.
	CountByLocalDate(ctx context.Context, userID string, lo, hi time.Time, loc *time.Location) (map[string]int, error)
}
