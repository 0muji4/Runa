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

// GenerateRefreshToken returns a new opaque base64url refresh token; the server must persist only its hash.
func GenerateRefreshToken() (string, error) {
	b := make([]byte, refreshTokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate refresh token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// HashRefreshToken returns the hex-encoded SHA-256 of a refresh token (unsalted: the token is already high-entropy).
func HashRefreshToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
