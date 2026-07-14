package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/mail"
	"time"

	"github.com/0muji4/Runa/apps/go/internal/auth"
	"github.com/0muji4/Runa/apps/go/internal/repository"
	"github.com/0muji4/Runa/apps/go/internal/service"
)

// minPasswordLength is the minimum accepted password length at signup.
const minPasswordLength = 8

// Auth is the HTTP transport for the authentication endpoints. It translates
// requests/responses and maps service errors to the shared error envelope; all
// logic lives in the service layer.
type Auth struct {
	svc    *service.AuthService
	logger *slog.Logger
}

// NewAuth constructs the auth handler from its service dependency.
func NewAuth(svc *service.AuthService, logger *slog.Logger) *Auth {
	return &Auth{svc: svc, logger: logger}
}

type signupRequest struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type appleRequest struct {
	IDToken     string `json:"id_token"`
	DisplayName string `json:"display_name"`
}

type googleRequest struct {
	IDToken string `json:"id_token"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type logoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type authTokensResponse struct {
	AccessToken  string        `json:"access_token"`
	RefreshToken string        `json:"refresh_token"`
	TokenType    string        `json:"token_type"`
	ExpiresIn    int           `json:"expires_in"`
	User         *userResponse `json:"user,omitempty"`
}

type userResponse struct {
	ID               string  `json:"id"`
	Email            *string `json:"email"`
	DisplayName      string  `json:"display_name"`
	AuthProvider     string  `json:"auth_provider"`
	IsPremium        bool    `json:"is_premium"`
	PremiumExpiresAt *string `json:"premium_expires_at"`
	CreatedAt        string  `json:"created_at"`
}

// Signup handles POST /api/v1/auth/signup (email + password).
func (a *Auth) Signup(w http.ResponseWriter, r *http.Request) {
	var req signupRequest
	if !a.decode(w, r, &req) {
		return
	}
	if details := validateSignup(req); len(details) > 0 {
		writeError(w, http.StatusBadRequest, CodeValidation, "validation failed", details, a.logger)
		return
	}

	res, err := a.svc.SignupEmail(r.Context(), req.Email, req.Password, req.DisplayName)
	if err != nil {
		if errors.Is(err, service.ErrEmailTaken) {
			writeError(w, http.StatusConflict, CodeEmailTaken, "email already registered", nil, a.logger)
			return
		}
		a.internal(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toAuthTokensResponse(res, true), a.logger)
}

// Login handles POST /api/v1/auth/login (email + password).
func (a *Auth) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if !a.decode(w, r, &req) {
		return
	}
	if req.Email == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, CodeValidation, "email and password are required", nil, a.logger)
		return
	}

	res, err := a.svc.LoginEmail(r.Context(), req.Email, req.Password)
	if err != nil {
		a.writeLoginError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toAuthTokensResponse(res, true), a.logger)
}

// Apple handles POST /api/v1/auth/apple (Apple ID token).
func (a *Auth) Apple(w http.ResponseWriter, r *http.Request) {
	var req appleRequest
	if !a.decode(w, r, &req) {
		return
	}
	if req.IDToken == "" {
		writeError(w, http.StatusBadRequest, CodeValidation, "id_token is required", nil, a.logger)
		return
	}

	res, err := a.svc.LoginApple(r.Context(), req.IDToken, req.DisplayName)
	if err != nil {
		a.writeProviderError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toAuthTokensResponse(res, true), a.logger)
}

// Google handles POST /api/v1/auth/google (Google ID token).
func (a *Auth) Google(w http.ResponseWriter, r *http.Request) {
	var req googleRequest
	if !a.decode(w, r, &req) {
		return
	}
	if req.IDToken == "" {
		writeError(w, http.StatusBadRequest, CodeValidation, "id_token is required", nil, a.logger)
		return
	}

	res, err := a.svc.LoginGoogle(r.Context(), req.IDToken)
	if err != nil {
		a.writeProviderError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toAuthTokensResponse(res, true), a.logger)
}

// Refresh handles POST /api/v1/auth/refresh (refresh-token rotation).
func (a *Auth) Refresh(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if !a.decode(w, r, &req) {
		return
	}
	if req.RefreshToken == "" {
		writeError(w, http.StatusBadRequest, CodeValidation, "refresh_token is required", nil, a.logger)
		return
	}

	tokens, err := a.svc.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		if errors.Is(err, service.ErrInvalidRefreshToken) {
			writeError(w, http.StatusUnauthorized, CodeTokenInvalid, "refresh token is invalid or expired", nil, a.logger)
			return
		}
		a.internal(w, r, err)
		return
	}
	// Refresh does not re-send the user object.
	writeJSON(w, http.StatusOK, authTokensResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    tokens.ExpiresIn,
	}, a.logger)
}

// Logout handles POST /api/v1/auth/logout (revoke refresh token). Idempotent.
func (a *Auth) Logout(w http.ResponseWriter, r *http.Request) {
	var req logoutRequest
	if !a.decode(w, r, &req) {
		return
	}
	if req.RefreshToken == "" {
		writeError(w, http.StatusBadRequest, CodeValidation, "refresh_token is required", nil, a.logger)
		return
	}
	if err := a.svc.Logout(r.Context(), req.RefreshToken); err != nil {
		a.internal(w, r, err)
		return
	}
	writeJSON(w, http.StatusNoContent, nil, a.logger)
}

// Me handles GET /api/v1/me (Bearer-protected; returns the caller's record).
func (a *Auth) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		// Should not happen behind RequireAuth, but fail closed.
		a.Unauthorized(w, r, auth.ErrInvalidToken)
		return
	}

	user, err := a.svc.Me(r.Context(), userID)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			a.Unauthorized(w, r, auth.ErrInvalidToken)
			return
		}
		a.internal(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toUserResponse(user), a.logger)
}

// Unauthorized writes the 401 body for failed Bearer verification, choosing the
// code from the underlying error so the client can tell "expired" from "bad".
func (a *Auth) Unauthorized(w http.ResponseWriter, r *http.Request, err error) {
	code := CodeUnauthorized
	message := "authentication required"
	switch {
	case errors.Is(err, auth.ErrTokenExpired):
		code, message = CodeTokenExpired, "access token expired"
	case errors.Is(err, auth.ErrInvalidToken):
		code, message = CodeTokenInvalid, "access token is invalid"
	}
	writeError(w, http.StatusUnauthorized, code, message, nil, a.logger)
}

// RateLimited writes the 429 body when an auth endpoint is rate limited.
func (a *Auth) RateLimited(w http.ResponseWriter, _ *http.Request, _ error) {
	writeError(w, http.StatusTooManyRequests, CodeRateLimited, "too many requests, please try again later", nil, a.logger)
}

// decode reads a JSON body into dst, rejecting unknown fields. It writes a 400
// on failure and reports whether decoding succeeded.
func (a *Auth) decode(w http.ResponseWriter, r *http.Request, dst any) bool {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		writeError(w, http.StatusBadRequest, CodeValidation, "invalid JSON body", nil, a.logger)
		return false
	}
	return true
}

func (a *Auth) writeLoginError(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, service.ErrInvalidCredentials) {
		writeError(w, http.StatusUnauthorized, CodeInvalidCredentials, "email or password is incorrect", nil, a.logger)
		return
	}
	a.internal(w, r, err)
}

func (a *Auth) writeProviderError(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, auth.ErrProviderVerification) {
		writeError(w, http.StatusUnauthorized, CodeProviderVerification, "could not verify the provider token", nil, a.logger)
		return
	}
	a.internal(w, r, err)
}

func (a *Auth) internal(w http.ResponseWriter, r *http.Request, err error) {
	a.logger.ErrorContext(r.Context(), "auth handler internal error", slog.Any("error", err))
	writeError(w, http.StatusInternalServerError, CodeInternal, "an unexpected error occurred", nil, a.logger)
}

func validateSignup(req signupRequest) []FieldError {
	var details []FieldError
	if _, err := mail.ParseAddress(req.Email); err != nil {
		details = append(details, FieldError{Field: "email", Message: "must be a valid email address"})
	}
	if len(req.Password) < minPasswordLength {
		details = append(details, FieldError{Field: "password", Message: "must be at least 8 characters"})
	}
	return details
}

func toAuthTokensResponse(res service.AuthResult, includeUser bool) authTokensResponse {
	out := authTokensResponse{
		AccessToken:  res.Tokens.AccessToken,
		RefreshToken: res.Tokens.RefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    res.Tokens.ExpiresIn,
	}
	if includeUser {
		u := toUserResponse(res.User)
		out.User = &u
	}
	return out
}

func toUserResponse(u repository.User) userResponse {
	out := userResponse{
		ID:           u.ID,
		Email:        u.Email,
		DisplayName:  u.DisplayName,
		AuthProvider: u.AuthProvider,
		IsPremium:    u.IsPremium,
		CreatedAt:    u.CreatedAt.UTC().Format(time.RFC3339),
	}
	if u.PremiumExpiresAt != nil {
		s := u.PremiumExpiresAt.UTC().Format(time.RFC3339)
		out.PremiumExpiresAt = &s
	}
	return out
}
