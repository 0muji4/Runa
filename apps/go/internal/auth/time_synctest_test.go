package auth

import (
	"errors"
	"testing"
	"testing/synctest"
	"time"
)

// ratelimit_test.go and jwt_test.go move time by overriding the unexported `now`
// field, so the code never runs against the real time package. These cover the
// same behaviour without the seam: inside a synctest bubble time.Now is virtual
// and a sleep past a TTL is instant.

func TestRateLimiterWindowWithRealClock(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const (
			max    = 3
			window = time.Minute
		)
		// No rl.now override: this is the limiter cmd/api constructs.
		rl := NewRateLimiter(max, window)

		for i := 0; i < max; i++ {
			if !rl.Allow("client-1") {
				t.Fatalf("request %d of %d was denied, want allowed", i+1, max)
			}
		}
		if rl.Allow("client-1") {
			t.Errorf("request %d exceeded the limit of %d but was allowed", max+1, max)
		}

		time.Sleep(window + time.Second)

		if !rl.Allow("client-1") {
			t.Error("the first request after the window elapsed was denied, want the counter reset")
		}
	})
}

func TestRateLimiterIsPerClientWithRealClock(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		rl := NewRateLimiter(1, time.Minute)

		if !rl.Allow("client-1") {
			t.Fatal("client-1's first request was denied, want allowed")
		}
		if rl.Allow("client-1") {
			t.Error("client-1's second request was allowed, want denied")
		}
		// One caller exhausting its quota must not lock anybody else out.
		if !rl.Allow("client-2") {
			t.Error("client-2's first request was denied, want allowed")
		}
	})
}

func TestAccessTokenExpiresWithRealClock(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const ttl = 15 * time.Minute
		// No ti.now override: signing and verification both use time.Now.
		ti := NewTokenIssuer("secret", ttl)

		token, expiresIn, err := ti.Issue("user-1")
		if err != nil {
			t.Fatalf("Issue() error = %v, want nil", err)
		}
		if want := int(ttl.Seconds()); expiresIn != want {
			t.Errorf("Issue() expiresIn = %d, want %d", expiresIn, want)
		}

		time.Sleep(ttl - time.Second)
		if _, err := ti.Verify(token); err != nil {
			t.Errorf("Verify() one second before expiry error = %v, want nil", err)
		}

		time.Sleep(2 * time.Second)
		if _, err := ti.Verify(token); !errors.Is(err, ErrTokenExpired) {
			t.Errorf("Verify() after expiry error = %v, want %v", err, ErrTokenExpired)
		}
	})
}
