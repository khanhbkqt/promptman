package history

import (
	"github.com/khanhnguyen/promptman/internal/request"
)

// Adapter bridges the request engine's HistoryAppender interface
// with the history module's Service. It converts request.HistoryEntry
// to history.HistoryEntry and delegates to Service.Append.
type Adapter struct {
	svc *Service
}

// NewAdapter creates a new Adapter around the given history Service.
func NewAdapter(svc *Service) *Adapter {
	return &Adapter{svc: svc}
}

// Append implements request.HistoryAppender. It converts the request
// engine's entry format to the history module's format and appends it.
// This method is safe for concurrent use (Service.Append is goroutine-safe).
func (a *Adapter) Append(entry request.HistoryEntry) {
	// Extract fields from the response if available.
	var method, url string
	var status, duration int
	if entry.Response != nil {
		method = entry.Response.Method
		url = entry.Response.URL
		status = entry.Response.Status
		if entry.Response.Timing != nil {
			duration = entry.Response.Timing.Total
		}
	}

	he := &HistoryEntry{
		Timestamp:   entry.ExecutedAt,
		RequestID:   entry.RequestID,
		Collection:  entry.CollectionID,
		Method:      method,
		URL:         url,
		Status:      status,
		Duration:    duration,
		Environment: entry.Environment,
		Source:      entry.Source,
	}

	// Best-effort append — errors are silently dropped because
	// history logging must never block or fail the request pipeline.
	_ = a.svc.Append(he)
}
