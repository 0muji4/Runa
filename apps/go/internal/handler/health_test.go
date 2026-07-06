package handler

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/0muji4/Runa/apps/go/internal/service"
)

// TestHealthz verifies the health endpoint returns 200, the correct JSON body,
// and the application/json content type — the exact cross-layer API contract.
func TestHealthz(t *testing.T) {
	h := NewHealth(service.NewHealth(), slog.New(slog.NewJSONHandler(io.Discard, nil)))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/healthz", nil)
	rec := httptest.NewRecorder()

	h.Healthz(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.StatusCode, http.StatusOK)
	}

	if ct := res.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/json")
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}

	var got healthzResponse
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("unmarshal body %q: %v", string(body), err)
	}

	if got.Status != "ok" {
		t.Errorf("status = %q, want %q", got.Status, "ok")
	}
}
