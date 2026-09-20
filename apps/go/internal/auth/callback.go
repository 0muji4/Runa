package auth

import (
	"crypto/subtle"
	"errors"
	"net/http"
)

// CallbackTokenHeader carries the shared secret the scheduler sends on every callback.
const CallbackTokenHeader = "X-Push-Callback-Token"

// ErrCallbackForbidden means the callback token was missing, wrong, or none is configured.
var ErrCallbackForbidden = errors.New("auth: callback access forbidden")

// RequireCallbackToken gates the scheduler callback behind a shared token
// compared in constant time; an empty serverToken rejects every request.
func RequireCallbackToken(serverToken string, onForbidden ErrorResponder) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			presented := r.Header.Get(CallbackTokenHeader)
			if serverToken == "" || subtle.ConstantTimeCompare([]byte(presented), []byte(serverToken)) != 1 {
				onForbidden(w, r, ErrCallbackForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
