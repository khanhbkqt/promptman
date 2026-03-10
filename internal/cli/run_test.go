package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/khanhnguyen/promptman/pkg/envelope"
)

func TestParsePath(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		wantColl    string
		wantReq     string
		wantErr     bool
		errContains string
	}{
		{
			name:     "simple path",
			path:     "users/health",
			wantColl: "users",
			wantReq:  "health",
		},
		{
			name:     "nested path",
			path:     "users/admin/list-admins",
			wantColl: "users",
			wantReq:  "admin/list-admins",
		},
		{
			name:     "deeply nested path",
			path:     "api/v1/auth/login",
			wantColl: "api",
			wantReq:  "v1/auth/login",
		},
		{
			name:        "no slash",
			path:        "users",
			wantErr:     true,
			errContains: "must be <collection>/<request>",
		},
		{
			name:        "empty collection",
			path:        "/health",
			wantErr:     true,
			errContains: "collection name cannot be empty",
		},
		{
			name:        "empty request",
			path:        "users/",
			wantErr:     true,
			errContains: "request name cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			coll, req, err := parsePath(tt.path)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("error %q should contain %q", err.Error(), tt.errContains)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if coll != tt.wantColl {
				t.Errorf("collection = %q, want %q", coll, tt.wantColl)
			}
			if req != tt.wantReq {
				t.Errorf("request = %q, want %q", req, tt.wantReq)
			}
		})
	}
}

func TestExtractExitCode(t *testing.T) {
	tests := []struct {
		name     string
		data     any
		wantCode int
	}{
		{
			name:     "nil data",
			data:     nil,
			wantCode: 0,
		},
		{
			name: "2xx status",
			data: map[string]any{
				"status": float64(200),
				"method": "GET",
			},
			wantCode: 0,
		},
		{
			name: "201 created",
			data: map[string]any{
				"status": float64(201),
			},
			wantCode: 0,
		},
		{
			name: "404 not found",
			data: map[string]any{
				"status": float64(404),
			},
			wantCode: 404,
		},
		{
			name: "500 server error",
			data: map[string]any{
				"status": float64(500),
			},
			wantCode: 500,
		},
		{
			name: "collection all 2xx",
			data: []map[string]any{
				{"status": float64(200)},
				{"status": float64(201)},
			},
			wantCode: 0,
		},
		{
			name: "collection with failure",
			data: []map[string]any{
				{"status": float64(200)},
				{"status": float64(503)},
			},
			wantCode: 503,
		},
		{
			name:     "no status field",
			data:     map[string]any{"method": "GET"},
			wantCode: 0,
		},
		{
			name:     "string data",
			data:     "plain string",
			wantCode: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractExitCode(tt.data)
			if got != tt.wantCode {
				t.Errorf("extractExitCode() = %d, want %d", got, tt.wantCode)
			}
		})
	}
}

func TestStatusToExitCode(t *testing.T) {
	tests := []struct {
		name     string
		m        map[string]any
		wantCode int
	}{
		{"200 OK", map[string]any{"status": float64(200)}, 0},
		{"204 No Content", map[string]any{"status": float64(204)}, 0},
		{"299 boundary", map[string]any{"status": float64(299)}, 0},
		{"300 redirect", map[string]any{"status": float64(300)}, 300},
		{"400 bad request", map[string]any{"status": float64(400)}, 400},
		{"no status", map[string]any{"method": "GET"}, 0},
		{"non-numeric status", map[string]any{"status": "200"}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := statusToExitCode(tt.m)
			if got != tt.wantCode {
				t.Errorf("statusToExitCode() = %d, want %d", got, tt.wantCode)
			}
		})
	}
}

func TestExitError_Error(t *testing.T) {
	e := &ExitError{Code: 42}
	want := "exit status 42"
	if got := e.Error(); got != want {
		t.Errorf("ExitError.Error() = %q, want %q", got, want)
	}
}

func TestExitError_IsDetectedByErrorsAs(t *testing.T) {
	err := error(&ExitError{Code: 1})
	var exitErr *ExitError
	if !errors.As(err, &exitErr) {
		t.Fatal("errors.As should detect ExitError")
	}
	if exitErr.Code != 1 {
		t.Errorf("exit code = %d, want 1", exitErr.Code)
	}
}

func TestRenderAndExit_SuccessJSON(t *testing.T) {
	root := NewRootCommand()
	var buf bytes.Buffer
	root.SetOut(&buf)

	globals := &GlobalFlags{Format: FormatJSON}
	env := envelope.Success(map[string]any{
		"status": float64(200),
		"method": "GET",
		"url":    "http://localhost/health",
		"body":   "OK",
	})

	cmd := root
	cmd.SetOut(&buf)

	err := renderAndExit(cmd, globals, env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have written JSON output.
	if buf.Len() == 0 {
		t.Fatal("expected JSON output, got empty")
	}

	// Parse output as envelope.
	var output envelope.Envelope
	if err := json.Unmarshal(buf.Bytes(), &output); err != nil {
		t.Fatalf("output is not valid JSON: %v\nGot: %s", err, buf.String())
	}
	if !output.OK {
		t.Error("expected ok=true in output")
	}
}

func TestRenderAndExit_ErrorEnvelope(t *testing.T) {
	root := NewRootCommand()
	var buf bytes.Buffer
	root.SetOut(&buf)

	globals := &GlobalFlags{Format: FormatJSON}
	env := envelope.Fail("NOT_FOUND", "request not found")

	err := renderAndExit(root, globals, env)
	var exitErr *ExitError
	if !errors.As(err, &exitErr) {
		t.Fatal("expected ExitError")
	}
	if exitErr.Code != 1 {
		t.Errorf("exit code = %d, want 1", exitErr.Code)
	}
}

func TestRenderAndExit_Non2xxStatus(t *testing.T) {
	root := NewRootCommand()
	var buf bytes.Buffer
	root.SetOut(&buf)

	globals := &GlobalFlags{Format: FormatJSON}
	env := envelope.Success(map[string]any{
		"status": float64(404),
		"method": "GET",
		"url":    "http://localhost/missing",
	})

	err := renderAndExit(root, globals, env)
	var exitErr *ExitError
	if !errors.As(err, &exitErr) {
		t.Fatal("expected ExitError for non-2xx")
	}
	if exitErr.Code != 404 {
		t.Errorf("exit code = %d, want 404", exitErr.Code)
	}
}

func TestWriteErrorEnvelope(t *testing.T) {
	root := NewRootCommand()
	var buf bytes.Buffer
	root.SetOut(&buf)

	globals := &GlobalFlags{Format: FormatJSON}
	err := writeErrorEnvelope(root, globals, "TEST_ERROR", "something went wrong")

	var exitErr *ExitError
	if !errors.As(err, &exitErr) {
		t.Fatal("expected ExitError")
	}

	// Check that error envelope was written.
	var env envelope.Envelope
	if jsonErr := json.Unmarshal(buf.Bytes(), &env); jsonErr != nil {
		t.Fatalf("output is not valid JSON: %v", jsonErr)
	}
	if env.OK {
		t.Error("expected ok=false")
	}
	if env.Error.Code != "TEST_ERROR" {
		t.Errorf("error code = %q, want TEST_ERROR", env.Error.Code)
	}
}

func TestNewRunCommand_Help(t *testing.T) {
	globals := &GlobalFlags{}
	cmd := newRunCommand(globals)

	if cmd.Use != "run [collection/request]" {
		t.Errorf("unexpected Use: %s", cmd.Use)
	}
	if cmd.Short == "" {
		t.Error("Short description should not be empty")
	}

	// Check flags are registered.
	flags := []string{"env", "timeout", "insecure", "collection", "stop-on-error"}
	for _, name := range flags {
		if cmd.Flags().Lookup(name) == nil {
			t.Errorf("flag --%s not registered", name)
		}
	}
}

func TestRunCommand_MutualExclusion(t *testing.T) {
	root := NewRootCommand()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)

	// Both positional arg and --collection should error.
	root.SetArgs([]string{"run", "--collection", "users", "users/health"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for mutual exclusion")
	}
}

func TestRunCommand_MissingArgs(t *testing.T) {
	root := NewRootCommand()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)

	// Neither positional arg nor --collection should error.
	root.SetArgs([]string{"run"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for missing arguments")
	}
}

// TestRunSingle_Integration tests the full single request flow using a mock HTTP server.
func TestRunSingle_Integration(t *testing.T) {
	// Create a mock daemon server that responds to POST /api/v1/run.
	mockResp := envelope.Success(map[string]any{
		"requestId": "health",
		"method":    "GET",
		"url":       "http://localhost:8080/health",
		"status":    float64(200),
		"body":      `{"status":"ok"}`,
		"headers":   map[string]any{"Content-Type": "application/json"},
		"timing": map[string]any{
			"dns": float64(1), "connect": float64(2), "tls": float64(0),
			"ttfb": float64(15), "transfer": float64(3), "total": float64(21),
		},
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the request.
		if r.URL.Path != "/api/v1/run" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method: %s", r.Method)
		}

		// Verify auth header.
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("missing or wrong auth header: %s", r.Header.Get("Authorization"))
		}

		// Verify body.
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decoding request body: %v", err)
		}
		if body["collection"] != "users" {
			t.Errorf("collection = %v, want users", body["collection"])
		}
		if body["requestId"] != "health" {
			t.Errorf("requestId = %v, want health", body["requestId"])
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(mockResp)
	}))
	defer server.Close()

	// Create a client that points to the mock server.
	client := &Client{
		baseURL:    server.URL + "/api/v1",
		token:      "test-token",
		httpClient: server.Client(),
	}

	root := NewRootCommand()
	var buf bytes.Buffer
	root.SetOut(&buf)

	globals := &GlobalFlags{Format: FormatJSON}
	rf := &runFlags{}

	err := runSingle(root, []string{"users/health"}, globals, rf, client)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify JSON output.
	var output envelope.Envelope
	if jsonErr := json.Unmarshal(buf.Bytes(), &output); jsonErr != nil {
		t.Fatalf("invalid JSON output: %v\nGot: %s", jsonErr, buf.String())
	}
	if !output.OK {
		t.Error("expected ok=true")
	}
}

// TestRunCollection_Integration tests collection run with a mock server.
func TestRunCollection_Integration(t *testing.T) {
	mockResp := envelope.Success([]map[string]any{
		{
			"requestId": "health",
			"method":    "GET",
			"url":       "http://localhost/health",
			"status":    float64(200),
			"body":      "OK",
		},
		{
			"requestId": "list",
			"method":    "GET",
			"url":       "http://localhost/list",
			"status":    float64(200),
			"body":      "[]",
		},
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/run/collection" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decoding body: %v", err)
		}
		if body["collection"] != "users" {
			t.Errorf("collection = %v, want users", body["collection"])
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(mockResp)
	}))
	defer server.Close()

	client := &Client{
		baseURL:    server.URL + "/api/v1",
		token:      "test-token",
		httpClient: server.Client(),
	}

	root := NewRootCommand()
	var buf bytes.Buffer
	root.SetOut(&buf)

	globals := &GlobalFlags{Format: FormatJSON}
	rf := &runFlags{
		collection: "users",
	}

	err := runCollection(root, globals, rf, client)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var output envelope.Envelope
	if jsonErr := json.Unmarshal(buf.Bytes(), &output); jsonErr != nil {
		t.Fatalf("invalid JSON output: %v", jsonErr)
	}
	if !output.OK {
		t.Error("expected ok=true")
	}
}

// TestRunSingle_WithEnvFlag tests that the --env flag is passed to the daemon.
func TestRunSingle_WithEnvFlag(t *testing.T) {
	var receivedBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&receivedBody)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(envelope.Success(map[string]any{"status": float64(200)}))
	}))
	defer server.Close()

	client := &Client{
		baseURL:    server.URL + "/api/v1",
		token:      "tok",
		httpClient: server.Client(),
	}

	root := NewRootCommand()
	var buf bytes.Buffer
	root.SetOut(&buf)

	globals := &GlobalFlags{Format: FormatJSON}
	rf := &runFlags{env: "staging"}

	_ = runSingle(root, []string{"api/endpoint"}, globals, rf, client)

	if receivedBody["env"] != "staging" {
		t.Errorf("env = %v, want staging", receivedBody["env"])
	}
}

// TestRunSingle_InsecureFlag tests that --insecure flag is passed.
func TestRunSingle_InsecureFlag(t *testing.T) {
	var receivedBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&receivedBody)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(envelope.Success(map[string]any{"status": float64(200)}))
	}))
	defer server.Close()

	client := &Client{
		baseURL:    server.URL + "/api/v1",
		token:      "tok",
		httpClient: server.Client(),
	}

	root := NewRootCommand()
	var buf bytes.Buffer
	root.SetOut(&buf)

	globals := &GlobalFlags{Format: FormatJSON}
	rf := &runFlags{insecure: true}

	_ = runSingle(root, []string{"api/test"}, globals, rf, client)

	if receivedBody["skipTlsVerify"] != true {
		t.Errorf("skipTlsVerify = %v, want true", receivedBody["skipTlsVerify"])
	}
}

// TestRunCollection_StopOnError tests --stop-on-error flag.
func TestRunCollection_StopOnError(t *testing.T) {
	var receivedBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&receivedBody)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(envelope.Success([]map[string]any{{"status": float64(200)}}))
	}))
	defer server.Close()

	client := &Client{
		baseURL:    server.URL + "/api/v1",
		token:      "tok",
		httpClient: server.Client(),
	}

	root := NewRootCommand()
	var buf bytes.Buffer
	root.SetOut(&buf)

	globals := &GlobalFlags{Format: FormatJSON}
	rf := &runFlags{collection: "api", stopOnError: true}

	_ = runCollection(root, globals, rf, client)

	if receivedBody["stopOnError"] != true {
		t.Errorf("stopOnError = %v, want true", receivedBody["stopOnError"])
	}
}

// TestRunSingle_DaemonError tests error handling when daemon returns an error.
func TestRunSingle_DaemonError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(envelope.Fail("REQUEST_NOT_FOUND", "request 'missing' not found"))
	}))
	defer server.Close()

	client := &Client{
		baseURL:    server.URL + "/api/v1",
		token:      "tok",
		httpClient: server.Client(),
	}

	root := NewRootCommand()
	var buf bytes.Buffer
	root.SetOut(&buf)

	globals := &GlobalFlags{Format: FormatJSON}
	rf := &runFlags{}

	err := runSingle(root, []string{"users/missing"}, globals, rf, client)

	var exitErr *ExitError
	if !errors.As(err, &exitErr) {
		t.Fatal("expected ExitError for daemon error response")
	}
	if exitErr.Code != 1 {
		t.Errorf("exit code = %d, want 1", exitErr.Code)
	}
}

// contains is a helper to check string containment.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchSubstring(s, substr)
}

func searchSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
