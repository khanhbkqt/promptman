package daemon

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/khanhnguyen/promptman/internal/collection"
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
// It uses errors.As to unwrap wrapped domain errors, so errors returned as
// fmt.Errorf("...: %w", domainErr) are handled correctly.
func (rr *RequestRegistrar) writeEngineError(w http.ResponseWriter, err error) {
	// request.DomainError — REQUEST_TIMEOUT, REQUEST_FAILED, etc.
	var rde *request.DomainError
	if errors.As(err, &rde) {
		statusCode := envelope.HTTPStatusForCode(rde.Code)
		envelope.WriteError(w, statusCode, rde.Code, rde.Message)
		return
	}
	// collection.DomainError — COLLECTION_NOT_FOUND, REQUEST_NOT_FOUND, etc.
	// These arrive wrapped (e.g. "loading collection: %w") from engine.go.
	var cde *collection.DomainError
	if errors.As(err, &cde) {
		statusCode := envelope.HTTPStatusForCode(cde.Code)
		envelope.WriteError(w, statusCode, cde.Code, cde.Message)
		return
	}
	// Unknown error — do not leak internal details.
	envelope.WriteError(w, http.StatusInternalServerError,
		envelope.CodeInternalError, "internal server error")
}
