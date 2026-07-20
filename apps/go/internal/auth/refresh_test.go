package auth

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
				require.NoError(t, err)
				require.NotEmpty(t, token)
				assert.Len(t, token, wantLen)

				decoded, err := base64.RawURLEncoding.DecodeString(token)
				require.NoError(t, err)
				assert.Len(t, decoded, refreshTokenBytes)

				assert.False(t, seen[token], "duplicate token generated: %q", token)
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

			assert.Len(t, ha, wantLen)
			_, err := hex.DecodeString(ha)
			assert.NoError(t, err)

			if tt.same {
				assert.Equal(t, ha, hb)
			} else {
				assert.NotEqual(t, ha, hb)
			}
		})
	}
}
