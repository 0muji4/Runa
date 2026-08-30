package auth

import (
	"strings"
	"testing"
)

// Both targets cover hand-written parsers on the login path, where a panic is a
// crash rather than a failed login.

func FuzzDecodeArgon2Hash(f *testing.F) {
	// A valid hash, then each way the layout can be wrong.
	valid, err := HashPassword("seed", DefaultArgon2Params())
	if err != nil {
		f.Fatalf("HashPassword() error = %v, want nil", err)
	}
	seeds := []string{
		valid,
		"",
		"$",
		"$argon2id$v=19$m=19456,t=2,p=1$c2FsdA$aGFzaA",
		"$argon2id$v=18$m=19456,t=2,p=1$c2FsdA$aGFzaA",
		"$argon2i$v=19$m=19456,t=2,p=1$c2FsdA$aGFzaA",
		"$argon2id$v=notanumber$m=1,t=1,p=1$c2FsdA$aGFzaA",
		"$argon2id$v=19$m=,t=,p=$c2FsdA$aGFzaA",
		"$argon2id$v=19$m=99999999999999999999,t=2,p=1$c2FsdA$aGFzaA",
		"$argon2id$v=19$m=19456,t=2,p=1$!!!not-base64!!!$aGFzaA",
		"$argon2id$v=19$m=19456,t=2,p=1$c2FsdA$!!!not-base64!!!",
		strings.Repeat("$", 64),
	}
	for _, s := range seeds {
		f.Add(s)
	}

	// Any input may error; none may panic.
	f.Fuzz(func(t *testing.T, encoded string) {
		_, _ = VerifyPassword("password", encoded)
	})
}

func FuzzParseJWKS(f *testing.F) {
	seeds := []string{
		`{"keys":[]}`,
		`{"keys":[{"kty":"RSA","kid":"a","n":"AQAB","e":"AQAB"}]}`,
		`{"keys":[{"kty":"EC","kid":"a","crv":"P-256"}]}`,
		`{"keys":[{"kty":"RSA","kid":"a","n":"!!!","e":"AQAB"}]}`,
		`{"keys":[{"kty":"RSA","kid":"a","n":"","e":""}]}`,
		`{"keys":null}`,
		`{}`,
		``,
		`not json at all`,
		`{"keys":[{"kty":"RSA","kid":"a","n":"` + strings.Repeat("A", 4096) + `","e":"AQAB"}]}`,
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, doc string) {
		keys, err := parseJWKS(strings.NewReader(doc))
		if err != nil {
			return
		}
		// A successful parse must not hand back a key that panics on use.
		for kid, key := range keys {
			if key == nil {
				t.Errorf("parseJWKS returned a nil key for kid %q", kid)
				continue
			}
			if key.N == nil {
				t.Errorf("parseJWKS returned a key with a nil modulus for kid %q", kid)
			}
		}
	})
}
