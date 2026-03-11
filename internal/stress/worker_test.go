package stress

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// mockExecutor implements RequestExecutor for testing.
type mockExecutor struct {
	latencyUs int64
	status    int
	bodySize  int64
	callCount int64
	mu        sync.Mutex
	calls     []executorCall
}

type executorCall struct {
	CollectionID string
	RequestID    string
}

func (m *mockExecutor) Execute(_ context.Context, collectionID, requestID string) (int, int64, int64, error) {
	atomic.AddInt64(&m.callCount, 1)
	m.mu.Lock()
	m.calls = append(m.calls, executorCall{collectionID, requestID})
	m.mu.Unlock()
	return m.status, m.latencyUs, m.bodySize, nil
}

func (m *mockExecutor) CallCount() int64 {
	return atomic.LoadInt64(&m.callCount)
}

func TestWorker_WeightDistribution(t *testing.T) {
	exec := &mockExecutor{status: 200, latencyUs: 1000, bodySize: 100}
	metrics := NewMetricsCollector()
	var connections int64

	scenarios := []resolvedScenario{
		{Name: "heavy", CollectionID: "c", RequestID: "r1", Weight: 70},
		{Name: "light", CollectionID: "c", RequestID: "r2", Weight: 30},
	}

	w := newWorker(scenarios, exec, metrics, &connections)

	// Run for enough iterations to verify distribution.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	w.run(ctx)

	total := exec.CallCount()
	if total < 100 {
		t.Skipf("not enough iterations (%d) for statistical significance", total)
	}

	// Count calls per request.
	exec.mu.Lock()
	counts := make(map[string]int)
	for _, call := range exec.calls {
		counts[call.RequestID]++
	}
	exec.mu.Unlock()

	r1Pct := float64(counts["r1"]) / float64(total) * 100
	r2Pct := float64(counts["r2"]) / float64(total) * 100

	// Allow ±10% tolerance (wider tolerance for 2s test).
	if r1Pct < 55 || r1Pct > 85 {
		t.Errorf("r1 distribution = %.1f%%, expected ~70%% (±15%%)", r1Pct)
	}
	if r2Pct < 15 || r2Pct > 45 {
		t.Errorf("r2 distribution = %.1f%%, expected ~30%% (±15%%)", r2Pct)
	}
}

func TestWorker_SingleScenario(t *testing.T) {
	exec := &mockExecutor{status: 200, latencyUs: 500, bodySize: 50}
	metrics := NewMetricsCollector()
	var connections int64

	scenarios := []resolvedScenario{
		{Name: "only", CollectionID: "c", RequestID: "r", Weight: 100},
	}

	w := newWorker(scenarios, exec, metrics, &connections)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	w.run(ctx)

	if got := exec.CallCount(); got == 0 {
		t.Error("worker executed zero requests")
	}

	summary := metrics.Summary()
	if summary.TotalRequests == 0 {
		t.Error("metrics recorded zero requests")
	}
}

func TestWorker_ThinkTime(t *testing.T) {
	exec := &mockExecutor{status: 200, latencyUs: 100, bodySize: 10}
	metrics := NewMetricsCollector()
	var connections int64

	scenarios := []resolvedScenario{
		{Name: "slow", CollectionID: "c", RequestID: "r", Weight: 100, ThinkTime: 200 * time.Millisecond},
	}

	w := newWorker(scenarios, exec, metrics, &connections)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	start := time.Now()
	w.run(ctx)
	elapsed := time.Since(start)

	// With 200ms think time in 500ms, we should get ~2-3 requests max.
	if got := exec.CallCount(); got > 5 {
		t.Errorf("with 200ms thinkTime in %v: got %d requests, expected ≤5", elapsed, got)
	}
}

func TestWorker_PeakConnections(t *testing.T) {
	// Use a slow executor to ensure concurrent connections.
	exec := &slowExecutor{delay: 50 * time.Millisecond, status: 200}
	metrics := NewMetricsCollector()
	var connections int64

	scenarios := []resolvedScenario{
		{Name: "test", CollectionID: "c", RequestID: "r", Weight: 100},
	}

	// Spawn multiple workers to see peak connections > 1.
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w := newWorker(scenarios, exec, metrics, &connections)
			w.run(ctx)
		}()
	}

	wg.Wait()

	summary := metrics.Summary()
	if summary.PeakConnections < 2 {
		t.Errorf("peak connections = %d, expected ≥2 with 5 concurrent workers", summary.PeakConnections)
	}
}

func TestWorker_GracefulShutdown(t *testing.T) {
	// Use a slow executor — worker should finish current request.
	exec := &slowExecutor{delay: 100 * time.Millisecond, status: 200}
	metrics := NewMetricsCollector()
	var connections int64

	scenarios := []resolvedScenario{
		{Name: "test", CollectionID: "c", RequestID: "r", Weight: 100},
	}

	w := newWorker(scenarios, exec, metrics, &connections)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		w.run(ctx)
		close(done)
	}()

	// Let worker start one request, then cancel.
	time.Sleep(50 * time.Millisecond)
	cancel()

	// Worker should exit within a reasonable time (finish current request).
	select {
	case <-done:
		// Good — worker exited.
	case <-time.After(2 * time.Second):
		t.Fatal("worker did not exit within timeout after cancellation")
	}
}

func TestWorker_SelectScenario_AllWeights(t *testing.T) {
	// Verify selectScenario covers all scenarios.
	w := &worker{
		scenarios: []resolvedScenario{
			{Name: "a", Weight: 33},
			{Name: "b", Weight: 34},
			{Name: "c", Weight: 33},
		},
	}

	seen := make(map[string]bool)
	for i := 0; i < 1000; i++ {
		s := w.selectScenario()
		seen[s.Name] = true
	}

	for _, name := range []string{"a", "b", "c"} {
		if !seen[name] {
			t.Errorf("scenario %q was never selected in 1000 iterations", name)
		}
	}
}

// slowExecutor simulates a slow HTTP request for testing.
type slowExecutor struct {
	delay  time.Duration
	status int
}

func (s *slowExecutor) Execute(ctx context.Context, _, _ string) (int, int64, int64, error) {
	select {
	case <-time.After(s.delay):
		return s.status, int64(s.delay.Microseconds()), 100, nil
	case <-ctx.Done():
		return 0, 0, 0, ctx.Err()
	}
}
