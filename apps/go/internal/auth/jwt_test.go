package auth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			require.NoError(t, err)
			assert.Equal(t, int(tt.ttl.Seconds()), expiresIn)

			userID, err := ti.Verify(token)
			require.NoError(t, err)
			assert.Equal(t, "user-1", userID)
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
				ti := NewTokenIssuer("secret", time.Minute)
				token, _, err := ti.Issue("user-1")
				require.NoError(t, err)
				return ti, token
			},
			wantErr: nil,
		},
		{
			name: "期限切れのトークンはErrTokenExpired",
			setup: func(t *testing.T) (*TokenIssuer, string) {
				ti := NewTokenIssuer("secret", time.Minute)
				ti.now = func() time.Time { return base }
				token, _, err := ti.Issue("user-1")
				require.NoError(t, err)
				ti.now = func() time.Time { return base.Add(2 * time.Minute) }
				return ti, token
			},
			wantErr: ErrTokenExpired,
		},
		{
			name: "別のsecretで検証するとErrInvalidToken",
			setup: func(t *testing.T) (*TokenIssuer, string) {
				token, _, err := NewTokenIssuer("secret-a", time.Minute).Issue("user-1")
				require.NoError(t, err)
				return NewTokenIssuer("secret-b", time.Minute), token
			},
			wantErr: ErrInvalidToken,
		},
		{
			name: "壊れた文字列はErrInvalidToken",
			setup: func(t *testing.T) (*TokenIssuer, string) {
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
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, "user-1", userID)
		})
	}
}
