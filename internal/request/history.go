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
}

// HistoryAppender appends request execution results to a history log.
// Implementations must be safe for concurrent use since Append is
// called from a goroutine (fire-and-forget) for non-blocking operation.
type HistoryAppender interface {
	Append(entry HistoryEntry)
}

// NoOpAppender is the default HistoryAppender that discards all entries.
// It is used when no history storage (M7) is configured.
type NoOpAppender struct{}

// Append is a no-op — the entry is silently discarded.
func (NoOpAppender) Append(HistoryEntry) {}
