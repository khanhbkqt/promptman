package daemon

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/khanhnguyen/promptman/internal/request"
	"github.com/khanhnguyen/promptman/pkg/envelope"
)

// RequestRegistrar registers request execution endpoints on the daemon router.
type RequestRegistrar struct {
	engine *request.Engine
}

// NewRequestRegistrar creates a RequestRegistrar for the given Engine.
func NewRequestRegistrar(engine *request.Engine) *RequestRegistrar {
	return &RequestRegistrar{engine: engine}
}

// RegisterRoutes mounts the request execution endpoints under the given prefix.
func (rr *RequestRegistrar) RegisterRoutes(mux *http.ServeMux, prefix string) {
	mux.HandleFunc("POST "+prefix+"run", rr.handleRunSingle())
	mux.HandleFunc("POST "+prefix+"run/collection", rr.handleRunCollection())
}

// handleRunSingle handles POST /api/v1/run — executes a single request.
func (rr *RequestRegistrar) handleRunSingle() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input request.ExecuteInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			envelope.WriteError(w, http.StatusBadRequest,
				envelope.CodeInvalidInput, "invalid JSON body: "+err.Error())
			return
		}

		// Validate required fields.
		if input.CollectionID == "" {
			envelope.WriteError(w, http.StatusBadRequest,
				envelope.CodeInvalidInput, "field 'collection' is required")
			return
		}
		if input.RequestID == "" {
			envelope.WriteError(w, http.StatusBadRequest,
				envelope.CodeInvalidInput, "field 'requestId' is required")
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
		defer cancel()

		resp, err := rr.engine.Execute(ctx, input)
		if err != nil {
			rr.writeEngineError(w, err)
			return
		}

		envelope.WriteSuccess(w, http.StatusOK, resp)
	}
}

// handleRunCollection handles POST /api/v1/run/collection — runs all requests.
func (rr *RequestRegistrar) handleRunCollection() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var opts request.CollectionRunOpts
		if err := json.NewDecoder(r.Body).Decode(&opts); err != nil {
			envelope.WriteError(w, http.StatusBadRequest,
				envelope.CodeInvalidInput, "invalid JSON body: "+err.Error())
			return
		}

		// Validate required fields.
		if opts.CollectionID == "" {
			envelope.WriteError(w, http.StatusBadRequest,
				envelope.CodeInvalidInput, "field 'collection' is required")
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
		defer cancel()

		results, err := rr.engine.ExecuteCollection(ctx, opts)
		if err != nil {
			rr.writeEngineError(w, err)
			return
		}

		envelope.WriteSuccess(w, http.StatusOK, results)
	}
}

// writeEngineError writes an appropriate error envelope based on the engine error type.
func (rr *RequestRegistrar) writeEngineError(w http.ResponseWriter, err error) {
	de, ok := err.(*request.DomainError)
	if ok {
		statusCode := envelope.HTTPStatusForCode(de.Code)
		envelope.WriteError(w, statusCode, de.Code, de.Message)
		return
	}
	envelope.WriteError(w, http.StatusInternalServerError,
		envelope.CodeInternalError, err.Error())
}
