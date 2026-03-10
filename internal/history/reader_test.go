package history

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// writeTestEntries writes JSONL entries to a file for testing.
func writeTestEntries(t *testing.T, dir, date string, entries []HistoryEntry) {
	t.Helper()
	path := filepath.Join(dir, date+".jsonl")
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create test file: %v", err)
	}
	defer func() { _ = f.Close() }()

	enc := json.NewEncoder(f)
	for _, e := range entries {
		if err := enc.Encode(e); err != nil {
			t.Fatalf("encode entry: %v", err)
		}
	}
}

func TestQuery_AllFilters(t *testing.T) {
	dir := t.TempDir()

	ts := time.Date(2026, 3, 10, 10, 0, 0, 0, time.UTC)
	entries := []HistoryEntry{
		{Timestamp: ts, RequestID: "r1", Collection: "users", Method: "GET", URL: "http://x/users", Status: 200, Duration: 10, Environment: "dev", Source: SourceCLI},
		{Timestamp: ts.Add(time.Minute), RequestID: "r2", Collection: "auth", Method: "POST", URL: "http://x/auth", Status: 401, Duration: 20, Environment: "staging", Source: SourceGUI},
		{Timestamp: ts.Add(2 * time.Minute), RequestID: "r3", Collection: "users", Method: "PUT", URL: "http://x/users/1", Status: 200, Duration: 30, Environment: "dev", Source: SourceTest},
		{Timestamp: ts.Add(3 * time.Minute), RequestID: "r4", Collection: "orders", Method: "GET", URL: "http://x/orders", Status: 500, Duration: 40, Environment: "prod", Source: SourceCLI},
	}
	writeTestEntries(t, dir, "2026-03-10", entries)

	tests := []struct {
		name    string
		query   HistoryQuery
		wantIDs []string
	}{
		{
			name:    "no filters — all entries, newest first",
			query:   HistoryQuery{},
			wantIDs: []string{"r4", "r3", "r2", "r1"},
		},
		{
			name:    "filter by collection",
			query:   HistoryQuery{Collection: "users"},
			wantIDs: []string{"r3", "r1"},
		},
		{
			name:    "filter by environment",
			query:   HistoryQuery{Environment: "dev"},
			wantIDs: []string{"r3", "r1"},
		},
		{
			name:    "filter by status",
			query:   HistoryQuery{Status: intPtr(401)},
			wantIDs: []string{"r2"},
		},
		{
			name:    "filter by source",
			query:   HistoryQuery{Source: SourceCLI},
			wantIDs: []string{"r4", "r1"},
		},
		{
			name:    "filter by since",
			query:   HistoryQuery{Since: timePtr(ts.Add(2 * time.Minute))},
			wantIDs: []string{"r4", "r3"},
		},
		{
			name:    "filter by until",
			query:   HistoryQuery{Until: timePtr(ts.Add(time.Minute))},
			wantIDs: []string{"r2", "r1"},
		},
		{
			name:    "combined filters",
			query:   HistoryQuery{Collection: "users", Environment: "dev", Source: SourceCLI},
			wantIDs: []string{"r1"},
		},
		{
			name:    "no match",
			query:   HistoryQuery{Collection: "nonexistent"},
			wantIDs: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := Query(dir, &tt.query)
			if err != nil {
				t.Fatalf("Query: %v", err)
			}
			gotIDs := extractIDs(results)
			assertEqualIDs(t, tt.wantIDs, gotIDs)
		})
	}
}

func TestQuery_Pagination(t *testing.T) {
	dir := t.TempDir()

	ts := time.Date(2026, 3, 10, 10, 0, 0, 0, time.UTC)
	var entries []HistoryEntry
	for i := 0; i < 10; i++ {
		entries = append(entries, HistoryEntry{
			Timestamp:   ts.Add(time.Duration(i) * time.Minute),
			RequestID:   fmt.Sprintf("r%d", i),
			Collection:  "test",
			Method:      "GET",
			URL:         "http://x/test",
			Status:      200,
			Duration:    10,
			Environment: "dev",
			Source:      SourceCLI,
		})
	}
	writeTestEntries(t, dir, "2026-03-10", entries)

	tests := []struct {
		name    string
		limit   int
		offset  int
		wantIDs []string
	}{
		{
			name:    "first page",
			limit:   3,
			offset:  0,
			wantIDs: []string{"r9", "r8", "r7"},
		},
		{
			name:    "second page",
			limit:   3,
			offset:  3,
			wantIDs: []string{"r6", "r5", "r4"},
		},
		{
			name:    "last partial page",
			limit:   3,
			offset:  9,
			wantIDs: []string{"r0"},
		},
		{
			name:    "offset beyond results",
			limit:   3,
			offset:  20,
			wantIDs: []string{},
		},
		{
			name:    "default limit",
			limit:   0,
			offset:  0,
			wantIDs: []string{"r9", "r8", "r7", "r6", "r5", "r4", "r3", "r2", "r1", "r0"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := &HistoryQuery{Limit: tt.limit, Offset: tt.offset}
			results, err := Query(dir, q)
			if err != nil {
				t.Fatalf("Query: %v", err)
			}
			gotIDs := extractIDs(results)
			assertEqualIDs(t, tt.wantIDs, gotIDs)
		})
	}
}

func TestQuery_MultipleFiles_ReverseChronological(t *testing.T) {
	dir := t.TempDir()

	// Day 1 entries (older)
	day1 := time.Date(2026, 3, 8, 10, 0, 0, 0, time.UTC)
	writeTestEntries(t, dir, "2026-03-08", []HistoryEntry{
		{Timestamp: day1, RequestID: "d1-r1", Collection: "users", Method: "GET", URL: "http://x/users", Status: 200, Duration: 10, Environment: "dev", Source: SourceCLI},
		{Timestamp: day1.Add(time.Hour), RequestID: "d1-r2", Collection: "users", Method: "GET", URL: "http://x/users", Status: 200, Duration: 20, Environment: "dev", Source: SourceCLI},
	})

	// Day 2 entries (newer)
	day2 := time.Date(2026, 3, 10, 10, 0, 0, 0, time.UTC)
	writeTestEntries(t, dir, "2026-03-10", []HistoryEntry{
		{Timestamp: day2, RequestID: "d2-r1", Collection: "users", Method: "GET", URL: "http://x/users", Status: 200, Duration: 30, Environment: "dev", Source: SourceCLI},
		{Timestamp: day2.Add(time.Hour), RequestID: "d2-r2", Collection: "users", Method: "GET", URL: "http://x/users", Status: 200, Duration: 40, Environment: "dev", Source: SourceCLI},
	})

	results, err := Query(dir, &HistoryQuery{})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}

	wantIDs := []string{"d2-r2", "d2-r1", "d1-r2", "d1-r1"}
	gotIDs := extractIDs(results)
	assertEqualIDs(t, wantIDs, gotIDs)
}

func TestQuery_DateRangeFileSelection(t *testing.T) {
	dir := t.TempDir()

	for _, date := range []string{"2026-03-05", "2026-03-08", "2026-03-10", "2026-03-12"} {
		ts, _ := time.Parse(dateFormat, date)
		writeTestEntries(t, dir, date, []HistoryEntry{
			{Timestamp: ts.Add(10 * time.Hour), RequestID: "r-" + date, Collection: "test", Method: "GET", URL: "http://x", Status: 200, Duration: 10, Environment: "dev", Source: SourceCLI},
		})
	}

	since := time.Date(2026, 3, 8, 0, 0, 0, 0, time.UTC)
	until := time.Date(2026, 3, 10, 23, 59, 59, 0, time.UTC)
	results, err := Query(dir, &HistoryQuery{Since: &since, Until: &until})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}

	wantIDs := []string{"r-2026-03-10", "r-2026-03-08"}
	gotIDs := extractIDs(results)
	assertEqualIDs(t, wantIDs, gotIDs)
}

func TestQuery_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	results, err := Query(dir, nil)
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestQuery_NonexistentDir(t *testing.T) {
	results, err := Query("/nonexistent/path", nil)
	if err != nil {
		t.Fatalf("expected nil error for nonexistent dir, got: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestQuery_CorruptedLines(t *testing.T) {
	dir := t.TempDir()

	// Write a file with some corrupted lines mixed in.
	path := filepath.Join(dir, "2026-03-10.jsonl")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	ts := time.Date(2026, 3, 10, 10, 0, 0, 0, time.UTC)
	good := HistoryEntry{Timestamp: ts, RequestID: "good1", Collection: "test", Method: "GET", URL: "http://x", Status: 200, Duration: 10, Environment: "dev", Source: SourceCLI}
	data, _ := json.Marshal(good)
	fmt.Fprintln(f, string(data))
	fmt.Fprintln(f, "{corrupted json line}")
	fmt.Fprintln(f, "")
	good.RequestID = "good2"
	good.Timestamp = ts.Add(time.Minute)
	data, _ = json.Marshal(good)
	fmt.Fprintln(f, string(data))
	_ = f.Close()

	results, err := Query(dir, nil)
	if err != nil {
		t.Fatalf("Query: %v", err)
	}

	wantIDs := []string{"good2", "good1"}
	gotIDs := extractIDs(results)
	assertEqualIDs(t, wantIDs, gotIDs)
}

// --- Helpers ---

func intPtr(v int) *int              { return &v }
func timePtr(v time.Time) *time.Time { return &v }

func extractIDs(entries []HistoryEntry) []string {
	ids := make([]string, len(entries))
	for i, e := range entries {
		ids[i] = e.RequestID
	}
	return ids
}

func assertEqualIDs(t *testing.T, want, got []string) {
	t.Helper()
	if len(want) != len(got) {
		t.Fatalf("result count: got %d, want %d\n  got:  %v\n  want: %v", len(got), len(want), got, want)
	}
	for i := range want {
		if want[i] != got[i] {
			t.Errorf("  [%d]: got %q, want %q", i, got[i], want[i])
		}
	}
}

// --- Benchmarks ---

func BenchmarkQuery_100K(b *testing.B) {
	dir := b.TempDir()

	// Generate 100K entries across 10 days (10K per day).
	baseDate := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	for day := 0; day < 10; day++ {
		date := baseDate.AddDate(0, 0, day)
		dateStr := date.Format(dateFormat)
		path := filepath.Join(dir, dateStr+".jsonl")
		f, err := os.Create(path)
		if err != nil {
			b.Fatal(err)
		}
		enc := json.NewEncoder(f)
		for i := 0; i < 10000; i++ {
			entry := HistoryEntry{
				Timestamp:   date.Add(time.Duration(i) * time.Second),
				RequestID:   fmt.Sprintf("r%d-%d", day, i),
				Collection:  fmt.Sprintf("col%d", i%5),
				Method:      "GET",
				URL:         fmt.Sprintf("http://x/api/%d", i),
				Status:      200,
				Duration:    i % 100,
				Environment: "dev",
				Source:      SourceCLI,
			}
			if err := enc.Encode(entry); err != nil {
				b.Fatal(err)
			}
		}
		_ = f.Close()
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		q := &HistoryQuery{Collection: "col0", Limit: 20}
		results, err := Query(dir, q)
		if err != nil {
			b.Fatal(err)
		}
		if len(results) != 20 {
			b.Fatalf("expected 20 results, got %d", len(results))
		}
	}
}
