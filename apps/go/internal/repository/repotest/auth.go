package repotest

import (
	"errors"
	"testing"
	"time"

	"github.com/0muji4/Runa/apps/go/internal/repository"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

// RunAuthStoreSuite exercises the AuthStore contract.
func RunAuthStoreSuite(t *testing.T, newFixture NewFixture) {
	t.Run("CreateUserRoundTrip", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)
		ctx := t.Context()

		email := "round@example.com"
		want := repository.CreateUserParams{
			Email:        &email,
			AuthProvider: "email",
			DisplayName:  "Round",
			PasswordHash: ptr("$argon2id$hash"),
		}
		created, err := f.Auth.CreateUser(ctx, want)
		if err != nil {
			t.Fatalf("CreateUser(%+v) error = %v, want nil", want, err)
		}
		if created.ID == "" {
			t.Error("CreateUser() id is empty, want a generated id")
		}
		if created.CreatedAt.IsZero() {
			t.Error("CreateUser() created_at is zero, want the insert time")
		}

		byID, err := f.Auth.GetUserByID(ctx, created.ID)
		if err != nil {
			t.Fatalf("GetUserByID(%q) error = %v, want nil", created.ID, err)
		}
		byEmail, err := f.Auth.GetUserByEmail(ctx, email)
		if err != nil {
			t.Fatalf("GetUserByEmail(%q) error = %v, want nil", email, err)
		}
		// The DB stores created_at at coarser precision than Go's clock.
		opt := cmpopts.EquateApproxTime(time.Second)
		if diff := cmp.Diff(created, byID, opt); diff != "" {
			t.Errorf("GetUserByID mismatch (-created +got):\n%s", diff)
		}
		if diff := cmp.Diff(created, byEmail, opt); diff != "" {
			t.Errorf("GetUserByEmail mismatch (-created +got):\n%s", diff)
		}
	})

	t.Run("DuplicateEmailIsRejected", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)
		ctx := t.Context()

		email := "dup@example.com"
		p := repository.CreateUserParams{Email: &email, AuthProvider: "email", DisplayName: "First"}
		if _, err := f.Auth.CreateUser(ctx, p); err != nil {
			t.Fatalf("first CreateUser() error = %v, want nil", err)
		}
		p.DisplayName = "Second"
		if _, err := f.Auth.CreateUser(ctx, p); !errors.Is(err, repository.ErrEmailTaken) {
			t.Errorf("second CreateUser(%q) error = %v, want %v", email, err, repository.ErrEmailTaken)
		}
	})

	t.Run("SocialUsersWithoutEmailDoNotCollide", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)
		ctx := t.Context()

		// The email uniqueness index is partial (WHERE email IS NOT NULL).
		first, err := f.Auth.CreateUser(ctx, repository.CreateUserParams{
			AuthProvider: "apple", AppleSub: ptr("apple-sub-1"), DisplayName: "A",
		})
		if err != nil {
			t.Fatalf("CreateUser(apple #1) error = %v, want nil", err)
		}
		second, err := f.Auth.CreateUser(ctx, repository.CreateUserParams{
			AuthProvider: "apple", AppleSub: ptr("apple-sub-2"), DisplayName: "B",
		})
		if err != nil {
			t.Fatalf("CreateUser(apple #2) error = %v, want nil", err)
		}
		if first.ID == second.ID {
			t.Errorf("two email-less accounts share id %q, want distinct rows", first.ID)
		}
	})

	t.Run("GetUserByProviderSub", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)
		ctx := t.Context()

		apple, err := f.Auth.CreateUser(ctx, repository.CreateUserParams{
			AuthProvider: "apple", AppleSub: ptr("apple-sub"), DisplayName: "Apple",
		})
		if err != nil {
			t.Fatalf("CreateUser(apple) error = %v, want nil", err)
		}
		google, err := f.Auth.CreateUser(ctx, repository.CreateUserParams{
			AuthProvider: "google", GoogleSub: ptr("google-sub"), DisplayName: "Google",
		})
		if err != nil {
			t.Fatalf("CreateUser(google) error = %v, want nil", err)
		}

		tests := []struct {
			name          string
			provider, sub string
			wantID        string
			wantErr       error
		}{
			{
				name:     "appleのsubjectでapple利用者が引ける",
				provider: "apple",
				sub:      "apple-sub",
				wantID:   apple.ID,
				wantErr:  nil,
			},
			{
				name:     "googleのsubjectでgoogle利用者が引ける",
				provider: "google",
				sub:      "google-sub",
				wantID:   google.ID,
				wantErr:  nil,
			},
			{
				// A subject is only valid for its own provider column.
				name:     "provider違いのsubjectは引けない",
				provider: "google",
				sub:      "apple-sub",
				wantID:   "",
				wantErr:  repository.ErrNotFound,
			},
			{
				name:     "未知のsubjectは引けない",
				provider: "apple",
				sub:      "unknown",
				wantID:   "",
				wantErr:  repository.ErrNotFound,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got, err := f.Auth.GetUserByProviderSub(ctx, tt.provider, tt.sub)
				if tt.wantErr != nil {
					if !errors.Is(err, tt.wantErr) {
						t.Fatalf("GetUserByProviderSub(%q, %q) error = %v, want %v",
							tt.provider, tt.sub, err, tt.wantErr)
					}
					return
				}
				if err != nil {
					t.Fatalf("GetUserByProviderSub(%q, %q) error = %v, want nil",
						tt.provider, tt.sub, err)
				}
				if got.ID != tt.wantID {
					t.Errorf("GetUserByProviderSub(%q, %q) id = %q, want %q",
						tt.provider, tt.sub, got.ID, tt.wantID)
				}
			})
		}
	})

	t.Run("LookupsMissTheUnknown", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)
		ctx := t.Context()

		const unknownID = "99999999-9999-4999-8999-999999999999"
		tests := []struct {
			name   string
			lookup func() error
		}{
			{
				name: "GetUserByID",
				lookup: func() error {
					_, err := f.Auth.GetUserByID(ctx, unknownID)
					return err
				},
			},
			{
				name: "GetUserByEmail",
				lookup: func() error {
					_, err := f.Auth.GetUserByEmail(ctx, "nobody@example.com")
					return err
				},
			},
			{
				name: "GetRefreshTokenByHash",
				lookup: func() error {
					_, err := f.Auth.GetRefreshTokenByHash(ctx, "never-issued")
					return err
				},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if err := tt.lookup(); !errors.Is(err, repository.ErrNotFound) {
					t.Errorf("%s(unknown) error = %v, want %v", tt.name, err, repository.ErrNotFound)
				}
			})
		}
	})

	t.Run("UpdateDisplayName", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)
		ctx := t.Context()

		id := f.NewUser(t)
		updated, err := f.Auth.UpdateDisplayName(ctx, id, "新しい名前")
		if err != nil {
			t.Fatalf("UpdateDisplayName(%q) error = %v, want nil", id, err)
		}
		if updated.DisplayName != "新しい名前" {
			t.Errorf("UpdateDisplayName() display_name = %q, want %q", updated.DisplayName, "新しい名前")
		}
		// It must persist, not just be echoed back.
		got, err := f.Auth.GetUserByID(ctx, id)
		if err != nil {
			t.Fatalf("GetUserByID(%q) error = %v, want nil", id, err)
		}
		if got.DisplayName != "新しい名前" {
			t.Errorf("re-read display_name = %q, want %q", got.DisplayName, "新しい名前")
		}

		const unknownID = "99999999-9999-4999-8999-999999999999"
		if _, err := f.Auth.UpdateDisplayName(ctx, unknownID, "x"); !errors.Is(err, repository.ErrNotFound) {
			t.Errorf("UpdateDisplayName(unknown) error = %v, want %v", err, repository.ErrNotFound)
		}
	})

	t.Run("DeleteUser", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)
		ctx := t.Context()

		id := f.NewUser(t)
		if err := f.Auth.DeleteUser(ctx, id); err != nil {
			t.Fatalf("DeleteUser(%q) error = %v, want nil", id, err)
		}
		if _, err := f.Auth.GetUserByID(ctx, id); !errors.Is(err, repository.ErrNotFound) {
			t.Errorf("GetUserByID after delete error = %v, want %v", err, repository.ErrNotFound)
		}
		if err := f.Auth.DeleteUser(ctx, id); !errors.Is(err, repository.ErrNotFound) {
			t.Errorf("second DeleteUser(%q) error = %v, want %v", id, err, repository.ErrNotFound)
		}
	})

	t.Run("RefreshTokenLifecycle", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)
		ctx := t.Context()

		id := f.NewUser(t)
		expires := time.Now().Add(time.Hour).UTC().Truncate(time.Second)
		const hash = "sha256-hash-of-the-token"

		if err := f.Auth.InsertRefreshToken(ctx, repository.InsertRefreshTokenParams{
			UserID: id, TokenHash: hash, ExpiresAt: expires,
		}); err != nil {
			t.Fatalf("InsertRefreshToken() error = %v, want nil", err)
		}

		got, err := f.Auth.GetRefreshTokenByHash(ctx, hash)
		if err != nil {
			t.Fatalf("GetRefreshTokenByHash() error = %v, want nil", err)
		}
		if got.UserID != id {
			t.Errorf("refresh token user_id = %q, want %q", got.UserID, id)
		}
		if got.Revoked {
			t.Error("a freshly inserted refresh token is revoked, want not revoked")
		}
		if !got.ExpiresAt.Equal(expires) {
			t.Errorf("refresh token expires_at = %s, want %s", got.ExpiresAt.UTC(), expires)
		}

		if err := f.Auth.RevokeRefreshToken(ctx, hash); err != nil {
			t.Fatalf("RevokeRefreshToken() error = %v, want nil", err)
		}
		revoked, err := f.Auth.GetRefreshTokenByHash(ctx, hash)
		if err != nil {
			t.Fatalf("GetRefreshTokenByHash() after revoke error = %v, want nil", err)
		}
		if !revoked.Revoked {
			t.Error("refresh token revoked = false after RevokeRefreshToken, want true")
		}

		// Logout is idempotent, so revoking an unknown token is a no-op.
		if err := f.Auth.RevokeRefreshToken(ctx, "never-issued"); err != nil {
			t.Errorf("RevokeRefreshToken(unknown) error = %v, want nil", err)
		}
		if _, err := f.Auth.GetRefreshTokenByHash(ctx, "never-issued"); !errors.Is(err, repository.ErrNotFound) {
			t.Errorf("GetRefreshTokenByHash(unknown) error = %v, want %v", err, repository.ErrNotFound)
		}
	})
}
