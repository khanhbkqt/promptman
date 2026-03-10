package history

import (
	"encoding/json"
	"testing"
	"time"
)

func TestHistoryEntry_JSONRoundTrip(t *testing.T) {
	ts := time.Date(2026, 3, 10, 10, 30, 0, 0, time.UTC)
	entry := HistoryEntry{
		Timestamp:   ts,
		RequestID:   "admin/list-admins",
		Collection:  "users",
		Method:      "GET",
		URL:         "http://localhost:3000/api/v1/admin/users",
		Status:      200,
		Duration:    45,
		Environment: "dev",
		Source:      SourceCLI,
	}

	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	// Verify JSON field names match the M7 spec.
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal raw: %v", err)
	}

	expectedFields := []string{"ts", "reqId", "collection", "method", "url", "status", "time", "env", "source"}
	for _, f := range expectedFields {
		if _, ok := raw[f]; !ok {
			t.Errorf("missing JSON field %q", f)
		}
	}

	// Round-trip: unmarshal back.
	var decoded HistoryEntry
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if !decoded.Timestamp.Equal(entry.Timestamp) {
		t.Errorf("timestamp: got %v, want %v", decoded.Timestamp, entry.Timestamp)
	}
	if decoded.RequestID != entry.RequestID {
		t.Errorf("reqId: got %q, want %q", decoded.RequestID, entry.RequestID)
	}
	if decoded.Duration != entry.Duration {
		t.Errorf("time: got %d, want %d", decoded.Duration, entry.Duration)
	}
	if decoded.Source != entry.Source {
		t.Errorf("source: got %q, want %q", decoded.Source, entry.Source)
	}
}

func TestHistoryQuery_OmitEmpty(t *testing.T) {
	q := HistoryQuery{Limit: 20}
	data, err := json.Marshal(q)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal raw: %v", err)
	}

	// Only limit should be present (non-zero).
	if _, ok := raw["limit"]; !ok {
		t.Error("expected 'limit' field to be present")
	}
	// collection should be omitted (empty string).
	if _, ok := raw["collection"]; ok {
		t.Error("expected 'collection' field to be omitted")
	}
}

func TestClearOpts_All(t *testing.T) {
	opts := ClearOpts{All: true}
	data, err := json.Marshal(opts)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded ClearOpts
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if !decoded.All {
		t.Error("expected All to be true")
	}
}

func TestClearOpts_DateRange(t *testing.T) {
	before := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	opts := ClearOpts{Before: &before}

	data, err := json.Marshal(opts)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded ClearOpts
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.Before == nil {
		t.Fatal("expected Before to be non-nil")
	}
	if !decoded.Before.Equal(before) {
		t.Errorf("before: got %v, want %v", decoded.Before, before)
	}
}
