package daemon

import (
	"context"
	"time"

	testing "github.com/khanhnguyen/promptman/internal/testing"
	"github.com/khanhnguyen/promptman/internal/testing/core"
)

// SuiteRunnerAdapter adapts core.Runner (which takes *core.TestOpts) to
// the SuiteRunner interface (which takes time.Duration) expected by
// TestRunRegistrar.
type SuiteRunnerAdapter struct {
	runner *core.Runner
}

// NewSuiteRunnerAdapter wraps a core.Runner to satisfy the SuiteRunner interface.
func NewSuiteRunnerAdapter(runner *core.Runner) *SuiteRunnerAdapter {
	return &SuiteRunnerAdapter{runner: runner}
}

// RunSuite delegates to core.Runner.RunSuite, converting the suiteTimeout
// duration into a *core.TestOpts value.
func (a *SuiteRunnerAdapter) RunSuite(ctx context.Context, collID, env string, suiteTimeout time.Duration) (*testing.TestResult, error) {
	var opts *core.TestOpts
	if suiteTimeout > 0 {
		opts = &core.TestOpts{PerSuiteTimeout: suiteTimeout}
	}
	return a.runner.RunSuite(ctx, collID, env, opts)
}
