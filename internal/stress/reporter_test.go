package stress

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFormatTable_BasicReport(t *testing.T) {
	report := &StressReport{
		Scenario: "Users API Load Test",
		Duration: 60000,
		Summary: StressSummary{
			TotalRequests:   3000,
			RPS:             50.0,
			Latency:         LatencyMetrics{P50: 45, P95: 120, P99: 350},
			ErrorRate:       2.5,
			Throughput:      102400,
			PeakConnections: 100,
		},
		Timeline: []TimelinePoint{},
	}

	out := FormatTable(report)

	// Verify key sections are present.
	checks := []string{
		"Users API Load Test",
		"3000",
		"50.00 req/s",
		"2.50%",
		"100.00 KB/s",
		"100",
		"45 ms",
		"120 ms",
		"350 ms",
	}
	for _, want := range checks {
		if !strings.Contains(out, want) {
			t.Errorf("FormatTable() missing %q\nGot:\n%s", want, out)
		}
	}
}

func TestFormatTable_WithThresholds(t *testing.T) {
	report := &StressReport{
		Scenario: "test",
		Duration: 5000,
		Summary: StressSummary{
			TotalRequests: 100,
			RPS:           20.0,
			Latency:       LatencyMetrics{P50: 10, P95: 50, P99: 100},
			ErrorRate:     0,
		},
		Thresholds: []ThresholdResult{
			{Name: "p95_latency", Operator: "<", Expected: 500, Actual: 50, Passed: true},
			{Name: "error_rate", Operator: "<", Expected: 5, Actual: 10, Passed: false},
		},
	}

	out := FormatTable(report)

	if !strings.Contains(out, "✓ PASS") {
		t.Error("expected passing threshold marker")
	}
	if !strings.Contains(out, "✗ FAIL") {
		t.Error("expected failing threshold marker")
	}
}

func TestFormatTable_NoThresholds(t *testing.T) {
	report := &StressReport{
		Scenario: "simple",
		Duration: 1000,
		Summary:  StressSummary{TotalRequests: 10, RPS: 10},
	}

	out := FormatTable(report)

	if strings.Contains(out, "Thresholds") {
		t.Error("empty thresholds should not show Thresholds section")
	}
}

func TestWriteJSON_Success(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "report.json")

	report := &StressReport{
		Scenario: "json-test",
		Duration: 30000,
		Summary: StressSummary{
			TotalRequests: 500,
			RPS:           16.67,
			Latency:       LatencyMetrics{P50: 20, P95: 80, P99: 200},
			ErrorRate:     1.0,
			Throughput:    51200,
		},
		Timeline: []TimelinePoint{
			{Elapsed: 1, RPS: 15, P95: 75, ErrorRate: 0, ActiveUsers: 10},
		},
	}

	if err := WriteJSON(report, path); err != nil {
		t.Fatalf("WriteJSON() error = %v", err)
	}

	// Read and verify the file.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading output file: %v", err)
	}

	var decoded StressReport
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshalling output: %v", err)
	}

	if decoded.Scenario != "json-test" {
		t.Errorf("Scenario = %q, want %q", decoded.Scenario, "json-test")
	}
	if decoded.Summary.TotalRequests != 500 {
		t.Errorf("TotalRequests = %d, want 500", decoded.Summary.TotalRequests)
	}
	if len(decoded.Timeline) != 1 {
		t.Errorf("Timeline length = %d, want 1", len(decoded.Timeline))
	}
}

func TestWriteJSON_InvalidPath(t *testing.T) {
	report := &StressReport{Scenario: "fail"}

	err := WriteJSON(report, "/nonexistent/dir/report.json")
	if err == nil {
		t.Error("WriteJSON() with invalid path should return error")
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name string
		ms   int64
		want string
	}{
		{"sub-second", 500, "500ms"},
		{"one second", 1000, "1s"},
		{"minutes", 65000, "1m5s"},
		{"with millis", 1500, "1.5s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatDuration(tt.ms)
			if got != tt.want {
				t.Errorf("formatDuration(%d) = %q, want %q", tt.ms, got, tt.want)
			}
		})
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		name string
		b    int64
		want string
	}{
		{"bytes", 512, "512 B"},
		{"kilobytes", 2048, "2.00 KB"},
		{"megabytes", 1048576, "1.00 MB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatBytes(tt.b)
			if got != tt.want {
				t.Errorf("formatBytes(%d) = %q, want %q", tt.b, got, tt.want)
			}
		})
	}
}
