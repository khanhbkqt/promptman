package daemon

import (
	"encoding/json"
	"net/http"

	"github.com/khanhnguyen/promptman/internal/collection"
	"github.com/khanhnguyen/promptman/pkg/envelope"
)

// CollectionRegistrar registers collection browsing and editing endpoints
// on the daemon router.
type CollectionRegistrar struct {
	svc *collection.Service
}

// NewCollectionRegistrar creates a CollectionRegistrar for the given Service.
func NewCollectionRegistrar(svc *collection.Service) *CollectionRegistrar {
	return &CollectionRegistrar{svc: svc}
}

// RegisterRoutes mounts the collection endpoints under the given prefix.
func (cr *CollectionRegistrar) RegisterRoutes(mux *http.ServeMux, prefix string) {
	mux.HandleFunc("GET "+prefix+"collections", cr.handleList())
	mux.HandleFunc("GET "+prefix+"collections/{id}", cr.handleGet())
	mux.HandleFunc("PUT "+prefix+"collections/{id}", cr.handleUpdate())
}

// handleList handles GET /api/v1/collections — list all collections.
func (cr *CollectionRegistrar) handleList() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		summaries, err := cr.svc.List()
		if err != nil {
			cr.writeCollError(w, err)
			return
		}

		// Return empty array instead of null when no collections exist.
		if summaries == nil {
			summaries = []collection.CollectionSummary{}
		}

		envelope.WriteSuccess(w, http.StatusOK, summaries)
	}
}

// handleGet handles GET /api/v1/collections/{id} — get a full collection.
func (cr *CollectionRegistrar) handleGet() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			envelope.WriteError(w, http.StatusBadRequest,
				envelope.CodeInvalidInput, "collection id is required")
			return
		}

		coll, err := cr.svc.Get(id)
		if err != nil {
			cr.writeCollError(w, err)
			return
		}

		envelope.WriteSuccess(w, http.StatusOK, coll)
	}
}

// handleUpdate handles PUT /api/v1/collections/{id} — partial update.
func (cr *CollectionRegistrar) handleUpdate() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			envelope.WriteError(w, http.StatusBadRequest,
				envelope.CodeInvalidInput, "collection id is required")
			return
		}

		var input collection.UpdateCollectionInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			envelope.WriteError(w, http.StatusBadRequest,
				envelope.CodeInvalidInput, "invalid JSON body: "+err.Error())
			return
		}

		updated, err := cr.svc.Update(id, &input)
		if err != nil {
			cr.writeCollError(w, err)
			return
		}

		envelope.WriteSuccess(w, http.StatusOK, updated)
	}
}

// writeCollError writes an appropriate error envelope based on the collection error type.
func (cr *CollectionRegistrar) writeCollError(w http.ResponseWriter, err error) {
	de, ok := err.(*collection.DomainError)
	if ok {
		statusCode := envelope.HTTPStatusForCode(de.Code)
		envelope.WriteError(w, statusCode, de.Code, de.Message)
		return
	}
	envelope.WriteError(w, http.StatusInternalServerError,
		envelope.CodeInternalError, "internal server error")
}
