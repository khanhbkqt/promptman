package daemon

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/khanhnguyen/promptman/internal/history"
	"github.com/khanhnguyen/promptman/pkg/envelope"
)

// setupHistoryHandler creates a history service with test data and returns
// a configured ServeMux with the HistoryRegistrar routes.
func setupHistoryHandler(t *testing.T) *http.ServeMux {
	t.Helper()

	dir := t.TempDir()
	svc, err := history.NewService(dir)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	t.Cleanup(func() { svc.Close() })

	// Seed some test entries.
	entries := []*history.HistoryEntry{
		{
			Timestamp:   time.Date(2026, 3, 10, 10, 0, 0, 0, time.UTC),
			RequestID:   "health",
			Collection:  "users",
			Method:      "GET",
			URL:         "http://localhost:3000/api/v1/health",
			Status:      200,
			Duration:    42,
			Environment: "dev",
			Source:      "cli",
		},
		{
			Timestamp:   time.Date(2026, 3, 10, 10, 5, 0, 0, time.UTC),
			RequestID:   "login",
			Collection:  "auth",
			Method:      "POST",
			URL:         "http://localhost:3000/api/v1/auth/login",
			Status:      200,
			Duration:    120,
			Environment: "dev",
			Source:      "gui",
		},
		{
			Timestamp:   time.Date(2026, 3, 10, 10, 10, 0, 0, time.UTC),
			RequestID:   "list-users",
			Collection:  "users",
			Method:      "GET",
			URL:         "http://localhost:3000/api/v1/users",
			Status:      401,
			Duration:    15,
			Environment: "staging",
			Source:      "test",
		},
	}
	for _, e := range entries {
		if err := svc.Append(e); err != nil {
			t.Fatalf("Append: %v", err)
		}
	}
	// Flush writer.
	svc.Close()

	// Re-create for reading.
	svc2, err := history.NewService(dir)
	if err != nil {
		t.Fatalf("NewService (re-open): %v", err)
	}
	t.Cleanup(func() { svc2.Close() })

	mux := http.NewServeMux()
	reg := NewHistoryRegistrar(svc2)
	reg.RegisterRoutes(mux, apiPrefix)

	return mux
}

func TestHandleListHistory(t *testing.T) {
	mux := setupHistoryHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/history", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var env envelope.Envelope
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if !env.OK {
		t.Fatalf("expected ok=true, got error: %+v", env.Error)
	}

	raw, _ := json.Marshal(env.Data)
	var resp historyListResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		t.Fatalf("decoding data: %v; raw: %s", err, raw)
	}

	if len(resp.Data) != 3 {
		t.Errorf("got %d entries, want 3", len(resp.Data))
	}
	if resp.Total != 3 {
		t.Errorf("total = %d, want 3", resp.Total)
	}
	if resp.Limit != 50 {
		t.Errorf("limit = %d, want 50 (default)", resp.Limit)
	}
}

func TestHandleListHistory_WithCollectionFilter(t *testing.T) {
	mux := setupHistoryHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/history?collection=users", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var env envelope.Envelope
	json.Unmarshal(w.Body.Bytes(), &env)

	raw, _ := json.Marshal(env.Data)
	var resp historyListResponse
	json.Unmarshal(raw, &resp)

	if len(resp.Data) != 2 {
		t.Errorf("got %d entries for collection=users, want 2", len(resp.Data))
	}
}

func TestHandleListHistory_WithLimitAndOffset(t *testing.T) {
	mux := setupHistoryHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/history?limit=1&offset=1", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var env envelope.Envelope
	json.Unmarshal(w.Body.Bytes(), &env)

	raw, _ := json.Marshal(env.Data)
	var resp historyListResponse
	json.Unmarshal(raw, &resp)

	if len(resp.Data) != 1 {
		t.Errorf("got %d entries with limit=1, want 1", len(resp.Data))
	}
	if resp.Limit != 1 {
		t.Errorf("limit = %d, want 1", resp.Limit)
	}
	if resp.Offset != 1 {
		t.Errorf("offset = %d, want 1", resp.Offset)
	}
}

func TestHandleListHistory_InvalidLimit(t *testing.T) {
	mux := setupHistoryHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/history?limit=abc", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	var env envelope.Envelope
	json.Unmarshal(w.Body.Bytes(), &env)

	if env.OK {
		t.Fatal("expected ok=false for invalid limit")
	}
	if env.Error.Code != envelope.CodeInvalidInput {
		t.Errorf("error code = %q, want %q", env.Error.Code, envelope.CodeInvalidInput)
	}
}

func TestHandleListHistory_InvalidStatus(t *testing.T) {
	mux := setupHistoryHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/history?status=abc", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	var env envelope.Envelope
	json.Unmarshal(w.Body.Bytes(), &env)

	if env.OK {
		t.Fatal("expected ok=false for invalid status")
	}
}

func TestHandleClearHistory(t *testing.T) {
	mux := setupHistoryHandler(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/history", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var env envelope.Envelope
	json.Unmarshal(w.Body.Bytes(), &env)
	if !env.OK {
		t.Fatalf("expected ok=true, got error: %+v", env.Error)
	}

	raw, _ := json.Marshal(env.Data)
	var resp historyClearResponse
	json.Unmarshal(raw, &resp)

	if resp.Message != "all history cleared" {
		t.Errorf("message = %q, want 'all history cleared'", resp.Message)
	}
}

func TestHandleClearHistory_WithBefore(t *testing.T) {
	mux := setupHistoryHandler(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/history?before=2026-03-11", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var env envelope.Envelope
	json.Unmarshal(w.Body.Bytes(), &env)
	if !env.OK {
		t.Fatalf("expected ok=true, got error: %+v", env.Error)
	}
}

func TestHandleClearHistory_InvalidBefore(t *testing.T) {
	mux := setupHistoryHandler(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/history?before=not-a-date", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	var env envelope.Envelope
	json.Unmarshal(w.Body.Bytes(), &env)

	if env.OK {
		t.Fatal("expected ok=false for invalid before")
	}
	if env.Error.Code != envelope.CodeInvalidInput {
		t.Errorf("error code = %q, want %q", env.Error.Code, envelope.CodeInvalidInput)
	}
}
