package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/go-cmp/cmp"
)

const testKID = "test-key-1"

func signedIDToken(t *testing.T, priv *rsa.PrivateKey, claims jwt.MapClaims) string {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tok.Header["kid"] = testKID
	signed, err := tok.SignedString(priv)
	if err != nil {
		t.Fatalf("SignedString() error = %v, want nil", err)
	}
	return signed
}

func newVerifier(t *testing.T, priv *rsa.PrivateKey, issuers, audiences []string) *OIDCVerifier {
	t.Helper()
	return NewOIDCVerifier(issuers, audiences, StaticKeySource{testKID: &priv.PublicKey})
}

func baseClaims(now time.Time) jwt.MapClaims {
	return jwt.MapClaims{
		"iss":            "https://accounts.google.com",
		"aud":            "client-123",
		"sub":            "google-sub-1",
		"email":          "user@example.com",
		"email_verified": true,
		"name":           "Test User",
		"iat":            now.Unix(),
		"exp":            now.Add(time.Hour).Unix(),
	}
}

func appleClaims(now time.Time) jwt.MapClaims {
	c := baseClaims(now)
	c["iss"] = "https://appleid.apple.com"
	c["aud"] = "com.runa.app"
	// Apple encodes email_verified as the JSON string "true", not a boolean.
	c["email_verified"] = "true"
	return c
}

func TestOIDCVerifier_Verify(t *testing.T) {
	t.Parallel()

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa.GenerateKey() error = %v, want nil", err)
	}
	other, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa.GenerateKey() error = %v, want nil", err)
	}
	now := time.Now()

	wantIdentity := OIDCIdentity{
		Subject:       "google-sub-1",
		Email:         "user@example.com",
		EmailVerified: true,
		Name:          "Test User",
	}

	tests := []struct {
		name         string
		verifier     *OIDCVerifier
		token        string
		wantErr      error
		wantIdentity OIDCIdentity
	}{
		{
			name:         "有効なGoogleトークンはidentityを返す",
			verifier:     newVerifier(t, priv, GoogleIssuers, []string{"client-123"}),
			token:        signedIDToken(t, priv, baseClaims(now)),
			wantErr:      nil,
			wantIdentity: wantIdentity,
		},
		{
			name:         "AppleのemailVerifiedは文字列trueでも真",
			verifier:     newVerifier(t, priv, AppleIssuers, []string{"com.runa.app"}),
			token:        signedIDToken(t, priv, appleClaims(now)),
			wantErr:      nil,
			wantIdentity: wantIdentity,
		},
		{
			name:         "aud不一致は拒否される",
			verifier:     newVerifier(t, priv, GoogleIssuers, []string{"someone-else"}),
			token:        signedIDToken(t, priv, baseClaims(now)),
			wantErr:      ErrProviderVerification,
			wantIdentity: OIDCIdentity{},
		},
		{
			name:         "iss不一致は拒否される",
			verifier:     newVerifier(t, priv, AppleIssuers, []string{"client-123"}),
			token:        signedIDToken(t, priv, baseClaims(now)),
			wantErr:      ErrProviderVerification,
			wantIdentity: OIDCIdentity{},
		},
		{
			name:         "期限切れトークンは拒否される",
			verifier:     newVerifier(t, priv, GoogleIssuers, []string{"client-123"}),
			token:        signedIDToken(t, priv, baseClaims(now.Add(-2*time.Hour))),
			wantErr:      ErrProviderVerification,
			wantIdentity: OIDCIdentity{},
		},
		{
			name:         "未知の署名鍵は拒否される",
			verifier:     NewOIDCVerifier(GoogleIssuers, []string{"client-123"}, StaticKeySource{testKID: &other.PublicKey}),
			token:        signedIDToken(t, priv, baseClaims(now)),
			wantErr:      ErrProviderVerification,
			wantIdentity: OIDCIdentity{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			id, err := tt.verifier.Verify(context.Background(), tt.token)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("Verify() error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Verify() error = %v, want nil", err)
			}
			if diff := cmp.Diff(tt.wantIdentity, id); diff != "" {
				t.Errorf("Verify() identity mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
