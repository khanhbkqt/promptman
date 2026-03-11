package stress

import (
	"sync"
	"sync/atomic"
	"time"

	hdr "github.com/HdrHistogram/hdrhistogram-go"
)

const (
	// hdrMinLatency is the minimum trackable latency in microseconds (1µs).
	hdrMinLatency = 1
	// hdrMaxLatency is the maximum trackable latency in microseconds (30s).
	hdrMaxLatency = 30_000_000
	// hdrSignificantFigures is the number of significant value digits for the histogram.
	hdrSignificantFigures = 3
)

// MetricsCollector aggregates request metrics using an HDR histogram
// for accurate latency percentile calculation. It is safe for concurrent
// use by multiple goroutines.
type MetricsCollector struct {
	mu sync.Mutex

	// Cumulative HDR histogram for the entire test run.
	histogram *hdr.Histogram

	// Per-window HDR histogram, reset on each Snapshot() call.
	windowHistogram *hdr.Histogram

	// Cumulative counters (entire test).
	totalRequests int64
	totalErrors   int64
	totalBytes    int64

	// Per-window counters, reset on each Snapshot() call.
	windowRequests int64
	windowErrors   int64

	// Peak tracking.
	peakConnections int32 // atomic, set externally via SetActiveUsers

	// Start time for RPS calculation.
	startTime time.Time
}

// NewMetricsCollector creates a new MetricsCollector ready for recording.
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		histogram:       hdr.New(hdrMinLatency, hdrMaxLatency, hdrSignificantFigures),
		windowHistogram: hdr.New(hdrMinLatency, hdrMaxLatency, hdrSignificantFigures),
		startTime:       time.Now(),
	}
}

// Record records a single request result. It is safe to call from
// multiple goroutines concurrently.
func (m *MetricsCollector) Record(latencyUs int64, statusCode int, bytesSent int64) {
	isError := statusCode >= 500 || statusCode == 0

	m.mu.Lock()
	defer m.mu.Unlock()

	// Clamp latency to valid range.
	if latencyUs < hdrMinLatency {
		latencyUs = hdrMinLatency
	}

	_ = m.histogram.RecordValue(latencyUs)
	_ = m.windowHistogram.RecordValue(latencyUs)

	m.totalRequests++
	m.windowRequests++
	m.totalBytes += bytesSent

	if isError {
		m.totalErrors++
		m.windowErrors++
	}
}

// SetPeakConnections atomically updates the peak connections if the
// given value exceeds the current peak.
func (m *MetricsCollector) SetPeakConnections(n int) {
	for {
		current := atomic.LoadInt32(&m.peakConnections)
		newVal := int32(n)
		if newVal <= current {
			return
		}
		if atomic.CompareAndSwapInt32(&m.peakConnections, current, newVal) {
			return
		}
	}
}

// Snapshot returns a per-second metrics snapshot and resets the
// per-window counters. The elapsed and activeUsers values must be
// provided by the caller (from the scheduler).
func (m *MetricsCollector) Snapshot(elapsed float64, activeUsers int) TimelinePoint {
	m.mu.Lock()
	defer m.mu.Unlock()

	var rps float64
	if elapsed > 0 {
		rps = float64(m.windowRequests)
	}

	var errorRate float64
	if m.windowRequests > 0 {
		errorRate = float64(m.windowErrors) / float64(m.windowRequests) * 100
	}

	var p95 int64
	if m.windowHistogram.TotalCount() > 0 {
		p95 = m.windowHistogram.ValueAtPercentile(95) / 1000 // µs → ms
	}

	point := TimelinePoint{
		Elapsed:     elapsed,
		RPS:         rps,
		P95:         p95,
		ErrorRate:   errorRate,
		ActiveUsers: activeUsers,
	}

	// Reset per-window state.
	m.windowHistogram.Reset()
	m.windowRequests = 0
	m.windowErrors = 0

	return point
}

// Summary returns the final aggregate metrics for the entire test run.
func (m *MetricsCollector) Summary() StressSummary {
	m.mu.Lock()
	defer m.mu.Unlock()

	elapsed := time.Since(m.startTime).Seconds()

	var rps float64
	if elapsed > 0 {
		rps = float64(m.totalRequests) / elapsed
	}

	var errorRate float64
	if m.totalRequests > 0 {
		errorRate = float64(m.totalErrors) / float64(m.totalRequests) * 100
	}

	var throughput int64
	if elapsed > 0 {
		throughput = int64(float64(m.totalBytes) / elapsed)
	}

	var latency LatencyMetrics
	if m.histogram.TotalCount() > 0 {
		latency = LatencyMetrics{
			P50: m.histogram.ValueAtPercentile(50) / 1000, // µs → ms
			P95: m.histogram.ValueAtPercentile(95) / 1000,
			P99: m.histogram.ValueAtPercentile(99) / 1000,
		}
	}

	return StressSummary{
		TotalRequests:   int(m.totalRequests),
		RPS:             rps,
		Latency:         latency,
		ErrorRate:       errorRate,
		Throughput:      throughput,
		PeakConnections: int(atomic.LoadInt32(&m.peakConnections)),
	}
}

// Reset clears all state, preparing the collector for reuse.
func (m *MetricsCollector) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.histogram.Reset()
	m.windowHistogram.Reset()
	m.totalRequests = 0
	m.totalErrors = 0
	m.totalBytes = 0
	m.windowRequests = 0
	m.windowErrors = 0
	atomic.StoreInt32(&m.peakConnections, 0)
	m.startTime = time.Now()
}
