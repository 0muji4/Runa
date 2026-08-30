package auth

import (
	"errors"
	"strings"
	"testing"
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
			if err != nil {
				t.Fatalf("HashPassword(%q) error = %v, want nil", tt.password, err)
			}
			const wantPrefix = "$argon2id$v="
			if !strings.HasPrefix(first, wantPrefix) {
				t.Errorf("HashPassword(%q) = %q, want prefix %q", tt.password, first, wantPrefix)
			}

			// The salt is random per call, so the same password must never hash twice alike.
			second, err := HashPassword(tt.password, p)
			if err != nil {
				t.Fatalf("HashPassword(%q) error = %v, want nil", tt.password, err)
			}
			if first == second {
				t.Errorf("HashPassword(%q) returned the same hash twice (%q); salt is not random",
					tt.password, first)
			}
		})
	}
}

func TestVerifyPassword(t *testing.T) {
	t.Parallel()

	p := DefaultArgon2Params()
	encoded, err := HashPassword("right", p)
	if err != nil {
		t.Fatalf("HashPassword(%q) error = %v, want nil", "right", err)
	}

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
		{
			// argon2.IDKey panics on t=0 instead of erroring.
			name:    "反復回数0はパニックせずErrInvalidHash",
			input:   "x",
			encoded: "$argon2id$v=19$m=19456,t=0,p=1$c2FsdA$aGFzaA",
			wantOK:  false,
			wantErr: ErrInvalidHash,
		},
		{
			// argon2.IDKey panics on p=0 instead of erroring.
			name:    "並列度0はパニックせずErrInvalidHash",
			input:   "x",
			encoded: "$argon2id$v=19$m=19456,t=2,p=0$c2FsdA$aGFzaA",
			wantOK:  false,
			wantErr: ErrInvalidHash,
		},
		{
			// 4 GiB would allocate 4 GiB and run for minutes.
			name:    "過大なメモリ指定はErrInvalidHash",
			input:   "x",
			encoded: "$argon2id$v=19$m=4000000,t=2,p=1$c2FsdA$aGFzaA",
			wantOK:  false,
			wantErr: ErrInvalidHash,
		},
		{
			name:    "過大な反復回数はErrInvalidHash",
			input:   "x",
			encoded: "$argon2id$v=19$m=19456,t=1000,p=1$c2FsdA$aGFzaA",
			wantOK:  false,
			wantErr: ErrInvalidHash,
		},
		{
			name:    "空のソルトはErrInvalidHash",
			input:   "x",
			encoded: "$argon2id$v=19$m=19456,t=2,p=1$$aGFzaA",
			wantOK:  false,
			wantErr: ErrInvalidHash,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ok, err := VerifyPassword(tt.input, tt.encoded)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("VerifyPassword(%q, %q) error = %v, want %v",
						tt.input, tt.encoded, err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("VerifyPassword(%q, ...) error = %v, want nil", tt.input, err)
			}
			if ok != tt.wantOK {
				t.Errorf("VerifyPassword(%q, ...) = %t, want %t", tt.input, ok, tt.wantOK)
			}
		})
	}
}
