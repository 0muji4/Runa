package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const testKID = "test-key-1"

// signedIDToken mints an RS256 token with the given claims, signed by priv and
// tagged with testKID, mimicking an Apple/Google ID token.
func signedIDToken(t *testing.T, priv *rsa.PrivateKey, claims jwt.MapClaims) string {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tok.Header["kid"] = testKID
	signed, err := tok.SignedString(priv)
	if err != nil {
		t.Fatalf("sign id token: %v", err)
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

func TestOIDCVerifierSuccess(t *testing.T) {
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	v := newVerifier(t, priv, GoogleIssuers, []string{"client-123"})

	id, err := v.Verify(context.Background(), signedIDToken(t, priv, baseClaims(time.Now())))
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if id.Subject != "google-sub-1" || id.Email != "user@example.com" || !id.EmailVerified {
		t.Fatalf("identity = %+v, unexpected", id)
	}
}

func TestOIDCVerifierAppleStringEmailVerified(t *testing.T) {
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	v := newVerifier(t, priv, AppleIssuers, []string{"com.runa.app"})
	claims := baseClaims(time.Now())
	claims["iss"] = "https://appleid.apple.com"
	claims["aud"] = "com.runa.app"
	claims["email_verified"] = "true" // Apple sends a string

	id, err := v.Verify(context.Background(), signedIDToken(t, priv, claims))
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if !id.EmailVerified {
		t.Fatal("EmailVerified = false for Apple's string \"true\", want true")
	}
}

func TestOIDCVerifierRejectsWrongAudience(t *testing.T) {
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	v := newVerifier(t, priv, GoogleIssuers, []string{"someone-else"})
	if _, err := v.Verify(context.Background(), signedIDToken(t, priv, baseClaims(time.Now()))); !errors.Is(err, ErrProviderVerification) {
		t.Fatalf("Verify error = %v, want ErrProviderVerification", err)
	}
}

func TestOIDCVerifierRejectsWrongIssuer(t *testing.T) {
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	v := newVerifier(t, priv, AppleIssuers, []string{"client-123"}) // token iss is Google
	if _, err := v.Verify(context.Background(), signedIDToken(t, priv, baseClaims(time.Now()))); !errors.Is(err, ErrProviderVerification) {
		t.Fatalf("Verify error = %v, want ErrProviderVerification", err)
	}
}

func TestOIDCVerifierRejectsExpired(t *testing.T) {
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	v := newVerifier(t, priv, GoogleIssuers, []string{"client-123"})
	claims := baseClaims(time.Now().Add(-2 * time.Hour)) // exp one hour in the past
	if _, err := v.Verify(context.Background(), signedIDToken(t, priv, claims)); !errors.Is(err, ErrProviderVerification) {
		t.Fatalf("Verify error = %v, want ErrProviderVerification", err)
	}
}

func TestOIDCVerifierRejectsUnknownKey(t *testing.T) {
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	other, _ := rsa.GenerateKey(rand.Reader, 2048)
	// Verifier trusts `other`, but the token is signed by `priv`.
	v := NewOIDCVerifier(GoogleIssuers, []string{"client-123"}, StaticKeySource{testKID: &other.PublicKey})
	if _, err := v.Verify(context.Background(), signedIDToken(t, priv, baseClaims(time.Now()))); !errors.Is(err, ErrProviderVerification) {
		t.Fatalf("Verify error = %v, want ErrProviderVerification", err)
	}
}
