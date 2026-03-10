package history

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const defaultLimit = 50

// Query reads JSONL history files from dir and returns entries matching
// the given filters, ordered in reverse chronological order (newest first).
func Query(dir string, q *HistoryQuery) ([]HistoryEntry, error) {
	if q == nil {
		q = &HistoryQuery{}
	}
	limit := q.Limit
	if limit <= 0 {
		limit = defaultLimit
	}

	files, err := listFiles(dir, q.Since, q.Until)
	if err != nil {
		return nil, err
	}

	// Reverse sort: newest files first for reverse chronological order.
	sort.Sort(sort.Reverse(sort.StringSlice(files)))

	var all []HistoryEntry
	needed := q.Offset + limit

	for _, f := range files {
		entries, err := readFile(f, q)
		if err != nil {
			return nil, err
		}
		// Reverse entries within each file so newest come first.
		reverseEntries(entries)
		all = append(all, entries...)
		// Early exit: we have enough entries after filtering.
		if len(all) >= needed {
			break
		}
	}

	// Apply offset and limit.
	if q.Offset >= len(all) {
		return []HistoryEntry{}, nil
	}
	end := q.Offset + limit
	if end > len(all) {
		end = len(all)
	}
	return all[q.Offset:end], nil
}

// listFiles returns sorted absolute paths of JSONL files in dir,
// filtered by the since/until date range from the query.
func listFiles(dir string, since, until *time.Time) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read history dir %s: %w", dir, err)
	}

	var files []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".jsonl") {
			continue
		}
		dateStr := strings.TrimSuffix(e.Name(), ".jsonl")
		fileDate, err := time.Parse(dateFormat, dateStr)
		if err != nil {
			continue // skip files with non-date names
		}

		// File-level date range filtering: each file covers one UTC day.
		if since != nil && fileDate.AddDate(0, 0, 1).Before(truncateToDay(*since)) {
			continue // file's last possible entry is before since
		}
		if until != nil && fileDate.After(truncateToDay(*until)) {
			continue // file starts after until
		}

		files = append(files, filepath.Join(dir, e.Name()))
	}
	sort.Strings(files)
	return files, nil
}

// truncateToDay truncates a time to the start of its UTC day.
func truncateToDay(t time.Time) time.Time {
	t = t.UTC()
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// readFile reads a single JSONL file and returns entries matching the filters.
func readFile(path string, q *HistoryQuery) ([]HistoryEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open history file %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	var entries []HistoryEntry
	scanner := bufio.NewScanner(f)
	// Set a generous max line size for entries with long URLs.
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var entry HistoryEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			// Skip corrupted lines but continue processing.
			continue
		}

		if matchesFilters(&entry, q) {
			entries = append(entries, entry)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan history file %s: %w", path, err)
	}
	return entries, nil
}

// matchesFilters checks if an entry matches all query filters.
func matchesFilters(e *HistoryEntry, q *HistoryQuery) bool {
	if q.Collection != "" && e.Collection != q.Collection {
		return false
	}
	if q.Environment != "" && e.Environment != q.Environment {
		return false
	}
	if q.Status != nil && e.Status != *q.Status {
		return false
	}
	if q.Source != "" && e.Source != q.Source {
		return false
	}
	if q.Since != nil && e.Timestamp.Before(*q.Since) {
		return false
	}
	if q.Until != nil && e.Timestamp.After(*q.Until) {
		return false
	}
	return true
}

// reverseEntries reverses a slice of HistoryEntry in place.
func reverseEntries(s []HistoryEntry) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}
