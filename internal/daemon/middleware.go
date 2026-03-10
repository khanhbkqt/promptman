package daemon

import (
	"net/http"
	"strings"

	"github.com/khanhnguyen/promptman/pkg/envelope"
)

// AuthMiddleware returns HTTP middleware that validates the auth token.
// It checks the Authorization header first (Bearer token), and falls
// back to the "token" query parameter for WebSocket upgrade requests
// that cannot easily set custom headers.
func AuthMiddleware(token string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var provided string

			// Try Authorization header first (REST API).
			auth := r.Header.Get("Authorization")
			if auth != "" {
				const prefix = "Bearer "
				if !strings.HasPrefix(auth, prefix) {
					envelope.WriteError(w, http.StatusUnauthorized,
						envelope.CodeUnauthorized, "invalid Authorization format, expected: Bearer <token>")
					return
				}
				provided = auth[len(prefix):]
			} else {
				// Fall back to query parameter (WebSocket).
				provided = r.URL.Query().Get("token")
			}

			if provided == "" {
				envelope.WriteError(w, http.StatusUnauthorized,
					envelope.CodeUnauthorized, "missing authentication: provide Authorization header or token query parameter")
				return
			}

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
