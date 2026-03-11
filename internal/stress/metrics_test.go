package stress

import (
	"math"
	"sync"
	"testing"
)

func TestMetricsCollector_Record(t *testing.T) {
	mc := NewMetricsCollector()

	// Record some requests with known latencies (in microseconds).
	mc.Record(10_000, 200, 1024) // 10ms, OK
	mc.Record(20_000, 200, 2048) // 20ms, OK
	mc.Record(50_000, 500, 512)  // 50ms, 500 error

	summary := mc.Summary()

	if summary.TotalRequests != 3 {
		t.Errorf("TotalRequests = %d, want 3", summary.TotalRequests)
	}
	if summary.ErrorRate < 33 || summary.ErrorRate > 34 {
		t.Errorf("ErrorRate = %f, want ~33.33", summary.ErrorRate)
	}
}

func TestMetricsCollector_HDRPercentiles(t *testing.T) {
	mc := NewMetricsCollector()

	// Record 100 requests with latencies from 1ms to 100ms.
	for i := int64(1); i <= 100; i++ {
		mc.Record(i*1000, 200, 100) // i ms in µs
	}

	summary := mc.Summary()

	// P50 should be around 50ms (±2ms due to HDR precision).
	if summary.Latency.P50 < 48 || summary.Latency.P50 > 52 {
		t.Errorf("P50 = %d, want ~50", summary.Latency.P50)
	}

	// P95 should be around 95ms.
	if summary.Latency.P95 < 93 || summary.Latency.P95 > 97 {
		t.Errorf("P95 = %d, want ~95", summary.Latency.P95)
	}

	// P99 should be around 99ms.
	if summary.Latency.P99 < 97 || summary.Latency.P99 > 101 {
		t.Errorf("P99 = %d, want ~99", summary.Latency.P99)
	}
}

func TestMetricsCollector_ConcurrentRecording(t *testing.T) {
	mc := NewMetricsCollector()

	const goroutines = 100
	const requestsPerGoroutine = 1000

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for g := 0; g < goroutines; g++ {
		go func() {
			defer wg.Done()
			for i := 0; i < requestsPerGoroutine; i++ {
				mc.Record(5000, 200, 256) // 5ms, OK
			}
		}()
	}

	wg.Wait()

	summary := mc.Summary()

	expectedTotal := goroutines * requestsPerGoroutine
	if summary.TotalRequests != expectedTotal {
		t.Errorf("TotalRequests = %d, want %d", summary.TotalRequests, expectedTotal)
	}
	if summary.ErrorRate != 0 {
		t.Errorf("ErrorRate = %f, want 0", summary.ErrorRate)
	}
}

func TestMetricsCollector_Snapshot(t *testing.T) {
	mc := NewMetricsCollector()

	// Record some requests in the first window.
	mc.Record(10_000, 200, 1024) // 10ms
	mc.Record(20_000, 200, 2048) // 20ms
	mc.Record(50_000, 500, 512)  // 50ms, error

	// Take a snapshot.
	point := mc.Snapshot(1.0, 50)

	if point.Elapsed != 1.0 {
		t.Errorf("Elapsed = %f, want 1.0", point.Elapsed)
	}
	if point.RPS != 3 {
		t.Errorf("RPS = %f, want 3", point.RPS)
	}
	if point.ActiveUsers != 50 {
		t.Errorf("ActiveUsers = %d, want 50", point.ActiveUsers)
	}
	// ErrorRate should be ~33.33%.
	if point.ErrorRate < 33 || point.ErrorRate > 34 {
		t.Errorf("ErrorRate = %f, want ~33.33", point.ErrorRate)
	}

	// After snapshot, window should be reset.
	// Record 1 more request and take another snapshot.
	mc.Record(5_000, 200, 100) // 5ms
	point2 := mc.Snapshot(2.0, 50)

	if point2.RPS != 1 {
		t.Errorf("RPS after reset = %f, want 1", point2.RPS)
	}
	if point2.ErrorRate != 0 {
		t.Errorf("ErrorRate after reset = %f, want 0", point2.ErrorRate)
	}

	// But cumulative totals should still reflect all 4 requests.
	summary := mc.Summary()
	if summary.TotalRequests != 4 {
		t.Errorf("TotalRequests = %d, want 4", summary.TotalRequests)
	}
}

func TestMetricsCollector_SnapshotEmpty(t *testing.T) {
	mc := NewMetricsCollector()

	point := mc.Snapshot(0.0, 0)

	if point.RPS != 0 {
		t.Errorf("RPS = %f, want 0", point.RPS)
	}
	if point.ErrorRate != 0 {
		t.Errorf("ErrorRate = %f, want 0", point.ErrorRate)
	}
	if point.P95 != 0 {
		t.Errorf("P95 = %d, want 0", point.P95)
	}
}

func TestMetricsCollector_SetPeakConnections(t *testing.T) {
	mc := NewMetricsCollector()

	mc.SetPeakConnections(50)
	mc.SetPeakConnections(100)
	mc.SetPeakConnections(75) // Should not reduce peak.

	summary := mc.Summary()
	if summary.PeakConnections != 100 {
		t.Errorf("PeakConnections = %d, want 100", summary.PeakConnections)
	}
}

func TestMetricsCollector_Reset(t *testing.T) {
	mc := NewMetricsCollector()

	mc.Record(10_000, 200, 1024)
	mc.Record(20_000, 500, 512)
	mc.SetPeakConnections(50)

	mc.Reset()

	summary := mc.Summary()
	if summary.TotalRequests != 0 {
		t.Errorf("TotalRequests after reset = %d, want 0", summary.TotalRequests)
	}
	if summary.PeakConnections != 0 {
		t.Errorf("PeakConnections after reset = %d, want 0", summary.PeakConnections)
	}
}

func TestMetricsCollector_ErrorClassification(t *testing.T) {
	mc := NewMetricsCollector()

	// 2xx, 3xx, 4xx are NOT errors. 5xx and 0 (connection failure) are errors.
	mc.Record(1000, 200, 100)
	mc.Record(1000, 301, 100)
	mc.Record(1000, 404, 100)
	mc.Record(1000, 500, 100) // error
	mc.Record(1000, 502, 100) // error
	mc.Record(1000, 0, 0)     // connection failure = error

	summary := mc.Summary()
	if summary.TotalRequests != 6 {
		t.Errorf("TotalRequests = %d, want 6", summary.TotalRequests)
	}
	// 3 errors out of 6 = 50%.
	if math.Abs(summary.ErrorRate-50) > 0.1 {
		t.Errorf("ErrorRate = %f, want 50", summary.ErrorRate)
	}
}

func TestMetricsCollector_Throughput(t *testing.T) {
	mc := NewMetricsCollector()

	// Record 10 requests of 1000 bytes each.
	for i := 0; i < 10; i++ {
		mc.Record(1000, 200, 1000)
	}

	summary := mc.Summary()
	// Total bytes = 10000, throughput = 10000 / elapsed.
	// We primarily verify it's non-zero and positive.
	if summary.Throughput <= 0 {
		t.Errorf("Throughput = %d, want > 0", summary.Throughput)
	}
}

func BenchmarkMetricsCollector_Record(b *testing.B) {
	mc := NewMetricsCollector()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mc.Record(5000, 200, 256)
	}
}

func BenchmarkMetricsCollector_ConcurrentRecord(b *testing.B) {
	mc := NewMetricsCollector()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			mc.Record(5000, 200, 256)
		}
	})
}
