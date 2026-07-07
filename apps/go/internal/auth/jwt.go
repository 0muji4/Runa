package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	// ErrInvalidToken means the access token failed signature/structural checks.
	ErrInvalidToken = errors.New("auth: invalid access token")
	// ErrTokenExpired means the access token was well-formed but past its exp.
	ErrTokenExpired = errors.New("auth: access token expired")
)

// TokenIssuer mints and verifies short-lived HS256 access tokens. A symmetric
// secret is sufficient because the same service both signs and verifies.
type TokenIssuer struct {
	secret []byte
	ttl    time.Duration
	now    func() time.Time
}

// NewTokenIssuer builds an issuer from the HS256 secret and access-token TTL.
func NewTokenIssuer(secret string, ttl time.Duration) *TokenIssuer {
	return &TokenIssuer{secret: []byte(secret), ttl: ttl, now: time.Now}
}

// accessClaims are the registered claims plus a token-type marker so a refresh
// or provider token can never be replayed as an access token.
type accessClaims struct {
	Type string `json:"typ"`
	jwt.RegisteredClaims
}

// Issue returns a signed access token for userID plus its lifetime in seconds.
func (ti *TokenIssuer) Issue(userID string) (token string, expiresIn int, err error) {
	now := ti.now()
	claims := accessClaims{
		Type: "access",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ti.ttl)),
		},
	}

	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(ti.secret)
	if err != nil {
		return "", 0, fmt.Errorf("sign access token: %w", err)
	}
	return signed, int(ti.ttl.Seconds()), nil
}

// Verify parses and validates an access token, returning its subject (user id).
// It reports ErrTokenExpired specifically so the caller can distinguish an
// expired token (client should refresh) from a malformed one.
func (ti *TokenIssuer) Verify(tokenString string) (string, error) {
	var claims accessClaims
	_, err := jwt.ParseWithClaims(tokenString, &claims,
		func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method %v", t.Header["alg"])
			}
			return ti.secret, nil
		},
		jwt.WithValidMethods([]string{"HS256"}),
		jwt.WithTimeFunc(ti.now),
	)
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return "", ErrTokenExpired
		}
		return "", ErrInvalidToken
	}
	if claims.Type != "access" || claims.Subject == "" {
		return "", ErrInvalidToken
	}
	return claims.Subject, nil
}
