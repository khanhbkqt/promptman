package cli

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/khanhnguyen/promptman/pkg/envelope"
)

func TestStatus_JSON(t *testing.T) {
	statusData := map[string]any{
		"pid":        float64(12345),
		"port":       float64(48721),
		"projectDir": "/tmp/test-project",
		"startedAt":  "2026-03-10T10:00:00Z",
		"uptime":     "2h30m",
	}
	mockResp := envelope.Success(statusData)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/status" {
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

	env, err := client.Get("/status")
	if err != nil {
		t.Fatalf("GET /status failed: %v", err)
	}

	globals := &GlobalFlags{Format: FormatJSON}
	renderErr := renderStatus(root, globals, env)
	if renderErr != nil {
		t.Fatalf("renderStatus error: %v", renderErr)
	}

	var output envelope.Envelope
	if jsonErr := json.Unmarshal(buf.Bytes(), &output); jsonErr != nil {
		t.Fatalf("invalid JSON: %v\nGot: %s", jsonErr, buf.String())
	}
	if !output.OK {
		t.Error("expected ok=true")
	}
}

func TestStatus_Table(t *testing.T) {
	statusData := map[string]any{
		"pid":        float64(12345),
		"port":       float64(48721),
		"projectDir": "/tmp/test-project",
		"startedAt":  "2026-03-10T10:00:00Z",
		"uptime":     "2h30m",
		"activeEnv":  "dev",
	}
	mockResp := envelope.Success(statusData)

	root := NewRootCommand()
	var buf bytes.Buffer
	root.SetOut(&buf)

	globals := &GlobalFlags{Format: FormatTable}
	renderErr := renderStatus(root, globals, mockResp)
	if renderErr != nil {
		t.Fatalf("renderStatus error: %v", renderErr)
	}

	output := buf.String()
	expectedFields := []string{"PID", "Port", "Uptime", "Project Dir", "Active Env"}
	for _, field := range expectedFields {
		if !strings.Contains(output, field) {
			t.Errorf("expected %q in table output, got: %s", field, output)
		}
	}

	// Verify actual values.
	if !strings.Contains(output, "12345") {
		t.Error("expected PID 12345 in output")
	}
	if !strings.Contains(output, "48721") {
		t.Error("expected port 48721 in output")
	}
	if !strings.Contains(output, "dev") {
		t.Error("expected active env 'dev' in output")
	}
}

func TestStatus_Minimal(t *testing.T) {
	statusData := map[string]any{
		"pid":        float64(12345),
		"port":       float64(48721),
		"projectDir": "/tmp/test-project",
		"startedAt":  "2026-03-10T10:00:00Z",
		"uptime":     "2h30m",
	}
	mockResp := envelope.Success(statusData)

	root := NewRootCommand()
	var buf bytes.Buffer
	root.SetOut(&buf)

	globals := &GlobalFlags{Format: FormatMinimal}
	renderErr := renderStatus(root, globals, mockResp)
	if renderErr != nil {
		t.Fatalf("renderStatus error: %v", renderErr)
	}

	output := buf.String()
	if !strings.Contains(output, "running") {
		t.Errorf("expected 'running' in minimal output, got: %s", output)
	}
	if !strings.Contains(output, "pid=12345") {
		t.Errorf("expected 'pid=12345' in minimal output, got: %s", output)
	}
	if !strings.Contains(output, "port=48721") {
		t.Errorf("expected 'port=48721' in minimal output, got: %s", output)
	}
}

func TestStatus_DaemonNotRunning(t *testing.T) {
	root := NewRootCommand()
	var buf bytes.Buffer
	root.SetOut(&buf)

	globals := &GlobalFlags{Format: FormatJSON}
	err := renderDaemonNotRunning(root, globals)
	if err != nil {
		t.Fatalf("renderDaemonNotRunning error: %v", err)
	}

	var output envelope.Envelope
	if jsonErr := json.Unmarshal(buf.Bytes(), &output); jsonErr != nil {
		t.Fatalf("invalid JSON: %v\nGot: %s", jsonErr, buf.String())
	}
	if !output.OK {
		t.Error("expected ok=true for not-running response")
	}

	// Verify message is present.
	raw, _ := json.Marshal(output.Data)
	if !strings.Contains(string(raw), "not_running") {
		t.Errorf("expected 'not_running' status, got: %s", raw)
	}
}

func TestStatus_DaemonNotRunning_Minimal(t *testing.T) {
	root := NewRootCommand()
	var buf bytes.Buffer
	root.SetOut(&buf)

	globals := &GlobalFlags{Format: FormatMinimal}
	err := renderDaemonNotRunning(root, globals)
	if err != nil {
		t.Fatalf("renderDaemonNotRunning error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "not_running") {
		t.Errorf("expected 'not_running' in minimal output, got: %s", output)
	}
}

func TestFormatUptime(t *testing.T) {
	tests := []struct {
		name      string
		uptime    string
		startedAt string
		want      string
	}{
		{"with uptime", "2h30m", "", "2h30m"},
		{"empty uptime no start", "", "", "unknown"},
		{"invalid start", "", "not-a-date", "not-a-date"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatUptime(tt.uptime, tt.startedAt)
			if got != tt.want {
				t.Errorf("formatUptime(%q, %q) = %q, want %q", tt.uptime, tt.startedAt, got, tt.want)
			}
		})
	}
}

func TestFormatActiveEnv(t *testing.T) {
	tests := []struct {
		name string
		env  string
		want string
	}{
		{"with name", "dev", "dev"},
		{"empty", "", "(none)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatActiveEnv(tt.env)
			if got != tt.want {
				t.Errorf("formatActiveEnv(%q) = %q, want %q", tt.env, got, tt.want)
			}
		})
	}
}
