package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/0muji4/Runa/apps/go/internal/repository"
	"github.com/0muji4/Runa/apps/go/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	require.NoError(t, err)
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
				_, err := svc.CreateQuote(ctx, day(d), "月あかり")
				require.NoError(t, err)
			}
			if tt.seedSong {
				seedSong(t, svc, ctx, d, "夜想曲")
			}

			content, err := svc.Today(ctx, day(d))
			require.NoError(t, err)

			if tt.wantQuote == nil {
				assert.Nil(t, content.Quote)
			} else {
				require.NotNil(t, content.Quote)
				assert.Equal(t, *tt.wantQuote, content.Quote.BodyText)
			}
			if tt.wantSong == nil {
				assert.Nil(t, content.Song)
			} else {
				require.NotNil(t, content.Song)
				assert.Equal(t, *tt.wantSong, content.Song.Title)
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
			require.NoError(t, err)
			assert.Len(t, page1.Songs, tt.wantPage1Len)
			assert.Equal(t, tt.wantPage1Cursor, page1.NextCursor != nil)
			for i := 1; i < len(page1.Songs); i++ {
				assert.False(t, page1.Songs[i-1].Date.Before(page1.Songs[i].Date), "page1 not newest-first at index %d", i)
			}
			if !tt.wantPage1Cursor {
				return
			}

			page2, err := svc.Archive(ctx, tt.limit, page1.NextCursor)
			require.NoError(t, err)
			assert.Len(t, page2.Songs, tt.wantPage2Len)
			assert.Equal(t, tt.wantPage2Cursor, page2.NextCursor != nil)
			if len(page2.Songs) > 0 {
				assert.True(t, page2.Songs[0].Date.Before(page1.Songs[len(page1.Songs)-1].Date), "pages overlap across the cursor boundary")
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
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			assert.NoError(t, err)
		})
	}
}
