package service

import (
	"context"
	"errors"
	"sort"
	"time"

	"github.com/0muji4/Runa/apps/go/internal/repository"
)

// Diary pagination bounds: limit is clamped into [1, MaxDiaryLimit], absent → DefaultDiaryLimit.
const (
	DefaultDiaryLimit = 20
	MaxDiaryLimit     = 50
)

// ErrDiaryNotFound means the entry does not exist, was deleted, or belongs to another user.
var ErrDiaryNotFound = errors.New("service: diary entry not found")

// CreateDiaryInput is the create/upsert payload; ClientID and CreatedAt are client-supplied.
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

// DiaryDelta is a sync pull; ServerTime is what the client sends as the next `since`.
type DiaryDelta struct {
	Entries    []repository.DiaryEntry
	ServerTime time.Time
}

// DiaryService implements the diary use cases over a DiaryStore.
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

// Create idempotently creates (or updates, on a repeated client_id) an entry; the bool reports created.
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

// List returns one page of the user's entries, newest first.
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

// Get returns a single entry.
func (s *DiaryService) Get(ctx context.Context, userID, id string) (repository.DiaryEntry, error) {
	entry, err := s.store.GetEntry(ctx, userID, id)
	return entry, mapNotFound(err)
}

// Update replaces the body/mood of an entry.
func (s *DiaryService) Update(ctx context.Context, userID, id, bodyText string, mood *string) (repository.DiaryEntry, error) {
	entry, err := s.store.UpdateEntry(ctx, userID, id, repository.UpdateDiaryParams{BodyText: bodyText, Mood: mood})
	return entry, mapNotFound(err)
}

// Delete soft-deletes an entry; idempotent for an already-deleted own entry,
// ErrDiaryNotFound only when the id is not the caller's.
func (s *DiaryService) Delete(ctx context.Context, userID, id string) error {
	return mapNotFound(s.store.SoftDeleteEntry(ctx, userID, id))
}

// Sync returns the delta since `since`. ServerTime is captured before the query
// so a concurrent write is never stranded on the wrong side of the watermark.
func (s *DiaryService) Sync(ctx context.Context, userID string, since time.Time) (DiaryDelta, error) {
	serverTime := s.now()
	entries, err := s.store.ListChangedSince(ctx, userID, since)
	if err != nil {
		return DiaryDelta{}, err
	}
	return DiaryDelta{Entries: entries, ServerTime: serverTime}, nil
}

// DiaryCalendarDay is one local date in a month with at least one entry.
type DiaryCalendarDay struct {
	Date  string // local calendar date "YYYY-MM-DD"
	Count int
}

// Calendar returns per-local-date entry counts for the month window
// [first-of-month 00:00 loc, first-of-next-month 00:00 loc), ascending by date.
func (s *DiaryService) Calendar(ctx context.Context, userID string, year, month int, loc *time.Location) ([]DiaryCalendarDay, error) {
	lo := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, loc)
	hi := lo.AddDate(0, 1, 0)

	counts, err := s.store.CountByLocalDate(ctx, userID, lo, hi, loc)
	if err != nil {
		return nil, err
	}

	days := make([]DiaryCalendarDay, 0, len(counts))
	for date, count := range counts {
		days = append(days, DiaryCalendarDay{Date: date, Count: count})
	}
	sort.Slice(days, func(i, j int) bool { return days[i].Date < days[j].Date })
	return days, nil
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
