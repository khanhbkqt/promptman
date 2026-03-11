package daemon

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/khanhnguyen/promptman/internal/collection"
	"github.com/khanhnguyen/promptman/pkg/envelope"
)

// setupCollHandler creates test collections on disk and returns
// a configured ServeMux with the CollectionRegistrar routes.
func setupCollHandler(t *testing.T) *http.ServeMux {
	t.Helper()

	tmpDir := t.TempDir()
	collDir := filepath.Join(tmpDir, ".promptman", "collections")
	if err := os.MkdirAll(collDir, 0o755); err != nil {
		t.Fatalf("creating coll dir: %v", err)
	}

	// Create "users-api" collection.
	usersYAML := `name: Users API
baseUrl: https://api.example.com
requests:
  - id: list-users
    method: GET
    path: /users
  - id: get-user
    method: GET
    path: /users/1
`
	if err := os.WriteFile(filepath.Join(collDir, "users-api.yaml"), []byte(usersYAML), 0o644); err != nil {
		t.Fatalf("writing users-api.yaml: %v", err)
	}

	// Create "auth-api" collection with folders.
	authYAML := `name: Auth API
baseUrl: https://auth.example.com
requests:
  - id: login
    method: POST
    path: /login
folders:
  - id: admin
    name: Admin
    requests:
      - id: list-admins
        method: GET
        path: /admins
`
	if err := os.WriteFile(filepath.Join(collDir, "auth-api.yaml"), []byte(authYAML), 0o644); err != nil {
		t.Fatalf("writing auth-api.yaml: %v", err)
	}

	repo := collection.NewFileRepository(collDir)
	svc := collection.NewService(repo)

	mux := http.NewServeMux()
	reg := NewCollectionRegistrar(svc)
	reg.RegisterRoutes(mux, apiPrefix)

	return mux
}

// setupEmptyCollHandler creates a handler with no collections on disk.
func setupEmptyCollHandler(t *testing.T) *http.ServeMux {
	t.Helper()

	tmpDir := t.TempDir()
	collDir := filepath.Join(tmpDir, ".promptman", "collections")
	if err := os.MkdirAll(collDir, 0o755); err != nil {
		t.Fatalf("creating coll dir: %v", err)
	}

	repo := collection.NewFileRepository(collDir)
	svc := collection.NewService(repo)

	mux := http.NewServeMux()
	reg := NewCollectionRegistrar(svc)
	reg.RegisterRoutes(mux, apiPrefix)

	return mux
}

func TestHandleListCollections(t *testing.T) {
	mux := setupCollHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/collections", nil)
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
		t.Fatal("expected ok=true")
	}

	raw, _ := json.Marshal(env.Data)
	var items []collection.CollectionSummary
	if err := json.Unmarshal(raw, &items); err != nil {
		t.Fatalf("decoding data: %v; raw: %s", err, raw)
	}

	if len(items) != 2 {
		t.Fatalf("got %d items, want 2", len(items))
	}

	// Verify at least one has requests counted.
	found := false
	for _, item := range items {
		if item.ID == "users-api" && item.RequestCount == 2 {
			found = true
		}
	}
	if !found {
		t.Error("expected users-api with requestCount=2")
	}
}

func TestHandleListCollections_Empty(t *testing.T) {
	mux := setupEmptyCollHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/collections", nil)
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
		t.Fatal("expected ok=true")
	}

	raw, _ := json.Marshal(env.Data)
	var items []collection.CollectionSummary
	if err := json.Unmarshal(raw, &items); err != nil {
		t.Fatalf("decoding data: %v; raw: %s", err, raw)
	}

	if len(items) != 0 {
		t.Fatalf("got %d items, want 0", len(items))
	}
}

func TestHandleGetCollection(t *testing.T) {
	mux := setupCollHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/collections/users-api", nil)
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
		t.Fatal("expected ok=true")
	}

	raw, _ := json.Marshal(env.Data)
	var coll map[string]any
	if err := json.Unmarshal(raw, &coll); err != nil {
		t.Fatalf("decoding collection data: %v", err)
	}
	if coll["name"] != "Users API" {
		t.Errorf("name = %v, want Users API", coll["name"])
	}

	requests, ok := coll["requests"].([]any)
	if !ok {
		t.Fatal("expected requests array in response")
	}
	if len(requests) != 2 {
		t.Errorf("got %d requests, want 2", len(requests))
	}
}

func TestHandleGetCollection_NotFound(t *testing.T) {
	mux := setupCollHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/collections/nonexistent", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	var env envelope.Envelope
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if env.OK {
		t.Fatal("expected ok=false for missing collection")
	}
	if env.Error.Code != envelope.CodeCollectionNotFound {
		t.Errorf("error code = %q, want %q", env.Error.Code, envelope.CodeCollectionNotFound)
	}
}

func TestHandleUpdateCollection(t *testing.T) {
	mux := setupCollHandler(t)

	newName := "Updated Users API"
	input := map[string]any{"name": newName}
	body, _ := json.Marshal(input)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/collections/users-api", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
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
	var coll map[string]any
	if err := json.Unmarshal(raw, &coll); err != nil {
		t.Fatalf("decoding collection data: %v", err)
	}
	if coll["name"] != newName {
		t.Errorf("name = %v, want %s", coll["name"], newName)
	}
}

func TestHandleUpdateCollection_NotFound(t *testing.T) {
	mux := setupCollHandler(t)

	input := map[string]any{"name": "nope"}
	body, _ := json.Marshal(input)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/collections/nonexistent", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	var env envelope.Envelope
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if env.OK {
		t.Fatal("expected ok=false for missing collection")
	}
	if env.Error.Code != envelope.CodeCollectionNotFound {
		t.Errorf("error code = %q, want %q", env.Error.Code, envelope.CodeCollectionNotFound)
	}
}

func TestHandleUpdateCollection_InvalidBody(t *testing.T) {
	mux := setupCollHandler(t)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/collections/users-api",
		bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	var env envelope.Envelope
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if env.OK {
		t.Fatal("expected ok=false for invalid JSON")
	}
	if env.Error.Code != envelope.CodeInvalidInput {
		t.Errorf("error code = %q, want %q", env.Error.Code, envelope.CodeInvalidInput)
	}
}
