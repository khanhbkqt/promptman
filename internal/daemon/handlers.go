package daemon

import (
	"net/http"

	"github.com/khanhnguyen/promptman/pkg/envelope"
)

// StatusHandler returns an http.HandlerFunc that responds with the
// current daemon info (pid, port, uptime, projectDir, startedAt)
// wrapped in the standard envelope format.
func StatusHandler(mgr *Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		info, err := mgr.Status()
		if err != nil {
			de, ok := err.(*DomainError)
			if ok {
				statusCode := envelope.HTTPStatusForCode(de.Code)
				envelope.WriteError(w, statusCode, de.Code, de.Message)
				return
			}
			envelope.WriteError(w, http.StatusInternalServerError,
				envelope.CodeInternalError, err.Error())
			return
		}

		envelope.WriteSuccess(w, http.StatusOK, info)
	}
}

// shutdownResponse is the JSON payload returned by the shutdown endpoint.
type shutdownResponse struct {
	Message string `json:"message"`
}

// ShutdownHandler returns an http.HandlerFunc that triggers a graceful
// shutdown of the HTTP server. The response is sent before the server
// begins draining, so the client receives confirmation.
func ShutdownHandler(srv *Server) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		envelope.WriteSuccess(w, http.StatusOK, shutdownResponse{
			Message: "shutting down",
		})

		// Shutdown in a goroutine so the response can be flushed first.
		go func() {
			_ = srv.Shutdown()
		}()
	}
}
