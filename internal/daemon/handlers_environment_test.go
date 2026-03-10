package daemon

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/khanhnguyen/promptman/internal/environment"
	"github.com/khanhnguyen/promptman/pkg/envelope"
)

// setupEnvHandler creates a test environment on disk and returns
// a configured ServeMux with the EnvironmentRegistrar routes and
// a cleanup function.
func setupEnvHandler(t *testing.T) (*http.ServeMux, string) {
	t.Helper()

	tmpDir := t.TempDir()
	envDir := filepath.Join(tmpDir, ".promptman", "environments")
	if err := os.MkdirAll(envDir, 0o755); err != nil {
		t.Fatalf("creating env dir: %v", err)
	}

	// Create "dev" environment file.
	devYAML := `name: dev
variables:
  host: localhost
  port: 8080
`
	if err := os.WriteFile(filepath.Join(envDir, "dev.yaml"), []byte(devYAML), 0o644); err != nil {
		t.Fatalf("writing dev.yaml: %v", err)
	}

	// Create "staging" environment file.
	stagingYAML := `name: staging
variables:
  host: staging.example.com
  port: 443
`
	if err := os.WriteFile(filepath.Join(envDir, "staging.yaml"), []byte(stagingYAML), 0o644); err != nil {
		t.Fatalf("writing staging.yaml: %v", err)
	}

	repo := environment.NewFileRepository(envDir)
	svc := environment.NewService(repo)

	mux := http.NewServeMux()
	reg := NewEnvironmentRegistrar(svc)
	reg.RegisterRoutes(mux, apiPrefix)

	return mux, envDir
}

func TestHandleListEnvironments(t *testing.T) {
	mux, _ := setupEnvHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/environments", nil)
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

	// Data should be a list of envListItem.
	raw, _ := json.Marshal(env.Data)
	var items []envListItem
	if err := json.Unmarshal(raw, &items); err != nil {
		t.Fatalf("decoding data: %v; raw: %s", err, raw)
	}

	if len(items) != 2 {
		t.Fatalf("got %d items, want 2", len(items))
	}

	// No active env set, so all should be false.
	for _, item := range items {
		if item.Active {
			t.Errorf("%s should not be active", item.Name)
		}
	}
}

func TestHandleListEnvironments_WithActive(t *testing.T) {
	mux, tmpDir := setupEnvHandler(t)

	// Set "dev" as active via the underlying service.
	repo := environment.NewFileRepository(tmpDir)
	svc := environment.NewService(repo)
	_ = tmpDir // tmpDir is the envDir returned from setupEnvHandler
	if err := svc.SetActive("dev"); err != nil {
		t.Fatalf("setting active: %v", err)
	}

	// Re-create mux with the same repo so it reads the active state.
	mux = http.NewServeMux()
	reg := NewEnvironmentRegistrar(svc)
	reg.RegisterRoutes(mux, apiPrefix)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/environments", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	var env envelope.Envelope
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("decoding response: %v", err)
	}

	raw, _ := json.Marshal(env.Data)
	var items []envListItem
	if err := json.Unmarshal(raw, &items); err != nil {
		t.Fatalf("decoding data: %v", err)
	}

	var activeCount int
	for _, item := range items {
		if item.Active {
			activeCount++
			if item.Name != "dev" {
				t.Errorf("active env = %q, want dev", item.Name)
			}
		}
	}
	if activeCount != 1 {
		t.Errorf("active count = %d, want 1", activeCount)
	}
}

func TestHandleGetEnvironment(t *testing.T) {
	mux, _ := setupEnvHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/environments/dev", nil)
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

	// Verify data contains expected variables.
	raw, _ := json.Marshal(env.Data)
	var envData map[string]any
	if err := json.Unmarshal(raw, &envData); err != nil {
		t.Fatalf("decoding env data: %v", err)
	}
	if envData["name"] != "dev" {
		t.Errorf("name = %v, want dev", envData["name"])
	}
}

func TestHandleGetEnvironment_NotFound(t *testing.T) {
	mux, _ := setupEnvHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/environments/nonexistent", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	var env envelope.Envelope
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if env.OK {
		t.Fatal("expected ok=false for missing env")
	}
}

func TestHandleSetActive(t *testing.T) {
	mux, _ := setupEnvHandler(t)

	body, _ := json.Marshal(setActiveRequest{Name: "dev"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/environments/active", bytes.NewReader(body))
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
}

func TestHandleSetActive_MissingName(t *testing.T) {
	mux, _ := setupEnvHandler(t)

	body, _ := json.Marshal(setActiveRequest{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/environments/active", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	var env envelope.Envelope
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if env.OK {
		t.Fatal("expected ok=false for missing name")
	}
	if env.Error.Code != envelope.CodeInvalidInput {
		t.Errorf("error code = %q, want %q", env.Error.Code, envelope.CodeInvalidInput)
	}
}

func TestHandleSetActive_NotFound(t *testing.T) {
	mux, _ := setupEnvHandler(t)

	body, _ := json.Marshal(setActiveRequest{Name: "nonexistent"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/environments/active", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	var env envelope.Envelope
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if env.OK {
		t.Fatal("expected ok=false for nonexistent env")
	}
}

func TestHandleUpdateEnvironment(t *testing.T) {
	mux, _ := setupEnvHandler(t)

	newVars := map[string]any{"host": "updated.example.com", "port": float64(9090)}
	input := map[string]any{"variables": newVars}
	body, _ := json.Marshal(input)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/environments/dev", bytes.NewReader(body))
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

	// Verify updated values.
	raw, _ := json.Marshal(env.Data)
	var envData map[string]any
	if err := json.Unmarshal(raw, &envData); err != nil {
		t.Fatalf("decoding env data: %v", err)
	}

	vars, ok := envData["variables"].(map[string]any)
	if !ok {
		t.Fatal("expected variables map in response")
	}
	if vars["host"] != "updated.example.com" {
		t.Errorf("host = %v, want updated.example.com", vars["host"])
	}
}

// TestHandleUpdateEnvironment_PutNewEnv verifies that PUT /environments/{name}
// creates the environment when it does not yet exist (upsert semantics, HTTP 201).
func TestHandleUpdateEnvironment_PutNewEnv(t *testing.T) {
	mux, _ := setupEnvHandler(t)

	newVars := map[string]any{"BASE_URL": "https://api.dev.example.com"}
	input := map[string]any{"variables": newVars}
	body, _ := json.Marshal(input)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/environments/dev-new", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	// Should return 201 Created, not 404.
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d (upsert should create); body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var env envelope.Envelope
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if !env.OK {
		t.Fatalf("expected ok=true for upsert, got error: %+v", env.Error)
	}

	// Verify the returned environment has the correct name and variables.
	raw, _ := json.Marshal(env.Data)
	var envData map[string]any
	if err := json.Unmarshal(raw, &envData); err != nil {
		t.Fatalf("decoding env data: %v", err)
	}
	if envData["name"] != "dev-new" {
		t.Errorf("name = %v, want dev-new", envData["name"])
	}
	vars, ok := envData["variables"].(map[string]any)
	if !ok {
		t.Fatal("expected variables map in response")
	}
	if vars["BASE_URL"] != "https://api.dev.example.com" {
		t.Errorf("BASE_URL = %v, want https://api.dev.example.com", vars["BASE_URL"])
	}
}

// TestHandleUpdateEnvironment_PutExistingEnvReturns200 verifies that PUT to an
// existing environment returns 200 OK (not 201), preserving the update path.
func TestHandleUpdateEnvironment_PutExistingEnvReturns200(t *testing.T) {
	mux, _ := setupEnvHandler(t)

	input := map[string]any{"variables": map[string]any{"host": "newhost"}}
	body, _ := json.Marshal(input)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/environments/dev", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d for existing env update; body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}
