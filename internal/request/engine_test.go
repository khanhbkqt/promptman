package request

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
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
