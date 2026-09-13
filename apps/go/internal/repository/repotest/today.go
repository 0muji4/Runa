package repotest

import (
	"errors"
	"testing"
	"time"

	"github.com/0muji4/Runa/apps/go/internal/repository"
	"github.com/google/go-cmp/cmp"
)

// day builds a UTC midnight instant, the shape the date columns store.
func day(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// songParams builds an InsertSongParams with metadata derived from a label, so a
// test can tell rows apart by title/URL without spelling every field out.
func songParams(date time.Time, trackID int64, label string) repository.InsertSongParams {
	return repository.InsertSongParams{
		Date:          date,
		ITunesTrackID: trackID,
		SongMetadata: repository.SongMetadata{
			Title:      label,
			Artist:     "月詠",
			ArtworkURL: "https://x/" + label + ".jpg",
			PreviewURL: "https://x/" + label + ".m4a",
			StoreURL:   "https://music.apple.com/jp/album/x?i=" + label,
			ResolvedAt: date,
		},
	}
}

// RunTodayStoreSuite exercises the TodayStore contract.
func RunTodayStoreSuite(t *testing.T, newFixture NewFixture) {
	t.Run("QuoteUpsertsByDate", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)
		ctx := t.Context()
		date := day(2026, 7, 11)

		first, err := f.Today.InsertQuote(ctx, repository.InsertQuoteParams{Date: date, BodyText: "v1"})
		if err != nil {
			t.Fatalf("InsertQuote() error = %v, want nil", err)
		}
		// One quote per calendar day: re-inserting replaces it.
		if _, err := f.Today.InsertQuote(ctx, repository.InsertQuoteParams{Date: date, BodyText: "v2"}); err != nil {
			t.Fatalf("second InsertQuote() error = %v, want nil", err)
		}

		got, err := f.Today.GetQuoteForDate(ctx, date)
		if err != nil {
			t.Fatalf("GetQuoteForDate() error = %v, want nil", err)
		}
		if got.BodyText != "v2" {
			t.Errorf("quote body_text = %q, want %q", got.BodyText, "v2")
		}
		if got.ID != first.ID {
			t.Errorf("quote id = %q, want the original %q", got.ID, first.ID)
		}
		if !got.Date.Equal(date) {
			t.Errorf("quote date = %s, want %s", got.Date.UTC(), date)
		}
	})

	t.Run("SongUpsertsByDate", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)
		ctx := t.Context()
		date := day(2026, 7, 11)

		if _, err := f.Today.InsertSong(ctx, songParams(date, 1001, "夜想曲")); err != nil {
			t.Fatalf("InsertSong() error = %v, want nil", err)
		}
		second := songParams(date, 1002, "薄明")
		if _, err := f.Today.InsertSong(ctx, second); err != nil {
			t.Fatalf("second InsertSong() error = %v, want nil", err)
		}

		got, err := f.Today.GetSongForDate(ctx, date)
		if err != nil {
			t.Fatalf("GetSongForDate() error = %v, want nil", err)
		}
		if got.ITunesTrackID != second.ITunesTrackID {
			t.Errorf("song itunes_track_id = %d, want %d", got.ITunesTrackID, second.ITunesTrackID)
		}
		if diff := cmp.Diff(second.SongMetadata, got.SongMetadata); diff != "" {
			t.Errorf("song metadata mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("SongMetadataUpdatesInPlace", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)
		ctx := t.Context()
		date := day(2026, 7, 11)

		song, err := f.Today.InsertSong(ctx, songParams(date, 1001, "夜想曲"))
		if err != nil {
			t.Fatalf("InsertSong() error = %v, want nil", err)
		}
		// A background re-fetch replaces the metadata but keeps the row (id and
		// track id), so play history stays attached.
		refreshed := songParams(date, 1001, "夜想曲-v2").SongMetadata
		refreshed.ResolvedAt = date.Add(48 * time.Hour)
		if err := f.Today.UpdateSongMetadata(ctx, song.ID, refreshed); err != nil {
			t.Fatalf("UpdateSongMetadata() error = %v, want nil", err)
		}

		got, err := f.Today.GetSongForDate(ctx, date)
		if err != nil {
			t.Fatalf("GetSongForDate() error = %v, want nil", err)
		}
		if got.ID != song.ID || got.ITunesTrackID != song.ITunesTrackID {
			t.Errorf("song identity = (%s, %d), want (%s, %d)", got.ID, got.ITunesTrackID, song.ID, song.ITunesTrackID)
		}
		if diff := cmp.Diff(refreshed, got.SongMetadata); diff != "" {
			t.Errorf("song metadata mismatch (-want +got):\n%s", diff)
		}

		const unknownSong = "99999999-9999-4999-8999-999999999999"
		if err := f.Today.UpdateSongMetadata(ctx, unknownSong, refreshed); !errors.Is(err, repository.ErrNotFound) {
			t.Errorf("UpdateSongMetadata(unknown song) error = %v, want %v", err, repository.ErrNotFound)
		}
	})

	t.Run("LookupsAreExactDay", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)
		ctx := t.Context()
		date := day(2026, 7, 11)

		if _, err := f.Today.InsertQuote(ctx, repository.InsertQuoteParams{Date: date, BodyText: "x"}); err != nil {
			t.Fatalf("InsertQuote() error = %v, want nil", err)
		}

		// Only 2026-07-11 has a quote, and no day has a song.
		tests := []struct {
			name   string
			lookup func() error
		}{
			{
				name: "翌日のquoteは無い",
				lookup: func() error {
					_, err := f.Today.GetQuoteForDate(ctx, day(2026, 7, 12))
					return err
				},
			},
			{
				name: "前日のquoteは無い",
				lookup: func() error {
					_, err := f.Today.GetQuoteForDate(ctx, day(2026, 7, 10))
					return err
				},
			},
			{
				name: "同日でもsongは別テーブルなので無い",
				lookup: func() error {
					_, err := f.Today.GetSongForDate(ctx, date)
					return err
				},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if err := tt.lookup(); !errors.Is(err, repository.ErrNotFound) {
					t.Errorf("error = %v, want %v", err, repository.ErrNotFound)
				}
			})
		}
	})

	t.Run("ArchiveIsNewestFirstAndPagesWithoutGaps", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)
		ctx := t.Context()

		// 07-12 is registered ahead of time: with Until = 07-11 it must stay out.
		dates := []time.Time{day(2026, 7, 9), day(2026, 7, 10), day(2026, 7, 11), day(2026, 7, 12)}
		for i, d := range dates {
			if _, err := f.Today.InsertSong(ctx, songParams(d, int64(1000+i), d.Format("2006-01-02"))); err != nil {
				t.Fatalf("InsertSong(%s) error = %v, want nil", d, err)
			}
		}
		until := day(2026, 7, 11)

		page1, err := f.Today.ListSongs(ctx, repository.ListSongsParams{Until: until, Limit: 2})
		if err != nil {
			t.Fatalf("ListSongs(page 1) error = %v, want nil", err)
		}
		if diff := cmp.Diff([]string{"2026-07-11", "2026-07-10"}, songTitles(page1)); diff != "" {
			t.Errorf("archive page 1 mismatch (-want +got):\n%s", diff)
		}

		last := page1[len(page1)-1]
		page2, err := f.Today.ListSongs(ctx, repository.ListSongsParams{
			Until: until, Limit: 2, Cursor: &repository.SongCursor{Date: last.Date, ID: last.ID},
		})
		if err != nil {
			t.Fatalf("ListSongs(page 2) error = %v, want nil", err)
		}
		if diff := cmp.Diff([]string{"2026-07-09"}, songTitles(page2)); diff != "" {
			t.Errorf("archive page 2 mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("RecordPlayNeedsAKnownSong", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)
		ctx := t.Context()
		user := f.NewUser(t)

		song, err := f.Today.InsertSong(ctx, songParams(day(2026, 7, 11), 1001, "夜想曲"))
		if err != nil {
			t.Fatalf("InsertSong() error = %v, want nil", err)
		}

		if err := f.Today.RecordPlay(ctx, user, song.ID, time.Now().UTC()); err != nil {
			t.Fatalf("RecordPlay() error = %v, want nil", err)
		}
		// History accumulates; replaying is not an error.
		if err := f.Today.RecordPlay(ctx, user, song.ID, time.Now().UTC()); err != nil {
			t.Errorf("second RecordPlay() error = %v, want nil", err)
		}

		const unknownSong = "99999999-9999-4999-8999-999999999999"
		if err := f.Today.RecordPlay(ctx, user, unknownSong, time.Now().UTC()); !errors.Is(err, repository.ErrNotFound) {
			t.Errorf("RecordPlay(unknown song) error = %v, want %v", err, repository.ErrNotFound)
		}
	})
}

// songTitles lists a page's titles so an ordering assertion reads as one list.
func songTitles(songs []repository.Song) []string {
	out := make([]string, 0, len(songs))
	for _, s := range songs {
		out = append(out, s.Title)
	}
	return out
}
