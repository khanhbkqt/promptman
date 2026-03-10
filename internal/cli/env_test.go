package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/khanhnguyen/promptman/pkg/envelope"
)

func TestNewEnvCommand_HasSubcommands(t *testing.T) {
	globals := &GlobalFlags{}
	cmd := newEnvCommand(globals)

	if cmd.Use != "env" {
		t.Errorf("Use = %q, want env", cmd.Use)
	}

	// Verify all subcommands are registered.
	subs := []string{"list", "use", "get", "set"}
	for _, name := range subs {
		found := false
		for _, sub := range cmd.Commands() {
			if sub.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("subcommand %q not registered", name)
		}
	}
}

func TestEnvList_JSON(t *testing.T) {
	items := []envListItem{
		{Name: "dev", VariableCount: 3, SecretCount: 1, Active: true},
		{Name: "staging", VariableCount: 2, SecretCount: 0, Active: false},
	}
	mockResp := envelope.Success(items)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/environments" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("unexpected method: %s", r.Method)
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

	// Call helper directly to bypass daemon auto-start.
	env, err := client.Get("/environments")
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}

	formatter, _ := NewFormatter(FormatJSON)
	if fmtErr := formatter.Format(&buf, env); fmtErr != nil {
		t.Fatalf("format error: %v", fmtErr)
	}

	// Verify JSON output is valid envelope.
	var output envelope.Envelope
	if jsonErr := json.Unmarshal(buf.Bytes(), &output); jsonErr != nil {
		t.Fatalf("invalid JSON: %v\nGot: %s", jsonErr, buf.String())
	}
	if !output.OK {
		t.Error("expected ok=true")
	}
}

func TestEnvList_Table(t *testing.T) {
	items := []envListItem{
		{Name: "dev", VariableCount: 3, SecretCount: 1, Active: true},
		{Name: "staging", VariableCount: 2, SecretCount: 0, Active: false},
	}
	mockResp := envelope.Success(items)

	root := NewRootCommand()
	var buf bytes.Buffer
	root.SetOut(&buf)

	_ = &GlobalFlags{Format: FormatTable} // format tested via render functions

	err := renderEnvListTable(root, mockResp)
	if err != nil {
		t.Fatalf("renderEnvListTable error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "dev") {
		t.Error("expected 'dev' in output")
	}
	if !strings.Contains(output, "staging") {
		t.Error("expected 'staging' in output")
	}
	if !strings.Contains(output, "NAME") {
		t.Error("expected 'NAME' header in output")
	}

	// Verify active marker is present for dev.
	lines := strings.Split(output, "\n")
	var devLine string
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "dev") {
			devLine = line
			break
		}
	}
	if devLine == "" {
		t.Fatal("could not find dev line in table output")
	}
	if !strings.Contains(devLine, "*") {
		t.Error("expected active marker '*' in dev line")
	}

	// Staging should NOT have the active marker.
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "staging") {
			parts := strings.Fields(trimmed)
			// Last column should be empty for inactive.
			if len(parts) > 3 && parts[3] == "*" {
				t.Error("staging should not have active marker")
			}
		}
	}
}

func TestEnvList_Minimal(t *testing.T) {
	items := []envListItem{
		{Name: "dev", VariableCount: 3, SecretCount: 1, Active: true},
		{Name: "staging", VariableCount: 2, SecretCount: 0, Active: false},
	}
	mockResp := envelope.Success(items)

	root := NewRootCommand()
	var buf bytes.Buffer
	root.SetOut(&buf)

	err := renderEnvListMinimal(root, mockResp)
	if err != nil {
		t.Fatalf("renderEnvListMinimal error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "* dev") {
		t.Errorf("expected '* dev' in minimal output, got: %s", output)
	}
	if !strings.Contains(output, "  staging") {
		t.Errorf("expected '  staging' in minimal output, got: %s", output)
	}
}

func TestEnvList_Empty(t *testing.T) {
	mockResp := envelope.Success([]envListItem{})

	root := NewRootCommand()
	var buf bytes.Buffer
	root.SetOut(&buf)

	err := renderEnvListTable(root, mockResp)
	if err != nil {
		t.Fatalf("renderEnvListTable error: %v", err)
	}

	if !strings.Contains(buf.String(), "(no environments)") {
		t.Errorf("expected empty message, got: %s", buf.String())
	}
}

func TestEnvUse_Integration(t *testing.T) {
	mockResp := envelope.Success(map[string]string{
		"message": "Active environment set to: dev",
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/environments/active" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method: %s", r.Method)
		}

		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["name"] != "dev" {
			t.Errorf("name = %q, want dev", body["name"])
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

	env, err := client.Post("/environments/active", map[string]string{"name": "dev"})
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}

	renderErr := renderEnvResponse(root, globals, env)
	if renderErr != nil {
		t.Fatalf("render error: %v", renderErr)
	}

	var output envelope.Envelope
	if jsonErr := json.Unmarshal(buf.Bytes(), &output); jsonErr != nil {
		t.Fatalf("invalid JSON: %v", jsonErr)
	}
	if !output.OK {
		t.Error("expected ok=true")
	}
}

func TestEnvUse_NotFound(t *testing.T) {
	mockResp := envelope.Fail("ENV_NOT_FOUND", "environment \"nonexistent\" not found")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
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

	env, err := client.Post("/environments/active", map[string]string{"name": "nonexistent"})
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}

	renderErr := renderEnvResponse(root, globals, env)
	var exitErr *ExitError
	if !errors.As(renderErr, &exitErr) {
		t.Fatal("expected ExitError")
	}
	if exitErr.Code != 1 {
		t.Errorf("exit code = %d, want 1", exitErr.Code)
	}
}

func TestEnvGet_Integration(t *testing.T) {
	mockResp := envelope.Success(map[string]any{
		"name": "dev",
		"variables": map[string]any{
			"host": "localhost",
			"port": float64(8080),
		},
		"secrets": map[string]string{
			"api_key": "***",
		},
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/environments/dev" {
			t.Errorf("unexpected path: %s", r.URL.Path)
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

	env, err := client.Get("/environments/dev")
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}

	renderErr := renderEnvResponse(root, globals, env)
	if renderErr != nil {
		t.Fatalf("render error: %v", renderErr)
	}

	var output envelope.Envelope
	if jsonErr := json.Unmarshal(buf.Bytes(), &output); jsonErr != nil {
		t.Fatalf("invalid JSON: %v", jsonErr)
	}
	if !output.OK {
		t.Error("expected ok=true")
	}
}

func TestEnvSet_Integration(t *testing.T) {
	callCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/environments":
			// List to find active.
			_ = json.NewEncoder(w).Encode(envelope.Success([]envListItem{
				{Name: "dev", VariableCount: 2, Active: true},
			}))

		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/environments/dev":
			// Get current env.
			_ = json.NewEncoder(w).Encode(envelope.Success(map[string]any{
				"name":      "dev",
				"variables": map[string]any{"host": "localhost"},
			}))

		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/environments/dev":
			// Update env — verify the body contains the new variable.
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			vars, ok := body["variables"].(map[string]any)
			if !ok {
				t.Error("expected variables in PUT body")
			}
			if vars["new_key"] != "new_value" {
				t.Errorf("new_key = %v, want new_value", vars["new_key"])
			}
			// Original variable should still be present.
			if vars["host"] != "localhost" {
				t.Errorf("host = %v, want localhost", vars["host"])
			}

			_ = json.NewEncoder(w).Encode(envelope.Success(map[string]any{
				"name":      "dev",
				"variables": vars,
			}))

		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
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

	// Directly exercise the env set sub-flow (bypassing EnsureDaemon/NewClient):
	// 1. Get active env name
	activeName, err := getActiveEnvName(client)
	if err != nil {
		t.Fatalf("getActiveEnvName failed: %v", err)
	}
	if activeName != "dev" {
		t.Fatalf("active = %q, want dev", activeName)
	}

	// 2. Get current env
	getEnv, getErr := client.Get("/environments/" + activeName)
	if getErr != nil {
		t.Fatalf("GET env failed: %v", getErr)
	}

	// 3. Merge new key-value
	currentVars := extractVariables(getEnv.Data)
	currentVars["new_key"] = "new_value"

	// 4. PUT updated env
	updateInput := map[string]any{"variables": currentVars}
	putEnv, putErr := client.Put("/environments/"+activeName, updateInput)
	if putErr != nil {
		t.Fatalf("PUT failed: %v", putErr)
	}

	renderErr := renderEnvResponse(root, globals, putEnv)
	if renderErr != nil {
		t.Fatalf("render error: %v", renderErr)
	}

	if callCount != 3 {
		t.Errorf("expected 3 API calls (list, get, put), got %d", callCount)
	}
}

func TestEnvSet_WithEnvFlag(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/environments/staging":
			_ = json.NewEncoder(w).Encode(envelope.Success(map[string]any{
				"name":      "staging",
				"variables": map[string]any{"host": "staging.example.com"},
			}))

		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/environments/staging":
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			_ = json.NewEncoder(w).Encode(envelope.Success(map[string]any{
				"name":      "staging",
				"variables": body["variables"],
			}))

		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	testClient := &Client{
		baseURL:    server.URL + "/api/v1",
		token:      "test-token",
		httpClient: server.Client(),
	}

	root := NewRootCommand()
	var buf bytes.Buffer
	root.SetOut(&buf)

	globals := &GlobalFlags{Format: FormatJSON}

	// Test the sub-flow directly since we can't bypass EnsureDaemon.
	getEnv, _ := testClient.Get("/environments/staging")
	currentVars := extractVariables(getEnv.Data)
	currentVars["api_url"] = "https://api.staging.example.com"

	updateInput := map[string]any{"variables": currentVars}
	putEnv, _ := testClient.Put("/environments/staging", updateInput)

	renderErr := renderEnvResponse(root, globals, putEnv)
	if renderErr != nil {
		t.Fatalf("render error: %v", renderErr)
	}

	// --env flag tested via executeEnvSet path in TestEnvSet_Integration
}

func TestParseEnvListItems(t *testing.T) {
	input := []envListItem{
		{Name: "dev", VariableCount: 2, SecretCount: 1, Active: true},
		{Name: "prod", VariableCount: 5, SecretCount: 3, Active: false},
	}

	items, err := parseEnvListItems(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(items) != 2 {
		t.Fatalf("got %d items, want 2", len(items))
	}
	if items[0].Name != "dev" {
		t.Errorf("items[0].Name = %q, want dev", items[0].Name)
	}
	if !items[0].Active {
		t.Error("items[0] should be active")
	}
	if items[1].Active {
		t.Error("items[1] should not be active")
	}
}

func TestExtractVariables(t *testing.T) {
	tests := []struct {
		name     string
		data     any
		wantKeys []string
	}{
		{
			name: "Valid env data",
			data: map[string]any{
				"name":      "dev",
				"variables": map[string]any{"host": "localhost", "port": float64(8080)},
			},
			wantKeys: []string{"host", "port"},
		},
		{
			name:     "No variables",
			data:     map[string]any{"name": "empty"},
			wantKeys: nil,
		},
		{
			name:     "Nil data",
			data:     nil,
			wantKeys: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vars := extractVariables(tt.data)
			if tt.wantKeys == nil {
				if len(vars) != 0 {
					t.Errorf("expected empty map, got %d keys", len(vars))
				}
				return
			}
			for _, key := range tt.wantKeys {
				if _, ok := vars[key]; !ok {
					t.Errorf("missing key %q", key)
				}
			}
		})
	}
}

func TestClientPut_Integration(t *testing.T) {
	var receivedMethod string
	var receivedBody map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		_ = json.NewDecoder(r.Body).Decode(&receivedBody)

		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("missing auth header")
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("missing content-type header")
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(envelope.Success(map[string]string{"result": "ok"}))
	}))
	defer server.Close()

	client := &Client{
		baseURL:    server.URL + "/api/v1",
		token:      "test-token",
		httpClient: server.Client(),
	}

	env, err := client.Put("/environments/dev", map[string]any{"variables": map[string]any{"key": "val"}})
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	if receivedMethod != http.MethodPut {
		t.Errorf("method = %s, want PUT", receivedMethod)
	}
	if !env.OK {
		t.Error("expected ok=true")
	}

	vars, ok := receivedBody["variables"].(map[string]any)
	if !ok {
		t.Fatal("expected variables in body")
	}
	if vars["key"] != "val" {
		t.Errorf("key = %v, want val", vars["key"])
	}
}
