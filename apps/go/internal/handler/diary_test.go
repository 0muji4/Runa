package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/0muji4/Runa/apps/go/internal/auth"
	"github.com/0muji4/Runa/apps/go/internal/repository/memdiary"
	"github.com/0muji4/Runa/apps/go/internal/service"
)

const testUser = "11111111-1111-4111-8111-111111111111"

// stubAccessVerifier makes RequireAuth resolve every Bearer token to a fixed
// user id, so the diary handlers run behind the real middleware (which also
// populates chi URL params) without minting JWTs.
type stubAccessVerifier struct{ userID string }

func (s stubAccessVerifier) Verify(string) (string, error) { return s.userID, nil }

// newDiaryRouter mounts the diary routes behind stub auth over an in-memory store.
func newDiaryRouter() http.Handler {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
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

// errBody decodes the shared error envelope for code assertions.
type errBody struct {
	Error struct {
		Code    string `json:"code"`
		Details []struct {
			Field string `json:"field"`
		} `json:"details"`
	} `json:"error"`
}

func decodeErr(t *testing.T, res *http.Response) errBody {
	t.Helper()
	var e errBody
	if err := json.NewDecoder(res.Body).Decode(&e); err != nil {
		t.Fatalf("decode error envelope: %v", err)
	}
	return e
}

func TestDiaryCreateValidation(t *testing.T) {
	r := newDiaryRouter()

	// Empty body_text and a non-UUID client_id both fail validation.
	res := doDiary(t, r, http.MethodPost, "/diary", `{"body_text":"  ","client_id":"nope"}`)
	defer res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", res.StatusCode)
	}
	body := decodeErr(t, res)
	if body.Error.Code != string(CodeValidation) {
		t.Errorf("code = %q, want validation_error", body.Error.Code)
	}
	if len(body.Error.Details) != 2 {
		t.Errorf("details len = %d, want 2 (body_text + client_id)", len(body.Error.Details))
	}
}

func TestDiaryCreateIdempotentStatus(t *testing.T) {
	r := newDiaryRouter()
	payload := `{"body_text":"月を見た","client_id":"aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"}`

	first := doDiary(t, r, http.MethodPost, "/diary", payload)
	defer first.Body.Close()
	if first.StatusCode != http.StatusCreated {
		t.Fatalf("first POST status = %d, want 201", first.StatusCode)
	}

	second := doDiary(t, r, http.MethodPost, "/diary", payload)
	defer second.Body.Close()
	if second.StatusCode != http.StatusOK {
		t.Fatalf("repeat POST status = %d, want 200 (idempotent)", second.StatusCode)
	}
}

func TestDiaryGetNotFound(t *testing.T) {
	r := newDiaryRouter()

	// Well-formed but unknown id → 404 not_found.
	res := doDiary(t, r, http.MethodGet, "/diary/99999999-9999-4999-8999-999999999999", "")
	defer res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("unknown id status = %d, want 404", res.StatusCode)
	}
	if code := decodeErr(t, res).Error.Code; code != string(CodeNotFound) {
		t.Errorf("code = %q, want not_found", code)
	}

	// Malformed id also 404 (rejected before hitting the store).
	bad := doDiary(t, r, http.MethodGet, "/diary/not-a-uuid", "")
	defer bad.Body.Close()
	if bad.StatusCode != http.StatusNotFound {
		t.Errorf("malformed id status = %d, want 404", bad.StatusCode)
	}
}

func TestDiaryListShape(t *testing.T) {
	r := newDiaryRouter()
	doDiary(t, r, http.MethodPost, "/diary", `{"body_text":"一件目","client_id":"aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"}`).Body.Close()

	res := doDiary(t, r, http.MethodGet, "/diary", "")
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", res.StatusCode)
	}
	var got diaryListResponse
	if err := json.NewDecoder(res.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got.Entries) != 1 {
		t.Fatalf("entries len = %d, want 1", len(got.Entries))
	}
	if got.NextCursor != nil {
		t.Errorf("next_cursor = %v, want null on a single-page result", *got.NextCursor)
	}
}

func TestDiaryUpdateRequiresBody(t *testing.T) {
	r := newDiaryRouter()
	res := doDiary(t, r, http.MethodPatch, "/diary/99999999-9999-4999-8999-999999999999", `{"mood":"calm"}`)
	defer res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (body_text required)", res.StatusCode)
	}
}

func TestDiarySyncRejectsBadSince(t *testing.T) {
	r := newDiaryRouter()
	res := doDiary(t, r, http.MethodGet, "/diary/sync?since=not-a-time", "")
	defer res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", res.StatusCode)
	}
}
