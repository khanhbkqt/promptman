package request

import (
	"time"
)

// HistoryEntry captures the result of a single request execution for logging.
type HistoryEntry struct {
	CollectionID string    `json:"collectionId"`
	RequestID    string    `json:"requestId"`
	Response     *Response `json:"response"`
	ExecutedAt   time.Time `json:"executedAt"`
	Source       string    `json:"source,omitempty"` // cli | gui | test
	Environment  string    `json:"env,omitempty"`    // environment name
}

// HistoryAppender appends request execution results to a history log.
//
// Implementations MUST be safe for concurrent use: Append is invoked
// from a fire-and-forget goroutine inside Execute, so multiple calls to
// Append may occur simultaneously (e.g., when ExecuteCollection runs
// several requests in a loop). Concrete implementations must protect
// shared state with synchronization primitives such as sync.Mutex.
type HistoryAppender interface {
	Append(entry HistoryEntry)
}

// NoOpAppender is the default HistoryAppender that discards all entries.
// It is used when no history storage (M7) is configured.
type NoOpAppender struct{}

// Append is a no-op — the entry is silently discarded.
func (NoOpAppender) Append(HistoryEntry) {}
