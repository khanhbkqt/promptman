package stress

import (
	"context"

	"github.com/khanhnguyen/promptman/internal/request"
)

// EngineExecutor wraps a *request.Engine to implement RequestExecutor.
// It allows the daemon to run stress tests without going through the
// daemon's own HTTP endpoint — the Engine handles collection lookup,
// environment resolution, and HTTP dispatch directly in-process.
type EngineExecutor struct {
	engine *request.Engine
}

// NewRequestExecutorFromEngine creates a RequestExecutor backed by a
// *request.Engine. Use this when the daemon wants to run stress tests
// itself (as opposed to the CLI, which proxies through the daemon API).
func NewRequestExecutorFromEngine(engine *request.Engine) *EngineExecutor {
	return &EngineExecutor{engine: engine}
}

// Execute sends a single HTTP request through the Engine and extracts
// the metrics needed for stress test recording.
func (e *EngineExecutor) Execute(
	ctx context.Context,
	collectionID, requestID string,
) (statusCode int, latencyUs int64, bodySize int64, err error) {
	resp, execErr := e.engine.Execute(ctx, request.ExecuteInput{
		CollectionID: collectionID,
		RequestID:    requestID,
		Source:       "stress",
	})
	if execErr != nil {
		return 0, 0, 0, execErr
	}

	var latency int64
	if resp.Timing != nil {
		latency = int64(resp.Timing.Total) * 1000 // ms → µs
	}

	return resp.Status, latency, int64(len(resp.Body)), nil
}
