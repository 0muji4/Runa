package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
)

// refreshTokenBytes is the entropy of an opaque refresh token (256 bits).
const refreshTokenBytes = 32

// GenerateRefreshToken returns a new opaque refresh token: 256 random bits,
// base64url-encoded. The plaintext is returned to the client exactly once; the
// server persists only its hash (see HashRefreshToken).
func GenerateRefreshToken() (string, error) {
	b := make([]byte, refreshTokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate refresh token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// HashRefreshToken returns the hex-encoded SHA-256 of a refresh token. Tokens
// are high-entropy random values, so a plain (unsalted) hash is sufficient and
// lets the server look a token up by its hash in one indexed query.
func HashRefreshToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
