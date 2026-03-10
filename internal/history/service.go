package history

// HistoryService defines the contract for request history operations.
type HistoryService interface {
	// Append records a new history entry asynchronously.
	// The call is non-blocking; the entry is buffered for background writing.
	Append(entry *HistoryEntry) error

	// Query returns history entries matching the given filters.
	Query(filters *HistoryQuery) ([]HistoryEntry, error)

	// Clear removes history entries matching the given options.
	Clear(opts *ClearOpts) error
}
