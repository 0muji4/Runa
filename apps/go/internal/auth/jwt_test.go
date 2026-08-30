package auth

import (
	"errors"
	"testing"
	"time"
)

func TestTokenIssuer_Issue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		ttl  time.Duration
	}{
		{
			name: "15分TTLで発行し、同じsubjectに検証で戻る",
			ttl:  15 * time.Minute,
		},
		{
			name: "1時間TTLで発行できる",
			ttl:  time.Hour,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ti := NewTokenIssuer("secret", tt.ttl)

			token, expiresIn, err := ti.Issue("user-1")
			if err != nil {
				t.Fatalf("Issue(%q) error = %v, want nil", "user-1", err)
			}
			if want := int(tt.ttl.Seconds()); expiresIn != want {
				t.Errorf("Issue(%q) expiresIn = %d, want %d", "user-1", expiresIn, want)
			}

			userID, err := ti.Verify(token)
			if err != nil {
				t.Fatalf("Verify(Issue(%q)) error = %v, want nil", "user-1", err)
			}
			if userID != "user-1" {
				t.Errorf("Verify(Issue(%q)) = %q, want %q", "user-1", userID, "user-1")
			}
		})
	}
}

func TestTokenIssuer_Verify(t *testing.T) {
	t.Parallel()

	base := time.Now()
	tests := []struct {
		name    string
		setup   func(t *testing.T) (issuer *TokenIssuer, token string)
		wantErr error
	}{
		{
			name: "有効なトークンはsubjectを返す",
			setup: func(t *testing.T) (*TokenIssuer, string) {
				t.Helper()
				ti := NewTokenIssuer("secret", time.Minute)
				token, _, err := ti.Issue("user-1")
				if err != nil {
					t.Fatalf("Issue(%q) error = %v, want nil", "user-1", err)
				}
				return ti, token
			},
			wantErr: nil,
		},
		{
			name: "期限切れのトークンはErrTokenExpired",
			setup: func(t *testing.T) (*TokenIssuer, string) {
				t.Helper()
				ti := NewTokenIssuer("secret", time.Minute)
				ti.now = func() time.Time { return base }
				token, _, err := ti.Issue("user-1")
				if err != nil {
					t.Fatalf("Issue(%q) error = %v, want nil", "user-1", err)
				}
				ti.now = func() time.Time { return base.Add(2 * time.Minute) }
				return ti, token
			},
			wantErr: ErrTokenExpired,
		},
		{
			name: "別のsecretで検証するとErrInvalidToken",
			setup: func(t *testing.T) (*TokenIssuer, string) {
				t.Helper()
				token, _, err := NewTokenIssuer("secret-a", time.Minute).Issue("user-1")
				if err != nil {
					t.Fatalf("Issue(%q) error = %v, want nil", "user-1", err)
				}
				return NewTokenIssuer("secret-b", time.Minute), token
			},
			wantErr: ErrInvalidToken,
		},
		{
			name: "壊れた文字列はErrInvalidToken",
			setup: func(t *testing.T) (*TokenIssuer, string) {
				t.Helper()
				return NewTokenIssuer("secret", time.Minute), "not.a.jwt"
			},
			wantErr: ErrInvalidToken,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			issuer, token := tt.setup(t)

			userID, err := issuer.Verify(token)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("Verify(%q) error = %v, want %v", token, err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Verify(%q) error = %v, want nil", token, err)
			}
			if userID != "user-1" {
				t.Errorf("Verify(%q) = %q, want %q", token, userID, "user-1")
			}
		})
	}
}
