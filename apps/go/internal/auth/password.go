// Package auth holds the cryptographic and token primitives for authentication:
// password hashing, access-token minting/verification, refresh-token generation,
// OIDC (Apple/Google) ID-token verification, the Bearer middleware and a simple
// rate limiter. It has no knowledge of HTTP responses or persistence so each
// piece is independently testable.
package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Argon2Params configures the argon2id password hash. Defaults follow the OWASP
// Password Storage Cheat Sheet (m=19 MiB, t=2, p=1), the current first-choice
// algorithm for password storage.
type Argon2Params struct {
	Memory      uint32 // KiB
	Iterations  uint32
	Parallelism uint8
	SaltLength  uint32 // bytes
	KeyLength   uint32 // bytes
}

// DefaultArgon2Params returns the OWASP-recommended argon2id parameters.
func DefaultArgon2Params() Argon2Params {
	return Argon2Params{
		Memory:      19 * 1024, // 19 MiB
		Iterations:  2,
		Parallelism: 1,
		SaltLength:  16,
		KeyLength:   32,
	}
}

var (
	// ErrInvalidHash is returned when an encoded hash is not a valid argon2id
	// PHC string.
	ErrInvalidHash = errors.New("auth: invalid argon2 hash format")
	// ErrIncompatibleVersion is returned when the encoded argon2 version differs
	// from the one this build links against.
	ErrIncompatibleVersion = errors.New("auth: incompatible argon2 version")
)

// HashPassword returns a PHC-formatted argon2id hash embedding the random salt
// and parameters, e.g. $argon2id$v=19$m=19456,t=2,p=1$<salt>$<hash>. Verifying
// later needs only the encoded string.
func HashPassword(password string, p Argon2Params) (string, error) {
	salt := make([]byte, p.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate salt: %w", err)
	}

	key := argon2.IDKey([]byte(password), salt, p.Iterations, p.Memory, p.Parallelism, p.KeyLength)

	b64 := base64.RawStdEncoding
	return fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, p.Memory, p.Iterations, p.Parallelism,
		b64.EncodeToString(salt), b64.EncodeToString(key),
	), nil
}

// VerifyPassword reports whether password matches the encoded argon2id hash. The
// comparison is constant-time. A non-nil error means the hash was malformed.
func VerifyPassword(password, encoded string) (bool, error) {
	p, salt, want, err := decodeArgon2Hash(encoded)
	if err != nil {
		return false, err
	}

	got := argon2.IDKey([]byte(password), salt, p.Iterations, p.Memory, p.Parallelism, p.KeyLength)
	if subtle.ConstantTimeEq(int32(len(want)), int32(len(got))) == 0 {
		return false, nil
	}
	return subtle.ConstantTimeCompare(want, got) == 1, nil
}

// decodeArgon2Hash parses a PHC argon2id string back into its params, salt and
// derived key.
func decodeArgon2Hash(encoded string) (Argon2Params, []byte, []byte, error) {
	// Layout: ["", "argon2id", "v=19", "m=..,t=..,p=..", <salt>, <key>].
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return Argon2Params{}, nil, nil, ErrInvalidHash
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return Argon2Params{}, nil, nil, ErrInvalidHash
	}
	if version != argon2.Version {
		return Argon2Params{}, nil, nil, ErrIncompatibleVersion
	}

	var p Argon2Params
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &p.Memory, &p.Iterations, &p.Parallelism); err != nil {
		return Argon2Params{}, nil, nil, ErrInvalidHash
	}

	b64 := base64.RawStdEncoding
	salt, err := b64.DecodeString(parts[4])
	if err != nil {
		return Argon2Params{}, nil, nil, ErrInvalidHash
	}
	key, err := b64.DecodeString(parts[5])
	if err != nil {
		return Argon2Params{}, nil, nil, ErrInvalidHash
	}

	p.SaltLength = uint32(len(salt))
	p.KeyLength = uint32(len(key))
	return p, salt, key, nil
}
