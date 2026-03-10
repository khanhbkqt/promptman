package sandbox

import (
	"context"
	"time"

	"github.com/dop251/goja"
	testing "github.com/khanhnguyen/promptman/internal/testing"
)

// Default timeout values for test execution.
const (
	DefaultPerTestTimeout  = 10 * time.Second
	DefaultPerSuiteTimeout = 120 * time.Second
)

// TimeoutLevel indicates which timeout level triggered.
type TimeoutLevel string

const (
	// TimeoutLevelTest indicates a per-test timeout was exceeded.
	TimeoutLevelTest TimeoutLevel = "test"
	// TimeoutLevelSuite indicates the suite-wide timeout was exceeded.
	TimeoutLevelSuite TimeoutLevel = "suite"
)

// TimeoutManager manages dual timeouts: per-test and per-suite.
// Per-test timeout interrupts individual script execution via vm.Interrupt().
// Per-suite timeout wraps all test execution with context.WithTimeout.
type TimeoutManager struct {
	perTest  time.Duration
	perSuite time.Duration
}

// NewTimeoutManager creates a TimeoutManager with the given durations.
// Zero or negative values are replaced with defaults.
func NewTimeoutManager(perTest, perSuite time.Duration) *TimeoutManager {
	if perTest <= 0 {
		perTest = DefaultPerTestTimeout
	}
	if perSuite <= 0 {
		perSuite = DefaultPerSuiteTimeout
	}
	return &TimeoutManager{
		perTest:  perTest,
		perSuite: perSuite,
	}
}

// PerTest returns the per-test timeout duration.
func (tm *TimeoutManager) PerTest() time.Duration {
	return tm.perTest
}

// PerSuite returns the per-suite timeout duration.
func (tm *TimeoutManager) PerSuite() time.Duration {
	return tm.perSuite
}

// SuiteContext creates a suite-level context with the configured timeout.
// The caller MUST call the returned CancelFunc when the suite completes.
func (tm *TimeoutManager) SuiteContext(parent context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, tm.perSuite)
}

// timeoutSentinel is the value passed to vm.Interrupt() so we can
// distinguish timeout interrupts from other interrupt reasons.
const timeoutSentinel = "timeout:test_execution_exceeded"

// RunWithTimeout executes fn with a per-test timeout. If the suite context
// (ctx) is already cancelled, it returns immediately with a suite-level
// timeout error. If the per-test deadline fires, it calls vm.Interrupt()
// to abort the running script and returns ErrTestTimeout.
//
// After each call, vm.ClearInterrupt() is called so the VM can be reused.
func (tm *TimeoutManager) RunWithTimeout(ctx context.Context, vm *goja.Runtime, fn func() error) error {
	// Check if suite context is already expired.
	if err := ctx.Err(); err != nil {
		return testing.ErrTestTimeout.Wrapf("suite timeout exceeded")
	}

	// Determine effective timeout: the lesser of per-test and remaining suite time.
	deadline, hasDeadline := ctx.Deadline()
	effectiveTimeout := tm.perTest
	if hasDeadline {
		remaining := time.Until(deadline)
		if remaining < effectiveTimeout {
			effectiveTimeout = remaining
		}
	}

	// Create per-test context for the timer goroutine.
	testCtx, testCancel := context.WithTimeout(ctx, effectiveTimeout)
	defer testCancel()

	// Schedule interrupt on timeout.
	done := make(chan struct{})
	go func() {
		select {
		case <-testCtx.Done():
			// Timeout or suite cancellation — interrupt the VM.
			vm.Interrupt(timeoutSentinel)
		case <-done:
			// Function completed normally — nothing to do.
		}
	}()

	// Run the function.
	err := fn()

	// Signal the goroutine to stop, then clear interrupt for VM reuse.
	close(done)
	vm.ClearInterrupt()

	if err != nil {
		return classifyTimeoutError(ctx, err)
	}
	return nil
}

// classifyTimeoutError checks whether an error was caused by a timeout
// interrupt and returns the appropriate ErrTestTimeout.
func classifyTimeoutError(ctx context.Context, err error) error {
	// Check if this is a goja interrupt (our timeout sentinel).
	if isInterruptError(err) {
		if ctx.Err() != nil {
			return testing.ErrTestTimeout.Wrapf("suite timeout exceeded")
		}
		return testing.ErrTestTimeout.Wrapf("per-test timeout exceeded")
	}
	return err
}

// isInterruptError returns true if the error is a goja InterruptedError.
func isInterruptError(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(*goja.InterruptedError)
	return ok
}
