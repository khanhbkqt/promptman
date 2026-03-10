package core

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/khanhnguyen/promptman/internal/request"
	tmod "github.com/khanhnguyen/promptman/internal/testing"
)

// --- Mock dependencies ---

type mockFileLoader struct {
	scripts map[string]string
}

func (m *mockFileLoader) Load(collID string) (string, error) {
	s, ok := m.scripts[collID]
	if !ok {
		return "", errors.New("test file not found")
	}
	return s, nil
}

type mockRequestRunner struct {
	responses map[string]*request.Response
	errors    map[string]error
}

func (m *mockRequestRunner) Execute(_ context.Context, input request.ExecuteInput) (*request.Response, error) {
	if e, ok := m.errors[input.RequestID]; ok {
		return nil, e
	}
	resp, ok := m.responses[input.RequestID]
	if !ok {
		return &request.Response{
			RequestID: input.RequestID,
			Status:    200,
			Headers:   map[string]string{"Content-Type": "application/json"},
			Body:      `{"ok":true}`,
			Timing:    &request.RequestTiming{Total: 42},
		}, nil
	}
	return resp, nil
}

type mockCollectionLister struct {
	paths map[string][]string
}

func (m *mockCollectionLister) ListRequestPaths(collID string) ([]string, error) {
	p, ok := m.paths[collID]
	if !ok {
		return nil, errors.New("collection not found")
	}
	return p, nil
}

// --- Helper to create module.exports script ---

const basicTestScript = `
module.exports = {
	"users/list": function(pm) {
		pm.test("status is 200", function() {
			pm.expect(pm.response.status).to.equal(200);
		});
	},
	"health": function(pm) {
		pm.test("health check passes", function() {
			pm.expect(pm.response.status).to.equal(200);
		});
	},
};
`

const scriptWithHooks = `
module.exports = {
	beforeAll: function(pm) {
		console.log("running beforeAll");
	},
	afterAll: function(pm) {
		console.log("running afterAll");
	},
	beforeEach: function(pm) {
		console.log("running beforeEach");
	},
	afterEach: function(pm) {
		console.log("running afterEach");
	},
	"users/list": function(pm) {
		console.log("running users/list");
		pm.test("status is 200", function() {
			console.log("running test inner");
			pm.expect(pm.response.status).to.equal(200);
		});
	},
};
`

const scriptWithFailingTest = `
module.exports = {
	"users/list": function(pm) {
		pm.test("status should be 404", function() {
			pm.expect(pm.response.status).to.equal(404);
		});
	},
};
`

const scriptWithWildcard = `
module.exports = {
	"admin/*": function(pm) {
		pm.test("admin endpoint", function() {
			pm.expect(pm.response.status).to.equal(200);
		});
	},
	"health": function(pm) {
		pm.test("health check", function() {
			pm.expect(pm.response.status).to.equal(200);
		});
	},
};
`

const scriptWithConsole = `
module.exports = {
	"health": function(pm) {
		console.log("running health test");
		pm.test("health check", function() {
			pm.expect(pm.response.status).to.equal(200);
		});
	},
};
`

// --- Tests ---

func newTestRunner(scripts map[string]string, paths map[string][]string) *Runner {
	return NewRunner(
		&mockFileLoader{scripts: scripts},
		&mockRequestRunner{responses: make(map[string]*request.Response)},
		&mockCollectionLister{paths: paths},
	)
}

func TestRunSuite_BasicPassingTests(t *testing.T) {
	r := newTestRunner(
		map[string]string{"api": basicTestScript},
		map[string][]string{"api": {"users/list", "health"}},
	)

	result, err := r.RunSuite(context.Background(), "api", "dev", nil)
	if err != nil {
		t.Fatalf("RunSuite error: %v", err)
	}

	if result.Collection != "api" {
		t.Errorf("expected collection 'api', got %q", result.Collection)
	}
	if result.Env != "dev" {
		t.Errorf("expected env 'dev', got %q", result.Env)
	}
	if result.Summary.Total != 2 {
		t.Errorf("expected 2 total tests, got %d", result.Summary.Total)
	}
	if result.Summary.Passed != 2 {
		t.Errorf("expected 2 passed, got %d", result.Summary.Passed)
	}
	if result.Summary.Failed != 0 {
		t.Errorf("expected 0 failed, got %d", result.Summary.Failed)
	}
}

func TestRunSuite_FailedTestIncludesResponse(t *testing.T) {
	r := newTestRunner(
		map[string]string{"api": scriptWithFailingTest},
		map[string][]string{"api": {"users/list"}},
	)

	result, err := r.RunSuite(context.Background(), "api", "dev", nil)
	if err != nil {
		t.Fatalf("RunSuite error: %v", err)
	}

	if result.Summary.Failed != 1 {
		t.Fatalf("expected 1 failed, got %d", result.Summary.Failed)
	}

	tc := result.Tests[0]
	if tc.Status != "failed" {
		t.Errorf("expected status 'failed', got %q", tc.Status)
	}
	if tc.Response == nil {
		t.Error("expected response snapshot on failed test")
	}
	if tc.Response != nil && tc.Response.Status != 200 {
		t.Errorf("expected response status 200, got %d", tc.Response.Status)
	}
}

func TestRunSuite_PassedTestOmitsResponse(t *testing.T) {
	r := newTestRunner(
		map[string]string{"api": basicTestScript},
		map[string][]string{"api": {"health"}},
	)

	result, err := r.RunSuite(context.Background(), "api", "dev", nil)
	if err != nil {
		t.Fatalf("RunSuite error: %v", err)
	}

	tc := result.Tests[0]
	if tc.Status != "passed" {
		t.Errorf("expected status 'passed', got %q", tc.Status)
	}
	if tc.Response != nil {
		t.Error("expected no response snapshot on passed test")
	}
}

func TestRunSuite_SkippedTests(t *testing.T) {
	r := newTestRunner(
		map[string]string{"api": basicTestScript},
		map[string][]string{"api": {"users/list", "admin/settings"}},
	)

	result, err := r.RunSuite(context.Background(), "api", "dev", nil)
	if err != nil {
		t.Fatalf("RunSuite error: %v", err)
	}

	if result.Summary.Skipped != 1 {
		t.Errorf("expected 1 skipped, got %d", result.Summary.Skipped)
	}

	// Find the skipped test.
	found := false
	for _, tc := range result.Tests {
		if tc.Request == "admin/settings" {
			found = true
			if tc.Status != "skipped" {
				t.Errorf("expected status 'skipped', got %q", tc.Status)
			}
		}
	}
	if !found {
		t.Error("expected skipped test for admin/settings")
	}
}

func TestRunSuite_WildcardMatching(t *testing.T) {
	r := newTestRunner(
		map[string]string{"api": scriptWithWildcard},
		map[string][]string{"api": {"health", "admin/list", "admin/get"}},
	)

	result, err := r.RunSuite(context.Background(), "api", "dev", nil)
	if err != nil {
		t.Fatalf("RunSuite error: %v", err)
	}

	if result.Summary.Total != 3 {
		t.Errorf("expected 3 total tests, got %d", result.Summary.Total)
	}
	if result.Summary.Passed != 3 {
		t.Errorf("expected 3 passed, got %d", result.Summary.Passed)
	}
}

func TestRunSuite_WithHooks(t *testing.T) {
	r := newTestRunner(
		map[string]string{"api": scriptWithHooks},
		map[string][]string{"api": {"users/list"}},
	)

	result, err := r.RunSuite(context.Background(), "api", "dev", nil)
	if err != nil {
		t.Fatalf("RunSuite error: %v", err)
	}

	if result.Summary.Total != 1 {
		t.Errorf("expected 1 total test, got %d", result.Summary.Total)
	}
	if result.Summary.Passed != 1 {
		errMsg := ""
		console := []string{}
		duration := -1
		if len(result.Tests) > 0 {
			if result.Tests[0].Error != nil {
				errMsg = result.Tests[0].Error.Message
			}
			console = result.Tests[0].Console
			duration = result.Tests[0].Duration
		}
		t.Errorf("expected 1 passed, got %d. Test case status: %s, duration: %d, err: %s, console: %v", result.Summary.Passed, result.Tests[0].Status, duration, errMsg, console)
	}
}

func TestRunSuite_ConsoleCapture(t *testing.T) {
	r := newTestRunner(
		map[string]string{"api": scriptWithConsole},
		map[string][]string{"api": {"health"}},
	)

	result, err := r.RunSuite(context.Background(), "api", "dev", nil)
	if err != nil {
		t.Fatalf("RunSuite error: %v", err)
	}

	// Console output should be captured at test level.
	tc := result.Tests[0]
	if len(tc.Console) == 0 {
		t.Error("expected console output from test")
	}
}

func TestRunSuite_TestFileNotFound(t *testing.T) {
	r := newTestRunner(
		map[string]string{},
		map[string][]string{"api": {"health"}},
	)

	_, err := r.RunSuite(context.Background(), "api", "dev", nil)
	if err == nil {
		t.Fatal("expected error for missing test file")
	}
}

func TestRunSuite_SuiteTimeout(t *testing.T) {
	slowScript := `
		module.exports = {
			"health": function(pm) {
				var start = Date.now();
				while (Date.now() - start < 5000) {}
				pm.test("slow test", function() {});
			},
		};
	`
	r := newTestRunner(
		map[string]string{"api": slowScript},
		map[string][]string{"api": {"health"}},
	)

	opts := &TestOpts{
		PerTestTimeout:  100 * time.Millisecond,
		PerSuiteTimeout: 200 * time.Millisecond,
	}
	result, err := r.RunSuite(context.Background(), "api", "dev", opts)
	if err != nil {
		t.Fatalf("RunSuite error: %v", err)
	}

	if result.Summary.Total != 1 {
		t.Fatalf("expected 1 total test, got %d", result.Summary.Total)
	}
	tc := result.Tests[0]
	if tc.Status != "timeout" {
		t.Errorf("expected status 'timeout', got %q", tc.Status)
	}
}

func TestRunSingle_BasicPassing(t *testing.T) {
	r := newTestRunner(
		map[string]string{"api": basicTestScript},
		map[string][]string{"api": {"users/list", "health"}},
	)

	result, err := r.RunSingle(context.Background(), "api", "health", "dev")
	if err != nil {
		t.Fatalf("RunSingle error: %v", err)
	}

	if result.Summary.Total != 1 {
		t.Errorf("expected 1 total, got %d", result.Summary.Total)
	}
	if result.Summary.Passed != 1 {
		t.Errorf("expected 1 passed, got %d", result.Summary.Passed)
	}
}

func TestRunSingle_NoMatchReturnsSkipped(t *testing.T) {
	r := newTestRunner(
		map[string]string{"api": basicTestScript},
		map[string][]string{"api": {"users/list", "health"}},
	)

	result, err := r.RunSingle(context.Background(), "api", "admin/settings", "dev")
	if err != nil {
		t.Fatalf("RunSingle error: %v", err)
	}

	if result.Summary.Total != 1 {
		t.Errorf("expected 1 total, got %d", result.Summary.Total)
	}
	if result.Summary.Skipped != 1 {
		t.Errorf("expected 1 skipped, got %d", result.Summary.Skipped)
	}
}

func TestRunSingle_RequestExecutionError(t *testing.T) {
	runner := NewRunner(
		&mockFileLoader{scripts: map[string]string{"api": basicTestScript}},
		&mockRequestRunner{
			responses: make(map[string]*request.Response),
			errors:    map[string]error{"health": errors.New("connection refused")},
		},
		&mockCollectionLister{paths: map[string][]string{"api": {"health"}}},
	)

	result, err := runner.RunSingle(context.Background(), "api", "health", "dev")
	if err != nil {
		t.Fatalf("RunSingle error: %v", err)
	}

	tc := result.Tests[0]
	if tc.Status != "error" {
		t.Errorf("expected status 'error', got %q", tc.Status)
	}
	if tc.Error == nil {
		t.Fatal("expected error details on failed request")
	}
}

func TestSnapshotFromResponse_Nil(t *testing.T) {
	s := snapshotFromResponse(nil)
	if s != nil {
		t.Error("expected nil snapshot for nil response")
	}
}

func TestSnapshotFromResponse_WithTiming(t *testing.T) {
	resp := &request.Response{
		Status:  404,
		Headers: map[string]string{"X-Custom": "val"},
		Body:    `{"error":"not found"}`,
		Timing:  &request.RequestTiming{Total: 123},
	}
	s := snapshotFromResponse(resp)
	if s == nil {
		t.Fatal("expected non-nil snapshot")
	}
	if s.Status != 404 {
		t.Errorf("expected status 404, got %d", s.Status)
	}
	if s.Time != 123 {
		t.Errorf("expected time 123, got %d", s.Time)
	}
}

func TestSnapshotFromResponse_NilTiming(t *testing.T) {
	resp := &request.Response{
		Status: 200,
		Body:   "ok",
	}
	s := snapshotFromResponse(resp)
	if s.Time != 0 {
		t.Errorf("expected time 0 for nil timing, got %d", s.Time)
	}
}

func TestRunSuite_EmptyCollection(t *testing.T) {
	r := newTestRunner(
		map[string]string{"api": basicTestScript},
		map[string][]string{"api": {}},
	)

	result, err := r.RunSuite(context.Background(), "api", "dev", nil)
	if err != nil {
		t.Fatalf("RunSuite error: %v", err)
	}

	if result.Summary.Total != 0 {
		t.Errorf("expected 0 total tests for empty collection, got %d", result.Summary.Total)
	}
}

func TestRunSuite_HasRunID(t *testing.T) {
	r := newTestRunner(
		map[string]string{"api": basicTestScript},
		map[string][]string{"api": {"health"}},
	)

	result, err := r.RunSuite(context.Background(), "api", "dev", nil)
	if err != nil {
		t.Fatalf("RunSuite error: %v", err)
	}

	if result.RunID == "" {
		t.Error("expected non-empty RunID")
	}
}

// Verify that snapshotFromResponse produces a valid tmod.ResponseSnapshot.
func TestSnapshotTypes(t *testing.T) {
	resp := &request.Response{
		Status:  200,
		Headers: map[string]string{"Content-Type": "application/json"},
		Body:    `{"ok":true}`,
		Timing:  &request.RequestTiming{Total: 42},
	}
	s := snapshotFromResponse(resp)

	// Verify it can be assigned to the model type.
	var _ *tmod.ResponseSnapshot = s
	if s.Status != 200 {
		t.Errorf("expected 200, got %d", s.Status)
	}
}
