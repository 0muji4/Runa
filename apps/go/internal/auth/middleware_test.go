package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

type stubVerifier struct {
	userID string
	err    error
}

func (s stubVerifier) Verify(string) (string, error) {
	return s.userID, s.err
}

func TestRequireAuth(t *testing.T) {
	t.Parallel()

	const wantUserID = "user-42"

	tests := []struct {
		name       string
		setHeader  bool
		authHeader string
		verifier   AccessVerifier
		wantStatus int
		wantNext   bool
		wantCtxID  string
	}{
		{
			name:       "有効なBearerは通過しコンテキストにIDを設定する",
			setHeader:  true,
			authHeader: "Bearer good-token",
			verifier:   stubVerifier{userID: wantUserID},
			wantStatus: http.StatusOK,
			wantNext:   true,
			wantCtxID:  wantUserID,
		},
		{
			name:       "Authorizationヘッダー無しは401",
			setHeader:  false,
			authHeader: "",
			verifier:   stubVerifier{userID: wantUserID},
			wantStatus: http.StatusUnauthorized,
			wantNext:   false,
			wantCtxID:  "",
		},
		{
			name:       "Basicスキームは401",
			setHeader:  true,
			authHeader: "Basic dXNlcjpwYXNz",
			verifier:   stubVerifier{userID: wantUserID},
			wantStatus: http.StatusUnauthorized,
			wantNext:   false,
			wantCtxID:  "",
		},
		{
			name:       "空のBearerトークンは401",
			setHeader:  true,
			authHeader: "Bearer ",
			verifier:   stubVerifier{userID: wantUserID},
			wantStatus: http.StatusUnauthorized,
			wantNext:   false,
			wantCtxID:  "",
		},
		{
			name:       "検証失敗は401",
			setHeader:  true,
			authHeader: "Bearer bad-token",
			verifier:   stubVerifier{err: ErrInvalidToken},
			wantStatus: http.StatusUnauthorized,
			wantNext:   false,
			wantCtxID:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var (
				nextCalled bool
				gotCtxID   string
			)
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextCalled = true
				gotCtxID, _ = UserIDFromContext(r.Context())
				w.WriteHeader(http.StatusOK)
			})
			onUnauthorized := func(w http.ResponseWriter, r *http.Request, err error) {
				w.WriteHeader(http.StatusUnauthorized)
			}
			h := RequireAuth(tt.verifier, onUnauthorized)(next)

			req := httptest.NewRequest(http.MethodGet, "/me", nil)
			if tt.setHeader {
				req.Header.Set("Authorization", tt.authHeader)
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			assert.Equal(t, tt.wantNext, nextCalled)
			if tt.wantNext {
				assert.Equal(t, tt.wantCtxID, gotCtxID)
			}
		})
	}
}

func TestUserIDFromContext(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		ctx    context.Context
		wantID string
		wantOK bool
	}{
		{
			name:   "値が存在する",
			ctx:    context.WithValue(context.Background(), userIDKey, "user-7"),
			wantID: "user-7",
			wantOK: true,
		},
		{
			name:   "値が無い",
			ctx:    context.Background(),
			wantID: "",
			wantOK: false,
		},
		{
			name:   "空文字は無しとして扱う",
			ctx:    context.WithValue(context.Background(), userIDKey, ""),
			wantID: "",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			id, ok := UserIDFromContext(tt.ctx)
			assert.Equal(t, tt.wantID, id)
			assert.Equal(t, tt.wantOK, ok)
		})
	}
}
