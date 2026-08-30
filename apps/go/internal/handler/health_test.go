package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/0muji4/Runa/apps/go/internal/service"
)

func TestHealth_Healthz(t *testing.T) {
	t.Parallel()

	h := NewHealth(service.NewHealth(), discardLogger())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/healthz", nil)
	rec := httptest.NewRecorder()

	h.Healthz(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/v1/healthz = %d, want %d", res.StatusCode, http.StatusOK)
	}
	if got, want := res.Header.Get("Content-Type"), "application/json"; got != want {
		t.Errorf("GET /api/v1/healthz Content-Type = %q, want %q", got, want)
	}
	if got, want := decodeJSON[healthzResponse](t, res).Status, "ok"; got != want {
		t.Errorf("GET /api/v1/healthz status = %q, want %q", got, want)
	}
}
