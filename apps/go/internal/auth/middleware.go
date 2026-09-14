package auth

import (
	"context"
	"net/http"
	"strings"
)

type contextKey string

const userIDKey contextKey = "auth.userID"

// AccessVerifier verifies an access token and returns its subject (user id).
type AccessVerifier interface {
	Verify(tokenString string) (string, error)
}

// ErrorResponder writes an error HTTP response; the handler layer supplies the JSON envelope.
type ErrorResponder func(w http.ResponseWriter, r *http.Request, err error)

// RequireAuth returns middleware that verifies the Bearer access token and
// stores the user id in the request context; failures go to onUnauthorized.
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

// UserIDFromContext returns the user id set by RequireAuth, and whether it was present.
func UserIDFromContext(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(userIDKey).(string)
	return id, ok && id != ""
}

func bearerToken(r *http.Request) (string, bool) {
	const prefix = "Bearer "
	h := r.Header.Get("Authorization")
	if len(h) <= len(prefix) || !strings.EqualFold(h[:len(prefix)], prefix) {
		return "", false
	}
	token := strings.TrimSpace(h[len(prefix):])
	return token, token != ""
}
