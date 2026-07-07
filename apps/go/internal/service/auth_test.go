package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/0muji4/Runa/apps/go/internal/auth"
	"github.com/0muji4/Runa/apps/go/internal/repository"
	"github.com/0muji4/Runa/apps/go/internal/repository/memauth"
	"github.com/0muji4/Runa/apps/go/internal/service"
)

// stubVerifier is a canned IDTokenVerifier for the social sign-in paths.
type stubVerifier struct {
	id  auth.OIDCIdentity
	err error
}

func (s stubVerifier) Verify(context.Context, string) (auth.OIDCIdentity, error) {
	return s.id, s.err
}

func newService(t *testing.T, store repository.AuthStore, apple, google auth.IDTokenVerifier) *service.AuthService {
	t.Helper()
	return service.NewAuthService(service.AuthConfig{
		Store:          store,
		Issuer:         auth.NewTokenIssuer("test-secret", time.Minute),
		Apple:          apple,
		Google:         google,
		PasswordParams: auth.DefaultArgon2Params(),
		RefreshTTL:     time.Hour,
	})
}

func TestSignupAndLoginEmail(t *testing.T) {
	svc := newService(t, memauth.New(), nil, nil)
	ctx := context.Background()

	res, err := svc.SignupEmail(ctx, "User@Example.com", "password123", "Runa")
	if err != nil {
		t.Fatalf("SignupEmail: %v", err)
	}
	if res.Tokens.AccessToken == "" || res.Tokens.RefreshToken == "" {
		t.Fatal("expected non-empty tokens after signup")
	}
	if res.User.Email == nil || *res.User.Email != "user@example.com" {
		t.Fatalf("email = %v, want normalized user@example.com", res.User.Email)
	}
	if res.User.AuthProvider != "email" {
		t.Errorf("auth_provider = %q, want email", res.User.AuthProvider)
	}

	if _, err := svc.LoginEmail(ctx, "user@example.com", "password123"); err != nil {
		t.Fatalf("LoginEmail with correct password: %v", err)
	}
	if _, err := svc.LoginEmail(ctx, "user@example.com", "wrong"); !errors.Is(err, service.ErrInvalidCredentials) {
		t.Fatalf("LoginEmail wrong password error = %v, want ErrInvalidCredentials", err)
	}
	if _, err := svc.LoginEmail(ctx, "missing@example.com", "whatever"); !errors.Is(err, service.ErrInvalidCredentials) {
		t.Fatalf("LoginEmail unknown user error = %v, want ErrInvalidCredentials", err)
	}
}

func TestSignupDuplicateEmail(t *testing.T) {
	svc := newService(t, memauth.New(), nil, nil)
	ctx := context.Background()
	if _, err := svc.SignupEmail(ctx, "dup@example.com", "password123", ""); err != nil {
		t.Fatalf("first signup: %v", err)
	}
	if _, err := svc.SignupEmail(ctx, "dup@example.com", "password123", ""); !errors.Is(err, service.ErrEmailTaken) {
		t.Fatalf("duplicate signup error = %v, want ErrEmailTaken", err)
	}
}

func TestSignupDerivesDisplayNameFromEmail(t *testing.T) {
	svc := newService(t, memauth.New(), nil, nil)
	res, err := svc.SignupEmail(context.Background(), "moon@example.com", "password123", "")
	if err != nil {
		t.Fatalf("SignupEmail: %v", err)
	}
	if res.User.DisplayName != "moon" {
		t.Errorf("display_name = %q, want derived %q", res.User.DisplayName, "moon")
	}
}

func TestRefreshRotation(t *testing.T) {
	svc := newService(t, memauth.New(), nil, nil)
	ctx := context.Background()

	res, err := svc.SignupEmail(ctx, "rot@example.com", "password123", "")
	if err != nil {
		t.Fatalf("SignupEmail: %v", err)
	}
	first := res.Tokens.RefreshToken

	rotated, err := svc.Refresh(ctx, first)
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if rotated.RefreshToken == first {
		t.Fatal("refresh token was not rotated")
	}

	// The old token is single-use and must now be rejected.
	if _, err := svc.Refresh(ctx, first); !errors.Is(err, service.ErrInvalidRefreshToken) {
		t.Fatalf("reusing rotated token error = %v, want ErrInvalidRefreshToken", err)
	}
	// The new token works.
	if _, err := svc.Refresh(ctx, rotated.RefreshToken); err != nil {
		t.Fatalf("Refresh with rotated token: %v", err)
	}
}

func TestLogoutRevokesRefreshToken(t *testing.T) {
	svc := newService(t, memauth.New(), nil, nil)
	ctx := context.Background()

	res, _ := svc.SignupEmail(ctx, "out@example.com", "password123", "")
	if err := svc.Logout(ctx, res.Tokens.RefreshToken); err != nil {
		t.Fatalf("Logout: %v", err)
	}
	if _, err := svc.Refresh(ctx, res.Tokens.RefreshToken); !errors.Is(err, service.ErrInvalidRefreshToken) {
		t.Fatalf("Refresh after logout error = %v, want ErrInvalidRefreshToken", err)
	}
	// Logout is idempotent.
	if err := svc.Logout(ctx, res.Tokens.RefreshToken); err != nil {
		t.Fatalf("second Logout: %v", err)
	}
}

func TestMe(t *testing.T) {
	svc := newService(t, memauth.New(), nil, nil)
	ctx := context.Background()

	res, _ := svc.SignupEmail(ctx, "me@example.com", "password123", "")
	got, err := svc.Me(ctx, res.User.ID)
	if err != nil {
		t.Fatalf("Me: %v", err)
	}
	if got.ID != res.User.ID {
		t.Errorf("id = %q, want %q", got.ID, res.User.ID)
	}
	if _, err := svc.Me(ctx, "does-not-exist"); !errors.Is(err, service.ErrUserNotFound) {
		t.Fatalf("Me unknown error = %v, want ErrUserNotFound", err)
	}
}

func TestLoginAppleCreatesThenReuses(t *testing.T) {
	apple := stubVerifier{id: auth.OIDCIdentity{Subject: "apple-sub-9", Email: "a@example.com", Name: "Apple User"}}
	svc := newService(t, memauth.New(), apple, nil)
	ctx := context.Background()

	first, err := svc.LoginApple(ctx, "id-token", "")
	if err != nil {
		t.Fatalf("first LoginApple: %v", err)
	}
	if first.User.AuthProvider != "apple" || first.User.AppleSub == nil {
		t.Fatalf("user = %+v, want apple provider with apple_sub", first.User)
	}

	second, err := svc.LoginApple(ctx, "id-token", "")
	if err != nil {
		t.Fatalf("second LoginApple: %v", err)
	}
	if second.User.ID != first.User.ID {
		t.Errorf("second sign-in created a new user (%q != %q)", second.User.ID, first.User.ID)
	}
}

func TestLoginGoogleVerificationError(t *testing.T) {
	google := stubVerifier{err: auth.ErrProviderVerification}
	svc := newService(t, memauth.New(), nil, google)
	if _, err := svc.LoginGoogle(context.Background(), "bad-token"); !errors.Is(err, auth.ErrProviderVerification) {
		t.Fatalf("LoginGoogle error = %v, want ErrProviderVerification", err)
	}
}
