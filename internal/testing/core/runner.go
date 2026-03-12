package core

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/dop251/goja"
	"github.com/khanhnguyen/promptman/internal/request"
	testing "github.com/khanhnguyen/promptman/internal/testing"
	"github.com/khanhnguyen/promptman/internal/testing/pmapi"
	"github.com/khanhnguyen/promptman/internal/testing/sandbox"
)

// TestFileLoader loads test script content for a given collection.
type TestFileLoader interface {
	// Load returns the JavaScript source for the test file associated
	// with the given collection ID.
	Load(collectionID string) (string, error)
}

// RequestRunner executes HTTP requests and returns responses.
type RequestRunner interface {
	// Execute runs a single HTTP request and returns the response.
	Execute(ctx context.Context, input request.ExecuteInput) (*request.Response, error)
}

// CollectionLister lists all request paths in a collection.
type CollectionLister interface {
	// ListRequestPaths returns all request paths (slash-separated) in
	// the collection. Example: ["health", "users/list", "admin/get"].
	ListRequestPaths(collectionID string) ([]string, error)
}

// TestOpts configures test execution.
type TestOpts struct {
	PerTestTimeout  time.Duration // override per-test timeout (0 = default)
	PerSuiteTimeout time.Duration // override per-suite timeout (0 = default)
}

// Runner orchestrates test suite execution by loading test scripts,
// matching test functions to requests via key patterns, executing
// lifecycle hooks in the correct order, and managing the dual timeout
// system.
type Runner struct {
	loader TestFileLoader
	runner RequestRunner
	lister CollectionLister
}

// NewRunner creates a Runner with the given dependencies.
func NewRunner(loader TestFileLoader, runner RequestRunner, lister CollectionLister) *Runner {
	return &Runner{
		loader: loader,
		runner: runner,
		lister: lister,
	}
}

// RunSuite executes all tests for a collection.
//
// Execution flow:
//  1. Load test file: .promptman/tests/<collId>.test.js
//  2. Parse module.exports → hook map + test key map
//  3. Execute hooks & tests:
//     beforeAll(pm)
//     ├── forEach request in collection:
//     │   beforeEach(pm)
//     │   ├── Execute request via RequestRunner
//     │   ├── Match request ID → test keys (specific > wildcard > glob)
//     │   ├── Run matched test function(pm)
//     │   afterEach(pm)
//     afterAll(pm)
//  4. Aggregate results → TestResult
func (r *Runner) RunSuite(ctx context.Context, collID, env string, opts *TestOpts) (*testing.TestResult, error) {
	// Load test script.
	source, err := r.loader.Load(collID)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, testing.ErrTestFileNotFound.Wrapf("no test file for collection %s", collID)
		}
		return nil, testing.ErrScriptParse.Wrapf("loading test file for %s: %v", collID, err)
	}

	// List all request paths in the collection.
	requestPaths, err := r.lister.ListRequestPaths(collID)
	if err != nil {
		return nil, fmt.Errorf("listing requests for %s: %w", collID, err)
	}

	// Create sandbox and set up module.exports pattern.
	sb := sandbox.New()
	_ = sb.VM().Set("module", sb.VM().NewObject())
	_, _ = sb.VM().RunString("module.exports = {};")

	// Evaluate the script.
	_, err = sb.Execute(source)
	if err != nil {
		return nil, err
	}

	// Get module.exports and parse.
	exports := sb.VM().Get("module").ToObject(sb.VM()).Get("exports")
	parsed, err := testing.ParseExports(sb.VM(), exports)
	if err != nil {
		return nil, err
	}

	// Set up timeout manager.
	perTest := sandbox.DefaultPerTestTimeout
	perSuite := sandbox.DefaultPerSuiteTimeout
	if opts != nil {
		if opts.PerTestTimeout > 0 {
			perTest = opts.PerTestTimeout
		}
		if opts.PerSuiteTimeout > 0 {
			perSuite = opts.PerSuiteTimeout
		}
	}
	tm := sandbox.NewTimeoutManager(perTest, perSuite)
	suiteCtx, suiteCancel := tm.SuiteContext(ctx)
	defer suiteCancel()

	result := &testing.TestResult{
		RunID:      fmt.Sprintf("%d", time.Now().UnixNano()),
		Collection: collID,
		Env:        env,
	}

	start := time.Now()
	testKeys := parsed.TestKeys()

	// Run beforeAll hook.
	if err := r.runHookInSandbox(suiteCtx, sb, tm, parsed.Hooks.BeforeAll); err != nil {
		return r.finalizeResult(result, start, err)
	}

	// Iterate over all requests in the collection.
	for _, reqPath := range requestPaths {
		// Check if suite context is cancelled (suite timeout).
		if suiteCtx.Err() != nil {
			break
		}

		// Find the best matching test key for this request.
		matchedKey, found := testing.FindBestMatch(testKeys, reqPath)
		if !found {
			// No matching test — mark as skipped.
			result.Tests = append(result.Tests, testing.TestCase{
				Request: reqPath,
				Name:    "(no test)",
				Status:  "skipped",
			})
			result.Summary.Skipped++
			continue
		}

		testFn := parsed.Tests[matchedKey]
		tc := r.executeOneRequest(suiteCtx, sb, tm, parsed, testFn, reqPath, collID, env)
		result.Tests = append(result.Tests, tc)

		// Update summary.
		switch tc.Status {
		case "passed":
			result.Summary.Passed++
		case "failed", "error":
			result.Summary.Failed++
		case "timeout":
			result.Summary.Failed++
		case "skipped":
			result.Summary.Skipped++
		}
	}

	// Run afterAll hook.
	_ = r.runHookInSandbox(suiteCtx, sb, tm, parsed.Hooks.AfterAll)

	// Collect all console output from the sandbox.
	result.Console = sb.Console()
	result.Summary.Total = len(result.Tests)
	result.Summary.Duration = int(time.Since(start).Milliseconds())

	return result, nil
}

// RunSingle executes the test for a single request within a collection.
func (r *Runner) RunSingle(ctx context.Context, collID, reqID, env string) (*testing.TestResult, error) {
	// Load test script.
	source, err := r.loader.Load(collID)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, testing.ErrTestFileNotFound.Wrapf("no test file for collection %s", collID)
		}
		return nil, testing.ErrScriptParse.Wrapf("loading test file for %s: %v", collID, err)
	}

	// Create sandbox and set up module.exports pattern.
	sb := sandbox.New()
	_ = sb.VM().Set("module", sb.VM().NewObject())
	_, _ = sb.VM().RunString("module.exports = {};")

	// Evaluate the script.
	_, err = sb.Execute(source)
	if err != nil {
		return nil, err
	}

	// Get module.exports and parse.
	exports := sb.VM().Get("module").ToObject(sb.VM()).Get("exports")
	parsed, err := testing.ParseExports(sb.VM(), exports)
	if err != nil {
		return nil, err
	}

	result := &testing.TestResult{
		RunID:      fmt.Sprintf("%d", time.Now().UnixNano()),
		Collection: collID,
		Env:        env,
	}

	start := time.Now()
	testKeys := parsed.TestKeys()

	// Find matching test.
	matchedKey, found := testing.FindBestMatch(testKeys, reqID)
	if !found {
		result.Tests = append(result.Tests, testing.TestCase{
			Request: reqID,
			Name:    "(no test)",
			Status:  "skipped",
		})
		result.Summary.Total = 1
		result.Summary.Skipped = 1
		result.Summary.Duration = int(time.Since(start).Milliseconds())
		return result, nil
	}

	tm := sandbox.NewTimeoutManager(sandbox.DefaultPerTestTimeout, sandbox.DefaultPerSuiteTimeout)

	// Run beforeAll hook.
	if err := r.runHookInSandbox(ctx, sb, tm, parsed.Hooks.BeforeAll); err != nil {
		return r.finalizeResult(result, start, err)
	}

	testFn := parsed.Tests[matchedKey]
	tc := r.executeOneRequest(ctx, sb, tm, parsed, testFn, reqID, collID, env)
	result.Tests = append(result.Tests, tc)

	// Run afterAll hook.
	_ = r.runHookInSandbox(ctx, sb, tm, parsed.Hooks.AfterAll)

	switch tc.Status {
	case "passed":
		result.Summary.Passed = 1
	default:
		result.Summary.Failed = 1
	}
	result.Summary.Total = 1
	result.Summary.Duration = int(time.Since(start).Milliseconds())
	result.Console = sb.Console()

	return result, nil
}

// executeOneRequest runs hooks + test for a single request.
func (r *Runner) executeOneRequest(
	ctx context.Context,
	sb *sandbox.Sandbox,
	tm *sandbox.TimeoutManager,
	parsed *testing.ParsedScript,
	testFn goja.Callable,
	reqPath, collID, env string,
) testing.TestCase {
	tc := testing.TestCase{
		Request: reqPath,
		Status:  "passed",
	}
	start := time.Now()

	// Run beforeEach hook.
	if err := r.runHookInSandbox(ctx, sb, tm, parsed.Hooks.BeforeEach); err != nil {
		tc.Status = "error"
		tc.Error = &testing.TestError{Message: fmt.Sprintf("beforeEach hook failed: %v", err)}
		tc.Duration = int(time.Since(start).Milliseconds())
		return tc
	}

	// Execute the HTTP request.
	resp, err := r.runner.Execute(ctx, request.ExecuteInput{
		CollectionID: collID,
		RequestID:    reqPath,
		Environment:  env,
	})

	var snapshot *testing.ResponseSnapshot
	if err != nil {
		tc.Status = "error"
		tc.Error = &testing.TestError{Message: fmt.Sprintf("request execution failed: %v", err)}
		tc.Duration = int(time.Since(start).Milliseconds())
		// Run afterEach even on request error.
		_ = r.runHookInSandbox(ctx, sb, tm, parsed.Hooks.AfterEach)
		return tc
	}

	snapshot = snapshotFromResponse(resp)

	// Reset console for this test to capture per-test output.
	sb.Reset()

	// Create PM instance with the response and inject into sandbox.
	pm := pmapi.NewPM(sb.VM(), resp)
	if err := pm.InjectInto(sb.VM()); err != nil {
		tc.Status = "error"
		tc.Error = &testing.TestError{Message: fmt.Sprintf("injecting pm API: %v", err)}
		tc.Duration = int(time.Since(start).Milliseconds())
		return tc
	}

	// Get the pm object to pass to the test function.
	pmObj := sb.VM().Get("pm")

	// Run the test function with timeout.
	runErr := tm.RunWithTimeout(ctx, sb.VM(), func() error {
		_, err := testFn(goja.Undefined(), pmObj)
		return err
	})

	// Capture per-test console output.
	tc.Console = sb.Console()

	if runErr != nil {
		if testing.IsDomainError(runErr, testing.ErrTestTimeout.Code) {
			tc.Status = "timeout"
			tc.Error = &testing.TestError{Message: runErr.Error()}
		} else {
			tc.Status = "error"
			tc.Error = &testing.TestError{Message: runErr.Error()}
		}
		tc.Response = snapshot // Include response on failure.
		tc.Duration = int(time.Since(start).Milliseconds())
		// Run afterEach even on test error.
		_ = r.runHookInSandbox(ctx, sb, tm, parsed.Hooks.AfterEach)
		return tc
	}

	// Collect test results from pm.test() calls.
	pmTests := pm.Tests()
	if len(pmTests) > 0 {
		// Use pm.test() results.
		tc.Name = pmTests[0].Name
		for _, pt := range pmTests {
			if pt.Status != "passed" {
				tc.Status = pt.Status
				tc.Error = pt.Error
				tc.Response = snapshot // Include response on failure.
				break
			}
		}
	}

	// If test passed, do NOT include the response snapshot.
	if tc.Status == "passed" {
		tc.Response = nil
	}

	tc.Duration = int(time.Since(start).Milliseconds())

	// Run afterEach hook.
	_ = r.runHookInSandbox(ctx, sb, tm, parsed.Hooks.AfterEach)

	return tc
}

// runHookInSandbox runs a lifecycle hook within the sandbox with timeout.
func (r *Runner) runHookInSandbox(
	ctx context.Context,
	sb *sandbox.Sandbox,
	tm *sandbox.TimeoutManager,
	hook goja.Callable,
) error {
	if hook == nil {
		return nil
	}

	pmObj := sb.VM().Get("pm")
	if pmObj == nil || goja.IsUndefined(pmObj) {
		pmObj = goja.Undefined()
	}

	return tm.RunWithTimeout(ctx, sb.VM(), func() error {
		return testing.RunHook(sb.VM(), hook, pmObj)
	})
}

// snapshotFromResponse creates a ResponseSnapshot from a request.Response.
func snapshotFromResponse(resp *request.Response) *testing.ResponseSnapshot {
	if resp == nil {
		return nil
	}
	totalTime := 0
	if resp.Timing != nil {
		totalTime = resp.Timing.Total
	}
	return &testing.ResponseSnapshot{
		Status:  resp.Status,
		Headers: resp.Headers,
		Body:    resp.Body,
		Time:    totalTime,
	}
}

// finalizeResult sets the summary duration and returns the result on error.
func (r *Runner) finalizeResult(result *testing.TestResult, start time.Time, err error) (*testing.TestResult, error) {
	result.Summary.Duration = int(time.Since(start).Milliseconds())
	return result, err
}
