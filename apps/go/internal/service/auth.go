package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/0muji4/Runa/apps/go/internal/auth"
	"github.com/0muji4/Runa/apps/go/internal/repository"
)

// Service-level auth errors. Handlers map these to HTTP status + error codes.
var (
	// ErrEmailTaken means signup hit an already-registered email.
	ErrEmailTaken = errors.New("service: email already registered")
	// ErrInvalidCredentials means email/password login failed. It is
	// deliberately generic so it does not reveal whether the email exists.
	ErrInvalidCredentials = errors.New("service: invalid credentials")
	// ErrInvalidRefreshToken means the presented refresh token is unknown,
	// revoked or expired.
	ErrInvalidRefreshToken = errors.New("service: invalid refresh token")
	// ErrUserNotFound means an authenticated request referenced a user that no
	// longer exists.
	ErrUserNotFound = errors.New("service: user not found")
)

// Tokens is the token bundle returned by every successful auth operation.
type Tokens struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int // access-token lifetime in seconds
}

// AuthResult pairs freshly issued tokens with the authenticated user.
type AuthResult struct {
	Tokens Tokens
	User   repository.User
}

// accessIssuer is the access-token seam (satisfied by *auth.TokenIssuer).
type accessIssuer interface {
	Issue(userID string) (token string, expiresIn int, err error)
}

// AuthConfig wires the AuthService dependencies.
type AuthConfig struct {
	Store          repository.AuthStore
	Issuer         accessIssuer
	Apple          auth.IDTokenVerifier
	Google         auth.IDTokenVerifier
	PasswordParams auth.Argon2Params
	RefreshTTL     time.Duration
	// Now is overridable in tests; defaults to time.Now.
	Now func() time.Time
}

// AuthService implements the authentication use cases: email signup/login,
// Apple/Google sign-in, refresh-token rotation, logout and self lookup.
type AuthService struct {
	cfg AuthConfig
	now func() time.Time
}

// NewAuthService constructs the service, defaulting Now to time.Now.
func NewAuthService(cfg AuthConfig) *AuthService {
	now := cfg.Now
	if now == nil {
		now = time.Now
	}
	return &AuthService{cfg: cfg, now: now}
}

// SignupEmail creates an email/password account and returns tokens + user.
func (s *AuthService) SignupEmail(ctx context.Context, email, password, displayName string) (AuthResult, error) {
	hash, err := auth.HashPassword(password, s.cfg.PasswordParams)
	if err != nil {
		return AuthResult{}, err
	}

	email = normalizeEmail(email)
	name := firstNonEmpty(strings.TrimSpace(displayName), localPart(email))

	user, err := s.cfg.Store.CreateUser(ctx, repository.CreateUserParams{
		Email:        &email,
		AuthProvider: "email",
		DisplayName:  name,
		PasswordHash: &hash,
	})
	if err != nil {
		if errors.Is(err, repository.ErrEmailTaken) {
			return AuthResult{}, ErrEmailTaken
		}
		return AuthResult{}, err
	}
	return s.completeLogin(ctx, user)
}

// LoginEmail authenticates an email/password account.
func (s *AuthService) LoginEmail(ctx context.Context, email, password string) (AuthResult, error) {
	user, err := s.cfg.Store.GetUserByEmail(ctx, normalizeEmail(email))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return AuthResult{}, ErrInvalidCredentials
		}
		return AuthResult{}, err
	}
	if user.PasswordHash == nil {
		// Social-only account: no password to check against.
		return AuthResult{}, ErrInvalidCredentials
	}
	ok, err := auth.VerifyPassword(password, *user.PasswordHash)
	if err != nil {
		return AuthResult{}, err
	}
	if !ok {
		return AuthResult{}, ErrInvalidCredentials
	}
	return s.completeLogin(ctx, user)
}

// LoginApple verifies an Apple ID token and signs the user in (creating the
// account on first sign-in).
func (s *AuthService) LoginApple(ctx context.Context, idToken, displayName string) (AuthResult, error) {
	identity, err := s.cfg.Apple.Verify(ctx, idToken)
	if err != nil {
		return AuthResult{}, err
	}
	return s.loginWithProvider(ctx, "apple", identity, displayName)
}

// LoginGoogle verifies a Google ID token and signs the user in.
func (s *AuthService) LoginGoogle(ctx context.Context, idToken string) (AuthResult, error) {
	identity, err := s.cfg.Google.Verify(ctx, idToken)
	if err != nil {
		return AuthResult{}, err
	}
	return s.loginWithProvider(ctx, "google", identity, "")
}

// loginWithProvider gets or creates a user keyed by (provider, subject).
//
// Scope note: it links by provider subject only. Cross-provider linking by
// shared email is deferred to a later slice because it is only safe for
// verified emails and touches account-takeover concerns.
func (s *AuthService) loginWithProvider(ctx context.Context, provider string, id auth.OIDCIdentity, displayName string) (AuthResult, error) {
	user, err := s.cfg.Store.GetUserByProviderSub(ctx, provider, id.Subject)
	switch {
	case err == nil:
		return s.completeLogin(ctx, user)
	case !errors.Is(err, repository.ErrNotFound):
		return AuthResult{}, err
	}

	params := repository.CreateUserParams{
		AuthProvider: provider,
		DisplayName:  firstNonEmpty(strings.TrimSpace(displayName), id.Name, localPart(id.Email), "Runa"),
	}
	if id.Email != "" {
		email := normalizeEmail(id.Email)
		params.Email = &email
	}
	sub := id.Subject
	if provider == "apple" {
		params.AppleSub = &sub
	} else {
		params.GoogleSub = &sub
	}

	user, err = s.cfg.Store.CreateUser(ctx, params)
	if err != nil {
		return AuthResult{}, err
	}
	return s.completeLogin(ctx, user)
}

// Refresh validates a refresh token, rotates it (revoke old + issue new) and
// returns a fresh token bundle.
func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (Tokens, error) {
	hash := auth.HashRefreshToken(refreshToken)
	stored, err := s.cfg.Store.GetRefreshTokenByHash(ctx, hash)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return Tokens{}, ErrInvalidRefreshToken
		}
		return Tokens{}, err
	}
	if stored.Revoked || !stored.ExpiresAt.After(s.now()) {
		return Tokens{}, ErrInvalidRefreshToken
	}

	// Rotate: the presented token is single-use.
	if err := s.cfg.Store.RevokeRefreshToken(ctx, hash); err != nil {
		return Tokens{}, err
	}
	return s.issueTokens(ctx, stored.UserID)
}

// Logout revokes the presented refresh token. It is idempotent: revoking an
// unknown token is not an error.
func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	return s.cfg.Store.RevokeRefreshToken(ctx, auth.HashRefreshToken(refreshToken))
}

// Me returns the authenticated user's record.
func (s *AuthService) Me(ctx context.Context, userID string) (repository.User, error) {
	user, err := s.cfg.Store.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return repository.User{}, ErrUserNotFound
		}
		return repository.User{}, err
	}
	return user, nil
}

// completeLogin issues tokens for an authenticated user and returns the pair.
func (s *AuthService) completeLogin(ctx context.Context, user repository.User) (AuthResult, error) {
	tokens, err := s.issueTokens(ctx, user.ID)
	if err != nil {
		return AuthResult{}, err
	}
	return AuthResult{Tokens: tokens, User: user}, nil
}

// issueTokens mints an access token and a rotated refresh token, persisting the
// refresh token's hash.
func (s *AuthService) issueTokens(ctx context.Context, userID string) (Tokens, error) {
	access, expiresIn, err := s.cfg.Issuer.Issue(userID)
	if err != nil {
		return Tokens{}, err
	}
	refresh, err := auth.GenerateRefreshToken()
	if err != nil {
		return Tokens{}, err
	}
	if err := s.cfg.Store.InsertRefreshToken(ctx, repository.InsertRefreshTokenParams{
		UserID:    userID,
		TokenHash: auth.HashRefreshToken(refresh),
		ExpiresAt: s.now().Add(s.cfg.RefreshTTL),
	}); err != nil {
		return Tokens{}, err
	}
	return Tokens{AccessToken: access, RefreshToken: refresh, ExpiresIn: expiresIn}, nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func localPart(email string) string {
	if at := strings.IndexByte(email, '@'); at > 0 {
		return email[:at]
	}
	return email
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
