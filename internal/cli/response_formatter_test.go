package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/khanhnguyen/promptman/pkg/envelope"
)

func TestFormatResponseTable_SingleResponse(t *testing.T) {
	resp := &ResponseDisplay{
		RequestID: "health",
		Method:    "GET",
		URL:       "http://localhost:8080/health",
		Status:    200,
		Headers:   map[string]string{"Content-Type": "application/json"},
		Body:      `{"status":"ok"}`,
		Timing: &TimingDisplay{
			DNS: 1, Connect: 2, TLS: 0, TTFB: 15, Transfer: 3, Total: 21,
		},
	}

	var buf bytes.Buffer
	if err := FormatResponseTable(&buf, resp); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()

	// Check status line.
	if !strings.Contains(output, "✓ 200") {
		t.Errorf("should contain '✓ 200': %s", output)
	}
	if !strings.Contains(output, "GET") {
		t.Errorf("should contain method: %s", output)
	}
	if !strings.Contains(output, "http://localhost:8080/health") {
		t.Errorf("should contain URL: %s", output)
	}

	// Check timing.
	if !strings.Contains(output, "DNS") {
		t.Errorf("should contain timing header: %s", output)
	}
	if !strings.Contains(output, "21ms") {
		t.Errorf("should contain total timing: %s", output)
	}

	// TLS should be skipped when 0.
	if strings.Contains(output, "TLS") {
		t.Errorf("should not contain TLS when 0ms: %s", output)
	}

	// Check headers.
	if !strings.Contains(output, "Content-Type") {
		t.Errorf("should contain header name: %s", output)
	}

	// Check body.
	if !strings.Contains(output, `{"status":"ok"}`) {
		t.Errorf("should contain body: %s", output)
	}
}

func TestFormatResponseTable_WithTLS(t *testing.T) {
	resp := &ResponseDisplay{
		Method: "POST",
		URL:    "https://secure.example.com/api",
		Status: 201,
		Body:   "created",
		Timing: &TimingDisplay{
			DNS: 5, Connect: 10, TLS: 25, TTFB: 40, Transfer: 5, Total: 85,
		},
	}

	var buf bytes.Buffer
	if err := FormatResponseTable(&buf, resp); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(buf.String(), "TLS") {
		t.Error("should contain TLS when > 0")
	}
	if !strings.Contains(buf.String(), "25ms") {
		t.Error("should show TLS timing")
	}
}

func TestFormatResponseTable_ErrorField(t *testing.T) {
	resp := &ResponseDisplay{
		Method: "GET",
		URL:    "http://localhost/timeout",
		Status: 0,
		Error:  "connection timeout",
	}

	var buf bytes.Buffer
	if err := FormatResponseTable(&buf, resp); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(buf.String(), "connection timeout") {
		t.Errorf("should contain error: %s", buf.String())
	}
}

func TestFormatResponseMinimal_Basic(t *testing.T) {
	resp := &ResponseDisplay{
		Method: "GET",
		URL:    "http://localhost/health",
		Status: 200,
		Body:   `{"ok":true}`,
		Timing: &TimingDisplay{Total: 42},
	}

	var buf bytes.Buffer
	if err := FormatResponseMinimal(&buf, resp); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "200 GET http://localhost/health (42ms)") {
		t.Errorf("unexpected format: %q", output)
	}
	if !strings.Contains(output, `{"ok":true}`) {
		t.Errorf("should contain body: %s", output)
	}
}

func TestFormatResponseMinimal_NoTiming(t *testing.T) {
	resp := &ResponseDisplay{
		Method: "POST",
		URL:    "http://example.com/api",
		Status: 500,
		Body:   "Internal Server Error",
	}

	var buf bytes.Buffer
	if err := FormatResponseMinimal(&buf, resp); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if strings.Contains(output, "ms)") {
		t.Errorf("should not contain timing when nil: %s", output)
	}
	if !strings.HasPrefix(output, "500 POST") {
		t.Errorf("should start with status: %s", output)
	}
}

func TestFormatResponseMinimal_WithError(t *testing.T) {
	resp := &ResponseDisplay{
		Method: "GET",
		URL:    "http://localhost/fail",
		Status: 0,
		Error:  "DNS lookup failed",
	}

	var buf bytes.Buffer
	if err := FormatResponseMinimal(&buf, resp); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(buf.String(), "DNS lookup failed") {
		t.Errorf("should contain error: %s", buf.String())
	}
}

func TestFormatRunResponse_JSON(t *testing.T) {
	env := envelope.Success(map[string]any{
		"requestId": "health",
		"method":    "GET",
		"status":    float64(200),
	})

	var buf bytes.Buffer
	if err := FormatRunResponse(&buf, FormatJSON, env); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should be valid JSON envelope.
	var output envelope.Envelope
	if jsonErr := json.Unmarshal(buf.Bytes(), &output); jsonErr != nil {
		t.Fatalf("output is not valid JSON: %v", jsonErr)
	}
	if !output.OK {
		t.Error("expected ok=true")
	}
}

func TestFormatRunResponse_Table(t *testing.T) {
	env := envelope.Success(map[string]any{
		"requestId": "health",
		"method":    "GET",
		"url":       "http://localhost/health",
		"status":    float64(200),
		"body":      "OK",
		"headers":   map[string]any{"X-Test": "yes"},
		"timing": map[string]any{
			"dns": float64(1), "connect": float64(2), "tls": float64(0),
			"ttfb": float64(10), "transfer": float64(2), "total": float64(15),
		},
	})

	var buf bytes.Buffer
	if err := FormatRunResponse(&buf, FormatTable, env); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "✓ 200") {
		t.Errorf("table should contain status indicator: %s", output)
	}
	if !strings.Contains(output, "15ms") {
		t.Errorf("table should contain timing: %s", output)
	}
}

func TestFormatRunResponse_Minimal(t *testing.T) {
	env := envelope.Success(map[string]any{
		"requestId": "health",
		"method":    "GET",
		"url":       "http://localhost/health",
		"status":    float64(200),
		"body":      "OK",
		"timing": map[string]any{
			"total": float64(25),
		},
	})

	var buf bytes.Buffer
	if err := FormatRunResponse(&buf, FormatMinimal, env); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "200 GET") {
		t.Errorf("minimal should start with status: %s", output)
	}
	if !strings.Contains(output, "(25ms)") {
		t.Errorf("minimal should contain timing: %s", output)
	}
}

func TestFormatRunResponse_ErrorEnvelope(t *testing.T) {
	env := envelope.Fail("NOT_FOUND", "request not found")

	var buf bytes.Buffer
	if err := FormatRunResponse(&buf, FormatTable, env); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(buf.String(), "NOT_FOUND") {
		t.Errorf("should contain error code: %s", buf.String())
	}
}

func TestFormatCollectionResponses_Table(t *testing.T) {
	responses := []ResponseDisplay{
		{
			Method: "GET", URL: "http://localhost/a", Status: 200, Body: "A",
			Timing: &TimingDisplay{Total: 10},
		},
		{
			Method: "POST", URL: "http://localhost/b", Status: 201, Body: "B",
			Timing: &TimingDisplay{Total: 20},
		},
	}

	var buf bytes.Buffer
	if err := formatCollectionResponses(&buf, FormatTable, responses); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	// Should have separator between responses.
	if !strings.Contains(output, "────") {
		t.Errorf("should contain separator: %s", output)
	}
	// Should have both responses.
	if !strings.Contains(output, "http://localhost/a") {
		t.Errorf("should contain first response: %s", output)
	}
	if !strings.Contains(output, "http://localhost/b") {
		t.Errorf("should contain second response: %s", output)
	}
}

func TestFormatCollectionResponses_Minimal(t *testing.T) {
	responses := []ResponseDisplay{
		{Method: "GET", URL: "http://localhost/a", Status: 200, Body: "a"},
		{Method: "GET", URL: "http://localhost/b", Status: 404, Body: "not found"},
	}

	var buf bytes.Buffer
	if err := formatCollectionResponses(&buf, FormatMinimal, responses); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "200 GET http://localhost/a") {
		t.Errorf("should contain first response: %s", output)
	}
	if !strings.Contains(output, "404 GET http://localhost/b") {
		t.Errorf("should contain second response: %s", output)
	}
}

func TestStatusEmoji(t *testing.T) {
	tests := []struct {
		status int
		want   string
	}{
		{200, "✓"},
		{201, "✓"},
		{299, "✓"},
		{301, "→"},
		{404, "✗"},
		{500, "⚠"},
	}

	for _, tt := range tests {
		t.Run(strings.Replace(string(rune(tt.status+'0')), "\x00", "", -1), func(t *testing.T) {
			got := statusEmoji(tt.status)
			if got != tt.want {
				t.Errorf("statusEmoji(%d) = %q, want %q", tt.status, got, tt.want)
			}
		})
	}
}

func TestFormatRunResponse_CollectionJSON(t *testing.T) {
	env := envelope.Success([]map[string]any{
		{"method": "GET", "url": "http://a", "status": float64(200)},
		{"method": "GET", "url": "http://b", "status": float64(201)},
	})

	var buf bytes.Buffer
	if err := FormatRunResponse(&buf, FormatJSON, env); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var output envelope.Envelope
	if jsonErr := json.Unmarshal(buf.Bytes(), &output); jsonErr != nil {
		t.Fatalf("invalid JSON: %v", jsonErr)
	}
	if !output.OK {
		t.Error("expected ok=true")
	}
}

func TestFormatRunResponse_CollectionTable(t *testing.T) {
	env := envelope.Success([]map[string]any{
		{"requestId": "a", "method": "GET", "url": "http://localhost/a", "status": float64(200), "body": "ok", "timing": map[string]any{"total": float64(10)}},
		{"requestId": "b", "method": "POST", "url": "http://localhost/b", "status": float64(500), "body": "err", "timing": map[string]any{"total": float64(30)}},
	})

	var buf bytes.Buffer
	if err := FormatRunResponse(&buf, FormatTable, env); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "http://localhost/a") {
		t.Errorf("should contain first response: %s", output)
	}
	if !strings.Contains(output, "http://localhost/b") {
		t.Errorf("should contain second response: %s", output)
	}
}
