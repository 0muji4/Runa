package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRateLimiter_Allow(t *testing.T) {
	t.Parallel()

	const (
		max    = 3
		window = time.Minute
	)
	base := time.Now()

	type step struct {
		advance     time.Duration
		wantAllowed bool
	}
	tests := []struct {
		name  string
		steps []step
	}{
		{
			name: "上限内のリクエストは許可される",
			steps: []step{
				{
					advance:     0,
					wantAllowed: true,
				},
				{
					advance:     0,
					wantAllowed: true,
				},
			},
		},
		{
			name: "上限ちょうどは許可され次は拒否される",
			steps: []step{
				{
					advance:     0,
					wantAllowed: true,
				},
				{
					advance:     0,
					wantAllowed: true,
				},
				{
					advance:     0,
					wantAllowed: true,
				},
				{
					advance:     0,
					wantAllowed: false,
				},
			},
		},
		{
			name: "ウィンドウ経過でカウンタがリセットされる",
			steps: []step{
				{
					advance:     0,
					wantAllowed: true,
				},
				{
					advance:     0,
					wantAllowed: true,
				},
				{
					advance:     0,
					wantAllowed: true,
				},
				{
					advance:     0,
					wantAllowed: false,
				},
				{
					advance:     window + time.Second,
					wantAllowed: true,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rl := NewRateLimiter(max, window)
			now := base
			rl.now = func() time.Time { return now }

			for i, s := range tt.steps {
				now = base.Add(s.advance)
				assert.Equal(t, s.wantAllowed, rl.Allow("client-1"), "step %d", i)
			}
		})
	}
}

func TestRateLimiter_Middleware(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		max        int
		requests   int
		wantNext   bool
		wantStatus int
	}{
		{
			name:       "上限内はnextを呼び200を返す",
			max:        5,
			requests:   1,
			wantNext:   true,
			wantStatus: http.StatusOK,
		},
		{
			name:       "上限超過はonLimitedを呼び429を返す",
			max:        1,
			requests:   2,
			wantNext:   false,
			wantStatus: http.StatusTooManyRequests,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rl := NewRateLimiter(tt.max, time.Minute)

			var nextCalled bool
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextCalled = true
				w.WriteHeader(http.StatusOK)
			})
			onLimited := func(w http.ResponseWriter, r *http.Request, err error) {
				w.WriteHeader(http.StatusTooManyRequests)
			}
			h := rl.Middleware(onLimited)(next)

			var rec *httptest.ResponseRecorder
			for i := 0; i < tt.requests; i++ {
				nextCalled = false
				rec = httptest.NewRecorder()
				// httptest.NewRequest sets a fixed RemoteAddr, so every request shares one limiter bucket.
				req := httptest.NewRequest(http.MethodPost, "/login", nil)
				h.ServeHTTP(rec, req)
			}

			assert.Equal(t, tt.wantNext, nextCalled)
			assert.Equal(t, tt.wantStatus, rec.Code)
		})
	}
}
