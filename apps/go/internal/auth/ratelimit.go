package auth

import (
	"net/http"
	"sync"
	"time"
)

// RateLimiter is a naive in-memory fixed-window limiter keyed by client IP. It
// is per-process (state resets on restart and is not shared across instances) —
// enough to blunt credential stuffing on a single-instance backend. A shared
// store (e.g. Redis) would be needed for a horizontally scaled deployment.
type RateLimiter struct {
	max    int
	window time.Duration
	now    func() time.Time

	mu      sync.Mutex
	buckets map[string]*bucket
}

type bucket struct {
	count int
	reset time.Time
}

// NewRateLimiter allows at most max requests per key within each window.
func NewRateLimiter(max int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		max:     max,
		window:  window,
		now:     time.Now,
		buckets: make(map[string]*bucket),
	}
}

// Allow records a hit for key and reports whether it is within the limit.
func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := rl.now()
	b, ok := rl.buckets[key]
	if !ok || now.After(b.reset) {
		rl.buckets[key] = &bucket{count: 1, reset: now.Add(rl.window)}
		return true
	}
	if b.count >= rl.max {
		return false
	}
	b.count++
	return true
}

// Middleware limits requests by client IP, delegating to onLimited (which sends
// the 429 body) when the window is exceeded.
func (rl *RateLimiter) Middleware(onLimited ErrorResponder) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !rl.Allow(clientKey(r)) {
				onLimited(w, r, nil)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// clientKey identifies the caller. The RealIP middleware upstream has already
// normalized RemoteAddr from X-Forwarded-For/X-Real-IP where trusted.
func clientKey(r *http.Request) string {
	return r.RemoteAddr
}
