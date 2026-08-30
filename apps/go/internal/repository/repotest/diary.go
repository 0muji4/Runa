package repotest

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/0muji4/Runa/apps/go/internal/repository"
	"github.com/google/go-cmp/cmp"
)

// clientID builds a UUID-shaped client id; the column is typed UUID.
func clientID(n int) string {
	return fmt.Sprintf("aaaaaaaa-aaaa-4aaa-8aaa-%012d", n)
}

// RunDiaryStoreSuite exercises the DiaryStore contract.
func RunDiaryStoreSuite(t *testing.T, newFixture NewFixture) {
	// seed writes n entries one minute apart, oldest first.
	seed := func(t *testing.T, f Fixture, userID string, n int, base time.Time) []repository.DiaryEntry {
		t.Helper()
		out := make([]repository.DiaryEntry, 0, n)
		for i := 1; i <= n; i++ {
			e, _, err := f.Diary.UpsertEntry(t.Context(), repository.UpsertDiaryParams{
				UserID:    userID,
				ClientID:  clientID(i),
				BodyText:  fmt.Sprintf("entry %d", i),
				CreatedAt: base.Add(time.Duration(i) * time.Minute),
			})
			if err != nil {
				t.Fatalf("seeding entry %d: UpsertEntry() error = %v, want nil", i, err)
			}
			out = append(out, e)
		}
		return out
	}

	t.Run("UpsertIsIdempotentPerClientID", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)
		ctx := t.Context()
		user := f.NewUser(t)
		authored := time.Now().Add(-time.Hour).UTC().Truncate(time.Millisecond)

		first, created, err := f.Diary.UpsertEntry(ctx, repository.UpsertDiaryParams{
			UserID: user, ClientID: clientID(1), BodyText: "v1", CreatedAt: authored,
		})
		if err != nil {
			t.Fatalf("first UpsertEntry() error = %v, want nil", err)
		}
		if !created {
			t.Error("first UpsertEntry() created = false, want true")
		}

		second, created, err := f.Diary.UpsertEntry(ctx, repository.UpsertDiaryParams{
			UserID: user, ClientID: clientID(1), BodyText: "v2", Mood: ptr("calm"), CreatedAt: authored.Add(time.Hour),
		})
		if err != nil {
			t.Fatalf("second UpsertEntry() error = %v, want nil", err)
		}
		if created {
			t.Error("second UpsertEntry() created = true, want false")
		}
		if second.ID != first.ID {
			t.Errorf("second UpsertEntry() id = %q, want the first id %q", second.ID, first.ID)
		}
		if second.BodyText != "v2" {
			t.Errorf("body_text = %q, want %q", second.BodyText, "v2")
		}
		if second.Mood == nil || *second.Mood != "calm" {
			t.Errorf("mood = %v, want %q", second.Mood, "calm")
		}
		// created_at is the client's authored time: a retried offline create must
		// not restamp the entry.
		if !second.CreatedAt.Equal(first.CreatedAt) {
			t.Errorf("created_at = %s after upsert, want the original %s",
				second.CreatedAt.UTC(), first.CreatedAt.UTC())
		}
	})

	t.Run("ClientIDIsScopedPerUser", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)
		ctx := t.Context()
		userA, userB := f.NewUser(t), f.NewUser(t)
		now := time.Now().UTC()

		a, _, err := f.Diary.UpsertEntry(ctx, repository.UpsertDiaryParams{
			UserID: userA, ClientID: clientID(1), BodyText: "a", CreatedAt: now,
		})
		if err != nil {
			t.Fatalf("UpsertEntry(userA) error = %v, want nil", err)
		}
		b, created, err := f.Diary.UpsertEntry(ctx, repository.UpsertDiaryParams{
			UserID: userB, ClientID: clientID(1), BodyText: "b", CreatedAt: now,
		})
		if err != nil {
			t.Fatalf("UpsertEntry(userB) error = %v, want nil", err)
		}
		if !created {
			t.Error("the same client_id under a different user reported created = false, want true")
		}
		if a.ID == b.ID {
			t.Errorf("two users sharing a client_id collided on id %q, want distinct rows", a.ID)
		}
	})

	t.Run("ListIsNewestFirstAndPagesWithoutGaps", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)
		ctx := t.Context()
		user := f.NewUser(t)
		base := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
		seeded := seed(t, f, user, 5, base)

		// Newest first, and the seeds went in oldest first.
		want := make([]string, 0, len(seeded))
		for i := len(seeded) - 1; i >= 0; i-- {
			want = append(want, seeded[i].ID)
		}

		// Whatever the page size, walking every page must rebuild the same list.
		tests := []struct {
			name  string
			limit int
		}{
			{
				name:  "1件ずつ",
				limit: 1,
			},
			{
				name:  "端数の出る2件ずつ",
				limit: 2,
			},
			{
				name:  "全件が1ページに収まる",
				limit: 10,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				var walked []string
				var cursor *repository.DiaryCursor
				for page := 0; page < len(seeded)+1; page++ {
					entries, err := f.Diary.ListEntries(ctx, repository.ListDiaryParams{
						UserID: user, Limit: tt.limit, Cursor: cursor,
					})
					if err != nil {
						t.Fatalf("ListEntries(limit=%d, page %d) error = %v, want nil",
							tt.limit, page, err)
					}
					if len(entries) == 0 {
						break
					}
					if len(entries) > tt.limit {
						t.Fatalf("ListEntries(limit=%d) returned %d entries, want at most %d",
							tt.limit, len(entries), tt.limit)
					}
					for i := 1; i < len(entries); i++ {
						if entries[i-1].CreatedAt.Before(entries[i].CreatedAt) {
							t.Errorf("limit=%d page %d is not newest-first at index %d",
								tt.limit, page, i)
						}
					}
					for _, e := range entries {
						walked = append(walked, e.ID)
					}
					last := entries[len(entries)-1]
					cursor = &repository.DiaryCursor{CreatedAt: last.CreatedAt, ID: last.ID}
				}
				if diff := cmp.Diff(want, walked); diff != "" {
					t.Errorf("limit=%d: paging lost, duplicated or reordered entries (-want +got):\n%s",
						tt.limit, diff)
				}
			})
		}
	})

	t.Run("ListExcludesDeletedAndOtherUsers", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)
		ctx := t.Context()
		user, stranger := f.NewUser(t), f.NewUser(t)
		entries := seed(t, f, user, 2, time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC))

		if err := f.Diary.SoftDeleteEntry(ctx, user, entries[0].ID); err != nil {
			t.Fatalf("SoftDeleteEntry() error = %v, want nil", err)
		}

		got, err := f.Diary.ListEntries(ctx, repository.ListDiaryParams{UserID: user, Limit: 10})
		if err != nil {
			t.Fatalf("ListEntries() error = %v, want nil", err)
		}
		if diff := cmp.Diff([]string{entries[1].ID}, ids(got)); diff != "" {
			t.Errorf("ListEntries() after a soft delete (-want +got):\n%s", diff)
		}

		strangers, err := f.Diary.ListEntries(ctx, repository.ListDiaryParams{UserID: stranger, Limit: 10})
		if err != nil {
			t.Fatalf("ListEntries(stranger) error = %v, want nil", err)
		}
		if len(strangers) != 0 {
			t.Errorf("a stranger sees %d entries, want 0", len(strangers))
		}
	})

	t.Run("EveryOperationIsOwnerScoped", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)
		ctx := t.Context()
		user, stranger := f.NewUser(t), f.NewUser(t)
		entry := seed(t, f, user, 1, time.Now().Add(-time.Hour).UTC())[0]

		// For a non-owner every operation must look like a missing row, so a 404
		// never reveals that the entry exists for somebody else.
		tests := []struct {
			op   string
			call func(userID string) error
		}{
			{
				op: "GetEntry",
				call: func(userID string) error {
					_, err := f.Diary.GetEntry(ctx, userID, entry.ID)
					return err
				},
			},
			{
				op: "UpdateEntry",
				call: func(userID string) error {
					_, err := f.Diary.UpdateEntry(ctx, userID, entry.ID,
						repository.UpdateDiaryParams{BodyText: "hijack"})
					return err
				},
			},
			{
				op: "SoftDeleteEntry",
				call: func(userID string) error {
					return f.Diary.SoftDeleteEntry(ctx, userID, entry.ID)
				},
			},
		}
		for _, tt := range tests {
			t.Run(tt.op, func(t *testing.T) {
				if err := tt.call(stranger); !errors.Is(err, repository.ErrNotFound) {
					t.Errorf("%s(stranger) error = %v, want %v", tt.op, err, repository.ErrNotFound)
				}
			})
		}

		// None of the rejected calls may have changed anything.
		owned, err := f.Diary.GetEntry(ctx, user, entry.ID)
		if err != nil {
			t.Fatalf("GetEntry(owner) error = %v, want nil", err)
		}
		if owned.BodyText != entry.BodyText {
			t.Errorf("body_text = %q after a stranger's update, want the original %q",
				owned.BodyText, entry.BodyText)
		}
	})

	t.Run("UpdateBumpsUpdatedAtAndClearsMood", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)
		ctx := t.Context()
		user := f.NewUser(t)

		entry, _, err := f.Diary.UpsertEntry(ctx, repository.UpsertDiaryParams{
			UserID: user, ClientID: clientID(1), BodyText: "v1",
			Mood: ptr("calm"), CreatedAt: time.Now().Add(-time.Hour).UTC(),
		})
		if err != nil {
			t.Fatalf("UpsertEntry() error = %v, want nil", err)
		}

		// Mood == nil clears the column.
		updated, err := f.Diary.UpdateEntry(ctx, user, entry.ID, repository.UpdateDiaryParams{BodyText: "v2"})
		if err != nil {
			t.Fatalf("UpdateEntry() error = %v, want nil", err)
		}
		if updated.BodyText != "v2" {
			t.Errorf("body_text = %q, want %q", updated.BodyText, "v2")
		}
		if updated.Mood != nil {
			t.Errorf("mood = %q after an update with a nil mood, want nil (cleared)", *updated.Mood)
		}
		// Sync is a watermark over updated_at, so it has to move.
		if !updated.UpdatedAt.After(entry.UpdatedAt) {
			t.Errorf("updated_at = %s, want it after the insert's %s",
				updated.UpdatedAt.UTC(), entry.UpdatedAt.UTC())
		}
	})

	t.Run("SoftDeleteIsIdempotent", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)
		ctx := t.Context()
		user := f.NewUser(t)
		entry := seed(t, f, user, 1, time.Now().Add(-time.Hour).UTC())[0]

		if err := f.Diary.SoftDeleteEntry(ctx, user, entry.ID); err != nil {
			t.Fatalf("first SoftDeleteEntry() error = %v, want nil", err)
		}
		if err := f.Diary.SoftDeleteEntry(ctx, user, entry.ID); err != nil {
			t.Errorf("second SoftDeleteEntry() error = %v, want nil", err)
		}
		if _, err := f.Diary.GetEntry(ctx, user, entry.ID); !errors.Is(err, repository.ErrNotFound) {
			t.Errorf("GetEntry() on a deleted entry error = %v, want %v", err, repository.ErrNotFound)
		}
		if _, err := f.Diary.UpdateEntry(ctx, user, entry.ID, repository.UpdateDiaryParams{BodyText: "x"}); !errors.Is(err, repository.ErrNotFound) {
			t.Errorf("UpdateEntry() on a deleted entry error = %v, want %v", err, repository.ErrNotFound)
		}
	})

	t.Run("ChangedSinceCarriesTombstonesOldestFirst", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)
		ctx := t.Context()
		user := f.NewUser(t)
		entries := seed(t, f, user, 2, time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC))

		// A zero watermark returns everything.
		all, err := f.Diary.ListChangedSince(ctx, user, time.Time{})
		if err != nil {
			t.Fatalf("ListChangedSince(epoch) error = %v, want nil", err)
		}
		if len(all) != 2 {
			t.Fatalf("ListChangedSince(epoch) returned %d entries, want 2", len(all))
		}
		for i := 1; i < len(all); i++ {
			if all[i].UpdatedAt.Before(all[i-1].UpdatedAt) {
				t.Errorf("ListChangedSince is not oldest-change-first at index %d", i)
			}
		}

		watermark := all[len(all)-1].UpdatedAt
		if err := f.Diary.SoftDeleteEntry(ctx, user, entries[0].ID); err != nil {
			t.Fatalf("SoftDeleteEntry() error = %v, want nil", err)
		}

		delta, err := f.Diary.ListChangedSince(ctx, user, watermark)
		if err != nil {
			t.Fatalf("ListChangedSince(watermark) error = %v, want nil", err)
		}
		if len(delta) != 1 {
			t.Fatalf("ListChangedSince(watermark) returned %d entries, want 1 tombstone", len(delta))
		}
		if delta[0].ID != entries[0].ID {
			t.Errorf("delta carries id %q, want the deleted %q", delta[0].ID, entries[0].ID)
		}
		// The tombstone is how other devices learn of the deletion.
		if delta[0].DeletedAt == nil {
			t.Error("the deleted entry's deleted_at is nil in the delta, want a tombstone timestamp")
		}
	})

	t.Run("CountByLocalDateGroupsInTheGivenZone", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)
		ctx := t.Context()
		user := f.NewUser(t)

		// Instants chosen so the local date differs in each direction:
		//   07-09 20:00Z -> 07-10 Tokyo (+09), 07-09 Honolulu (-10)
		//   07-10 16:00Z -> 07-11 Tokyo,       07-10 Honolulu
		//   07-10 18:00Z -> 07-11 Tokyo,       07-10 Honolulu
		//   07-10 05:00Z -> 07-10 Tokyo,       07-09 Honolulu  (rolls back)
		at := []time.Time{
			time.Date(2026, 7, 9, 20, 0, 0, 0, time.UTC),
			time.Date(2026, 7, 10, 16, 0, 0, 0, time.UTC),
			time.Date(2026, 7, 10, 18, 0, 0, 0, time.UTC),
			time.Date(2026, 7, 10, 5, 0, 0, 0, time.UTC),
		}
		for i, when := range at {
			if _, _, err := f.Diary.UpsertEntry(ctx, repository.UpsertDiaryParams{
				UserID: user, ClientID: clientID(i + 1), BodyText: "e", CreatedAt: when,
			}); err != nil {
				t.Fatalf("seeding entry at %s error = %v, want nil", when, err)
			}
		}

		tokyo, err := time.LoadLocation("Asia/Tokyo")
		if err != nil {
			t.Fatalf("LoadLocation(Asia/Tokyo) error = %v, want nil", err)
		}
		honolulu, err := time.LoadLocation("Pacific/Honolulu")
		if err != nil {
			t.Fatalf("LoadLocation(Pacific/Honolulu) error = %v, want nil", err)
		}
		lo := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
		hi := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)

		tests := []struct {
			name string
			loc  *time.Location
			want map[string]int
		}{
			{
				name: "UTC",
				loc:  time.UTC,
				want: map[string]int{"2026-07-09": 1, "2026-07-10": 3},
			},
			{
				name: "東（+09:00）は日付が進む",
				loc:  tokyo,
				want: map[string]int{"2026-07-10": 2, "2026-07-11": 2},
			},
			{
				name: "西（-10:00）は日付が戻る",
				loc:  honolulu,
				want: map[string]int{"2026-07-09": 2, "2026-07-10": 2},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got, err := f.Diary.CountByLocalDate(ctx, user, lo, hi, tt.loc)
				if err != nil {
					t.Fatalf("CountByLocalDate(%s) error = %v, want nil", tt.loc, err)
				}
				if diff := cmp.Diff(tt.want, got); diff != "" {
					t.Errorf("CountByLocalDate(%s) mismatch (-want +got):\n%s", tt.loc, diff)
				}
			})
		}
	})
}

// ids extracts entry ids so an ordering assertion reads as one list.
func ids(entries []repository.DiaryEntry) []string {
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		out = append(out, e.ID)
	}
	return out
}
