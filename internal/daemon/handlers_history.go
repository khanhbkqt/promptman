package daemon

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/khanhnguyen/promptman/internal/history"
	"github.com/khanhnguyen/promptman/pkg/envelope"
)

// historyListResponse is the paginated response shape for GET /api/v1/history.
type historyListResponse struct {
	Data   []history.HistoryEntry `json:"data"`
	Total  int                    `json:"total"`
	Limit  int                    `json:"limit"`
	Offset int                    `json:"offset"`
}

// historyClearResponse is the response shape for DELETE /api/v1/history.
type historyClearResponse struct {
	Deleted int    `json:"deleted"`
	Message string `json:"message"`
}

// HistoryRegistrar registers history query/clear endpoints on the daemon router.
type HistoryRegistrar struct {
	svc history.HistoryService
}

// NewHistoryRegistrar creates a HistoryRegistrar for the given service.
func NewHistoryRegistrar(svc history.HistoryService) *HistoryRegistrar {
	return &HistoryRegistrar{svc: svc}
}

// RegisterRoutes mounts the history endpoints under the given prefix.
func (hr *HistoryRegistrar) RegisterRoutes(mux *http.ServeMux, prefix string) {
	mux.HandleFunc("GET "+prefix+"history", hr.handleList())
	mux.HandleFunc("DELETE "+prefix+"history", hr.handleClear())
}

// handleList handles GET /api/v1/history — query with filters and pagination.
func (hr *HistoryRegistrar) handleList() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()

		query := &history.HistoryQuery{
			Collection:  q.Get("collection"),
			Environment: q.Get("env"),
			Source:      q.Get("source"),
		}

		// Parse optional status filter.
		if statusStr := q.Get("status"); statusStr != "" {
			status, err := strconv.Atoi(statusStr)
			if err != nil {
				envelope.WriteError(w, http.StatusBadRequest,
					envelope.CodeInvalidInput, "invalid 'status' param: must be integer")
				return
			}
			query.Status = &status
		}

		// Parse limit (default 50).
		query.Limit = 50
		if limitStr := q.Get("limit"); limitStr != "" {
			limit, err := strconv.Atoi(limitStr)
			if err != nil || limit < 1 {
				envelope.WriteError(w, http.StatusBadRequest,
					envelope.CodeInvalidInput, "invalid 'limit' param: must be positive integer")
				return
			}
			query.Limit = limit
		}

		// Parse offset (default 0).
		if offsetStr := q.Get("offset"); offsetStr != "" {
			offset, err := strconv.Atoi(offsetStr)
			if err != nil || offset < 0 {
				envelope.WriteError(w, http.StatusBadRequest,
					envelope.CodeInvalidInput, "invalid 'offset' param: must be non-negative integer")
				return
			}
			query.Offset = offset
		}

		results, err := hr.svc.Query(query)
		if err != nil {
			hr.writeHistoryError(w, err)
			return
		}

		if results == nil {
			results = []history.HistoryEntry{}
		}

		envelope.WriteSuccess(w, http.StatusOK, historyListResponse{
			Data:   results,
			Total:  len(results),
			Limit:  query.Limit,
			Offset: query.Offset,
		})
	}
}

// handleClear handles DELETE /api/v1/history — clears history entries.
func (hr *HistoryRegistrar) handleClear() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()

		opts := &history.ClearOpts{}

		// Parse optional "before" date filter.
		if beforeStr := q.Get("before"); beforeStr != "" {
			t, err := time.Parse(time.RFC3339, beforeStr)
			if err != nil {
				// Try date-only format.
				t, err = time.Parse("2006-01-02", beforeStr)
				if err != nil {
					envelope.WriteError(w, http.StatusBadRequest,
						envelope.CodeInvalidInput,
						"invalid 'before' param: must be ISO 8601 date (YYYY-MM-DD or RFC3339)")
					return
				}
			}
			opts.Before = &t
		} else {
			// No filter means clear all.
			opts.All = true
		}

		if err := hr.svc.Clear(opts); err != nil {
			hr.writeHistoryError(w, err)
			return
		}

		msg := "all history cleared"
		if opts.Before != nil {
			msg = fmt.Sprintf("history before %s cleared", opts.Before.Format("2006-01-02"))
		}

		envelope.WriteSuccess(w, http.StatusOK, historyClearResponse{
			Deleted: 0, // file-based clear doesn't return exact count
			Message: msg,
		})
	}
}

// writeHistoryError writes an appropriate error envelope for history domain errors.
func (hr *HistoryRegistrar) writeHistoryError(w http.ResponseWriter, err error) {
	var de *history.DomainError
	if errors.As(err, &de) {
		statusCode := envelope.HTTPStatusForCode(de.Code)
		envelope.WriteError(w, statusCode, de.Code, de.Message)
		return
	}
	envelope.WriteError(w, http.StatusInternalServerError,
		envelope.CodeInternalError, "internal server error")
}
