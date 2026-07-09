package service

import (
	"context"
	"errors"
	"time"

	"github.com/0muji4/Runa/apps/go/internal/repository"
)

// Diary pagination bounds. The handler clamps a client-supplied limit into
// [1, MaxDiaryLimit] and defaults an absent one to DefaultDiaryLimit.
const (
	DefaultDiaryLimit = 20
	MaxDiaryLimit     = 50
)

// ErrDiaryNotFound means the entry does not exist, was deleted, or belongs to
// another user. All three collapse to one error so the handler answers 404
// without revealing which case it was.
var ErrDiaryNotFound = errors.New("service: diary entry not found")

// CreateDiaryInput is the create/upsert payload. ClientID and CreatedAt are
// client-supplied: the client owns the identity of an offline-authored entry.
type CreateDiaryInput struct {
	ClientID  string
	BodyText  string
	Mood      *string
	CreatedAt time.Time
}

// DiaryPage is one keyset page. NextCursor is nil on the last page.
type DiaryPage struct {
	Entries    []repository.DiaryEntry
	NextCursor *repository.DiaryCursor
}

// DiaryDelta is the response of a sync pull: the entries changed since the
// requested watermark, plus the server time the client should send as the next
// `since`.
type DiaryDelta struct {
	Entries    []repository.DiaryEntry
	ServerTime time.Time
}

// DiaryService implements the diary use cases over a DiaryStore. Every method is
// scoped by userID so a caller can only ever touch their own entries.
type DiaryService struct {
	store repository.DiaryStore
	now   func() time.Time
}

// NewDiaryService constructs the service, defaulting now to time.Now.
func NewDiaryService(store repository.DiaryStore, now func() time.Time) *DiaryService {
	if now == nil {
		now = time.Now
	}
	return &DiaryService{store: store, now: now}
}

// Create idempotently creates (or updates, on a repeated client_id) an entry.
// created reports which happened, so the handler answers 201 vs 200.
func (s *DiaryService) Create(ctx context.Context, userID string, in CreateDiaryInput) (repository.DiaryEntry, bool, error) {
	createdAt := in.CreatedAt
	if createdAt.IsZero() {
		createdAt = s.now()
	}
	return s.store.UpsertEntry(ctx, repository.UpsertDiaryParams{
		UserID:    userID,
		ClientID:  in.ClientID,
		BodyText:  in.BodyText,
		Mood:      in.Mood,
		CreatedAt: createdAt,
	})
}

// List returns one page of the user's entries, newest first. It over-fetches by
// one row to decide whether a next page exists without a second query.
func (s *DiaryService) List(ctx context.Context, userID string, limit int, cursor *repository.DiaryCursor) (DiaryPage, error) {
	limit = clampLimit(limit)
	entries, err := s.store.ListEntries(ctx, repository.ListDiaryParams{
		UserID: userID,
		Limit:  limit + 1, // sentinel row reveals a further page
		Cursor: cursor,
	})
	if err != nil {
		return DiaryPage{}, err
	}

	page := DiaryPage{Entries: entries}
	if len(entries) > limit {
		page.Entries = entries[:limit]
		last := page.Entries[limit-1]
		page.NextCursor = &repository.DiaryCursor{CreatedAt: last.CreatedAt, ID: last.ID}
	}
	return page, nil
}

// Get returns a single entry, mapping the repository's not-found to
// ErrDiaryNotFound.
func (s *DiaryService) Get(ctx context.Context, userID, id string) (repository.DiaryEntry, error) {
	entry, err := s.store.GetEntry(ctx, userID, id)
	return entry, mapNotFound(err)
}

// Update replaces the body/mood of an entry.
func (s *DiaryService) Update(ctx context.Context, userID, id, bodyText string, mood *string) (repository.DiaryEntry, error) {
	entry, err := s.store.UpdateEntry(ctx, userID, id, repository.UpdateDiaryParams{BodyText: bodyText, Mood: mood})
	return entry, mapNotFound(err)
}

// Delete soft-deletes an entry. It is idempotent for an already-deleted own entry
// and returns ErrDiaryNotFound only when the id is not the caller's.
func (s *DiaryService) Delete(ctx context.Context, userID, id string) error {
	return mapNotFound(s.store.SoftDeleteEntry(ctx, userID, id))
}

// Sync returns the delta since `since`. ServerTime is captured before the query
// so a concurrent write is never stranded on the wrong side of the watermark;
// the client stores it as the next `since` (last_synced_at).
func (s *DiaryService) Sync(ctx context.Context, userID string, since time.Time) (DiaryDelta, error) {
	serverTime := s.now()
	entries, err := s.store.ListChangedSince(ctx, userID, since)
	if err != nil {
		return DiaryDelta{}, err
	}
	return DiaryDelta{Entries: entries, ServerTime: serverTime}, nil
}

func clampLimit(limit int) int {
	switch {
	case limit <= 0:
		return DefaultDiaryLimit
	case limit > MaxDiaryLimit:
		return MaxDiaryLimit
	default:
		return limit
	}
}

func mapNotFound(err error) error {
	if errors.Is(err, repository.ErrNotFound) {
		return ErrDiaryNotFound
	}
	return err
}
