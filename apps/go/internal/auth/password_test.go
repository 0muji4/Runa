package auth

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashPassword(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		password string
	}{
		{
			name:     "パスフレーズをハッシュ化できる",
			password: "correct horse battery staple",
		},
		{
			name:     "空パスワードでもハッシュ化できる",
			password: "",
		},
		{
			name:     "Unicodeパスワードをハッシュ化できる",
			password: "パスワード🔑",
		},
	}

	p := DefaultArgon2Params()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			first, err := HashPassword(tt.password, p)
			require.NoError(t, err)
			assert.True(t, strings.HasPrefix(first, "$argon2id$v="))

			second, err := HashPassword(tt.password, p)
			require.NoError(t, err)
			assert.NotEqual(t, first, second)
		})
	}
}

func TestVerifyPassword(t *testing.T) {
	t.Parallel()

	p := DefaultArgon2Params()
	encoded, err := HashPassword("right", p)
	require.NoError(t, err)

	tests := []struct {
		name    string
		input   string
		encoded string
		wantOK  bool
		wantErr error
	}{
		{
			name:    "正しいパスワードは一致する",
			input:   "right",
			encoded: encoded,
			wantOK:  true,
			wantErr: nil,
		},
		{
			name:    "誤ったパスワードは不一致",
			input:   "wrong",
			encoded: encoded,
			wantOK:  false,
			wantErr: nil,
		},
		{
			name:    "壊れたエンコードはErrInvalidHash",
			input:   "x",
			encoded: "not-a-phc-string",
			wantOK:  false,
			wantErr: ErrInvalidHash,
		},
		{
			name:    "非互換のargon2バージョンはErrIncompatibleVersion",
			input:   "x",
			encoded: "$argon2id$v=18$m=19456,t=2,p=1$c2FsdA$aGFzaA",
			wantOK:  false,
			wantErr: ErrIncompatibleVersion,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ok, err := VerifyPassword(tt.input, tt.encoded)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantOK, ok)
		})
	}
}
