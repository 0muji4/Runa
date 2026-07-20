package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"github.com/0muji4/Runa/apps/go/internal/auth"
	"github.com/0muji4/Runa/apps/go/internal/repository/memauth"
	"github.com/0muji4/Runa/apps/go/internal/repository/memdiary"
	"github.com/0muji4/Runa/apps/go/internal/service"
)

const testUser = "11111111-1111-4111-8111-111111111111"

func discardLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(io.Discard, nil))
}

func decodeJSON[T any](t *testing.T, res *http.Response) T {
	t.Helper()
	var v T
	require.NoError(t, json.NewDecoder(res.Body).Decode(&v))
	return v
}

func decodeError(t *testing.T, res *http.Response) errorEnvelope {
	t.Helper()
	return decodeJSON[errorEnvelope](t, res)
}

func newAuthHandler() *Auth {
	svc := service.NewAuthService(service.AuthConfig{
		Store:          memauth.New(),
		Issuer:         auth.NewTokenIssuer("test-secret", time.Minute),
		PasswordParams: auth.DefaultArgon2Params(),
		RefreshTTL:     time.Hour,
	})
	return NewAuth(svc, discardLogger())
}

func postJSON(t *testing.T, h http.HandlerFunc, body string) *http.Response {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()
	h(rec, req)
	return rec.Result()
}

// stubAccessVerifier resolves every Bearer token to a fixed user id, so the diary
// handlers run behind the real middleware without minting JWTs.
type stubAccessVerifier struct{ userID string }

func (s stubAccessVerifier) Verify(string) (string, error) { return s.userID, nil }

func newDiaryRouter() http.Handler {
	logger := discardLogger()
	h := NewDiary(service.NewDiaryService(memdiary.New(), nil), logger)

	onUnauthorized := func(w http.ResponseWriter, _ *http.Request, _ error) {
		writeError(w, http.StatusUnauthorized, CodeUnauthorized, "authentication required", nil, logger)
	}

	r := chi.NewRouter()
	r.Group(func(pr chi.Router) {
		pr.Use(auth.RequireAuth(stubAccessVerifier{testUser}, onUnauthorized))
		pr.Get("/diary", h.List)
		pr.Post("/diary", h.Create)
		pr.Get("/diary/sync", h.Sync)
		pr.Get("/diary/{id}", h.Get)
		pr.Patch("/diary/{id}", h.Update)
		pr.Delete("/diary/{id}", h.Delete)
	})
	return r
}

func doDiary(t *testing.T, r http.Handler, method, path, body string) *http.Response {
	t.Helper()
	var reader io.Reader
	if body != "" {
		reader = bytes.NewBufferString(body)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Authorization", "Bearer test") // stub verifier ignores the value
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec.Result()
}

func createDiaryEntry(t *testing.T, r http.Handler, clientID, bodyText string) diaryEntryResponse {
	t.Helper()
	res := doDiary(t, r, http.MethodPost, "/diary",
		fmt.Sprintf(`{"body_text":%q,"client_id":%q}`, bodyText, clientID))
	defer res.Body.Close()
	require.Equal(t, http.StatusCreated, res.StatusCode)
	return decodeJSON[diaryEntryResponse](t, res)
}
