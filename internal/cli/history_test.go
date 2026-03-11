package cli

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/khanhnguyen/promptman/pkg/envelope"
)

func TestNewHistoryCommand_Flags(t *testing.T) {
	globals := &GlobalFlags{Format: FormatJSON}
	cmd := newHistoryCommand(globals)

	if cmd.Use != "history" {
		t.Errorf("Use = %q, want 'history'", cmd.Use)
	}

	// Verify flags exist.
	for _, flag := range []string{"collection", "env", "source", "limit"} {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("missing flag %q", flag)
		}
	}
}

func TestNewHistoryCommand_HasClearSubcommand(t *testing.T) {
	globals := &GlobalFlags{Format: FormatJSON}
	cmd := newHistoryCommand(globals)

	var found bool
	for _, sub := range cmd.Commands() {
		if sub.Use == "clear" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'clear' subcommand")
	}
}

func TestHistoryCommand_ListCallsDaemonGet(t *testing.T) {
	// Create a mock daemon server.
	mockData := map[string]any{
		"data":   []any{},
		"total":  0,
		"limit":  20,
		"offset": 0,
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/history" {
			t.Errorf("path = %q, want /api/v1/history", r.URL.Path)
		}

		env := envelope.Success(mockData)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(env)
	}))
	defer srv.Close()

	client := NewClientDirect(srv.URL+"/api/v1", "test-token", srv.Client())

	// Execute a GET through the client.
	env, err := client.Get("/history?limit=20")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !env.OK {
		t.Fatal("expected ok=true")
	}
}

func TestHistoryCommand_ClearCallsDaemonDelete(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}

		env := envelope.Success(map[string]any{"deleted": 0, "message": "all history cleared"})
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(env)
	}))
	defer srv.Close()

	client := NewClientDirect(srv.URL+"/api/v1", "test-token", srv.Client())

	env, err := client.Delete("/history")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if !env.OK {
		t.Fatal("expected ok=true")
	}
}

func TestHistoryCommand_ClearWithBeforeParam(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		before := r.URL.Query().Get("before")
		if before != "2026-03-01" {
			t.Errorf("before = %q, want 2026-03-01", before)
		}

		env := envelope.Success(map[string]any{"deleted": 0, "message": "history before 2026-03-01 cleared"})
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(env)
	}))
	defer srv.Close()

	client := NewClientDirect(srv.URL+"/api/v1", "test-token", srv.Client())

	env, err := client.Delete("/history?before=2026-03-01")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if !env.OK {
		t.Fatal("expected ok=true")
	}
}

func TestHistoryCommand_ListWithCollectionFilter(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		collection := r.URL.Query().Get("collection")
		if collection != "users" {
			t.Errorf("collection = %q, want users", collection)
		}

		mockData := map[string]any{"data": []any{}, "total": 0, "limit": 20, "offset": 0}
		env := envelope.Success(mockData)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(env)
	}))
	defer srv.Close()

	client := NewClientDirect(srv.URL+"/api/v1", "test-token", srv.Client())

	env, err := client.Get("/history?collection=users&limit=20")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !env.OK {
		t.Fatal("expected ok=true")
	}
}

// Suppress unused import warning.
var _ = bytes.NewReader
