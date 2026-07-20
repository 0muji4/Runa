package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRequireAdmin(t *testing.T) {
	t.Parallel()

	const serverToken = "s3cr3t-admin-token"

	tests := []struct {
		name        string
		serverToken string
		setHeader   bool
		presented   string
		wantStatus  int
		wantNext    bool
	}{
		{
			name:        "正しいトークンは通過する",
			serverToken: serverToken,
			setHeader:   true,
			presented:   serverToken,
			wantStatus:  http.StatusOK,
			wantNext:    true,
		},
		{
			name:        "誤ったトークンは403",
			serverToken: serverToken,
			setHeader:   true,
			presented:   "wrong-token",
			wantStatus:  http.StatusForbidden,
			wantNext:    false,
		},
		{
			name:        "トークン無しは403",
			serverToken: serverToken,
			setHeader:   false,
			presented:   "",
			wantStatus:  http.StatusForbidden,
			wantNext:    false,
		},
		{
			name:        "サーバトークンが空だと管理機能は無効",
			serverToken: "",
			setHeader:   true,
			presented:   "anything",
			wantStatus:  http.StatusForbidden,
			wantNext:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var nextCalled bool
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextCalled = true
				w.WriteHeader(http.StatusOK)
			})
			onForbidden := func(w http.ResponseWriter, r *http.Request, err error) {
				w.WriteHeader(http.StatusForbidden)
			}
			h := RequireAdmin(tt.serverToken, onForbidden)(next)

			req := httptest.NewRequest(http.MethodPost, "/admin/quotes", nil)
			if tt.setHeader {
				req.Header.Set(AdminTokenHeader, tt.presented)
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			assert.Equal(t, tt.wantNext, nextCalled)
		})
	}
}
