package request

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/khanhnguyen/promptman/internal/collection"
	"github.com/khanhnguyen/promptman/internal/environment"
	"github.com/khanhnguyen/promptman/pkg/envelope"
)

// --- Mock services ---

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

// --- Integration Tests ---

func TestEngine_Execute_SuccessfulGET(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Custom", "test-value")
		w.WriteHeader(200)
		fmt.Fprint(w, `{"message":"hello"}`)
	}))
	defer srv.Close()

	collSvc := &mockCollectionFinder{
		requests: map[string]*collection.ResolvedRequest{
			"api/get-hello": {
				URL:    srv.URL + "/hello",
				Method: "GET",
			},
		},
	}
	envSvc := &mockEnvironmentResolver{activeErr: fmt.Errorf("no active env")}
	engine := NewEngine(collSvc, envSvc)

	resp, err := engine.Execute(context.Background(), ExecuteInput{
		CollectionID: "api",
		RequestID:    "get-hello",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Status != 200 {
		t.Errorf("Status = %d, want 200", resp.Status)
	}
	if resp.Body != `{"message":"hello"}` {
		t.Errorf("Body = %q, want %q", resp.Body, `{"message":"hello"}`)
	}
	if resp.Headers["X-Custom"] != "test-value" {
		t.Errorf("X-Custom header = %q, want %q", resp.Headers["X-Custom"], "test-value")
	}
	if resp.Method != "GET" {
		t.Errorf("Method = %q, want GET", resp.Method)
	}
	if resp.RequestID != "get-hello" {
		t.Errorf("RequestID = %q, want %q", resp.RequestID, "get-hello")
	}
	if resp.Timing == nil {
		t.Fatal("Timing should not be nil")
	}
	if resp.Timing.Total < 0 {
		t.Errorf("Timing.Total = %d, want >= 0", resp.Timing.Total)
	}
}

func TestEngine_Execute_POSTWithJSONBody(t *testing.T) {
	var receivedBody string
	var receivedContentType string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedContentType = r.Header.Get("Content-Type")
		buf := make([]byte, 1024)
		n, _ := r.Body.Read(buf)
		receivedBody = string(buf[:n])
		w.WriteHeader(201)
		fmt.Fprint(w, `{"id":1}`)
	}))
	defer srv.Close()

	collSvc := &mockCollectionFinder{
		requests: map[string]*collection.ResolvedRequest{
			"api/create-user": {
				URL:    srv.URL + "/users",
				Method: "POST",
				Body: &collection.RequestBody{
					Type:    "json",
					Content: map[string]any{"name": "{{user_name}}", "email": "test@example.com"},
				},
			},
		},
	}
	envSvc := &mockEnvironmentResolver{
		activeEnv: "dev",
		envs: map[string]*environment.Environment{
			"dev": {
				Name:      "dev",
				Variables: map[string]any{"user_name": "Alice"},
			},
		},
	}
	engine := NewEngine(collSvc, envSvc)

	resp, err := engine.Execute(context.Background(), ExecuteInput{
		CollectionID: "api",
		RequestID:    "create-user",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Status != 201 {
		t.Errorf("Status = %d, want 201", resp.Status)
	}
	if receivedContentType != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", receivedContentType)
	}
	if !strings.Contains(receivedBody, "Alice") {
		t.Errorf("body should contain resolved name 'Alice', got: %s", receivedBody)
	}
}

func TestEngine_Execute_VariableResolution(t *testing.T) {
	var receivedPath string
	var receivedAuthHeader string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		receivedAuthHeader = r.Header.Get("Authorization")
		w.WriteHeader(200)
	}))
	defer srv.Close()

	collSvc := &mockCollectionFinder{
		requests: map[string]*collection.ResolvedRequest{
			"api/get-user": {
				URL:    srv.URL + "/users/{{user_id}}",
				Method: "GET",
				Headers: map[string]string{
					"X-Correlation-ID": "{{trace_id}}",
				},
				Auth: &collection.AuthConfig{
					Type:   "bearer",
					Bearer: &collection.BearerAuth{Token: "{{api_token}}"},
				},
			},
		},
	}
	envSvc := &mockEnvironmentResolver{
		activeEnv: "staging",
		envs: map[string]*environment.Environment{
			"staging": {
				Name:      "staging",
				Variables: map[string]any{"user_id": "42", "trace_id": "abc-123"},
				Secrets:   map[string]string{"api_token": "secret-token"},
			},
		},
	}
	engine := NewEngine(collSvc, envSvc)

	resp, err := engine.Execute(context.Background(), ExecuteInput{
		CollectionID: "api",
		RequestID:    "get-user",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Status != 200 {
		t.Errorf("Status = %d, want 200", resp.Status)
	}
	if receivedPath != "/users/42" {
		t.Errorf("path = %q, want /users/42", receivedPath)
	}
	if receivedAuthHeader != "Bearer secret-token" {
		t.Errorf("Authorization = %q, want 'Bearer secret-token'", receivedAuthHeader)
	}
}

func TestEngine_Execute_RuntimeOverrides(t *testing.T) {
	var receivedPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		w.WriteHeader(200)
	}))
	defer srv.Close()

	collSvc := &mockCollectionFinder{
		requests: map[string]*collection.ResolvedRequest{
			"api/get-user": {
				URL:    srv.URL + "/users/{{user_id}}",
				Method: "GET",
			},
		},
	}
	envSvc := &mockEnvironmentResolver{
		activeEnv: "dev",
		envs: map[string]*environment.Environment{
			"dev": {
				Name:      "dev",
				Variables: map[string]any{"user_id": "1"},
			},
		},
	}
	engine := NewEngine(collSvc, envSvc)

	// Runtime override should win over env variable.
	resp, err := engine.Execute(context.Background(), ExecuteInput{
		CollectionID: "api",
		RequestID:    "get-user",
		Variables:    map[string]any{"user_id": "99"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != 200 {
		t.Errorf("Status = %d, want 200", resp.Status)
	}
	if receivedPath != "/users/99" {
		t.Errorf("path = %q, want /users/99 (runtime override)", receivedPath)
	}
}

func TestEngine_Execute_Timeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(200)
	}))
	defer srv.Close()

	timeout := 50 // 50ms
	collSvc := &mockCollectionFinder{
		requests: map[string]*collection.ResolvedRequest{
			"api/slow": {
				URL:     srv.URL + "/slow",
				Method:  "GET",
				Timeout: &timeout,
			},
		},
	}
	envSvc := &mockEnvironmentResolver{activeErr: fmt.Errorf("no active env")}
	engine := NewEngine(collSvc, envSvc)

	_, err := engine.Execute(context.Background(), ExecuteInput{
		CollectionID: "api",
		RequestID:    "slow",
	})
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if !IsDomainError(err, envelope.CodeRequestTimeout) {
		t.Errorf("expected REQUEST_TIMEOUT error, got: %v", err)
	}
}

func TestEngine_Execute_ConnectionRefused(t *testing.T) {
	collSvc := &mockCollectionFinder{
		requests: map[string]*collection.ResolvedRequest{
			"api/dead": {
				URL:    "http://127.0.0.1:1", // port 1 should be refused
				Method: "GET",
			},
		},
	}
	envSvc := &mockEnvironmentResolver{activeErr: fmt.Errorf("no active env")}
	engine := NewEngine(collSvc, envSvc, WithDefaultTimeout(2*time.Second))

	_, err := engine.Execute(context.Background(), ExecuteInput{
		CollectionID: "api",
		RequestID:    "dead",
	})
	if err == nil {
		t.Fatal("expected connection error, got nil")
	}
	if !IsDomainError(err, envelope.CodeRequestFailed) {
		t.Errorf("expected REQUEST_FAILED error, got: %v", err)
	}
}

func TestEngine_Execute_ContextCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Block until client disconnects.
		<-r.Context().Done()
	}))
	defer srv.Close()

	collSvc := &mockCollectionFinder{
		requests: map[string]*collection.ResolvedRequest{
			"api/long": {
				URL:    srv.URL + "/long",
				Method: "GET",
			},
		},
	}
	envSvc := &mockEnvironmentResolver{activeErr: fmt.Errorf("no active env")}
	engine := NewEngine(collSvc, envSvc)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := engine.Execute(ctx, ExecuteInput{
		CollectionID: "api",
		RequestID:    "long",
	})
	if err == nil {
		t.Fatal("expected error from context cancellation")
	}
	// Should be classified as either timeout or failed.
	de, ok := err.(*DomainError)
	if !ok {
		t.Fatalf("expected DomainError, got %T: %v", err, err)
	}
	if de.Code != envelope.CodeRequestTimeout && de.Code != envelope.CodeRequestFailed {
		t.Errorf("expected REQUEST_TIMEOUT or REQUEST_FAILED, got %q", de.Code)
	}
}

func TestEngine_Execute_TLSSkipVerify(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		fmt.Fprint(w, "tls-ok")
	}))
	defer srv.Close()

	collSvc := &mockCollectionFinder{
		requests: map[string]*collection.ResolvedRequest{
			"api/tls": {
				URL:    srv.URL + "/secure",
				Method: "GET",
			},
		},
	}
	envSvc := &mockEnvironmentResolver{activeErr: fmt.Errorf("no active env")}

	// Without skip-verify, should fail (self-signed cert).
	engine := NewEngine(collSvc, envSvc)
	_, err := engine.Execute(context.Background(), ExecuteInput{
		CollectionID:  "api",
		RequestID:     "tls",
		SkipTLSVerify: false,
	})
	if err == nil {
		t.Fatal("expected TLS error without skip-verify")
	}

	// With skip-verify, should succeed.
	// Use a transport that supports TLS.
	engine2 := NewEngine(collSvc, envSvc, WithTransport(&http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: false,
		},
	}))
	resp, err := engine2.Execute(context.Background(), ExecuteInput{
		CollectionID:  "api",
		RequestID:     "tls",
		SkipTLSVerify: true,
	})
	if err != nil {
		t.Fatalf("expected success with skip-verify, got: %v", err)
	}
	if resp.Status != 200 {
		t.Errorf("Status = %d, want 200", resp.Status)
	}
	if resp.Body != "tls-ok" {
		t.Errorf("Body = %q, want 'tls-ok'", resp.Body)
	}
}

func TestEngine_Execute_TimingPopulated(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		fmt.Fprint(w, "ok")
	}))
	defer srv.Close()

	collSvc := &mockCollectionFinder{
		requests: map[string]*collection.ResolvedRequest{
			"api/timing": {
				URL:    srv.URL + "/timing",
				Method: "GET",
			},
		},
	}
	envSvc := &mockEnvironmentResolver{activeErr: fmt.Errorf("no active env")}
	engine := NewEngine(collSvc, envSvc)

	resp, err := engine.Execute(context.Background(), ExecuteInput{
		CollectionID: "api",
		RequestID:    "timing",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Timing == nil {
		t.Fatal("Timing should not be nil")
	}
	if resp.Timing.Total < 0 {
		t.Errorf("Timing.Total = %d, want >= 0", resp.Timing.Total)
	}
	// TTFB should be >= 0 (may be 0 for localhost).
	if resp.Timing.TTFB < 0 {
		t.Errorf("Timing.TTFB = %d, want >= 0", resp.Timing.TTFB)
	}
}

func TestEngine_Execute_CollectionNotFound(t *testing.T) {
	collSvc := &mockCollectionFinder{
		err: fmt.Errorf("collection not found"),
	}
	envSvc := &mockEnvironmentResolver{activeErr: fmt.Errorf("no active env")}
	engine := NewEngine(collSvc, envSvc)

	_, err := engine.Execute(context.Background(), ExecuteInput{
		CollectionID: "missing",
		RequestID:    "req",
	})
	if err == nil {
		t.Fatal("expected error for missing collection")
	}
	if !strings.Contains(err.Error(), "finding request") {
		t.Errorf("error should mention finding request, got: %v", err)
	}
}

func TestEngine_Execute_ExplicitEnvironment(t *testing.T) {
	var receivedPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		w.WriteHeader(200)
	}))
	defer srv.Close()

	collSvc := &mockCollectionFinder{
		requests: map[string]*collection.ResolvedRequest{
			"api/get-user": {
				URL:    srv.URL + "/users/{{user_id}}",
				Method: "GET",
			},
		},
	}
	envSvc := &mockEnvironmentResolver{
		activeEnv: "dev",
		envs: map[string]*environment.Environment{
			"dev":     {Name: "dev", Variables: map[string]any{"user_id": "1"}},
			"staging": {Name: "staging", Variables: map[string]any{"user_id": "100"}},
		},
	}
	engine := NewEngine(collSvc, envSvc)

	// Explicit env should override active.
	resp, err := engine.Execute(context.Background(), ExecuteInput{
		CollectionID: "api",
		RequestID:    "get-user",
		Environment:  "staging",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != 200 {
		t.Errorf("Status = %d, want 200", resp.Status)
	}
	if receivedPath != "/users/100" {
		t.Errorf("path = %q, want /users/100 (from staging)", receivedPath)
	}
}

func TestEngine_Execute_NoEnvironment(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()

	collSvc := &mockCollectionFinder{
		requests: map[string]*collection.ResolvedRequest{
			"api/simple": {
				URL:    srv.URL + "/simple",
				Method: "GET",
			},
		},
	}
	// No active env, no env specified — should still work.
	envSvc := &mockEnvironmentResolver{activeErr: fmt.Errorf("no active env")}
	engine := NewEngine(collSvc, envSvc)

	resp, err := engine.Execute(context.Background(), ExecuteInput{
		CollectionID: "api",
		RequestID:    "simple",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != 200 {
		t.Errorf("Status = %d, want 200", resp.Status)
	}
}

func TestEngine_WithDefaultTimeout(t *testing.T) {
	engine := NewEngine(
		&mockCollectionFinder{},
		&mockEnvironmentResolver{},
		WithDefaultTimeout(5*time.Second),
	)

	if engine.timeout != 5*time.Second {
		t.Errorf("timeout = %v, want 5s", engine.timeout)
	}
}

// --- Additional mock services for collection runner ---

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

// mockHistoryAppender is a goroutine-safe HistoryAppender for tests.
// The mu guard is required because Append is called from fire-and-forget
// goroutines in engine.go and may be invoked concurrently.
type mockHistoryAppender struct {
	mu      sync.Mutex
	entries []HistoryEntry
	ch      chan struct{} // optional signal channel
}

func newMockHistoryAppender(expectedCalls int) *mockHistoryAppender {
	return &mockHistoryAppender{
		ch: make(chan struct{}, expectedCalls),
	}
}

func (m *mockHistoryAppender) Append(entry HistoryEntry) {
	m.mu.Lock()
	m.entries = append(m.entries, entry)
	m.mu.Unlock()
	if m.ch != nil {
		m.ch <- struct{}{}
	}
}

func (m *mockHistoryAppender) waitFor(n int, timeout time.Duration) bool {
	deadline := time.After(timeout)
	for i := 0; i < n; i++ {
		select {
		case <-m.ch:
		case <-deadline:
			return false
		}
	}
	return true
}

// --- Collection Runner Tests ---

func TestEngine_ExecuteCollection_Sequential(t *testing.T) {
	var callOrder []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callOrder = append(callOrder, r.URL.Path)
		w.WriteHeader(200)
		fmt.Fprint(w, "ok")
	}))
	defer srv.Close()

	collSvc := &mockCollectionFinder{
		requests: map[string]*collection.ResolvedRequest{
			"api/health":    {URL: srv.URL + "/health", Method: "GET"},
			"api/get-users": {URL: srv.URL + "/users", Method: "GET"},
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
	engine := NewEngine(collSvc, envSvc, WithCollectionGetter(collGetter))

	results, err := engine.ExecuteCollection(context.Background(), CollectionRunOpts{
		CollectionID: "api",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("got %d results, want 2", len(results))
	}
	for i, r := range results {
		if r.Status != 200 {
			t.Errorf("result[%d].Status = %d, want 200", i, r.Status)
		}
		if r.Error != "" {
			t.Errorf("result[%d].Error = %q, want empty", i, r.Error)
		}
	}
	if len(callOrder) != 2 {
		t.Fatalf("server received %d calls, want 2", len(callOrder))
	}
}

func TestEngine_ExecuteCollection_StopOnError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()

	// First request will fail (bad URL), second should not execute.
	collSvc := &mockCollectionFinder{
		requests: map[string]*collection.ResolvedRequest{
			"api/bad":  {URL: "http://127.0.0.1:1/fail", Method: "GET"},
			"api/good": {URL: srv.URL + "/ok", Method: "GET"},
		},
	}
	collGetter := &mockCollectionGetter{
		collections: map[string]*collection.Collection{
			"api": {
				Name: "API",
				Requests: []collection.Request{
					{ID: "bad", Method: "GET", Path: "/fail"},
					{ID: "good", Method: "GET", Path: "/ok"},
				},
			},
		},
	}
	envSvc := &mockEnvironmentResolver{activeErr: fmt.Errorf("no active env")}
	engine := NewEngine(collSvc, envSvc,
		WithCollectionGetter(collGetter),
		WithDefaultTimeout(1*time.Second),
	)

	results, err := engine.ExecuteCollection(context.Background(), CollectionRunOpts{
		CollectionID: "api",
		StopOnError:  true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1 (should stop on first error)", len(results))
	}
	if results[0].Error == "" {
		t.Error("first result should have an error")
	}
}

func TestEngine_ExecuteCollection_ContinueOnError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		fmt.Fprint(w, "ok")
	}))
	defer srv.Close()

	collSvc := &mockCollectionFinder{
		requests: map[string]*collection.ResolvedRequest{
			"api/bad":  {URL: "http://127.0.0.1:1/fail", Method: "GET"},
			"api/good": {URL: srv.URL + "/ok", Method: "GET"},
		},
	}
	collGetter := &mockCollectionGetter{
		collections: map[string]*collection.Collection{
			"api": {
				Name: "API",
				Requests: []collection.Request{
					{ID: "bad", Method: "GET", Path: "/fail"},
					{ID: "good", Method: "GET", Path: "/ok"},
				},
			},
		},
	}
	envSvc := &mockEnvironmentResolver{activeErr: fmt.Errorf("no active env")}
	engine := NewEngine(collSvc, envSvc,
		WithCollectionGetter(collGetter),
		WithDefaultTimeout(1*time.Second),
	)

	results, err := engine.ExecuteCollection(context.Background(), CollectionRunOpts{
		CollectionID: "api",
		StopOnError:  false,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("got %d results, want 2 (should continue past error)", len(results))
	}
	if results[0].Error == "" {
		t.Error("first result should have an error")
	}
	if results[1].Status != 200 {
		t.Errorf("second result status = %d, want 200", results[1].Status)
	}
}

func TestEngine_ExecuteCollection_EmptyCollection(t *testing.T) {
	collSvc := &mockCollectionFinder{}
	collGetter := &mockCollectionGetter{
		collections: map[string]*collection.Collection{
			"empty": {Name: "Empty"},
		},
	}
	envSvc := &mockEnvironmentResolver{activeErr: fmt.Errorf("no active env")}
	engine := NewEngine(collSvc, envSvc, WithCollectionGetter(collGetter))

	results, err := engine.ExecuteCollection(context.Background(), CollectionRunOpts{
		CollectionID: "empty",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("got %d results, want 0 for empty collection", len(results))
	}
}

func TestEngine_ExecuteCollection_NestedFolders(t *testing.T) {
	var receivedPaths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPaths = append(receivedPaths, r.URL.Path)
		w.WriteHeader(200)
	}))
	defer srv.Close()

	collSvc := &mockCollectionFinder{
		requests: map[string]*collection.ResolvedRequest{
			"api/health":             {URL: srv.URL + "/health", Method: "GET"},
			"api/admin/list-admins":  {URL: srv.URL + "/admin/list", Method: "GET"},
			"api/admin/settings/get": {URL: srv.URL + "/admin/settings", Method: "GET"},
		},
	}
	collGetter := &mockCollectionGetter{
		collections: map[string]*collection.Collection{
			"api": {
				Name: "API",
				Requests: []collection.Request{
					{ID: "health", Method: "GET", Path: "/health"},
				},
				Folders: []collection.Folder{
					{
						ID:   "admin",
						Name: "Admin",
						Requests: []collection.Request{
							{ID: "list-admins", Method: "GET", Path: "/admin/list"},
						},
						Folders: []collection.Folder{
							{
								ID:   "settings",
								Name: "Settings",
								Requests: []collection.Request{
									{ID: "get", Method: "GET", Path: "/admin/settings"},
								},
							},
						},
					},
				},
			},
		},
	}
	envSvc := &mockEnvironmentResolver{activeErr: fmt.Errorf("no active env")}
	engine := NewEngine(collSvc, envSvc, WithCollectionGetter(collGetter))

	results, err := engine.ExecuteCollection(context.Background(), CollectionRunOpts{
		CollectionID: "api",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("got %d results, want 3", len(results))
	}
	for _, r := range results {
		if r.Status != 200 {
			t.Errorf("result %q status = %d, want 200", r.RequestID, r.Status)
		}
	}
}

func TestEngine_ExecuteCollection_HistoryAppenderCalled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		fmt.Fprint(w, "ok")
	}))
	defer srv.Close()

	collSvc := &mockCollectionFinder{
		requests: map[string]*collection.ResolvedRequest{
			"api/r1": {URL: srv.URL + "/r1", Method: "GET"},
			"api/r2": {URL: srv.URL + "/r2", Method: "GET"},
		},
	}
	collGetter := &mockCollectionGetter{
		collections: map[string]*collection.Collection{
			"api": {
				Name: "API",
				Requests: []collection.Request{
					{ID: "r1", Method: "GET", Path: "/r1"},
					{ID: "r2", Method: "GET", Path: "/r2"},
				},
			},
		},
	}
	envSvc := &mockEnvironmentResolver{activeErr: fmt.Errorf("no active env")}
	history := newMockHistoryAppender(2)
	engine := NewEngine(collSvc, envSvc,
		WithCollectionGetter(collGetter),
		WithHistoryAppender(history),
	)

	results, err := engine.ExecuteCollection(context.Background(), CollectionRunOpts{
		CollectionID: "api",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("got %d results, want 2", len(results))
	}

	// Wait for async history appends (fire-and-forget goroutines).
	if !history.waitFor(2, 2*time.Second) {
		t.Fatalf("history appender received %d entries, want 2 (timed out)", len(history.entries))
	}
	if len(history.entries) != 2 {
		t.Errorf("history has %d entries, want 2", len(history.entries))
	}
}

func TestEngine_ExecuteCollection_NotFoundError(t *testing.T) {
	collSvc := &mockCollectionFinder{}
	collGetter := &mockCollectionGetter{
		err: fmt.Errorf("collection not found"),
	}
	envSvc := &mockEnvironmentResolver{activeErr: fmt.Errorf("no active env")}
	engine := NewEngine(collSvc, envSvc, WithCollectionGetter(collGetter))

	_, err := engine.ExecuteCollection(context.Background(), CollectionRunOpts{
		CollectionID: "missing",
	})
	if err == nil {
		t.Fatal("expected error for missing collection")
	}
	if !strings.Contains(err.Error(), "loading collection") {
		t.Errorf("error should mention loading collection, got: %v", err)
	}
}

// --- collectRequestPaths tests ---

func TestCollectRequestPaths(t *testing.T) {
	c := &collection.Collection{
		Name: "Test",
		Requests: []collection.Request{
			{ID: "root-req"},
		},
		Folders: []collection.Folder{
			{
				ID:   "auth",
				Name: "Auth",
				Requests: []collection.Request{
					{ID: "login"},
					{ID: "logout"},
				},
				Folders: []collection.Folder{
					{
						ID:   "admin",
						Name: "Admin",
						Requests: []collection.Request{
							{ID: "create-user"},
						},
					},
				},
			},
		},
	}

	paths := collectRequestPaths(c)

	expected := []string{
		"root-req",
		"auth/login",
		"auth/logout",
		"auth/admin/create-user",
	}

	if len(paths) != len(expected) {
		t.Fatalf("got %d paths, want %d: %v", len(paths), len(expected), paths)
	}
	for i, p := range paths {
		if p != expected[i] {
			t.Errorf("path[%d] = %q, want %q", i, p, expected[i])
		}
	}
}
