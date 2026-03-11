package stress

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"testing"
	"time"
)

func TestStressRunner_Run_BasicSingleScenario(t *testing.T) {
	exec := &mockExecutor{status: 200, latencyUs: 1000, bodySize: 256}
	runner := NewStressRunner(exec)

	report, err := runner.Run(&StressOpts{
		Collection: "test-col",
		RequestID:  "test-req",
		Users:      5,
		Duration:   "2s",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if report.Scenario != "test-req" {
		t.Errorf("scenario = %q, want %q", report.Scenario, "test-req")
	}
	if report.Summary.TotalRequests == 0 {
		t.Error("total requests = 0, want > 0")
	}
	if report.Summary.RPS <= 0 {
		t.Error("RPS = 0, want > 0")
	}
	if report.Summary.Latency.P50 <= 0 {
		t.Error("P50 latency = 0, want > 0")
	}
	if report.Duration < 1900 {
		t.Errorf("duration = %dms, want ~2000ms", report.Duration)
	}
	if len(report.Timeline) == 0 {
		t.Error("timeline is empty, want at least 1 point")
	}
}

func TestStressRunner_Run_WithRampUp(t *testing.T) {
	exec := &mockExecutor{status: 200, latencyUs: 500, bodySize: 100}
	runner := NewStressRunner(exec)

	report, err := runner.Run(&StressOpts{
		Collection: "test",
		RequestID:  "req",
		Users:      10,
		Duration:   "3s",
		RampUp:     "1s",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if report.Summary.TotalRequests == 0 {
		t.Error("no requests executed")
	}
	if len(report.Timeline) < 2 {
		t.Errorf("timeline has %d points, want >= 2", len(report.Timeline))
	}
}

func TestStressRunner_Run_InvalidOpts(t *testing.T) {
	exec := &mockExecutor{status: 200}
	runner := NewStressRunner(exec)

	_, err := runner.Run(&StressOpts{
		Collection: "",
		RequestID:  "req",
		Users:      10,
		Duration:   "10s",
	})
	if err == nil {
		t.Fatal("expected error for invalid opts")
	}
}

func TestStressRunner_RunFromConfig_ValidYAML(t *testing.T) {
	exec := &mockExecutor{status: 200, latencyUs: 500, bodySize: 64}
	runner := NewStressRunner(exec)

	yaml := `
name: Config Test
scenarios:
  - name: read
    request: test/read
    weight: 60
  - name: write
    request: test/write
    weight: 40

config:
  users: 5
  duration: 2s
`
	path := writeRunnerTestYAML(t, yaml)

	report, err := runner.RunFromConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if report.Scenario != "Config Test" {
		t.Errorf("scenario = %q, want %q", report.Scenario, "Config Test")
	}
	if report.Summary.TotalRequests == 0 {
		t.Error("no requests executed")
	}
}

func TestStressRunner_RunFromConfig_InvalidFile(t *testing.T) {
	exec := &mockExecutor{status: 200}
	runner := NewStressRunner(exec)

	_, err := runner.RunFromConfig("/nonexistent.yaml")
	if err == nil {
		t.Fatal("expected error for nonexistent config file")
	}
}

func TestStressRunner_Integration_WithHTTPServer(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	var requestCount int64

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&requestCount, 1)
		time.Sleep(5 * time.Millisecond) // Simulate work.
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"ok"}`)
	}))
	defer server.Close()

	// Create an executor that actually hits the test server.
	exec := &httpTestExecutor{serverURL: server.URL}
	runner := NewStressRunner(exec)

	report, err := runner.Run(&StressOpts{
		Collection: "integration",
		RequestID:  "test",
		Users:      10,
		Duration:   "3s",
		RampUp:     "500ms",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Validate report.
	if report.Summary.TotalRequests == 0 {
		t.Error("total requests = 0")
	}
	if report.Summary.RPS <= 0 {
		t.Errorf("RPS = %.2f, want > 0", report.Summary.RPS)
	}
	if report.Summary.Latency.P50 <= 0 {
		t.Errorf("P50 = %d, want > 0", report.Summary.Latency.P50)
	}
	if report.Summary.ErrorRate > 1.0 {
		t.Errorf("error rate = %.2f%%, want ≤1%% (small errors from ctx cancellation ok)", report.Summary.ErrorRate)
	}
	if len(report.Timeline) < 2 {
		t.Errorf("timeline points = %d, want >= 2", len(report.Timeline))
	}

	serverHits := atomic.LoadInt64(&requestCount)
	if serverHits == 0 {
		t.Error("HTTP server received 0 requests")
	}

	t.Logf("Integration: %d requests, %.0f RPS, P50=%dms, P95=%dms, P99=%dms",
		report.Summary.TotalRequests, report.Summary.RPS,
		report.Summary.Latency.P50, report.Summary.Latency.P95, report.Summary.Latency.P99)
}

func TestStressRunner_NoGoroutineLeaks(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping goroutine leak test in short mode")
	}

	// Measure baseline goroutines.
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	baseline := runtime.NumGoroutine()

	exec := &mockExecutor{status: 200, latencyUs: 100, bodySize: 10}
	runner := NewStressRunner(exec)

	report, err := runner.Run(&StressOpts{
		Collection: "leak-test",
		RequestID:  "req",
		Users:      50,
		Duration:   "2s",
		RampUp:     "500ms",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if report.Summary.TotalRequests == 0 {
		t.Error("no requests executed")
	}

	// Wait for goroutines to settle.
	time.Sleep(500 * time.Millisecond)
	runtime.GC()
	time.Sleep(100 * time.Millisecond)

	after := runtime.NumGoroutine()
	leaked := after - baseline

	// Allow some tolerance (Go runtime goroutines may fluctuate).
	if leaked > 5 {
		t.Errorf("goroutine leak: baseline=%d, after=%d, leaked=%d", baseline, after, leaked)
	}
}

func TestStressRunner_ErrorTracking(t *testing.T) {
	// Executor that returns server errors.
	exec := &mockExecutor{status: 500, latencyUs: 1000, bodySize: 0}
	runner := NewStressRunner(exec)

	report, err := runner.Run(&StressOpts{
		Collection: "errors",
		RequestID:  "fail",
		Users:      3,
		Duration:   "1s",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if report.Summary.ErrorRate != 100 {
		t.Errorf("error rate = %.1f%%, want 100%%", report.Summary.ErrorRate)
	}
}

// httpTestExecutor makes real HTTP requests to a test server.
type httpTestExecutor struct {
	serverURL string
}

func (e *httpTestExecutor) Execute(ctx context.Context, _, _ string) (int, int64, int64, error) {
	start := time.Now()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, e.serverURL, nil)
	if err != nil {
		return 0, 0, 0, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, time.Since(start).Microseconds(), 0, err
	}
	defer resp.Body.Close()

	latencyUs := time.Since(start).Microseconds()
	return resp.StatusCode, latencyUs, resp.ContentLength, nil
}

// writeRunnerTestYAML writes a YAML string to a temp file.
func writeRunnerTestYAML(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.stress.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("writing yaml: %v", err)
	}
	return path
}

// ─── Threshold wiring tests ───────────────────────────────────────────────────

func TestStressRunner_Run_WithThresholdsPass(t *testing.T) {
	exec := &mockExecutor{status: 200, latencyUs: 1000, bodySize: 64}
	runner := NewStressRunner(exec)

	report, err := runner.Run(&StressOpts{
		Collection: "test",
		RequestID:  "req",
		Users:      3,
		Duration:   "1s",
		Thresholds: []string{"error_rate<50%", "rps>0"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(report.Thresholds) != 2 {
		t.Errorf("len(Thresholds) = %d, want 2", len(report.Thresholds))
	}
	for _, r := range report.Thresholds {
		if !r.Passed {
			t.Errorf("threshold %q failed: actual=%.1f %s %.1f", r.Name, r.Actual, r.Operator, r.Expected)
		}
	}
}

func TestStressRunner_Run_WithThresholdsFail(t *testing.T) {
	// All 500s → error_rate=100, fails the <5% threshold.
	exec := &mockExecutor{status: 500, latencyUs: 1000, bodySize: 0}
	runner := NewStressRunner(exec)

	report, err := runner.Run(&StressOpts{
		Collection: "test",
		RequestID:  "req",
		Users:      3,
		Duration:   "1s",
		Thresholds: []string{"error_rate<5%"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(report.Thresholds) != 1 {
		t.Errorf("len(Thresholds) = %d, want 1", len(report.Thresholds))
	}
	if report.Thresholds[0].Passed {
		t.Errorf("threshold Passed = true, want false (error_rate=%.1f)", report.Thresholds[0].Actual)
	}
}

func TestStressRunner_Run_NoThresholds(t *testing.T) {
	exec := &mockExecutor{status: 200, latencyUs: 500, bodySize: 64}
	runner := NewStressRunner(exec)

	report, err := runner.Run(&StressOpts{
		Collection: "test",
		RequestID:  "req",
		Users:      2,
		Duration:   "1s",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.Thresholds != nil {
		t.Errorf("Thresholds = %v, want nil when none defined", report.Thresholds)
	}
}

func TestStressRunner_Run_InvalidThreshold(t *testing.T) {
	exec := &mockExecutor{status: 200}
	runner := NewStressRunner(exec)

	_, err := runner.Run(&StressOpts{
		Collection: "test",
		RequestID:  "req",
		Users:      2,
		Duration:   "1s",
		Thresholds: []string{"badexpression"},
	})
	if err == nil {
		t.Fatal("expected error for invalid threshold, got nil")
	}
}

func TestStressRunner_RunFromConfig_WithThresholdConfig(t *testing.T) {
	exec := &mockExecutor{status: 200, latencyUs: 500, bodySize: 64}
	runner := NewStressRunner(exec)

	yaml := `
name: Threshold Config Test
scenarios:
  - name: read
    request: test/list
    weight: 100
config:
  users: 3
  duration: 1s
thresholds:
  error_rate: "50%"
`
	path := writeRunnerTestYAML(t, yaml)

	report, err := runner.RunFromConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// error_rate < 50% should pass (executor returns 200s)
	if len(report.Thresholds) == 0 {
		t.Error("Thresholds is empty, want at least 1 result")
	}
	for _, r := range report.Thresholds {
		if !r.Passed {
			t.Errorf("threshold %q failed: actual=%.1f %s %.1f", r.Name, r.Actual, r.Operator, r.Expected)
		}
	}
}
