package auth

import (
	"context"
	"net/http"
	"strings"
)

// contextKey is unexported so only this package can set/read the value.
type contextKey string

const userIDKey contextKey = "auth.userID"

// AccessVerifier verifies an access token and returns its subject (user id).
// *TokenIssuer implements it.
type AccessVerifier interface {
	Verify(tokenString string) (string, error)
}

// ErrorResponder writes an error HTTP response. The handler layer supplies one
// that emits the shared JSON error envelope, keeping response formatting out of
// this package.
type ErrorResponder func(w http.ResponseWriter, r *http.Request, err error)

// RequireAuth returns middleware that verifies the Bearer access token and, on
// success, stores the user id in the request context for downstream handlers.
// On failure it delegates to onUnauthorized (which chooses status/body).
func RequireAuth(verifier AccessVerifier, onUnauthorized ErrorResponder) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := bearerToken(r)
			if !ok {
				onUnauthorized(w, r, ErrInvalidToken)
				return
			}
			userID, err := verifier.Verify(token)
			if err != nil {
				onUnauthorized(w, r, err)
				return
			}
			ctx := context.WithValue(r.Context(), userIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UserIDFromContext returns the authenticated user id previously set by
// RequireAuth, and whether it was present.
func UserIDFromContext(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(userIDKey).(string)
	return id, ok && id != ""
}

// bearerToken extracts the token from an "Authorization: Bearer <token>" header.
func bearerToken(r *http.Request) (string, bool) {
	const prefix = "Bearer "
	h := r.Header.Get("Authorization")
	if len(h) <= len(prefix) || !strings.EqualFold(h[:len(prefix)], prefix) {
		return "", false
	}
	token := strings.TrimSpace(h[len(prefix):])
	return token, token != ""
}
