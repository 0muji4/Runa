package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/0muji4/Runa/apps/go/internal/repository"
	"github.com/0muji4/Runa/apps/go/internal/service"
)

func day(s string) time.Time {
	d, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return d
}

func seedSong(t *testing.T, svc *service.TodayService, ctx context.Context, date, title string) repository.Song {
	t.Helper()
	song, err := svc.CreateSong(ctx, repository.InsertSongParams{
		Date: day(date), Title: title, Artist: "月詠",
		ArtworkURL: "https://x/a.jpg", AudioURL: "https://x/a.mp3",
	})
	if err != nil {
		t.Fatalf("CreateSong(%q, %q) error = %v, want nil", date, title, err)
	}
	return song
}

func TestTodayService_Today(t *testing.T) {
	t.Parallel()

	const d = "2026-07-11"
	tests := []struct {
		name      string
		seedQuote bool
		seedSong  bool
		wantQuote *string
		wantSong  *string
	}{
		{
			name:      "未登録の日はquoteもsongもnilを返す",
			seedQuote: false,
			seedSong:  false,
			wantQuote: nil,
			wantSong:  nil,
		},
		{
			name:      "登録済みの日はquoteとsongを返す",
			seedQuote: true,
			seedSong:  true,
			wantQuote: ptr("月あかり"),
			wantSong:  ptr("夜想曲"),
		},
		{
			name:      "quoteのみの日はsongがnil",
			seedQuote: true,
			seedSong:  false,
			wantQuote: ptr("月あかり"),
			wantSong:  nil,
		},
		{
			name:      "songのみの日はquoteがnil",
			seedQuote: false,
			seedSong:  true,
			wantQuote: nil,
			wantSong:  ptr("夜想曲"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc := newTodayService()
			ctx := context.Background()
			if tt.seedQuote {
				if _, err := svc.CreateQuote(ctx, day(d), "月あかり"); err != nil {
					t.Fatalf("CreateQuote(%q) error = %v, want nil", d, err)
				}
			}
			if tt.seedSong {
				seedSong(t, svc, ctx, d, "夜想曲")
			}

			content, err := svc.Today(ctx, day(d))
			if err != nil {
				t.Fatalf("Today(%q) error = %v, want nil", d, err)
			}

			switch {
			case tt.wantQuote == nil && content.Quote != nil:
				t.Errorf("Today(%q).Quote = %+v, want nil", d, content.Quote)
			case tt.wantQuote != nil && content.Quote == nil:
				t.Errorf("Today(%q).Quote = nil, want body_text %q", d, *tt.wantQuote)
			case tt.wantQuote != nil && content.Quote.BodyText != *tt.wantQuote:
				t.Errorf("Today(%q).Quote.body_text = %q, want %q",
					d, content.Quote.BodyText, *tt.wantQuote)
			}
			switch {
			case tt.wantSong == nil && content.Song != nil:
				t.Errorf("Today(%q).Song = %+v, want nil", d, content.Song)
			case tt.wantSong != nil && content.Song == nil:
				t.Errorf("Today(%q).Song = nil, want title %q", d, *tt.wantSong)
			case tt.wantSong != nil && content.Song.Title != *tt.wantSong:
				t.Errorf("Today(%q).Song.title = %q, want %q",
					d, content.Song.Title, *tt.wantSong)
			}
		})
	}
}

func TestTodayService_Archive(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		seedDates       []string
		limit           int
		wantPage1Len    int
		wantPage1Cursor bool
		wantPage2Len    int
		wantPage2Cursor bool
	}{
		{
			name:            "空アーカイブは何も返さない",
			seedDates:       nil,
			limit:           2,
			wantPage1Len:    0,
			wantPage1Cursor: false,
			wantPage2Len:    0,
			wantPage2Cursor: false,
		},
		{
			name:            "アーカイブは新しい順にページングする",
			seedDates:       []string{"2026-07-09", "2026-07-10", "2026-07-11"},
			limit:           2,
			wantPage1Len:    2,
			wantPage1Cursor: true,
			wantPage2Len:    1,
			wantPage2Cursor: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc := newTodayService()
			ctx := context.Background()
			for _, d := range tt.seedDates {
				seedSong(t, svc, ctx, d, d)
			}

			page1, err := svc.Archive(ctx, tt.limit, nil)
			if err != nil {
				t.Fatalf("Archive(limit=%d) error = %v, want nil", tt.limit, err)
			}
			if len(page1.Songs) != tt.wantPage1Len {
				t.Errorf("Archive(limit=%d) page 1 returned %d songs, want %d",
					tt.limit, len(page1.Songs), tt.wantPage1Len)
			}
			if got := page1.NextCursor != nil; got != tt.wantPage1Cursor {
				t.Errorf("Archive(limit=%d) page 1 has a next cursor = %t, want %t",
					tt.limit, got, tt.wantPage1Cursor)
			}
			for i := 1; i < len(page1.Songs); i++ {
				if page1.Songs[i-1].Date.Before(page1.Songs[i].Date) {
					t.Errorf("Archive() page 1 is not newest-first at index %d: %s before %s",
						i, page1.Songs[i-1].Date, page1.Songs[i].Date)
				}
			}
			if !tt.wantPage1Cursor {
				return
			}

			page2, err := svc.Archive(ctx, tt.limit, page1.NextCursor)
			if err != nil {
				t.Fatalf("Archive(limit=%d, cursor) error = %v, want nil", tt.limit, err)
			}
			if len(page2.Songs) != tt.wantPage2Len {
				t.Errorf("Archive(limit=%d, cursor) page 2 returned %d songs, want %d",
					tt.limit, len(page2.Songs), tt.wantPage2Len)
			}
			if got := page2.NextCursor != nil; got != tt.wantPage2Cursor {
				t.Errorf("Archive(limit=%d, cursor) page 2 has a next cursor = %t, want %t",
					tt.limit, got, tt.wantPage2Cursor)
			}
			if len(page2.Songs) > 0 {
				lastOfPage1 := page1.Songs[len(page1.Songs)-1].Date
				if !page2.Songs[0].Date.Before(lastOfPage1) {
					t.Errorf("pages overlap across the cursor boundary: page 2 starts at %s, page 1 ended at %s",
						page2.Songs[0].Date, lastOfPage1)
				}
			}
		})
	}
}

func TestTodayService_MarkPlayed(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		seedSong bool
		songID   string
		wantErr  error
	}{
		{
			name:     "未知のsongはErrSongNotFound",
			seedSong: false,
			songID:   "no-such-song",
			wantErr:  service.ErrSongNotFound,
		},
		{
			name:     "既知のsongは再生を記録する",
			seedSong: true,
			songID:   "",
			wantErr:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc := newTodayService()
			ctx := context.Background()
			songID := tt.songID
			if tt.seedSong {
				songID = seedSong(t, svc, ctx, "2026-07-11", "夜想曲").ID
			}

			// A zero playedAt exercises the default-to-server-clock branch.
			err := svc.MarkPlayed(ctx, userA, songID, time.Time{})
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("MarkPlayed(%q) error = %v, want %v", songID, err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("MarkPlayed(%q) error = %v, want nil", songID, err)
			}
		})
	}
}
