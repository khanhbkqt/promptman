package history

import (
	"testing"
	"time"

	"github.com/khanhnguyen/promptman/internal/request"
)

func TestAdapter_Append_FullEntry(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	svc, err := NewService(dir)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	defer svc.Close()

	adapter := NewAdapter(svc)

	now := time.Now().UTC()
	entry := request.HistoryEntry{
		CollectionID: "users",
		RequestID:    "health",
		Response: &request.Response{
			RequestID: "health",
			Method:    "GET",
			URL:       "http://localhost:3000/api/v1/health",
			Status:    200,
			Timing:    &request.RequestTiming{Total: 42},
		},
		ExecutedAt:  now,
		Source:      "cli",
		Environment: "dev",
	}

	adapter.Append(entry)

	// Flush writer to ensure entry is written.
	if err := svc.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Re-create service to read.
	svc2, err := NewService(dir)
	if err != nil {
		t.Fatalf("NewService (re-open): %v", err)
	}
	defer svc2.Close()

	results, err := svc2.Query(&HistoryQuery{Limit: 10})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	got := results[0]
	if got.Collection != "users" {
		t.Errorf("Collection = %q, want %q", got.Collection, "users")
	}
	if got.RequestID != "health" {
		t.Errorf("RequestID = %q, want %q", got.RequestID, "health")
	}
	if got.Method != "GET" {
		t.Errorf("Method = %q, want %q", got.Method, "GET")
	}
	if got.URL != "http://localhost:3000/api/v1/health" {
		t.Errorf("URL = %q, want %q", got.URL, "http://localhost:3000/api/v1/health")
	}
	if got.Status != 200 {
		t.Errorf("Status = %d, want %d", got.Status, 200)
	}
	if got.Duration != 42 {
		t.Errorf("Duration = %d, want %d", got.Duration, 42)
	}
	if got.Source != "cli" {
		t.Errorf("Source = %q, want %q", got.Source, "cli")
	}
	if got.Environment != "dev" {
		t.Errorf("Environment = %q, want %q", got.Environment, "dev")
	}
}

func TestAdapter_Append_NilResponse(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	svc, err := NewService(dir)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	defer svc.Close()

	adapter := NewAdapter(svc)

	entry := request.HistoryEntry{
		CollectionID: "auth",
		RequestID:    "login",
		Response:     nil, // nil response (e.g., network error)
		ExecutedAt:   time.Now().UTC(),
		Source:       "test",
		Environment:  "staging",
	}

	// Should not panic.
	adapter.Append(entry)

	if err := svc.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	svc2, err := NewService(dir)
	if err != nil {
		t.Fatalf("NewService (re-open): %v", err)
	}
	defer svc2.Close()

	results, err := svc2.Query(&HistoryQuery{Limit: 10})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	got := results[0]
	if got.Method != "" {
		t.Errorf("Method = %q, want empty", got.Method)
	}
	if got.Status != 0 {
		t.Errorf("Status = %d, want 0", got.Status)
	}
	if got.Source != "test" {
		t.Errorf("Source = %q, want %q", got.Source, "test")
	}
}

func TestAdapter_Append_NilTiming(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	svc, err := NewService(dir)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	defer svc.Close()

	adapter := NewAdapter(svc)

	entry := request.HistoryEntry{
		CollectionID: "users",
		RequestID:    "create",
		Response: &request.Response{
			RequestID: "create",
			Method:    "POST",
			URL:       "http://localhost:3000/api/v1/users",
			Status:    201,
			Timing:    nil, // nil timing
		},
		ExecutedAt:  time.Now().UTC(),
		Source:      "gui",
		Environment: "prod",
	}

	adapter.Append(entry)

	if err := svc.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	svc2, err := NewService(dir)
	if err != nil {
		t.Fatalf("NewService (re-open): %v", err)
	}
	defer svc2.Close()

	results, err := svc2.Query(&HistoryQuery{Limit: 10})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	got := results[0]
	if got.Duration != 0 {
		t.Errorf("Duration = %d, want 0 (nil timing)", got.Duration)
	}
	if got.Source != "gui" {
		t.Errorf("Source = %q, want %q", got.Source, "gui")
	}
}
