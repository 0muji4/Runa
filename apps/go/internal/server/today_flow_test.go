package server_test

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/0muji4/Runa/apps/go/internal/auth"
	"github.com/0muji4/Runa/apps/go/internal/handler"
	"github.com/0muji4/Runa/apps/go/internal/repository/memauth"
	"github.com/0muji4/Runa/apps/go/internal/repository/memdiary"
	"github.com/0muji4/Runa/apps/go/internal/repository/memtoday"
	"github.com/0muji4/Runa/apps/go/internal/server"
	"github.com/0muji4/Runa/apps/go/internal/service"
)

// adminToken is the shared seed token the test router is configured with.
const adminToken = "seed-secret"

// today response shapes mirror the JSON contract for decoding.
type quoteResp struct {
	ID       string `json:"id"`
	Date     string `json:"date"`
	BodyText string `json:"body_text"`
}

type songResp struct {
	ID         string `json:"id"`
	Date       string `json:"date"`
	Title      string `json:"title"`
	Artist     string `json:"artist"`
	ArtworkURL string `json:"artwork_url"`
	AudioURL   string `json:"audio_url"`
}

type todayResp struct {
	Date  string     `json:"date"`
	Quote *quoteResp `json:"quote"`
	Song  *songResp  `json:"song"`
}

type songsResp struct {
	Songs      []songResp `json:"songs"`
	NextCursor *string    `json:"next_cursor"`
}

// newTodayFlowRouter wires auth + today (with admin) over in-memory stores so the
// whole request path (real JWT middleware, real admin gate, real routing) runs.
func newTodayFlowRouter() http.Handler {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	issuer := auth.NewTokenIssuer("test-secret", time.Minute)
	authSvc := service.NewAuthService(service.AuthConfig{
		Store:          memauth.New(),
		Issuer:         issuer,
		PasswordParams: auth.DefaultArgon2Params(),
		RefreshTTL:     time.Hour,
	})
	ah := handler.NewAuth(authSvc, logger)
	th := handler.NewToday(service.NewTodayService(memtoday.New(), nil), logger)

	return server.New(server.Deps{
		Health:         handler.NewHealth(service.NewHealth(), logger),
		Auth:           ah,
		Diary:          handler.NewDiary(service.NewDiaryService(memdiary.New(), nil), logger),
		Today:          th,
		RequireAuth:    auth.RequireAuth(issuer, ah.Unauthorized),
		AuthRateLimit:  auth.NewRateLimiter(100, time.Minute).Middleware(ah.RateLimited),
		RequireAdmin:   auth.RequireAdmin(adminToken, th.Forbidden),
		AllowedOrigins: []string{"*"},
		Logger:         logger,
	})
}

// doAdmin issues a request carrying the X-Admin-Token header (the `do` helper
// only sets a Bearer token).
func doAdmin(t *testing.T, r http.Handler, method, path, token, body string) *http.Response {
	t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	if token != "" {
		req.Header.Set("X-Admin-Token", token)
	}
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec.Result()
}

// seedSong posts one curated song for a date via the admin endpoint and returns it.
func seedSong(t *testing.T, r http.Handler, date, title string) songResp {
	t.Helper()
	res := doAdmin(t, r, http.MethodPost, "/api/v1/admin/songs", adminToken,
		`{"date":"`+date+`","title":"`+title+`","artist":"月詠","artwork_url":"https://x/a.jpg","audio_url":"https://x/a.mp3"}`)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("seed song %s status = %d, want 201", date, res.StatusCode)
	}
	var s songResp
	decode(t, res, &s)
	return s
}

// TestTodayFlow runs the today lifecycle through the real router: admin seed →
// GET /today (hit + miss) → archive paging → played (204 + 404).
func TestTodayFlow(t *testing.T) {
	r := newTodayFlowRouter()
	token := signupToken(t, r, "today@example.com")

	// 1. Seed a quote + song for a specific day via the admin endpoints.
	res := doAdmin(t, r, http.MethodPost, "/api/v1/admin/quotes", adminToken,
		`{"date":"2026-07-11","body_text":"月あかりのはじまり"}`)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("seed quote status = %d, want 201", res.StatusCode)
	}
	july11 := seedSong(t, r, "2026-07-11", "夜想曲")

	// 2. GET /today on the seeded day returns both the quote and the song.
	res = do(t, r, http.MethodGet, "/api/v1/today?date=2026-07-11", token, "")
	if res.StatusCode != http.StatusOK {
		t.Fatalf("today status = %d, want 200", res.StatusCode)
	}
	var today todayResp
	decode(t, res, &today)
	if today.Quote == nil || today.Quote.BodyText != "月あかりのはじまり" {
		t.Fatalf("today quote = %+v, want the seeded quote", today.Quote)
	}
	if today.Song == nil || today.Song.ID != july11.ID {
		t.Fatalf("today song = %+v, want the seeded song", today.Song)
	}

	// 3. GET /today on an unseeded day returns nulls (not an error).
	res = do(t, r, http.MethodGet, "/api/v1/today?date=2000-01-01", token, "")
	decode(t, res, &today)
	if today.Quote != nil || today.Song != nil {
		t.Fatalf("unseeded today = %+v, want null quote and song", today)
	}

	// 4. Seed two more days, then page the archive two at a time (newest first).
	seedSong(t, r, "2026-07-10", "薄明")
	seedSong(t, r, "2026-07-09", "残響")
	res = do(t, r, http.MethodGet, "/api/v1/songs?limit=2", token, "")
	var page1 songsResp
	decode(t, res, &page1)
	if len(page1.Songs) != 2 || page1.NextCursor == nil {
		t.Fatalf("archive page1 = %+v, want 2 songs and a cursor", page1)
	}
	if page1.Songs[0].Date != "2026-07-11" || page1.Songs[1].Date != "2026-07-10" {
		t.Fatalf("archive page1 order = [%s %s], want newest first",
			page1.Songs[0].Date, page1.Songs[1].Date)
	}
	res = do(t, r, http.MethodGet, "/api/v1/songs?limit=2&cursor="+*page1.NextCursor, token, "")
	var page2 songsResp
	decode(t, res, &page2)
	if len(page2.Songs) != 1 || page2.NextCursor != nil {
		t.Fatalf("archive page2 = %+v, want the last song and no cursor", page2)
	}
	if page2.Songs[0].Date != "2026-07-09" {
		t.Fatalf("archive page2 song = %s, want 2026-07-09", page2.Songs[0].Date)
	}

	// 5. Record a play (204), then a play against an unknown song id (404).
	res = do(t, r, http.MethodPost, "/api/v1/songs/"+july11.ID+"/played", token, "")
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("played status = %d, want 204", res.StatusCode)
	}
	res = do(t, r, http.MethodPost, "/api/v1/songs/aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa/played", token, "")
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("played unknown song status = %d, want 404", res.StatusCode)
	}
}

// TestTodayRequiresAuth confirms the today endpoints are Bearer-protected.
func TestTodayRequiresAuth(t *testing.T) {
	r := newTodayFlowRouter()
	res := do(t, r, http.MethodGet, "/api/v1/today", "", "")
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauthenticated today status = %d, want 401", res.StatusCode)
	}
}

// TestAdminRequiresToken confirms the seed endpoints reject a missing or wrong
// admin token with 403.
func TestAdminRequiresToken(t *testing.T) {
	r := newTodayFlowRouter()
	body := `{"date":"2026-07-11","body_text":"x"}`

	res := doAdmin(t, r, http.MethodPost, "/api/v1/admin/quotes", "", body)
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("admin without token status = %d, want 403", res.StatusCode)
	}
	res = doAdmin(t, r, http.MethodPost, "/api/v1/admin/quotes", "wrong-token", body)
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("admin with wrong token status = %d, want 403", res.StatusCode)
	}
}
