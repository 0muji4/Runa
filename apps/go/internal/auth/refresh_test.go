package auth

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"testing"
)

func TestGenerateRefreshToken(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		count int
	}{
		{
			name:  "単一トークンが整形されている",
			count: 1,
		},
		{
			name:  "多数のトークンが一意である",
			count: 100,
		},
	}

	wantLen := base64.RawURLEncoding.EncodedLen(refreshTokenBytes)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			seen := make(map[string]bool, tt.count)
			for i := 0; i < tt.count; i++ {
				token, err := GenerateRefreshToken()
				if err != nil {
					t.Fatalf("GenerateRefreshToken() error = %v, want nil", err)
				}
				if len(token) != wantLen {
					t.Errorf("len(GenerateRefreshToken()) = %d, want %d", len(token), wantLen)
				}

				decoded, err := base64.RawURLEncoding.DecodeString(token)
				if err != nil {
					t.Fatalf("base64 decode of %q error = %v, want nil", token, err)
				}
				if len(decoded) != refreshTokenBytes {
					t.Errorf("decoded entropy = %d bytes, want %d", len(decoded), refreshTokenBytes)
				}

				if seen[token] {
					t.Fatalf("GenerateRefreshToken() returned a duplicate token %q at iteration %d", token, i)
				}
				seen[token] = true
			}
		})
	}
}

func TestHashRefreshToken(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		a    string
		b    string
		same bool
	}{
		{
			name: "同じ入力は決定的に同じダイジェスト",
			a:    "token-abc",
			b:    "token-abc",
			same: true,
		},
		{
			name: "異なる入力は異なるダイジェスト",
			a:    "token-abc",
			b:    "token-xyz",
			same: false,
		},
	}

	wantLen := hex.EncodedLen(sha256.Size)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ha := HashRefreshToken(tt.a)
			hb := HashRefreshToken(tt.b)

			if len(ha) != wantLen {
				t.Errorf("len(HashRefreshToken(%q)) = %d, want %d", tt.a, len(ha), wantLen)
			}
			if _, err := hex.DecodeString(ha); err != nil {
				t.Errorf("HashRefreshToken(%q) = %q, want hex-decodable: %v", tt.a, ha, err)
			}

			if got := ha == hb; got != tt.same {
				t.Errorf("HashRefreshToken(%q) == HashRefreshToken(%q) = %t, want %t",
					tt.a, tt.b, got, tt.same)
			}
		})
	}
}
