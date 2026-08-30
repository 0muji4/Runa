package repository_test

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/0muji4/Runa/apps/go/internal/repository"
)

// Behaviour that only exists in SQL, so no fake can stand in for it.

func requirePostgres(t *testing.T) {
	t.Helper()
	if skipReason != "" {
		t.Skip(skipReason)
	}
}

// TestUpsertReportsInsertVsUpdate pins the (xmax = 0) trick in UpsertEntry, the
// sole source of the handler's 201-vs-200 answer.
func TestUpsertReportsInsertVsUpdate(t *testing.T) {
	requirePostgres(t)
	t.Parallel()

	f := newFixture(t)
	ctx := t.Context()
	user := f.NewUser(t)
	const client = "aaaaaaaa-aaaa-4aaa-8aaa-000000000001"
	authored := time.Now().Add(-time.Hour).UTC()

	entry, created, err := f.Diary.UpsertEntry(ctx, repository.UpsertDiaryParams{
		UserID: user, ClientID: client, BodyText: "v1", CreatedAt: authored,
	})
	if err != nil {
		t.Fatalf("first UpsertEntry() error = %v, want nil", err)
	}
	if !created {
		t.Error("first UpsertEntry() created = false, want true")
	}

	_, created, err = f.Diary.UpsertEntry(ctx, repository.UpsertDiaryParams{
		UserID: user, ClientID: client, BodyText: "v2", CreatedAt: authored,
	})
	if err != nil {
		t.Fatalf("second UpsertEntry() error = %v, want nil", err)
	}
	if created {
		t.Error("second UpsertEntry() created = true, want false")
	}

	// A soft-deleted row still occupies the (user_id, client_id) unique index, so
	// re-syncing that client_id is an UPDATE, not a resurrection as a new entry.
	if err := f.Diary.SoftDeleteEntry(ctx, user, entry.ID); err != nil {
		t.Fatalf("SoftDeleteEntry() error = %v, want nil", err)
	}
	revived, created, err := f.Diary.UpsertEntry(ctx, repository.UpsertDiaryParams{
		UserID: user, ClientID: client, BodyText: "v3", CreatedAt: authored,
	})
	if err != nil {
		t.Fatalf("UpsertEntry() after a soft delete error = %v, want nil", err)
	}
	if created {
		t.Error("UpsertEntry() after a soft delete created = true, want false (the row still exists)")
	}
	if revived.ID != entry.ID {
		t.Errorf("id = %q after re-upserting a deleted client_id, want the original %q",
			revived.ID, entry.ID)
	}
}

// TestKeysetCursorBreaksTiesByID pins the (created_at, id) < ($1, $2) row
// comparison, which a naive "created_at < cursor" gets wrong for tied rows.
func TestKeysetCursorBreaksTiesByID(t *testing.T) {
	requirePostgres(t)
	t.Parallel()

	f := newFixture(t)
	ctx := t.Context()
	user := f.NewUser(t)

	// Sharing one created_at leaves only the id tiebreak to order them.
	const tied = 5
	sameInstant := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	for i := 1; i <= tied; i++ {
		if _, _, err := f.Diary.UpsertEntry(ctx, repository.UpsertDiaryParams{
			UserID:    user,
			ClientID:  fmt.Sprintf("aaaaaaaa-aaaa-4aaa-8aaa-%012d", i),
			BodyText:  fmt.Sprintf("tied %d", i),
			CreatedAt: sameInstant,
		}); err != nil {
			t.Fatalf("seeding tied entry %d error = %v, want nil", i, err)
		}
	}

	var walked []string
	var cursor *repository.DiaryCursor
	for page := 0; page < tied+1; page++ {
		got, err := f.Diary.ListEntries(ctx, repository.ListDiaryParams{
			UserID: user, Limit: 2, Cursor: cursor,
		})
		if err != nil {
			t.Fatalf("ListEntries(page %d) error = %v, want nil", page, err)
		}
		if len(got) == 0 {
			break
		}
		for _, e := range got {
			walked = append(walked, e.ID)
		}
		last := got[len(got)-1]
		cursor = &repository.DiaryCursor{CreatedAt: last.CreatedAt, ID: last.ID}
	}

	if len(walked) != tied {
		t.Errorf("paging over %d rows sharing one created_at walked %d ids, want %d",
			tied, len(walked), tied)
	}
	seen := make(map[string]bool, len(walked))
	for _, id := range walked {
		if seen[id] {
			t.Errorf("id %q returned twice across the cursor boundary", id)
		}
		seen[id] = true
	}
	for i := 1; i < len(walked); i++ {
		if walked[i-1] <= walked[i] {
			t.Errorf("ids are not strictly descending at index %d: %q then %q",
				i, walked[i-1], walked[i])
		}
	}
}

// TestCountByLocalDateUsesPostgresZoneNames pins the AT TIME ZONE grouping. The
// calendar endpoint passes loc.String() into the query, and Go and Postgres
// resolve zone names from separate tzdata.
func TestCountByLocalDateUsesPostgresZoneNames(t *testing.T) {
	requirePostgres(t)
	t.Parallel()

	f := newFixture(t)
	ctx := t.Context()
	user := f.NewUser(t)

	// Either side of a DST boundary, so the zone shifts by a different amount.
	at := []time.Time{
		time.Date(2026, 1, 15, 4, 30, 0, 0, time.UTC), // NY: 2026-01-14 23:30 (UTC-5)
		time.Date(2026, 7, 15, 3, 30, 0, 0, time.UTC), // NY: 2026-07-14 23:30 (UTC-4)
	}
	for i, when := range at {
		if _, _, err := f.Diary.UpsertEntry(ctx, repository.UpsertDiaryParams{
			UserID:    user,
			ClientID:  fmt.Sprintf("aaaaaaaa-aaaa-4aaa-8aaa-%012d", i+1),
			BodyText:  "e",
			CreatedAt: when,
		}); err != nil {
			t.Fatalf("seeding entry at %s error = %v, want nil", when, err)
		}
	}

	newYork, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatalf("LoadLocation(America/New_York) error = %v, want nil", err)
	}
	lo := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	hi := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)

	got, err := f.Diary.CountByLocalDate(ctx, user, lo, hi, newYork)
	if err != nil {
		t.Fatalf("CountByLocalDate(America/New_York) error = %v, want nil", err)
	}
	want := map[string]int{"2026-01-14": 1, "2026-07-14": 1}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("CountByLocalDate(America/New_York) mismatch (-want +got):\n%s", diff)
	}
}

// TestDeleteUserCascades pins the ON DELETE CASCADE account deletion relies on:
// the service removes only the user row and expects the rest to follow.
func TestDeleteUserCascades(t *testing.T) {
	requirePostgres(t)
	t.Parallel()

	f := newFixture(t)
	ctx := t.Context()
	user := f.NewUser(t)

	if _, _, err := f.Diary.UpsertEntry(ctx, repository.UpsertDiaryParams{
		UserID:   user,
		ClientID: "aaaaaaaa-aaaa-4aaa-8aaa-000000000001",
		BodyText: "月がきれい", CreatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("seeding a diary entry error = %v, want nil", err)
	}
	if _, err := f.Gallery.InsertImage(ctx, repository.InsertGalleryParams{
		UserID: user, ObjectKey: "gallery/" + user + "/k1",
		Width: 1, Height: 1, Theme: "pink",
	}); err != nil {
		t.Fatalf("seeding a gallery image error = %v, want nil", err)
	}
	if _, err := f.Devices.UpsertDevice(ctx, repository.UpsertDeviceParams{
		UserID: user, PushToken: "token", Platform: "ios",
		NotifyTime: "22:00", Enabled: true,
	}); err != nil {
		t.Fatalf("seeding a device error = %v, want nil", err)
	}
	song, err := f.Today.InsertSong(ctx, repository.InsertSongParams{
		Date: time.Date(2026, 7, 11, 0, 0, 0, 0, time.UTC), Title: "夜想曲", Artist: "月詠",
		ArtworkURL: "https://x/a.jpg", AudioURL: "https://x/a.mp3",
	})
	if err != nil {
		t.Fatalf("seeding a song error = %v, want nil", err)
	}
	if err := f.Today.RecordPlay(ctx, user, song.ID, time.Now().UTC()); err != nil {
		t.Fatalf("seeding play history error = %v, want nil", err)
	}

	if err := f.Auth.DeleteUser(ctx, user); err != nil {
		t.Fatalf("DeleteUser(%q) error = %v, want nil", user, err)
	}

	// Account deletion purges storage from this list, so a row surviving the
	// cascade would leave an object nothing ever removes.
	keys, err := f.Gallery.ListObjectKeys(ctx, user)
	if err != nil {
		t.Fatalf("ListObjectKeys() after cascade error = %v, want nil", err)
	}
	if len(keys) != 0 {
		t.Errorf("gallery_images kept %d rows after the user was deleted, want 0: %v", len(keys), keys)
	}
	entries, err := f.Diary.ListEntries(ctx, repository.ListDiaryParams{UserID: user, Limit: 10})
	if err != nil {
		t.Fatalf("ListEntries() after cascade error = %v, want nil", err)
	}
	if len(entries) != 0 {
		t.Errorf("diary_entries kept %d rows after the user was deleted, want 0", len(entries))
	}
	if _, err := f.Auth.GetUserByID(ctx, user); !errors.Is(err, repository.ErrNotFound) {
		t.Errorf("GetUserByID() after delete error = %v, want %v", err, repository.ErrNotFound)
	}

	// Curated content is not user-owned and must survive its listener.
	if _, err := f.Today.GetSongForDate(ctx, song.Date); err != nil {
		t.Errorf("GetSongForDate() after the listener was deleted error = %v, want nil", err)
	}
}
