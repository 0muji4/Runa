package server_test

import (
	"net/http"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// songDates pulls the dates out of an archive page, so a paging assertion reads
// as one ordered list rather than an index-by-index comparison.
func songDates(page songsResp) []string {
	dates := make([]string, 0, len(page.Songs))
	for _, s := range page.Songs {
		dates = append(dates, s.Date)
	}
	return dates
}

func seedSong(t *testing.T, r http.Handler, date, title string) songResp {
	t.Helper()
	res := doAdmin(t, r, http.MethodPost, "/api/v1/admin/songs", adminToken,
		`{"date":"`+date+`","title":"`+title+`","artist":"月詠","artwork_url":"https://x/a.jpg","audio_url":"https://x/a.mp3"}`)
	checkStatus(t, res, http.StatusCreated)
	var s songResp
	decode(t, res, &s)
	return s
}

func TestTodayFlow(t *testing.T) {
	t.Parallel()

	env := newRouter(t)
	token := signupToken(t, env.r, "today@example.com")

	res := doAdmin(t, env.r, http.MethodPost, "/api/v1/admin/quotes", adminToken,
		`{"date":"2026-07-11","body_text":"月あかりのはじまり"}`)
	checkStatus(t, res, http.StatusCreated)
	res.Body.Close()
	july11 := seedSong(t, env.r, "2026-07-11", "夜想曲")

	res = do(t, env.r, http.MethodGet, "/api/v1/today?date=2026-07-11", token, "")
	checkStatus(t, res, http.StatusOK)
	var today todayResp
	decode(t, res, &today)
	if today.Quote == nil {
		t.Fatal("GET /api/v1/today quote = nil, want the seeded quote")
	}
	if got, want := today.Quote.BodyText, "月あかりのはじまり"; got != want {
		t.Errorf("today quote body_text = %q, want %q", got, want)
	}
	if today.Song == nil {
		t.Fatal("GET /api/v1/today song = nil, want the seeded song")
	}
	if today.Song.ID != july11.ID {
		t.Errorf("today song id = %q, want %q", today.Song.ID, july11.ID)
	}

	res = do(t, env.r, http.MethodGet, "/api/v1/today?date=2000-01-01", token, "")
	decode(t, res, &today)
	// A day with no curated content answers 200 with both fields null.
	if today.Quote != nil {
		t.Errorf("today quote for an unseeded date = %+v, want nil", today.Quote)
	}
	if today.Song != nil {
		t.Errorf("today song for an unseeded date = %+v, want nil", today.Song)
	}

	// アーカイブは新しい順にページングする。
	seedSong(t, env.r, "2026-07-10", "薄明")
	seedSong(t, env.r, "2026-07-09", "残響")

	res = do(t, env.r, http.MethodGet, "/api/v1/songs?limit=2", token, "")
	var page1 songsResp
	decode(t, res, &page1)
	if len(page1.Songs) != 2 {
		t.Fatalf("songs page 1 returned %d songs, want 2", len(page1.Songs))
	}
	if page1.NextCursor == nil {
		t.Fatal("songs page 1 next_cursor = nil, want a cursor")
	}
	if diff := cmp.Diff([]string{"2026-07-11", "2026-07-10"}, songDates(page1)); diff != "" {
		t.Errorf("songs page 1 dates mismatch (-want +got):\n%s", diff)
	}

	res = do(t, env.r, http.MethodGet, "/api/v1/songs?limit=2&cursor="+*page1.NextCursor, token, "")
	var page2 songsResp
	decode(t, res, &page2)
	if len(page2.Songs) != 1 {
		t.Fatalf("songs page 2 returned %d songs, want 1", len(page2.Songs))
	}
	if page2.NextCursor != nil {
		t.Errorf("songs page 2 next_cursor = %q, want nil (last page)", *page2.NextCursor)
	}
	if diff := cmp.Diff([]string{"2026-07-09"}, songDates(page2)); diff != "" {
		t.Errorf("songs page 2 dates mismatch (-want +got):\n%s", diff)
	}

	res = do(t, env.r, http.MethodPost, "/api/v1/songs/"+july11.ID+"/played", token, "")
	checkStatus(t, res, http.StatusNoContent)
	res.Body.Close()

	res = do(t, env.r, http.MethodPost, "/api/v1/songs/aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa/played", token, "")
	checkStatus(t, res, http.StatusNotFound)
	res.Body.Close()
}

func TestTodayRequiresAuth(t *testing.T) {
	t.Parallel()

	env := newRouter(t)
	res := do(t, env.r, http.MethodGet, "/api/v1/today", "", "")
	checkStatus(t, res, http.StatusUnauthorized)
	res.Body.Close()
}

func TestAdminRequiresToken(t *testing.T) {
	t.Parallel()

	env := newRouter(t)
	body := `{"date":"2026-07-11","body_text":"x"}`

	tests := []struct {
		name  string
		token string
	}{
		{
			name:  "トークンなし",
			token: "",
		},
		{
			name:  "誤ったトークン",
			token: "wrong-token",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			res := doAdmin(t, env.r, http.MethodPost, "/api/v1/admin/quotes", tt.token, body)
			checkStatus(t, res, http.StatusForbidden)
			res.Body.Close()
		})
	}
}
