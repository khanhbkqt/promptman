package history

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWriter_BasicWrite(t *testing.T) {
	dir := t.TempDir()
	w, err := NewWriter(dir)
	if err != nil {
		t.Fatalf("NewWriter: %v", err)
	}

	ts := time.Date(2026, 3, 10, 10, 30, 0, 0, time.UTC)
	entry := &HistoryEntry{
		Timestamp:   ts,
		RequestID:   "users/list",
		Collection:  "users",
		Method:      "GET",
		URL:         "http://localhost/api/users",
		Status:      200,
		Duration:    42,
		Environment: "dev",
		Source:      SourceCLI,
	}

	if err := w.Write(entry); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Verify the file was created with the correct name.
	path := filepath.Join(dir, "2026-03-10.jsonl")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}

	var decoded HistoryEntry
	if err := json.Unmarshal(data[:len(data)-1], &decoded); err != nil { // strip newline
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.RequestID != "users/list" {
		t.Errorf("reqId: got %q, want %q", decoded.RequestID, "users/list")
	}
	if decoded.Duration != 42 {
		t.Errorf("time: got %d, want 42", decoded.Duration)
	}
}

func TestWriter_MultipleEntries(t *testing.T) {
	dir := t.TempDir()
	w, err := NewWriter(dir)
	if err != nil {
		t.Fatalf("NewWriter: %v", err)
	}

	ts := time.Date(2026, 3, 10, 10, 0, 0, 0, time.UTC)
	for i := 0; i < 100; i++ {
		entry := &HistoryEntry{
			Timestamp:   ts,
			RequestID:   "req",
			Collection:  "col",
			Method:      "GET",
			URL:         "http://test",
			Status:      200,
			Duration:    i,
			Environment: "dev",
			Source:      SourceCLI,
		}
		if err := w.Write(entry); err != nil {
			t.Fatalf("Write[%d]: %v", i, err)
		}
	}

	if err := w.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Count lines in the output file.
	path := filepath.Join(dir, "2026-03-10.jsonl")
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	count := 0
	for scanner.Scan() {
		count++
	}
	if count != 100 {
		t.Errorf("line count: got %d, want 100", count)
	}
}

func TestWriter_DailyRotation(t *testing.T) {
	dir := t.TempDir()
	w, err := NewWriter(dir)
	if err != nil {
		t.Fatalf("NewWriter: %v", err)
	}

	// Write entries on two different days.
	day1 := time.Date(2026, 3, 10, 10, 0, 0, 0, time.UTC)
	day2 := time.Date(2026, 3, 11, 10, 0, 0, 0, time.UTC)

	if err := w.Write(&HistoryEntry{Timestamp: day1, RequestID: "day1"}); err != nil {
		t.Fatalf("Write day1: %v", err)
	}
	if err := w.Write(&HistoryEntry{Timestamp: day2, RequestID: "day2"}); err != nil {
		t.Fatalf("Write day2: %v", err)
	}

	if err := w.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Both files should exist.
	for _, name := range []string{"2026-03-10.jsonl", "2026-03-11.jsonl"} {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file %s to exist", name)
		}
	}

	// Verify content of each file.
	checkFileEntry(t, filepath.Join(dir, "2026-03-10.jsonl"), "day1")
	checkFileEntry(t, filepath.Join(dir, "2026-03-11.jsonl"), "day2")
}

func TestWriter_FlushOnClose(t *testing.T) {
	dir := t.TempDir()
	w, err := NewWriter(dir)
	if err != nil {
		t.Fatalf("NewWriter: %v", err)
	}

	// Write entries then close immediately.
	for i := 0; i < 50; i++ {
		ts := time.Date(2026, 3, 10, 10, 0, 0, 0, time.UTC)
		entry := &HistoryEntry{Timestamp: ts, RequestID: "flush", Duration: i}
		if err := w.Write(entry); err != nil {
			t.Fatalf("Write[%d]: %v", i, err)
		}
	}

	if err := w.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// All 50 entries should be flushed.
	path := filepath.Join(dir, "2026-03-10.jsonl")
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	count := 0
	for scanner.Scan() {
		count++
	}
	if count != 50 {
		t.Errorf("flushed entries: got %d, want 50", count)
	}
}

func TestWriter_CloseIdempotent(t *testing.T) {
	dir := t.TempDir()
	w, err := NewWriter(dir)
	if err != nil {
		t.Fatalf("NewWriter: %v", err)
	}

	// Close multiple times should not panic.
	if err := w.Close(); err != nil {
		t.Fatalf("Close 1: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("Close 2: %v", err)
	}
}

func TestWriter_WriteAfterClose(t *testing.T) {
	dir := t.TempDir()
	w, err := NewWriter(dir)
	if err != nil {
		t.Fatalf("NewWriter: %v", err)
	}

	if err := w.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	ts := time.Date(2026, 3, 10, 10, 0, 0, 0, time.UTC)
	err = w.Write(&HistoryEntry{Timestamp: ts, RequestID: "late"})
	if err == nil {
		t.Fatal("expected error writing after close")
	}
}

func TestWriter_DirectoryCreation(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "deep", "history")
	w, err := NewWriter(dir)
	if err != nil {
		t.Fatalf("NewWriter: %v", err)
	}

	ts := time.Date(2026, 3, 10, 10, 0, 0, 0, time.UTC)
	if err := w.Write(&HistoryEntry{Timestamp: ts, RequestID: "nested"}); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	path := filepath.Join(dir, "2026-03-10.jsonl")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("expected file %s to exist", path)
	}
}

// BenchmarkWriter_Append measures the latency of the non-blocking Write call.
// The acceptance criterion requires < 1ms append latency.
func BenchmarkWriter_Append(b *testing.B) {
	dir := b.TempDir()
	w, err := NewWriter(dir)
	if err != nil {
		b.Fatalf("NewWriter: %v", err)
	}
	defer w.Close()

	ts := time.Date(2026, 3, 10, 10, 0, 0, 0, time.UTC)
	entry := &HistoryEntry{
		Timestamp:   ts,
		RequestID:   "bench/request",
		Collection:  "benchmark",
		Method:      "GET",
		URL:         "http://localhost/bench",
		Status:      200,
		Duration:    10,
		Environment: "bench",
		Source:      SourceCLI,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := w.Write(entry); err != nil {
			b.Fatalf("Write: %v", err)
		}
	}
}

// checkFileEntry reads the first line of a JSONL file and verifies the reqId.
func checkFileEntry(t *testing.T, path, expectedReqID string) {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	if !scanner.Scan() {
		t.Fatalf("no lines in %s", path)
	}

	var entry HistoryEntry
	if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
		t.Fatalf("unmarshal entry from %s: %v", path, err)
	}
	if entry.RequestID != expectedReqID {
		t.Errorf("reqId in %s: got %q, want %q", path, entry.RequestID, expectedReqID)
	}
}
