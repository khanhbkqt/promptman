package stress

import (
	"context"
	"math/rand"
	"sync/atomic"
	"time"
)

// worker represents a virtual user that repeatedly executes requests
// according to the weighted scenario distribution.
type worker struct {
	scenarios   []resolvedScenario
	executor    RequestExecutor
	metrics     *MetricsCollector
	connections *int64 // shared atomic peak connections counter
}

// newWorker creates a worker with the given dependencies.
func newWorker(
	scenarios []resolvedScenario,
	executor RequestExecutor,
	metrics *MetricsCollector,
	connections *int64,
) *worker {
	return &worker{
		scenarios:   scenarios,
		executor:    executor,
		metrics:     metrics,
		connections: connections,
	}
}

// run is the main worker loop. It selects a scenario by weight,
// executes the request, records metrics, waits for thinkTime, and
// repeats until the context is cancelled. The worker finishes the
// current request before exiting (graceful shutdown).
func (w *worker) run(ctx context.Context) {
	for {
		// Check if test is done before starting a new request.
		select {
		case <-ctx.Done():
			return
		default:
		}

		scenario := w.selectScenario()

		// Track peak connections: increment before, decrement after.
		current := atomic.AddInt64(w.connections, 1)
		w.metrics.SetPeakConnections(int(current))

		statusCode, latencyUs, bodySize, _ := w.executor.Execute(
			ctx, scenario.CollectionID, scenario.RequestID,
		)

		atomic.AddInt64(w.connections, -1)

		// Record metrics regardless of error (error requests are tracked by status code).
		w.metrics.Record(latencyUs, statusCode, bodySize)

		// Apply think time if configured.
		if scenario.ThinkTime > 0 {
			select {
			case <-ctx.Done():
				return
			case <-time.After(scenario.ThinkTime):
			}
		}
	}
}

// selectScenario picks a scenario using cumulative weight distribution.
// For a single scenario, this always returns that scenario.
func (w *worker) selectScenario() *resolvedScenario {
	if len(w.scenarios) == 1 {
		return &w.scenarios[0]
	}

	// Generate random number 1–100 and find the matching scenario
	// using cumulative weights.
	r := rand.Intn(100) + 1 // 1–100
	cumulative := 0

	for i := range w.scenarios {
		cumulative += w.scenarios[i].Weight
		if r <= cumulative {
			return &w.scenarios[i]
		}
	}

	// Fallback: return last scenario (should not happen if weights sum to 100).
	return &w.scenarios[len(w.scenarios)-1]
}
