package server_test

import (
	"io"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/0muji4/Runa/apps/go/internal/auth"
	"github.com/0muji4/Runa/apps/go/internal/handler"
	"github.com/0muji4/Runa/apps/go/internal/repository/memauth"
	"github.com/0muji4/Runa/apps/go/internal/repository/memdiary"
	"github.com/0muji4/Runa/apps/go/internal/server"
	"github.com/0muji4/Runa/apps/go/internal/service"
)

// diaryEntry mirrors the JSON shape of the diary entry response for decoding.
type diaryEntry struct {
	ID        string  `json:"id"`
	ClientID  string  `json:"client_id"`
	BodyText  string  `json:"body_text"`
	Mood      *string `json:"mood"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
	DeletedAt *string `json:"deleted_at"`
}

type diaryList struct {
	Entries    []diaryEntry `json:"entries"`
	NextCursor *string      `json:"next_cursor"`
}

type diarySync struct {
	Entries    []diaryEntry `json:"entries"`
	ServerTime string       `json:"server_time"`
}

// newDiaryFlowRouter wires auth + diary over in-memory stores so the whole
// request path (real JWT middleware, real routing, chi URL params) is exercised.
func newDiaryFlowRouter() http.Handler {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	issuer := auth.NewTokenIssuer("test-secret", time.Minute)
	authSvc := service.NewAuthService(service.AuthConfig{
		Store:          memauth.New(),
		Issuer:         issuer,
		PasswordParams: auth.DefaultArgon2Params(),
		RefreshTTL:     time.Hour,
	})
	ah := handler.NewAuth(authSvc, logger)
	dh := handler.NewDiary(service.NewDiaryService(memdiary.New(), nil), logger)

	return server.New(server.Deps{
		Health:         handler.NewHealth(service.NewHealth(), logger),
		Auth:           ah,
		Diary:          dh,
		RequireAuth:    auth.RequireAuth(issuer, ah.Unauthorized),
		AuthRateLimit:  auth.NewRateLimiter(100, time.Minute).Middleware(ah.RateLimited),
		AllowedOrigins: []string{"*"},
		Logger:         logger,
	})
}

// signupToken creates an account and returns its access token.
func signupToken(t *testing.T, r http.Handler, email string) string {
	t.Helper()
	res := do(t, r, http.MethodPost, "/api/v1/auth/signup", "",
		`{"email":"`+email+`","password":"password123","display_name":"U"}`)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("signup %s status = %d, want 201", email, res.StatusCode)
	}
	var tok struct {
		AccessToken string `json:"access_token"`
	}
	decode(t, res, &tok)
	return tok.AccessToken
}

// TestDiaryFlow runs the full lifecycle through the real router: idempotent POST
// -> list -> get -> patch -> sync delta -> soft delete -> tombstone in sync, and
// confirms another user can never reach the entry (404).
func TestDiaryFlow(t *testing.T) {
	r := newDiaryFlowRouter()
	token := signupToken(t, r, "diary@example.com")
	const cid = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"

	// 1. Create (201).
	res := do(t, r, http.MethodPost, "/api/v1/diary", token,
		`{"body_text":"月を見上げた","mood":"calm","client_id":"`+cid+`","created_at":"2026-07-01T12:00:00Z"}`)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("create status = %d, want 201", res.StatusCode)
	}
	var created diaryEntry
	decode(t, res, &created)
	if created.ClientID != cid || created.Mood == nil || *created.Mood != "calm" {
		t.Fatalf("created entry = %+v, unexpected", created)
	}

	// 2. Repeat POST with the same client_id is idempotent: 200, same id, new body.
	res = do(t, r, http.MethodPost, "/api/v1/diary", token,
		`{"body_text":"月を見上げた（推敲）","client_id":"`+cid+`","created_at":"2026-07-01T12:00:00Z"}`)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("repeat create status = %d, want 200", res.StatusCode)
	}
	var repeated diaryEntry
	decode(t, res, &repeated)
	if repeated.ID != created.ID {
		t.Fatalf("idempotent POST changed id: %q -> %q", created.ID, repeated.ID)
	}

	// 3. List shows exactly one entry, no next page.
	res = do(t, r, http.MethodGet, "/api/v1/diary", token, "")
	var list diaryList
	decode(t, res, &list)
	if len(list.Entries) != 1 || list.NextCursor != nil {
		t.Fatalf("list = %+v, want 1 entry and no cursor", list)
	}

	// 4. Get by id.
	res = do(t, r, http.MethodGet, "/api/v1/diary/"+created.ID, token, "")
	if res.StatusCode != http.StatusOK {
		t.Fatalf("get status = %d, want 200", res.StatusCode)
	}

	// 5. Full sync, capture the watermark.
	res = do(t, r, http.MethodGet, "/api/v1/diary/sync", token, "")
	var full diarySync
	decode(t, res, &full)
	if len(full.Entries) != 1 {
		t.Fatalf("full sync entries = %d, want 1", len(full.Entries))
	}

	// 6. Patch, then a delta sync from the watermark returns only the change.
	res = do(t, r, http.MethodPatch, "/api/v1/diary/"+created.ID, token,
		`{"body_text":"翌朝、読み返した","mood":"gentle"}`)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("patch status = %d, want 200", res.StatusCode)
	}
	res = do(t, r, http.MethodGet, "/api/v1/diary/sync?since="+full.ServerTime, token, "")
	var delta diarySync
	decode(t, res, &delta)
	if len(delta.Entries) != 1 || delta.Entries[0].BodyText != "翌朝、読み返した" {
		t.Fatalf("delta sync = %+v, want the single patched entry", delta.Entries)
	}

	// 7. Soft delete → gone from the list.
	res = do(t, r, http.MethodDelete, "/api/v1/diary/"+created.ID, token, "")
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("delete status = %d, want 204", res.StatusCode)
	}
	res = do(t, r, http.MethodGet, "/api/v1/diary", token, "")
	decode(t, res, &list)
	if len(list.Entries) != 0 {
		t.Fatalf("list after delete = %d, want 0", len(list.Entries))
	}

	// 8. The deletion propagates through sync as a tombstone.
	res = do(t, r, http.MethodGet, "/api/v1/diary/sync?since="+delta.ServerTime, token, "")
	var afterDelete diarySync
	decode(t, res, &afterDelete)
	if len(afterDelete.Entries) != 1 || afterDelete.Entries[0].DeletedAt == nil {
		t.Fatalf("sync after delete = %+v, want one tombstone", afterDelete.Entries)
	}

	// 9. Another user cannot touch the entry: every access is 404.
	other := signupToken(t, r, "other@example.com")
	for _, tc := range []struct {
		method, path, body string
	}{
		{http.MethodGet, "/api/v1/diary/" + created.ID, ""},
		{http.MethodPatch, "/api/v1/diary/" + created.ID, `{"body_text":"改ざん"}`},
		{http.MethodDelete, "/api/v1/diary/" + created.ID, ""},
	} {
		res = do(t, r, tc.method, tc.path, other, tc.body)
		if res.StatusCode != http.StatusNotFound {
			t.Errorf("%s by other user status = %d, want 404", tc.method, res.StatusCode)
		}
		res.Body.Close()
	}
}

// TestDiaryRequiresAuth confirms the diary collection is Bearer-protected.
func TestDiaryRequiresAuth(t *testing.T) {
	r := newDiaryFlowRouter()
	res := do(t, r, http.MethodGet, "/api/v1/diary", "", "")
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauthenticated list status = %d, want 401", res.StatusCode)
	}
}
