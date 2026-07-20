package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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

	require.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, "application/json", res.Header.Get("Content-Type"))
	assert.Equal(t, "ok", decodeJSON[healthzResponse](t, res).Status)
}
