package daemon

import (
	"net/http"
	"strings"

	"github.com/khanhnguyen/promptman/pkg/envelope"
)

// AuthMiddleware returns HTTP middleware that validates the Authorization
// header contains a valid Bearer token. Requests with a missing or invalid
// token receive a 401 response with the UNAUTHORIZED envelope error code.
func AuthMiddleware(token string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			if auth == "" {
				envelope.WriteError(w, http.StatusUnauthorized,
					envelope.CodeUnauthorized, "missing Authorization header")
				return
			}

			const prefix = "Bearer "
			if !strings.HasPrefix(auth, prefix) {
				envelope.WriteError(w, http.StatusUnauthorized,
					envelope.CodeUnauthorized, "invalid Authorization format, expected: Bearer <token>")
				return
			}

			provided := auth[len(prefix):]
			if provided != token {
				envelope.WriteError(w, http.StatusUnauthorized,
					envelope.CodeUnauthorized, "invalid token")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// IdleResetMiddleware returns HTTP middleware that resets the daemon's
// idle shutdown timer on every incoming request. This keeps the daemon
// alive as long as it is actively serving requests.
func IdleResetMiddleware(mgr *Manager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mgr.ResetIdleTimer()
			next.ServeHTTP(w, r)
		})
	}
}
