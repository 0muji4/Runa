package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/0muji4/Runa/apps/go/internal/repository"
	"github.com/0muji4/Runa/apps/go/internal/repository/memtoday"
	"github.com/0muji4/Runa/apps/go/internal/service"
)

const todayUser = "11111111-1111-4111-8111-111111111111"

func newTodayService() *service.TodayService {
	return service.NewTodayService(memtoday.New(), nil)
}

func day(s string) time.Time {
	d, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return d
}

// TestTodayMissingContentIsNotAnError verifies an unseeded day yields nil quote
// and song (not an error) so the home screen still renders.
func TestTodayMissingContentIsNotAnError(t *testing.T) {
	svc := newTodayService()
	content, err := svc.Today(context.Background(), day("2026-07-11"))
	if err != nil {
		t.Fatalf("Today on empty store: %v", err)
	}
	if content.Quote != nil || content.Song != nil {
		t.Fatalf("content = %+v, want nil quote and song", content)
	}
}

// TestTodayReturnsSeededContent verifies the exact-date lookup returns what was
// curated for that day.
func TestTodayReturnsSeededContent(t *testing.T) {
	svc := newTodayService()
	ctx := context.Background()
	if _, err := svc.CreateQuote(ctx, day("2026-07-11"), "月あかり"); err != nil {
		t.Fatalf("CreateQuote: %v", err)
	}
	if _, err := svc.CreateSong(ctx, repository.InsertSongParams{
		Date: day("2026-07-11"), Title: "夜想曲", Artist: "月詠",
		ArtworkURL: "https://x/a.jpg", AudioURL: "https://x/a.mp3",
	}); err != nil {
		t.Fatalf("CreateSong: %v", err)
	}

	content, err := svc.Today(ctx, day("2026-07-11"))
	if err != nil {
		t.Fatalf("Today: %v", err)
	}
	if content.Quote == nil || content.Quote.BodyText != "月あかり" {
		t.Fatalf("quote = %+v, want the seeded quote", content.Quote)
	}
	if content.Song == nil || content.Song.Title != "夜想曲" {
		t.Fatalf("song = %+v, want the seeded song", content.Song)
	}
}

// TestArchivePagingBoundary verifies the archive clamps the page to the limit,
// emits a next cursor only when more rows remain, and orders newest first.
func TestArchivePagingBoundary(t *testing.T) {
	svc := newTodayService()
	ctx := context.Background()
	for _, d := range []string{"2026-07-09", "2026-07-10", "2026-07-11"} {
		if _, err := svc.CreateSong(ctx, repository.InsertSongParams{
			Date: day(d), Title: d, Artist: "月詠",
			ArtworkURL: "https://x/a.jpg", AudioURL: "https://x/a.mp3",
		}); err != nil {
			t.Fatalf("CreateSong %s: %v", d, err)
		}
	}

	page1, err := svc.Archive(ctx, 2, nil)
	if err != nil {
		t.Fatalf("Archive page1: %v", err)
	}
	if len(page1.Songs) != 2 || page1.NextCursor == nil {
		t.Fatalf("page1 = %d songs, cursor=%v; want 2 and a cursor", len(page1.Songs), page1.NextCursor)
	}
	if page1.Songs[0].Date.Before(page1.Songs[1].Date) {
		t.Fatalf("page1 not newest-first: %v then %v", page1.Songs[0].Date, page1.Songs[1].Date)
	}

	page2, err := svc.Archive(ctx, 2, page1.NextCursor)
	if err != nil {
		t.Fatalf("Archive page2: %v", err)
	}
	if len(page2.Songs) != 1 || page2.NextCursor != nil {
		t.Fatalf("page2 = %d songs, cursor=%v; want 1 and no cursor", len(page2.Songs), page2.NextCursor)
	}
}

// TestMarkPlayedUnknownSong verifies a play against an unknown song id maps to
// ErrSongNotFound (a 404 at the handler).
func TestMarkPlayedUnknownSong(t *testing.T) {
	svc := newTodayService()
	err := svc.MarkPlayed(context.Background(), todayUser, "no-such-song", time.Time{})
	if !errors.Is(err, service.ErrSongNotFound) {
		t.Fatalf("MarkPlayed unknown song err = %v, want ErrSongNotFound", err)
	}
}
