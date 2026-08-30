package handler

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/0muji4/Runa/apps/go/internal/auth"
	"github.com/0muji4/Runa/apps/go/internal/repository/memdevices"
	"github.com/0muji4/Runa/apps/go/internal/service"
)

func newDevicesRouter() http.Handler {
	logger := discardLogger()
	h := NewDevices(service.NewDeviceService(memdevices.New(), nil), logger)

	onUnauthorized := func(w http.ResponseWriter, _ *http.Request, _ error) {
		writeError(w, http.StatusUnauthorized, CodeUnauthorized, "authentication required", nil, logger)
	}

	r := chi.NewRouter()
	r.Group(func(pr chi.Router) {
		pr.Use(auth.RequireAuth(stubAccessVerifier{testUser}, onUnauthorized))
		pr.Put("/devices", h.Register)
	})
	return r
}

func doDevices(t *testing.T, r http.Handler, bearer, body string) *http.Response {
	t.Helper()
	var reader io.Reader
	if body != "" {
		reader = bytes.NewBufferString(body)
	}
	req := httptest.NewRequest(http.MethodPut, "/devices", reader)
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec.Result()
}

func TestDevices_Register(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		bearer      string
		body        string
		wantStatus  int
		wantCode    ErrorCode
		wantDetails int
	}{
		{
			name:        "有効な登録は200",
			bearer:      "test",
			body:        `{"push_token":"tok","platform":"ios","notify_time":"22:00","enabled":true}`,
			wantStatus:  http.StatusOK,
			wantCode:    "",
			wantDetails: -1,
		},
		{
			name:        "全フィールド不正で検証エラー3件",
			bearer:      "test",
			body:        `{"push_token":"  ","platform":"web","notify_time":"nope","enabled":false}`,
			wantStatus:  http.StatusBadRequest,
			wantCode:    CodeValidation,
			wantDetails: 3,
		},
		{
			name:        "未知のフィールドは400",
			bearer:      "test",
			body:        `{"push_token":"tok","platform":"ios","notify_time":"22:00","enabled":true,"extra":1}`,
			wantStatus:  http.StatusBadRequest,
			wantCode:    CodeValidation,
			wantDetails: -1,
		},
		{
			name:        "未認証は401",
			bearer:      "",
			body:        `{"push_token":"tok","platform":"ios","notify_time":"22:00","enabled":true}`,
			wantStatus:  http.StatusUnauthorized,
			wantCode:    CodeUnauthorized,
			wantDetails: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := newDevicesRouter()
			res := doDevices(t, r, tt.bearer, tt.body)
			defer res.Body.Close()

			if res.StatusCode != tt.wantStatus {
				t.Fatalf("PUT /devices %s = %d, want %d", tt.body, res.StatusCode, tt.wantStatus)
			}
			checkErrorEnvelope(t, res, tt.wantCode, tt.wantDetails)
		})
	}
}
