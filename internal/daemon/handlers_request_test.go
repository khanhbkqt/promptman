package daemon

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/khanhnguyen/promptman/internal/collection"
	"github.com/khanhnguyen/promptman/internal/environment"
	"github.com/khanhnguyen/promptman/internal/request"
	"github.com/khanhnguyen/promptman/pkg/envelope"
)

// --- Mock services for handler tests ---

type mockCollectionFinder struct {
	requests map[string]*collection.ResolvedRequest
	err      error
}

func (m *mockCollectionFinder) FindRequest(collID, reqPath string) (*collection.ResolvedRequest, error) {
	if m.err != nil {
		return nil, m.err
	}
	key := collID + "/" + reqPath
	req, ok := m.requests[key]
	if !ok {
		return nil, fmt.Errorf("request %q not found in collection %q", reqPath, collID)
	}
	return req, nil
}

type mockEnvironmentResolver struct {
	envs      map[string]*environment.Environment
	activeEnv string
	activeErr error
}

func (m *mockEnvironmentResolver) GetRaw(name string) (*environment.Environment, error) {
	env, ok := m.envs[name]
	if !ok {
		return nil, fmt.Errorf("environment %q not found", name)
	}
	return env, nil
}

func (m *mockEnvironmentResolver) GetActive() (string, error) {
	if m.activeErr != nil {
		return "", m.activeErr
	}
	return m.activeEnv, nil
}

type mockCollectionGetter struct {
	collections map[string]*collection.Collection
	err         error
}

func (m *mockCollectionGetter) Get(id string) (*collection.Collection, error) {
	if m.err != nil {
		return nil, m.err
	}
	c, ok := m.collections[id]
	if !ok {
		return nil, fmt.Errorf("collection %q not found", id)
	}
	return c, nil
}

// --- Helper ---

func setupTestEngine(targetURL string) *request.Engine {
	collSvc := &mockCollectionFinder{
		requests: map[string]*collection.ResolvedRequest{
			"api/health":    {URL: targetURL + "/health", Method: "GET"},
			"api/get-users": {URL: targetURL + "/users", Method: "GET"},
		},
	}
	collGetter := &mockCollectionGetter{
		collections: map[string]*collection.Collection{
			"api": {
				Name: "API",
				Requests: []collection.Request{
					{ID: "health", Method: "GET", Path: "/health"},
					{ID: "get-users", Method: "GET", Path: "/users"},
				},
			},
		},
	}
	envSvc := &mockEnvironmentResolver{activeErr: fmt.Errorf("no active env")}
	return request.NewEngine(collSvc, envSvc,
		request.WithCollectionGetter(collGetter),
		request.WithDefaultTimeout(5*time.Second),
	)
}

func callHandler(handler http.HandlerFunc, body any) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	_ = json.NewEncoder(&buf).Encode(body)
	req := httptest.NewRequest("POST", "/api/v1/run", &buf)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler(w, req)
	return w
}

func decodeEnvelope(w *httptest.ResponseRecorder) *envelope.Envelope {
	var env envelope.Envelope
	_ = json.NewDecoder(w.Body).Decode(&env)
	return &env
}

// --- Tests ---

func TestHandleRunSingle_Success(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		fmt.Fprint(w, `{"status":"ok"}`)
	}))
	defer target.Close()

	engine := setupTestEngine(target.URL)
	reg := NewRequestRegistrar(engine)

	body := map[string]any{
		"collection": "api",
		"requestId":  "health",
	}
	w := callHandler(reg.handleRunSingle(), body)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	env := decodeEnvelope(w)
	if !env.OK {
		t.Errorf("envelope.OK = false, want true")
	}
	if env.Data == nil {
		t.Fatal("envelope.Data should not be nil")
	}
}

func TestHandleRunSingle_MissingCollection(t *testing.T) {
	engine := setupTestEngine("http://localhost")
	reg := NewRequestRegistrar(engine)

	body := map[string]any{
		"requestId": "health",
	}
	w := callHandler(reg.handleRunSingle(), body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
	env := decodeEnvelope(w)
	if env.OK {
		t.Error("envelope.OK should be false")
	}
	if env.Error == nil || env.Error.Code != envelope.CodeInvalidInput {
		t.Errorf("expected INVALID_INPUT error code, got: %v", env.Error)
	}
}

func TestHandleRunSingle_MissingRequestID(t *testing.T) {
	engine := setupTestEngine("http://localhost")
	reg := NewRequestRegistrar(engine)

	body := map[string]any{
		"collection": "api",
	}
	w := callHandler(reg.handleRunSingle(), body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
	env := decodeEnvelope(w)
	if env.Error == nil || env.Error.Code != envelope.CodeInvalidInput {
		t.Errorf("expected INVALID_INPUT, got: %v", env.Error)
	}
}

func TestHandleRunSingle_InvalidJSON(t *testing.T) {
	engine := setupTestEngine("http://localhost")
	reg := NewRequestRegistrar(engine)

	req := httptest.NewRequest("POST", "/api/v1/run", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	reg.handleRunSingle()(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
	env := decodeEnvelope(w)
	if env.Error == nil || env.Error.Code != envelope.CodeInvalidInput {
		t.Errorf("expected INVALID_INPUT, got: %v", env.Error)
	}
}

func TestHandleRunCollection_Success(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		fmt.Fprint(w, "ok")
	}))
	defer target.Close()

	engine := setupTestEngine(target.URL)
	reg := NewRequestRegistrar(engine)

	body := map[string]any{
		"collection": "api",
	}
	var buf bytes.Buffer
	_ = json.NewEncoder(&buf).Encode(body)
	req := httptest.NewRequest("POST", "/api/v1/run/collection", &buf)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	reg.handleRunCollection()(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	env := decodeEnvelope(w)
	if !env.OK {
		t.Error("envelope.OK should be true")
	}
}

func TestHandleRunCollection_MissingCollection(t *testing.T) {
	engine := setupTestEngine("http://localhost")
	reg := NewRequestRegistrar(engine)

	body := map[string]any{}
	var buf bytes.Buffer
	_ = json.NewEncoder(&buf).Encode(body)
	req := httptest.NewRequest("POST", "/api/v1/run/collection", &buf)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	reg.handleRunCollection()(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
	env := decodeEnvelope(w)
	if env.Error == nil || env.Error.Code != envelope.CodeInvalidInput {
		t.Errorf("expected INVALID_INPUT, got: %v", env.Error)
	}
}

func TestHandleRunCollection_InvalidJSON(t *testing.T) {
	engine := setupTestEngine("http://localhost")
	reg := NewRequestRegistrar(engine)

	req := httptest.NewRequest("POST", "/api/v1/run/collection", bytes.NewBufferString("{broken"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	reg.handleRunCollection()(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestRequestRegistrar_RegisterRoutes(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer target.Close()

	engine := setupTestEngine(target.URL)
	reg := NewRequestRegistrar(engine)

	mux := http.NewServeMux()
	reg.RegisterRoutes(mux, "/api/v1/")

	// Verify routes are registered by making requests to the mux.
	body := map[string]any{"collection": "api", "requestId": "health"}
	var buf bytes.Buffer
	_ = json.NewEncoder(&buf).Encode(body)

	req := httptest.NewRequest("POST", "/api/v1/run", &buf)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	// Should not be 404 (route exists).
	if w.Code == http.StatusNotFound {
		t.Error("POST /api/v1/run should be registered (got 404)")
	}
}
