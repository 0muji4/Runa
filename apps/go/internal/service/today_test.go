package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/0muji4/Runa/apps/go/internal/itunes"
	"github.com/0muji4/Runa/apps/go/internal/repository"
	"github.com/0muji4/Runa/apps/go/internal/service"
	"github.com/google/go-cmp/cmp"
)

func day(s string) time.Time {
	d, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return d
}

func seedSong(t *testing.T, svc *service.TodayService, lookup *fakeLookup, ctx context.Context, date, title string) repository.Song {
	t.Helper()
	song, err := svc.CreateSong(ctx, day(date), lookup.add(title))
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

			svc, lookup, _ := newTodayService()
			ctx := context.Background()
			if tt.seedQuote {
				if _, err := svc.CreateQuote(ctx, day(d), "月あかり"); err != nil {
					t.Fatalf("CreateQuote(%q) error = %v, want nil", d, err)
				}
			}
			if tt.seedSong {
				seedSong(t, svc, lookup, ctx, d, "夜想曲")
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

			svc, lookup, _ := newTodayService()
			ctx := context.Background()
			for _, d := range tt.seedDates {
				seedSong(t, svc, lookup, ctx, d, d)
			}

			page1, err := svc.Archive(ctx, day("2026-12-31"), tt.limit, nil)
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

			page2, err := svc.Archive(ctx, day("2026-12-31"), tt.limit, page1.NextCursor)
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

			svc, lookup, _ := newTodayService()
			ctx := context.Background()
			songID := tt.songID
			if tt.seedSong {
				songID = seedSong(t, svc, lookup, ctx, "2026-07-11", "夜想曲").ID
			}

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

func TestTodayService_CreateSong(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		prepare func(lookup *fakeLookup) int64
		wantErr error
	}{
		{
			name: "Appleにある曲は楽曲情報ごと登録される",
			prepare: func(lookup *fakeLookup) int64 {
				return lookup.add("夜想曲")
			},
			wantErr: nil,
		},
		{
			name: "Appleに無い曲はErrTrackNotFound",
			prepare: func(lookup *fakeLookup) int64 {
				return 999
			},
			wantErr: service.ErrTrackNotFound,
		},
		{
			name: "試聴の無い曲はErrTrackNoPreview",
			prepare: func(lookup *fakeLookup) int64 {
				id := lookup.add("夜想曲")
				lookup.set(id, itunes.Track{TrackID: id, Title: "夜想曲", Artist: "月詠", ArtworkURL: "https://x/a.jpg", StoreURL: "https://x/s"})
				return id
			},
			wantErr: service.ErrTrackNoPreview,
		},
		{
			name: "Appleが応答しなければErrTrackLookupFailed",
			prepare: func(lookup *fakeLookup) int64 {
				id := lookup.add("夜想曲")
				lookup.setErr(errors.New("dial tcp: i/o timeout"))
				return id
			},
			wantErr: service.ErrTrackLookupFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc, lookup, _ := newTodayService()
			ctx := context.Background()
			trackID := tt.prepare(lookup)

			song, err := svc.CreateSong(ctx, day("2026-07-11"), trackID)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("CreateSong() error = %v, want %v", err, tt.wantErr)
				}
				content, err := svc.Today(ctx, day("2026-07-11"))
				if err != nil {
					t.Fatalf("Today() error = %v, want nil", err)
				}
				if content.Song != nil {
					t.Errorf("Today().Song = %+v after a failed registration, want nil", content.Song)
				}
				return
			}
			if err != nil {
				t.Fatalf("CreateSong() error = %v, want nil", err)
			}
			want := repository.SongMetadata{
				Title: "夜想曲", Artist: "月詠",
				ArtworkURL: "https://x/夜想曲.jpg", PreviewURL: "https://x/夜想曲.m4a",
				StoreURL: "https://music.apple.com/jp/x?i=夜想曲", ResolvedAt: testNow,
			}
			if song.ITunesTrackID != trackID {
				t.Errorf("song.ITunesTrackID = %d, want %d", song.ITunesTrackID, trackID)
			}
			if diff := cmp.Diff(want, song.SongMetadata); diff != "" {
				t.Errorf("song metadata mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestTodayService_RefreshesStaleMetadataOnRead(t *testing.T) {
	t.Parallel()

	const d = "2026-07-11"
	tests := []struct {
		name        string
		advance     time.Duration
		appleDown   bool
		wantLookups int    // lookups triggered by the read (registration's own excluded)
		wantPreview string // preview after the read
	}{
		{
			name:        "24時間未満なら取得し直さない",
			advance:     service.RefreshTTL - time.Minute,
			wantLookups: 0,
			wantPreview: "https://x/夜想曲.m4a",
		},
		{
			name:        "24時間を過ぎた読み出しで楽曲情報が新しくなる",
			advance:     service.RefreshTTL,
			wantLookups: 1,
			wantPreview: "https://x/夜想曲-v2.m4a",
		},
		{
			name:        "Appleが応答しなくても保存済みの楽曲情報を返す",
			advance:     service.RefreshTTL,
			appleDown:   true,
			wantLookups: 1,
			wantPreview: "https://x/夜想曲.m4a",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc, lookup, clock := newTodayService()
			ctx := context.Background()
			song := seedSong(t, svc, lookup, ctx, d, "夜想曲")
			registrationLookups := lookup.callCount()

			lookup.set(song.ITunesTrackID, itunes.Track{
				TrackID: song.ITunesTrackID, Title: "夜想曲", Artist: "月詠",
				ArtworkURL: "https://x/夜想曲.jpg", PreviewURL: "https://x/夜想曲-v2.m4a",
				StoreURL: "https://music.apple.com/jp/x?i=夜想曲",
			})
			if tt.appleDown {
				lookup.setErr(errors.New("503"))
			}
			clock.Advance(tt.advance)

			// The read answers with what is stored; the inline refresh lands before the next read.
			first, err := svc.Today(ctx, day(d))
			if err != nil {
				t.Fatalf("Today() error = %v, want nil", err)
			}
			if first.Song == nil || first.Song.PreviewURL != "https://x/夜想曲.m4a" {
				t.Fatalf("first Today().Song = %+v, want the stored preview", first.Song)
			}
			if got := lookup.callCount() - registrationLookups; got != tt.wantLookups {
				t.Errorf("read triggered %d lookups, want %d", got, tt.wantLookups)
			}

			second, err := svc.Today(ctx, day(d))
			if err != nil {
				t.Fatalf("second Today() error = %v, want nil", err)
			}
			if second.Song == nil || second.Song.PreviewURL != tt.wantPreview {
				t.Errorf("second Today().Song.PreviewURL = %q, want %q", second.Song.PreviewURL, tt.wantPreview)
			}
		})
	}
}

func TestTodayService_RefreshKeepsVerifiedArtwork(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		stored      string // artwork already in the row
		fetched     string // artwork the re-fetch returns
		wantArtwork string
	}{
		{
			name:        "600pxの確認だけ通らなかった再取得は保存済みの600pxを保つ",
			stored:      "https://cdn/a/600x600bb.jpg",
			fetched:     "https://cdn/a/100x100bb.jpg",
			wantArtwork: "https://cdn/a/600x600bb.jpg",
		},
		{
			name:        "アートワーク自体が変わったら新しい方を保存する",
			stored:      "https://cdn/a/600x600bb.jpg",
			fetched:     "https://cdn/b/100x100bb.jpg",
			wantArtwork: "https://cdn/b/100x100bb.jpg",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc, lookup, clock := newTodayService()
			ctx := context.Background()
			id := lookup.add("夜想曲")
			lookup.set(id, itunes.Track{TrackID: id, Title: "夜想曲", Artist: "月詠", ArtworkURL: tt.stored, PreviewURL: "https://x/p.m4a", StoreURL: "https://x/s"})
			if _, err := svc.CreateSong(ctx, day("2026-07-11"), id); err != nil {
				t.Fatalf("CreateSong() error = %v, want nil", err)
			}

			lookup.set(id, itunes.Track{TrackID: id, Title: "夜想曲", Artist: "月詠", ArtworkURL: tt.fetched, PreviewURL: "https://x/p.m4a", StoreURL: "https://x/s"})
			clock.Advance(service.RefreshTTL)
			if _, err := svc.Today(ctx, day("2026-07-11")); err != nil {
				t.Fatalf("Today() error = %v, want nil", err)
			}
			content, err := svc.Today(ctx, day("2026-07-11"))
			if err != nil {
				t.Fatalf("second Today() error = %v, want nil", err)
			}
			if content.Song.ArtworkURL != tt.wantArtwork {
				t.Errorf("artwork after refresh = %q, want %q", content.Song.ArtworkURL, tt.wantArtwork)
			}
		})
	}
}

func TestTodayService_RefreshBackoffAndBatching(t *testing.T) {
	t.Parallel()

	t.Run("失敗した曲は5分間は取得し直さない", func(t *testing.T) {
		t.Parallel()
		svc, lookup, clock := newTodayService()
		ctx := context.Background()
		seedSong(t, svc, lookup, ctx, "2026-07-11", "夜想曲")
		base := lookup.callCount()

		lookup.setErr(errors.New("503"))
		clock.Advance(service.RefreshTTL)
		for range 3 {
			if _, err := svc.Today(ctx, day("2026-07-11")); err != nil {
				t.Fatalf("Today() error = %v, want nil", err)
			}
		}
		if got := lookup.callCount() - base; got != 1 {
			t.Errorf("three reads during an outage triggered %d lookups, want 1", got)
		}

		clock.Advance(service.RefreshRetryAfter)
		if _, err := svc.Today(ctx, day("2026-07-11")); err != nil {
			t.Fatalf("Today() error = %v, want nil", err)
		}
		if got := lookup.callCount() - base; got != 2 {
			t.Errorf("a read after the retry window triggered %d lookups in total, want 2", got)
		}
	})

	t.Run("アーカイブの1ページは1回の問い合わせでまとめて取得し直す", func(t *testing.T) {
		t.Parallel()
		svc, lookup, clock := newTodayService()
		ctx := context.Background()
		for _, d := range []string{"2026-07-09", "2026-07-10", "2026-07-11"} {
			seedSong(t, svc, lookup, ctx, d, d)
		}
		base := lookup.callCount()

		clock.Advance(service.RefreshTTL)
		if _, err := svc.Archive(ctx, day("2026-12-31"), 10, nil); err != nil {
			t.Fatalf("Archive() error = %v, want nil", err)
		}
		if got := lookup.callCount() - base; got != 1 {
			t.Fatalf("archive page triggered %d lookups, want 1", got)
		}
		if got := len(lookup.lastCall()); got != 3 {
			t.Errorf("the batch lookup carried %d ids, want 3", got)
		}
	})
}
