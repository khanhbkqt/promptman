package history

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Service is the concrete implementation of HistoryService.
// It wires the async Writer for appends and the reader for queries.
type Service struct {
	dir    string  // base directory for history files
	writer *Writer // async JSONL writer
}

// NewService creates a new Service backed by the given directory.
// The directory is created if it does not exist.
func NewService(dir string) (*Service, error) {
	w, err := NewWriter(dir)
	if err != nil {
		return nil, err
	}
	return &Service{dir: dir, writer: w}, nil
}

// Append records a new history entry asynchronously.
func (s *Service) Append(entry *HistoryEntry) error {
	return s.writer.Write(entry)
}

// Query returns history entries matching the given filters.
func (s *Service) Query(filters *HistoryQuery) ([]HistoryEntry, error) {
	return Query(s.dir, filters)
}

// Clear removes history entries matching the given options.
func (s *Service) Clear(opts *ClearOpts) error {
	if opts == nil || opts.All {
		return clearAll(s.dir)
	}
	return clearByDateRange(s.dir, opts.Before, opts.After)
}

// Retention deletes history files older than the given number of days.
// A value of 0 means no cleanup.
func (s *Service) Retention(days int) error {
	if days <= 0 {
		return nil
	}
	cutoff := time.Now().UTC().AddDate(0, 0, -days)
	return clearByDateRange(s.dir, &cutoff, nil)
}

// Close shuts down the writer, flushing any buffered entries.
func (s *Service) Close() error {
	return s.writer.Close()
}

// clearAll removes all .jsonl files from the history directory.
func clearAll(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read history dir %s: %w", dir, err)
	}

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".jsonl") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove history file %s: %w", path, err)
		}
	}
	return nil
}

// clearByDateRange removes .jsonl files whose date falls within the
// specified range. before=nil means no upper bound, after=nil means no lower bound.
// Files with dates strictly before "before" and strictly after "after" are removed.
func clearByDateRange(dir string, before, after *time.Time) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read history dir %s: %w", dir, err)
	}

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".jsonl") {
			continue
		}
		dateStr := strings.TrimSuffix(e.Name(), ".jsonl")
		fileDate, err := time.Parse(dateFormat, dateStr)
		if err != nil {
			continue
		}

		shouldRemove := false
		if before != nil && after != nil {
			// Remove files in [after, before) range — but typically
			// before means "delete files before this date"
			// after means "delete files after this date"
			shouldRemove = fileDate.Before(truncateToDay(*before)) &&
				!fileDate.Before(truncateToDay(*after))
		} else if before != nil {
			// Delete files older than "before" date.
			shouldRemove = fileDate.Before(truncateToDay(*before))
		} else if after != nil {
			// Delete files newer than "after" date.
			shouldRemove = fileDate.After(truncateToDay(*after))
		}

		if shouldRemove {
			path := filepath.Join(dir, e.Name())
			if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("remove history file %s: %w", path, err)
			}
		}
	}
	return nil
}
