package history

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestService_AppendAndQuery(t *testing.T) {
	dir := t.TempDir()
	svc, err := NewService(dir)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	defer func() { _ = svc.Close() }()

	ts := time.Date(2026, 3, 10, 10, 0, 0, 0, time.UTC)
	entries := []HistoryEntry{
		{Timestamp: ts, RequestID: "r1", Collection: "users", Method: "GET", URL: "http://x/users", Status: 200, Duration: 10, Environment: "dev", Source: SourceCLI},
		{Timestamp: ts.Add(time.Minute), RequestID: "r2", Collection: "auth", Method: "POST", URL: "http://x/auth", Status: 401, Duration: 20, Environment: "staging", Source: SourceGUI},
	}

	for i := range entries {
		if err := svc.Append(&entries[i]); err != nil {
			t.Fatalf("Append: %v", err)
		}
	}

	// Close to flush buffered writes.
	if err := svc.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Re-read using Query function directly.
	results, err := Query(dir, &HistoryQuery{})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	// Newest first.
	if results[0].RequestID != "r2" {
		t.Errorf("expected r2 first, got %s", results[0].RequestID)
	}
	if results[1].RequestID != "r1" {
		t.Errorf("expected r1 second, got %s", results[1].RequestID)
	}
}

func TestService_ClearAll(t *testing.T) {
	dir := t.TempDir()

	ts := time.Date(2026, 3, 10, 10, 0, 0, 0, time.UTC)
	writeTestEntries(t, dir, "2026-03-10", []HistoryEntry{
		{Timestamp: ts, RequestID: "r1", Collection: "test", Method: "GET", URL: "http://x", Status: 200, Duration: 10, Environment: "dev", Source: SourceCLI},
	})
	writeTestEntries(t, dir, "2026-03-09", []HistoryEntry{
		{Timestamp: ts.Add(-24 * time.Hour), RequestID: "r2", Collection: "test", Method: "GET", URL: "http://x", Status: 200, Duration: 10, Environment: "dev", Source: SourceCLI},
	})

	svc := &Service{dir: dir}
	if err := svc.Clear(nil); err != nil {
		t.Fatalf("Clear(nil): %v", err)
	}

	// Verify all files are gone.
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".jsonl" {
			t.Errorf("expected no jsonl files, found %s", e.Name())
		}
	}
}

func TestService_ClearAll_ExplicitFlag(t *testing.T) {
	dir := t.TempDir()

	ts := time.Date(2026, 3, 10, 10, 0, 0, 0, time.UTC)
	writeTestEntries(t, dir, "2026-03-10", []HistoryEntry{
		{Timestamp: ts, RequestID: "r1", Collection: "test", Method: "GET", URL: "http://x", Status: 200, Duration: 10, Environment: "dev", Source: SourceCLI},
	})

	svc := &Service{dir: dir}
	if err := svc.Clear(&ClearOpts{All: true}); err != nil {
		t.Fatalf("Clear(All): %v", err)
	}

	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".jsonl" {
			t.Errorf("expected no jsonl files, found %s", e.Name())
		}
	}
}

func TestService_ClearByDateRange(t *testing.T) {
	dir := t.TempDir()

	for _, date := range []string{"2026-03-05", "2026-03-08", "2026-03-10", "2026-03-12"} {
		ts, _ := time.Parse(dateFormat, date)
		writeTestEntries(t, dir, date, []HistoryEntry{
			{Timestamp: ts.Add(10 * time.Hour), RequestID: "r-" + date, Collection: "test", Method: "GET", URL: "http://x", Status: 200, Duration: 10, Environment: "dev", Source: SourceCLI},
		})
	}

	// Delete files before 2026-03-10.
	before := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
	svc := &Service{dir: dir}
	if err := svc.Clear(&ClearOpts{Before: &before}); err != nil {
		t.Fatalf("Clear(Before): %v", err)
	}

	// Only 2026-03-10 and 2026-03-12 should remain.
	entries, _ := os.ReadDir(dir)
	var remaining []string
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".jsonl" {
			remaining = append(remaining, e.Name())
		}
	}
	if len(remaining) != 2 {
		t.Fatalf("expected 2 remaining files, got %d: %v", len(remaining), remaining)
	}
}

func TestService_Retention(t *testing.T) {
	dir := t.TempDir()

	now := time.Now().UTC()
	// Create files at various ages.
	dates := []string{
		now.Format(dateFormat),                    // today
		now.AddDate(0, 0, -3).Format(dateFormat),  // 3 days ago
		now.AddDate(0, 0, -7).Format(dateFormat),  // 7 days ago
		now.AddDate(0, 0, -15).Format(dateFormat), // 15 days ago
		now.AddDate(0, 0, -30).Format(dateFormat), // 30 days ago
	}

	for _, date := range dates {
		ts, _ := time.Parse(dateFormat, date)
		writeTestEntries(t, dir, date, []HistoryEntry{
			{Timestamp: ts.Add(10 * time.Hour), RequestID: "r-" + date, Collection: "test", Method: "GET", URL: "http://x", Status: 200, Duration: 10, Environment: "dev", Source: SourceCLI},
		})
	}

	// Retain 7 days → should delete the 15-day and 30-day files.
	svc := &Service{dir: dir}
	if err := svc.Retention(7); err != nil {
		t.Fatalf("Retention: %v", err)
	}

	entries, _ := os.ReadDir(dir)
	var remaining []string
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".jsonl" {
			remaining = append(remaining, e.Name())
		}
	}

	// Today, 3 days ago, and 7 days ago should remain (3 files).
	if len(remaining) != 3 {
		t.Errorf("expected 3 remaining files, got %d: %v", len(remaining), remaining)
	}
}

func TestService_Retention_ZeroDays(t *testing.T) {
	dir := t.TempDir()

	writeTestEntries(t, dir, "2026-03-01", []HistoryEntry{
		{Timestamp: time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC), RequestID: "r1", Collection: "test", Method: "GET", URL: "http://x", Status: 200, Duration: 10, Environment: "dev", Source: SourceCLI},
	})

	svc := &Service{dir: dir}
	if err := svc.Retention(0); err != nil {
		t.Fatalf("Retention(0): %v", err)
	}

	// Nothing should be deleted.
	entries, _ := os.ReadDir(dir)
	count := 0
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".jsonl" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected 1 file, got %d", count)
	}
}

func TestService_ClearNonexistentDir(t *testing.T) {
	svc := &Service{dir: "/nonexistent/path"}
	if err := svc.Clear(&ClearOpts{All: true}); err != nil {
		t.Errorf("expected nil error for nonexistent dir, got: %v", err)
	}
}
