package server_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func seedSong(t *testing.T, r http.Handler, date, title string) songResp {
	t.Helper()
	res := doAdmin(t, r, http.MethodPost, "/api/v1/admin/songs", adminToken,
		`{"date":"`+date+`","title":"`+title+`","artist":"月詠","artwork_url":"https://x/a.jpg","audio_url":"https://x/a.mp3"}`)
	require.Equal(t, http.StatusCreated, res.StatusCode)
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
	require.Equal(t, http.StatusCreated, res.StatusCode)
	res.Body.Close()
	july11 := seedSong(t, env.r, "2026-07-11", "夜想曲")

	res = do(t, env.r, http.MethodGet, "/api/v1/today?date=2026-07-11", token, "")
	require.Equal(t, http.StatusOK, res.StatusCode)
	var today todayResp
	decode(t, res, &today)
	require.NotNil(t, today.Quote)
	assert.Equal(t, "月あかりのはじまり", today.Quote.BodyText)
	require.NotNil(t, today.Song)
	assert.Equal(t, july11.ID, today.Song.ID)

	res = do(t, env.r, http.MethodGet, "/api/v1/today?date=2000-01-01", token, "")
	decode(t, res, &today)
	assert.Nil(t, today.Quote)
	assert.Nil(t, today.Song)

	// アーカイブは新しい順にページングする。
	seedSong(t, env.r, "2026-07-10", "薄明")
	seedSong(t, env.r, "2026-07-09", "残響")

	res = do(t, env.r, http.MethodGet, "/api/v1/songs?limit=2", token, "")
	var page1 songsResp
	decode(t, res, &page1)
	require.Len(t, page1.Songs, 2)
	require.NotNil(t, page1.NextCursor)
	assert.Equal(t, "2026-07-11", page1.Songs[0].Date)
	assert.Equal(t, "2026-07-10", page1.Songs[1].Date)

	res = do(t, env.r, http.MethodGet, "/api/v1/songs?limit=2&cursor="+*page1.NextCursor, token, "")
	var page2 songsResp
	decode(t, res, &page2)
	require.Len(t, page2.Songs, 1)
	assert.Nil(t, page2.NextCursor)
	assert.Equal(t, "2026-07-09", page2.Songs[0].Date)

	res = do(t, env.r, http.MethodPost, "/api/v1/songs/"+july11.ID+"/played", token, "")
	require.Equal(t, http.StatusNoContent, res.StatusCode)
	res.Body.Close()

	res = do(t, env.r, http.MethodPost, "/api/v1/songs/aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa/played", token, "")
	require.Equal(t, http.StatusNotFound, res.StatusCode)
	res.Body.Close()
}

func TestTodayRequiresAuth(t *testing.T) {
	t.Parallel()

	env := newRouter(t)
	res := do(t, env.r, http.MethodGet, "/api/v1/today", "", "")
	require.Equal(t, http.StatusUnauthorized, res.StatusCode)
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
			require.Equal(t, http.StatusForbidden, res.StatusCode)
			res.Body.Close()
		})
	}
}
