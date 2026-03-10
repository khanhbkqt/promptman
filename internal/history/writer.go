package history

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
)

const (
	// channelSize is the buffer size for the async write channel.
	channelSize = 1000

	// dateFormat is the layout used for daily file rotation.
	dateFormat = "2006-01-02"
)

// Writer is an async JSONL writer that buffers entries via a channel
// and writes them to daily-rotated files in the background.
// It is safe for concurrent use from multiple goroutines.
type Writer struct {
	dir     string             // base directory for history files
	entries chan *HistoryEntry // buffered channel for async writes
	done    chan struct{}      // closed when the consumer goroutine exits
	once    sync.Once          // ensures Close is idempotent
	closed  atomic.Bool        // true after Close is called

	// mutable state owned exclusively by the consumer goroutine
	currentDate string   // YYYY-MM-DD of the currently open file
	currentFile *os.File // handle to the currently open file
}

// NewWriter creates a new async JSONL writer that stores history files
// in the given directory. The directory is created if it does not exist.
// Call Close to flush remaining entries and release resources.
func NewWriter(dir string) (*Writer, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create history dir %s: %w", dir, err)
	}

	w := &Writer{
		dir:     dir,
		entries: make(chan *HistoryEntry, channelSize),
		done:    make(chan struct{}),
	}
	go w.consume()
	return w, nil
}

// Write enqueues a history entry for background writing.
// Returns an error if the writer has been closed.
// This method is non-blocking as long as the channel buffer is not full.
func (w *Writer) Write(entry *HistoryEntry) error {
	if w.closed.Load() {
		return ErrHistoryWriteFailed.Wrap("writer is closed")
	}
	w.entries <- entry
	return nil
}

// Close signals the writer to stop, drains any remaining buffered
// entries, and closes the current file. It is safe to call multiple times.
func (w *Writer) Close() error {
	var closeErr error
	w.once.Do(func() {
		w.closed.Store(true)
		close(w.entries)
		<-w.done // wait for consumer to finish
		if w.currentFile != nil {
			closeErr = w.currentFile.Close()
		}
	})
	return closeErr
}

// consume is the background goroutine that reads entries from the channel
// and writes them to the appropriate daily JSONL file.
func (w *Writer) consume() {
	defer close(w.done)
	for entry := range w.entries {
		if err := w.writeEntry(entry); err != nil {
			// Log write errors but continue processing.
			// In production this could emit metrics or use a logger.
			_ = err
		}
	}
}

// writeEntry marshals a single entry to JSON and appends it to the
// current day's file, rotating files as needed.
func (w *Writer) writeEntry(entry *HistoryEntry) error {
	dateStr := entry.Timestamp.UTC().Format(dateFormat)

	// Rotate file if the date has changed.
	if dateStr != w.currentDate {
		if err := w.rotate(dateStr); err != nil {
			return err
		}
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshal history entry: %w", err)
	}
	data = append(data, '\n')

	if _, err := w.currentFile.Write(data); err != nil {
		return fmt.Errorf("write history entry: %w", err)
	}
	return nil
}

// rotate closes the current file (if any) and opens a new file for
// the given date string.
func (w *Writer) rotate(dateStr string) error {
	if w.currentFile != nil {
		if err := w.currentFile.Close(); err != nil {
			return fmt.Errorf("close history file: %w", err)
		}
	}

	path := filepath.Join(w.dir, dateStr+".jsonl")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open history file %s: %w", path, err)
	}

	w.currentFile = f
	w.currentDate = dateStr
	return nil
}
