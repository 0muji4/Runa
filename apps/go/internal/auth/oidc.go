package auth

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Well-known provider endpoints and issuers. Audiences (client IDs) are supplied
// per-deployment via config since they are app-specific.
const (
	AppleJWKSURL  = "https://appleid.apple.com/auth/keys"
	GoogleJWKSURL = "https://www.googleapis.com/oauth2/v3/certs"
)

var (
	// AppleIssuers is the accepted `iss` set for Apple ID tokens.
	AppleIssuers = []string{"https://appleid.apple.com"}
	// GoogleIssuers is the accepted `iss` set for Google ID tokens (Google uses
	// both the bare host and the https form).
	GoogleIssuers = []string{"https://accounts.google.com", "accounts.google.com"}
)

// ErrProviderVerification is returned when an Apple/Google ID token fails any
// verification step (signature, issuer, audience, expiry, subject).
var ErrProviderVerification = errors.New("auth: provider token verification failed")

// OIDCIdentity is the subset of verified ID-token claims the auth service needs.
type OIDCIdentity struct {
	Subject       string
	Email         string
	EmailVerified bool
	Name          string
}

// IDTokenVerifier verifies a provider ID token and returns its identity. Both
// the real OIDCVerifier and test fakes satisfy it.
type IDTokenVerifier interface {
	Verify(ctx context.Context, idToken string) (OIDCIdentity, error)
}

// KeySource returns RSA public keys keyed by their JWK "kid".
type KeySource interface {
	Keys(ctx context.Context) (map[string]*rsa.PublicKey, error)
}

// OIDCVerifier verifies RS256 ID tokens against a JWKS, an issuer allow-list and
// an audience allow-list.
type OIDCVerifier struct {
	issuers   []string
	audiences []string
	keys      KeySource
	now       func() time.Time
}

// NewOIDCVerifier builds a verifier for the given issuers, audiences (client
// IDs) and key source.
func NewOIDCVerifier(issuers, audiences []string, keys KeySource) *OIDCVerifier {
	return &OIDCVerifier{issuers: issuers, audiences: audiences, keys: keys, now: time.Now}
}

// idTokenClaims covers the claims used across Apple and Google. email_verified
// is `any` because Apple encodes it as the string "true" while Google uses a
// JSON boolean.
type idTokenClaims struct {
	Email         string `json:"email"`
	EmailVerified any    `json:"email_verified"`
	Name          string `json:"name"`
	jwt.RegisteredClaims
}

// Verify checks signature, issuer, audience and expiry, returning the identity.
func (v *OIDCVerifier) Verify(ctx context.Context, idToken string) (OIDCIdentity, error) {
	keys, err := v.keys.Keys(ctx)
	if err != nil {
		return OIDCIdentity{}, fmt.Errorf("%w: fetch keys: %v", ErrProviderVerification, err)
	}

	var claims idTokenClaims
	_, err = jwt.ParseWithClaims(idToken, &claims,
		func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, fmt.Errorf("unexpected signing method %v", t.Header["alg"])
			}
			kid, _ := t.Header["kid"].(string)
			key, ok := keys[kid]
			if !ok {
				return nil, fmt.Errorf("unknown key id %q", kid)
			}
			return key, nil
		},
		jwt.WithValidMethods([]string{"RS256"}),
		jwt.WithTimeFunc(v.now),
	)
	if err != nil {
		return OIDCIdentity{}, fmt.Errorf("%w: %v", ErrProviderVerification, err)
	}

	if !contains(v.issuers, claims.Issuer) {
		return OIDCIdentity{}, fmt.Errorf("%w: issuer %q not allowed", ErrProviderVerification, claims.Issuer)
	}
	if !audienceAllowed(claims.Audience, v.audiences) {
		return OIDCIdentity{}, fmt.Errorf("%w: audience not allowed", ErrProviderVerification)
	}
	if claims.Subject == "" {
		return OIDCIdentity{}, fmt.Errorf("%w: missing subject", ErrProviderVerification)
	}

	return OIDCIdentity{
		Subject:       claims.Subject,
		Email:         claims.Email,
		EmailVerified: truthy(claims.EmailVerified),
		Name:          claims.Name,
	}, nil
}

func contains(set []string, v string) bool {
	for _, s := range set {
		if s == v {
			return true
		}
	}
	return false
}

func audienceAllowed(aud jwt.ClaimStrings, allowed []string) bool {
	for _, a := range aud {
		if contains(allowed, a) {
			return true
		}
	}
	return false
}

func truthy(v any) bool {
	switch t := v.(type) {
	case bool:
		return t
	case string:
		return t == "true"
	default:
		return false
	}
}

// RemoteJWKS fetches and caches a provider's JWKS over HTTP. Providers rotate
// keys rarely, so a time-based cache avoids a network round trip per token.
type RemoteJWKS struct {
	url    string
	client *http.Client
	ttl    time.Duration
	now    func() time.Time

	mu      sync.Mutex
	cached  map[string]*rsa.PublicKey
	fetched time.Time
}

// NewRemoteJWKS builds a JWKS source for url with a one-hour cache.
func NewRemoteJWKS(url string) *RemoteJWKS {
	return &RemoteJWKS{
		url:    url,
		client: &http.Client{Timeout: 10 * time.Second},
		ttl:    time.Hour,
		now:    time.Now,
	}
}

// Keys returns the cached key set, refreshing it over HTTP when stale.
func (r *RemoteJWKS) Keys(ctx context.Context) (map[string]*rsa.PublicKey, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.cached != nil && r.now().Sub(r.fetched) < r.ttl {
		return r.cached, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, r.url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("jwks fetch: status %d", resp.StatusCode)
	}

	keys, err := parseJWKS(resp.Body)
	if err != nil {
		return nil, err
	}
	r.cached = keys
	r.fetched = r.now()
	return keys, nil
}

// StaticKeySource is a fixed in-memory key set, used in tests.
type StaticKeySource map[string]*rsa.PublicKey

// Keys returns the static key set.
func (s StaticKeySource) Keys(context.Context) (map[string]*rsa.PublicKey, error) {
	return s, nil
}

type jwksDocument struct {
	Keys []struct {
		Kty string `json:"kty"`
		Kid string `json:"kid"`
		N   string `json:"n"`
		E   string `json:"e"`
	} `json:"keys"`
}

func parseJWKS(r io.Reader) (map[string]*rsa.PublicKey, error) {
	var doc jwksDocument
	if err := json.NewDecoder(r).Decode(&doc); err != nil {
		return nil, fmt.Errorf("decode jwks: %w", err)
	}

	out := make(map[string]*rsa.PublicKey, len(doc.Keys))
	for _, k := range doc.Keys {
		if k.Kty != "RSA" {
			continue
		}
		pub, err := rsaPublicKeyFromJWK(k.N, k.E)
		if err != nil {
			return nil, err
		}
		out[k.Kid] = pub
	}
	return out, nil
}

// rsaPublicKeyFromJWK reconstructs an RSA public key from the base64url modulus
// and exponent of a JWK.
func rsaPublicKeyFromJWK(nStr, eStr string) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(nStr)
	if err != nil {
		return nil, fmt.Errorf("decode jwk modulus: %w", err)
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(eStr)
	if err != nil {
		return nil, fmt.Errorf("decode jwk exponent: %w", err)
	}
	return &rsa.PublicKey{
		N: new(big.Int).SetBytes(nBytes),
		E: int(new(big.Int).SetBytes(eBytes).Int64()),
	}, nil
}
