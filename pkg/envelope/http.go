package envelope

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"runtime/debug"
)

// WriteSuccess writes a success envelope as JSON to the HTTP response.
func WriteSuccess(w http.ResponseWriter, statusCode int, data any) {
	writeJSON(w, statusCode, Success(data))
}

// WriteError writes an error envelope as JSON to the HTTP response.
func WriteError(w http.ResponseWriter, statusCode int, code, message string) {
	writeJSON(w, statusCode, Fail(code, message))
}

// WriteErrorWithDetails writes an error envelope with additional details.
func WriteErrorWithDetails(w http.ResponseWriter, statusCode int, code, message string, details any) {
	writeJSON(w, statusCode, FailWithDetails(code, message, details))
}

// WrapHandler returns an http.HandlerFunc that recovers from panics and writes
// a 500 INTERNAL_ERROR envelope response. It logs the panic stack trace.
func WrapHandler(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("[PANIC] %v\n%s", rec, debug.Stack())
				WriteError(w, http.StatusInternalServerError,
					CodeInternalError,
					fmt.Sprintf("internal server error: %v", rec),
				)
			}
		}()
		next(w, r)
	}
}

func writeJSON(w http.ResponseWriter, statusCode int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("[envelope] failed to encode JSON response: %v", err)
	}
}
