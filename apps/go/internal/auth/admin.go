package auth

import (
	"crypto/subtle"
	"errors"
	"net/http"
)

// AdminTokenHeader is the header carrying the shared admin token for the curated
// seed endpoints (POST /admin/quotes, /admin/songs).
const AdminTokenHeader = "X-Admin-Token"

// ErrAdminForbidden means the admin token was missing, wrong, or the feature is
// disabled (no server token configured).
var ErrAdminForbidden = errors.New("auth: admin access forbidden")

// RequireAdmin returns middleware that gates the admin seed endpoints behind a
// static shared token compared in constant time.
//
// When serverToken is empty the admin surface is DISABLED: every request is
// rejected. This fails closed, so a deployment that never sets ADMIN_API_TOKEN
// cannot be seeded by an anonymous caller.
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
