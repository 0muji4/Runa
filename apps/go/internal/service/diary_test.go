package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/0muji4/Runa/apps/go/internal/repository/memdiary"
	"github.com/0muji4/Runa/apps/go/internal/service"
)

const (
	userA = "11111111-1111-4111-8111-111111111111"
	userB = "22222222-2222-4222-8222-222222222222"
)

func clientID(n string) string { return "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaa" + n }

func newDiaryService() *service.DiaryService {
	return service.NewDiaryService(memdiary.New(), nil)
}

func ptr(s string) *string { return &s }

// TestDiaryCreateIdempotent verifies POST is idempotent by client_id: a repeat
// upserts the same row (created=false) carrying the latest body, never a dup.
func TestDiaryCreateIdempotent(t *testing.T) {
	svc := newDiaryService()
	ctx := context.Background()
	cid := clientID("1")

	first, created, err := svc.Create(ctx, userA, service.CreateDiaryInput{ClientID: cid, BodyText: "夜の記録"})
	if err != nil || !created {
		t.Fatalf("first create: created=%v err=%v, want created=true", created, err)
	}

	second, created, err := svc.Create(ctx, userA, service.CreateDiaryInput{ClientID: cid, BodyText: "夜の記録（推敲）"})
	if err != nil {
		t.Fatalf("second create: %v", err)
	}
	if created {
		t.Error("second create reported created=true, want false (idempotent upsert)")
	}
	if second.ID != first.ID {
		t.Errorf("id changed on repeat client_id: %q -> %q", first.ID, second.ID)
	}
	if second.BodyText != "夜の記録（推敲）" {
		t.Errorf("body not updated on repeat: %q", second.BodyText)
	}

	// Only one row should exist for the user.
	page, err := svc.List(ctx, userA, 0, nil)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(page.Entries) != 1 {
		t.Fatalf("list len = %d, want 1 (no duplicate)", len(page.Entries))
	}
}

// TestDiaryOwnership verifies another user can never see or mutate an entry:
// every cross-user access collapses to ErrDiaryNotFound.
func TestDiaryOwnership(t *testing.T) {
	svc := newDiaryService()
	ctx := context.Background()

	entry, _, err := svc.Create(ctx, userA, service.CreateDiaryInput{ClientID: clientID("1"), BodyText: "秘密"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	if _, err := svc.Get(ctx, userB, entry.ID); !errors.Is(err, service.ErrDiaryNotFound) {
		t.Errorf("cross-user Get err = %v, want ErrDiaryNotFound", err)
	}
	if _, err := svc.Update(ctx, userB, entry.ID, "改ざん", nil); !errors.Is(err, service.ErrDiaryNotFound) {
		t.Errorf("cross-user Update err = %v, want ErrDiaryNotFound", err)
	}
	if err := svc.Delete(ctx, userB, entry.ID); !errors.Is(err, service.ErrDiaryNotFound) {
		t.Errorf("cross-user Delete err = %v, want ErrDiaryNotFound", err)
	}

	// Owner still sees it intact.
	if got, err := svc.Get(ctx, userA, entry.ID); err != nil || got.BodyText != "秘密" {
		t.Errorf("owner Get after cross-user attempts: body=%q err=%v", got.BodyText, err)
	}
}

// TestDiaryListPaginationCursor walks two keyset pages newest-first and checks
// the cursor stitches them without overlap or gap.
func TestDiaryListPaginationCursor(t *testing.T) {
	svc := newDiaryService()
	ctx := context.Background()
	base := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)

	// Five entries with increasing created_at (newest = e5).
	for i := 1; i <= 5; i++ {
		_, _, err := svc.Create(ctx, userA, service.CreateDiaryInput{
			ClientID:  clientID(itoa(i)),
			BodyText:  "entry",
			CreatedAt: base.Add(time.Duration(i) * time.Minute),
		})
		if err != nil {
			t.Fatalf("create %d: %v", i, err)
		}
	}

	page1, err := svc.List(ctx, userA, 2, nil)
	if err != nil {
		t.Fatalf("page1: %v", err)
	}
	if len(page1.Entries) != 2 || page1.NextCursor == nil {
		t.Fatalf("page1 len=%d cursor=%v, want 2 entries + cursor", len(page1.Entries), page1.NextCursor)
	}
	// Newest first: the first entry authored last has the largest created_at.
	if !page1.Entries[0].CreatedAt.After(page1.Entries[1].CreatedAt) {
		t.Error("page1 not sorted newest-first")
	}

	page2, err := svc.List(ctx, userA, 2, page1.NextCursor)
	if err != nil {
		t.Fatalf("page2: %v", err)
	}
	if len(page2.Entries) != 2 {
		t.Fatalf("page2 len=%d, want 2", len(page2.Entries))
	}
	// No overlap: page2's newest is strictly older than page1's oldest.
	if !page2.Entries[0].CreatedAt.Before(page1.Entries[1].CreatedAt) {
		t.Error("pages overlap across the cursor boundary")
	}
}

// TestDiarySoftDeleteAndSync verifies a deleted entry drops out of the list but
// still rides the sync delta as a tombstone (deleted_at set).
func TestDiarySoftDeleteAndSync(t *testing.T) {
	svc := newDiaryService()
	ctx := context.Background()

	entry, _, err := svc.Create(ctx, userA, service.CreateDiaryInput{ClientID: clientID("1"), BodyText: "消す"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := svc.Delete(ctx, userA, entry.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	// Idempotent: deleting again still succeeds.
	if err := svc.Delete(ctx, userA, entry.ID); err != nil {
		t.Fatalf("second delete: %v", err)
	}

	page, err := svc.List(ctx, userA, 0, nil)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(page.Entries) != 0 {
		t.Errorf("deleted entry still in list (len=%d)", len(page.Entries))
	}

	delta, err := svc.Sync(ctx, userA, time.Time{})
	if err != nil {
		t.Fatalf("sync: %v", err)
	}
	if len(delta.Entries) != 1 || delta.Entries[0].DeletedAt == nil {
		t.Fatalf("sync did not carry the tombstone: %+v", delta.Entries)
	}
}

// TestDiarySyncDelta verifies `since` filtering: a second sync from the returned
// server_time yields only what changed after it.
func TestDiarySyncDelta(t *testing.T) {
	svc := newDiaryService()
	ctx := context.Background()

	entry, _, err := svc.Create(ctx, userA, service.CreateDiaryInput{ClientID: clientID("1"), BodyText: "v1"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	full, err := svc.Sync(ctx, userA, time.Time{})
	if err != nil {
		t.Fatalf("full sync: %v", err)
	}
	if len(full.Entries) != 1 {
		t.Fatalf("full sync len=%d, want 1", len(full.Entries))
	}

	if _, err := svc.Update(ctx, userA, entry.ID, "v2", ptr("calm")); err != nil {
		t.Fatalf("update: %v", err)
	}

	delta, err := svc.Sync(ctx, userA, full.ServerTime)
	if err != nil {
		t.Fatalf("delta sync: %v", err)
	}
	if len(delta.Entries) != 1 {
		t.Fatalf("delta sync len=%d, want 1 (only the update)", len(delta.Entries))
	}
	if delta.Entries[0].BodyText != "v2" || delta.Entries[0].Mood == nil || *delta.Entries[0].Mood != "calm" {
		t.Errorf("delta entry = %+v, want body v2 mood calm", delta.Entries[0])
	}

	// Nothing changed since the latest update.
	after, err := svc.Sync(ctx, userA, delta.ServerTime)
	if err != nil {
		t.Fatalf("empty sync: %v", err)
	}
	if len(after.Entries) != 0 {
		t.Errorf("expected empty delta, got %d", len(after.Entries))
	}
}

func itoa(i int) string { return string(rune('0' + i)) }
