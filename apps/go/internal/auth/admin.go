package auth

import (
	"crypto/subtle"
	"errors"
	"net/http"
)

// AdminTokenHeader is the header carrying the shared admin token.
const AdminTokenHeader = "X-Admin-Token"

// ErrAdminForbidden means the admin token was missing, wrong, or no server token is configured.
var ErrAdminForbidden = errors.New("auth: admin access forbidden")

// RequireAdmin returns middleware that gates admin endpoints behind a shared token
// compared in constant time. An empty serverToken must reject every request (fail closed).
func RequireAdmin(serverToken string, onForbidden ErrorResponder) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			presented := r.Header.Get(AdminTokenHeader)
			if serverToken == "" || subtle.ConstantTimeCompare([]byte(presented), []byte(serverToken)) != 1 {
				onForbidden(w, r, ErrAdminForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
