package auth

import (
	"errors"
	"testing"
	"testing/synctest"
	"time"
)

// Unlike ratelimit_test.go / jwt_test.go, these do not override `now`: synctest makes time.Now virtual.

func TestRateLimiterWindowWithRealClock(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const (
			max    = 3
			window = time.Minute
		)

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

		if !rl.Allow("client-2") {
			t.Error("client-2's first request was denied, want allowed")
		}
	})
}

func TestAccessTokenExpiresWithRealClock(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const ttl = 15 * time.Minute

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
