package history

import "time"

// HistoryEntry represents a single request execution log entry.
type HistoryEntry struct {
	Timestamp   time.Time `json:"ts"`
	RequestID   string    `json:"reqId"`
	Collection  string    `json:"collection"`
	Method      string    `json:"method"`
	URL         string    `json:"url"`
	Status      int       `json:"status"`
	Duration    int       `json:"time"` // milliseconds
	Environment string    `json:"env"`
	Source      string    `json:"source"` // cli | gui | test
}

// Source constants for HistoryEntry.Source.
const (
	SourceCLI  = "cli"
	SourceGUI  = "gui"
	SourceTest = "test"
)

// HistoryQuery specifies filters for querying history entries.
type HistoryQuery struct {
	Collection  string     `json:"collection,omitempty"`
	Environment string     `json:"env,omitempty"`
	Status      *int       `json:"status,omitempty"`
	Source      string     `json:"source,omitempty"`
	Since       *time.Time `json:"since,omitempty"`
	Until       *time.Time `json:"until,omitempty"`
	Limit       int        `json:"limit,omitempty"` // default: 50
	Offset      int        `json:"offset,omitempty"`
}

// ClearOpts specifies which history entries to remove.
type ClearOpts struct {
	Before *time.Time `json:"before,omitempty"` // delete entries older than this
	After  *time.Time `json:"after,omitempty"`  // delete entries newer than this
	All    bool       `json:"all,omitempty"`    // delete everything
}
