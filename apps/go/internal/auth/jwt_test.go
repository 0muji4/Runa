package auth

import (
	"errors"
	"testing"
	"time"
)

func TestTokenIssuerIssueAndVerify(t *testing.T) {
	ti := NewTokenIssuer("secret", 15*time.Minute)

	token, expiresIn, err := ti.Issue("user-1")
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	if expiresIn != int((15 * time.Minute).Seconds()) {
		t.Errorf("expiresIn = %d, want %d", expiresIn, int((15*time.Minute).Seconds()))
	}

	userID, err := ti.Verify(token)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if userID != "user-1" {
		t.Errorf("subject = %q, want %q", userID, "user-1")
	}
}

func TestTokenIssuerRejectsExpired(t *testing.T) {
	ti := NewTokenIssuer("secret", time.Minute)
	base := time.Now()
	ti.now = func() time.Time { return base }

	token, _, err := ti.Issue("user-1")
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}

	// Advance past expiry.
	ti.now = func() time.Time { return base.Add(2 * time.Minute) }
	if _, err := ti.Verify(token); !errors.Is(err, ErrTokenExpired) {
		t.Fatalf("Verify error = %v, want ErrTokenExpired", err)
	}
}

func TestTokenIssuerRejectsWrongSecret(t *testing.T) {
	issued, _, _ := NewTokenIssuer("secret-a", time.Minute).Issue("user-1")
	if _, err := NewTokenIssuer("secret-b", time.Minute).Verify(issued); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("Verify error = %v, want ErrInvalidToken", err)
	}
}

func TestTokenIssuerRejectsGarbage(t *testing.T) {
	if _, err := NewTokenIssuer("secret", time.Minute).Verify("not.a.jwt"); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("Verify error = %v, want ErrInvalidToken", err)
	}
}
