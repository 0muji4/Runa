package server_test

import (
	"net/http"
	"strconv"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// songDates pulls the dates out of an archive page.
func songDates(page songsResp) []string {
	dates := make([]string, 0, len(page.Songs))
	for _, s := range page.Songs {
		dates = append(dates, s.Date)
	}
	return dates
}

// seedSong registers a track with the fake Apple and creates the day's song via the admin endpoint.
func seedSong(t *testing.T, env *testEnv, date, title string) songResp {
	t.Helper()
	id := env.apple.add(title)
	res := doAdmin(t, env.r, http.MethodPost, "/api/v1/admin/songs", adminToken,
		`{"date":"`+date+`","itunes_track_id":`+strconv.FormatInt(id, 10)+`}`)
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
	july11 := seedSong(t, env, "2026-07-11", "夜想曲")
	if july11.ArtworkURL != env.apple.srv.URL+"/art/600x600bb.jpg" {
		t.Errorf("registered song artwork_url = %q, want the 600px rendition", july11.ArtworkURL)
	}
	if july11.PreviewURL != "https://cdn.example/夜想曲.m4a" || july11.StoreURL == "" {
		t.Errorf("registered song preview/store = (%q, %q), want Apple's URLs", july11.PreviewURL, july11.StoreURL)
	}

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
	if today.Quote != nil {
		t.Errorf("today quote for an unseeded date = %+v, want nil", today.Quote)
	}
	if today.Song != nil {
		t.Errorf("today song for an unseeded date = %+v, want nil", today.Song)
	}

	// アーカイブは新しい順にページングし、until より先の日（先に登録した曲）は含めない。
	seedSong(t, env, "2026-07-10", "薄明")
	seedSong(t, env, "2026-07-09", "残響")
	seedSong(t, env, "2026-07-12", "明日の曲")

	res = do(t, env.r, http.MethodGet, "/api/v1/songs?until=2026-07-11&limit=2", token, "")
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

	res = do(t, env.r, http.MethodGet, "/api/v1/songs?until=2026-07-11&limit=2&cursor="+*page1.NextCursor, token, "")
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

func TestAdminCreateSongRejectsUnusableTracks(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		body       func(env *testEnv) string
		appleDown  bool
		wantStatus int
		wantCode   string
	}{
		{
			name:       "track idが無いのは400",
			body:       func(*testEnv) string { return `{"date":"2026-07-11"}` },
			wantStatus: http.StatusBadRequest,
			wantCode:   "validation_error",
		},
		{
			name:       "Appleに無いtrack idは422",
			body:       func(*testEnv) string { return `{"date":"2026-07-11","itunes_track_id":424242}` },
			wantStatus: http.StatusUnprocessableEntity,
			wantCode:   "validation_error",
		},
		{
			name: "Appleが応答しなければ502",
			body: func(env *testEnv) string {
				return `{"date":"2026-07-11","itunes_track_id":` + strconv.FormatInt(env.apple.add("夜想曲"), 10) + `}`
			},
			appleDown:  true,
			wantStatus: http.StatusBadGateway,
			wantCode:   "upstream_error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			env := newRouter(t)
			token := signupToken(t, env.r, "admin-song@example.com")
			env.apple.setDown(tt.appleDown)

			res := doAdmin(t, env.r, http.MethodPost, "/api/v1/admin/songs", adminToken, tt.body(env))
			checkStatus(t, res, tt.wantStatus)
			var body errorResp
			decode(t, res, &body)
			if body.Error.Code != tt.wantCode {
				t.Errorf("error code = %q, want %q", body.Error.Code, tt.wantCode)
			}

			res = do(t, env.r, http.MethodGet, "/api/v1/today?date=2026-07-11", token, "")
			var today todayResp
			decode(t, res, &today)
			if today.Song != nil {
				t.Errorf("today song after a rejected registration = %+v, want nil", today.Song)
			}
		})
	}
}
