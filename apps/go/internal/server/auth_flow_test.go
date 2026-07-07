package server_test

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/0muji4/Runa/apps/go/internal/auth"
	"github.com/0muji4/Runa/apps/go/internal/handler"
	"github.com/0muji4/Runa/apps/go/internal/repository/memauth"
	"github.com/0muji4/Runa/apps/go/internal/server"
	"github.com/0muji4/Runa/apps/go/internal/service"
)

// tokens mirrors the JSON shape of AuthTokens for decoding in tests.
type tokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	User         struct {
		ID          string `json:"id"`
		Email       string `json:"email"`
		DisplayName string `json:"display_name"`
	} `json:"user"`
}

func newRouter() http.Handler {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	issuer := auth.NewTokenIssuer("test-secret", time.Minute)
	svc := service.NewAuthService(service.AuthConfig{
		Store:          memauth.New(),
		Issuer:         issuer,
		PasswordParams: auth.DefaultArgon2Params(),
		RefreshTTL:     time.Hour,
	})
	h := handler.NewAuth(svc, logger)
	return server.New(server.Deps{
		Health:         handler.NewHealth(service.NewHealth(), logger),
		Auth:           h,
		RequireAuth:    auth.RequireAuth(issuer, h.Unauthorized),
		AuthRateLimit:  auth.NewRateLimiter(100, time.Minute).Middleware(h.RateLimited),
		AllowedOrigins: []string{"*"},
		Logger:         logger,
	})
}

func do(t *testing.T, r http.Handler, method, path, bearer, body string) *http.Response {
	t.Helper()
	var reader io.Reader
	if body != "" {
		reader = bytes.NewBufferString(body)
	}
	req := httptest.NewRequest(method, path, reader)
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec.Result()
}

// TestAuthFlow exercises the full protected-endpoint lifecycle end to end
// through the real router: signup -> /me -> refresh (rotation) -> logout ->
// /me/refresh rejected. No database is involved (in-memory store).
func TestAuthFlow(t *testing.T) {
	r := newRouter()

	// 1. Signup.
	res := do(t, r, http.MethodPost, "/api/v1/auth/signup", "", `{"email":"flow@example.com","password":"password123","display_name":"Flow"}`)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("signup status = %d, want 201", res.StatusCode)
	}
	var signed tokens
	decode(t, res, &signed)
	if signed.User.DisplayName != "Flow" {
		t.Errorf("display_name = %q, want Flow", signed.User.DisplayName)
	}

	// 2. /me with the access token.
	res = do(t, r, http.MethodGet, "/api/v1/me", signed.AccessToken, "")
	if res.StatusCode != http.StatusOK {
		t.Fatalf("/me status = %d, want 200", res.StatusCode)
	}
	var me struct {
		Email       string `json:"email"`
		DisplayName string `json:"display_name"`
	}
	decode(t, res, &me)
	if me.Email != "flow@example.com" || me.DisplayName != "Flow" {
		t.Errorf("/me = %+v, unexpected", me)
	}

	// 3. /me without a token is rejected.
	res = do(t, r, http.MethodGet, "/api/v1/me", "", "")
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("/me without token status = %d, want 401", res.StatusCode)
	}

	// 4. Refresh rotates the token pair.
	res = do(t, r, http.MethodPost, "/api/v1/auth/refresh", "", `{"refresh_token":"`+signed.RefreshToken+`"}`)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("refresh status = %d, want 200", res.StatusCode)
	}
	var rotated tokens
	decode(t, res, &rotated)
	if rotated.AccessToken == "" || rotated.RefreshToken == signed.RefreshToken {
		t.Fatal("refresh did not rotate the token pair")
	}

	// 5. The old refresh token is now rejected.
	res = do(t, r, http.MethodPost, "/api/v1/auth/refresh", "", `{"refresh_token":"`+signed.RefreshToken+`"}`)
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("reused refresh status = %d, want 401", res.StatusCode)
	}

	// 6. The rotated access token still works.
	res = do(t, r, http.MethodGet, "/api/v1/me", rotated.AccessToken, "")
	if res.StatusCode != http.StatusOK {
		t.Fatalf("/me with rotated token status = %d, want 200", res.StatusCode)
	}

	// 7. Logout revokes the current refresh token.
	res = do(t, r, http.MethodPost, "/api/v1/auth/logout", "", `{"refresh_token":"`+rotated.RefreshToken+`"}`)
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("logout status = %d, want 204", res.StatusCode)
	}

	// 8. Refresh after logout is rejected.
	res = do(t, r, http.MethodPost, "/api/v1/auth/refresh", "", `{"refresh_token":"`+rotated.RefreshToken+`"}`)
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("refresh after logout status = %d, want 401", res.StatusCode)
	}
}

func TestMeRejectsGarbageToken(t *testing.T) {
	r := newRouter()
	res := do(t, r, http.MethodGet, "/api/v1/me", "not-a-real-token", "")
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", res.StatusCode)
	}
}

func decode(t *testing.T, res *http.Response, dst any) {
	t.Helper()
	defer res.Body.Close()
	if err := json.NewDecoder(res.Body).Decode(dst); err != nil {
		t.Fatalf("decode response: %v", err)
	}
}
