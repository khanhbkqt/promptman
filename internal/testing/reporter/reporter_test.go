package reporter

import (
	"encoding/json"
	"strings"
	"testing"

	tmod "github.com/khanhnguyen/promptman/internal/testing"
)

// --- Shared test fixtures ---

func fixtureResult() *tmod.TestResult {
	return &tmod.TestResult{
		RunID:      "run-001",
		Collection: "users",
		Env:        "dev",
		Summary: tmod.TestSummary{
			Total:    4,
			Passed:   2,
			Failed:   1,
			Skipped:  1,
			Duration: 1245,
		},
		Tests: []tmod.TestCase{
			{
				Request:  "users/list",
				Name:     "status is 200",
				Status:   "passed",
				Duration: 42,
			},
			{
				Request:  "users/create",
				Name:     "creates user",
				Status:   "passed",
				Duration: 103,
			},
			{
				Request:  "users/delete",
				Name:     "returns 404",
				Status:   "failed",
				Duration: 55,
				Error: &tmod.TestError{
					Expected: 404,
					Actual:   200,
					Message:  "expected status 404 but got 200",
				},
				Response: &tmod.ResponseSnapshot{
					Status: 200,
					Body:   `{"ok":true}`,
					Time:   55,
				},
			},
			{
				Request: "users/admin",
				Name:    "admin check",
				Status:  "skipped",
			},
		},
		Console: []string{"debug: starting tests", "info: all set"},
	}
}

func fixtureAllPassing() *tmod.TestResult {
	return &tmod.TestResult{
		RunID:      "run-002",
		Collection: "health",
		Env:        "",
		Summary: tmod.TestSummary{
			Total:    2,
			Passed:   2,
			Failed:   0,
			Skipped:  0,
			Duration: 50,
		},
		Tests: []tmod.TestCase{
			{Request: "health", Name: "is 200", Status: "passed", Duration: 25},
			{Request: "ready", Name: "is ready", Status: "passed", Duration: 25},
		},
	}
}

// --- ForFormat tests ---

func TestForFormat_ValidFormats(t *testing.T) {
	formats := []string{"json", "table", "tap", "minimal"}
	for _, f := range formats {
		t.Run(f, func(t *testing.T) {
			r, err := ForFormat(f)
			if err != nil {
				t.Fatalf("ForFormat(%q) error: %v", f, err)
			}
			if r == nil {
				t.Fatalf("ForFormat(%q) returned nil", f)
			}
		})
	}
}

func TestForFormat_Unknown(t *testing.T) {
	_, err := ForFormat("html")
	if err == nil {
		t.Fatal("expected error for unknown format")
	}
	if !strings.Contains(err.Error(), "unknown report format") {
		t.Errorf("unexpected error: %v", err)
	}
}

// --- JSON Reporter tests ---

func TestJSONReporter_ValidJSON(t *testing.T) {
	r := &JSONReporter{}
	result := fixtureResult()
	out, err := r.Format(result)
	if err != nil {
		t.Fatalf("Format error: %v", err)
	}

	if !json.Valid(out) {
		t.Errorf("output is not valid JSON:\n%s", string(out))
	}
}

func TestJSONReporter_ContainsFields(t *testing.T) {
	r := &JSONReporter{}
	result := fixtureResult()
	out, err := r.Format(result)
	if err != nil {
		t.Fatalf("Format error: %v", err)
	}

	s := string(out)
	checks := []string{
		`"runId"`, `"run-001"`,
		`"collection"`, `"users"`,
		`"console"`, `"debug: starting tests"`,
		`"passed"`, `"failed"`, `"skipped"`,
		`"response"`, `"status"`,
	}
	for _, c := range checks {
		if !strings.Contains(s, c) {
			t.Errorf("JSON output missing %s", c)
		}
	}
}

func TestJSONReporter_RoundTrip(t *testing.T) {
	r := &JSONReporter{}
	result := fixtureResult()
	out, err := r.Format(result)
	if err != nil {
		t.Fatalf("Format error: %v", err)
	}

	var decoded tmod.TestResult
	if err := json.Unmarshal(out, &decoded); err != nil {
		t.Fatalf("cannot unmarshal JSON output: %v", err)
	}

	if decoded.RunID != "run-001" {
		t.Errorf("RunID = %q; want %q", decoded.RunID, "run-001")
	}
	if decoded.Summary.Total != 4 {
		t.Errorf("Total = %d; want 4", decoded.Summary.Total)
	}
	if len(decoded.Tests) != 4 {
		t.Errorf("len(Tests) = %d; want 4", len(decoded.Tests))
	}
}

// --- Table Reporter tests ---

func TestTableReporter_ContainsIcons(t *testing.T) {
	// Override terminal detection for deterministic output.
	isTerminalFn = func() bool { return false }
	defer func() { isTerminalFn = nil }()

	r := &TableReporter{}
	out, err := r.Format(fixtureResult())
	if err != nil {
		t.Fatalf("Format error: %v", err)
	}
	s := string(out)

	if !strings.Contains(s, "✓") {
		t.Error("table output missing pass icon ✓")
	}
	if !strings.Contains(s, "✗") {
		t.Error("table output missing fail icon ✗")
	}
	if !strings.Contains(s, "○") {
		t.Error("table output missing skip icon ○")
	}
}

func TestTableReporter_SummaryLine(t *testing.T) {
	isTerminalFn = func() bool { return false }
	defer func() { isTerminalFn = nil }()

	tests := []struct {
		name     string
		result   *tmod.TestResult
		contains string
	}{
		{
			name:     "with failures",
			result:   fixtureResult(),
			contains: "✗ 2/4 passed, 1 failed (1245ms)",
		},
		{
			name:     "all passing",
			result:   fixtureAllPassing(),
			contains: "✓ 2/2 passed (50ms)",
		},
	}

	r := &TableReporter{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := r.Format(tt.result)
			if err != nil {
				t.Fatalf("Format error: %v", err)
			}
			if !strings.Contains(string(out), tt.contains) {
				t.Errorf("table output missing summary %q\n%s", tt.contains, string(out))
			}
		})
	}
}

func TestTableReporter_ColorEnabled(t *testing.T) {
	isTerminalFn = func() bool { return true }
	defer func() { isTerminalFn = nil }()

	r := &TableReporter{}
	out, err := r.Format(fixtureResult())
	if err != nil {
		t.Fatalf("Format error: %v", err)
	}

	if !strings.Contains(string(out), colorGreen) {
		t.Error("expected ANSI green codes when terminal = true")
	}
	if !strings.Contains(string(out), colorRed) {
		t.Error("expected ANSI red codes when terminal = true")
	}
}

func TestTableReporter_NoColorWithoutTerminal(t *testing.T) {
	isTerminalFn = func() bool { return false }
	defer func() { isTerminalFn = nil }()

	r := &TableReporter{}
	out, err := r.Format(fixtureResult())
	if err != nil {
		t.Fatalf("Format error: %v", err)
	}

	if strings.Contains(string(out), "\033[") {
		t.Error("ANSI escape codes found when terminal = false")
	}
}

func TestTableReporter_ErrorTruncation(t *testing.T) {
	isTerminalFn = func() bool { return false }
	defer func() { isTerminalFn = nil }()

	result := &tmod.TestResult{
		RunID:      "run-trunc",
		Collection: "trunc",
		Summary:    tmod.TestSummary{Total: 1, Failed: 1, Duration: 10},
		Tests: []tmod.TestCase{
			{
				Request:  "test",
				Name:     "long error",
				Status:   "failed",
				Duration: 10,
				Error: &tmod.TestError{
					Message: strings.Repeat("x", 200),
				},
			},
		},
	}

	r := &TableReporter{}
	out, err := r.Format(result)
	if err != nil {
		t.Fatalf("Format error: %v", err)
	}

	if strings.Contains(string(out), strings.Repeat("x", 200)) {
		t.Error("full 200-char error message not truncated")
	}
	if !strings.Contains(string(out), "…") {
		t.Error("truncated error should end with …")
	}
}

func TestTableReporter_FailedTestShowsBody(t *testing.T) {
	isTerminalFn = func() bool { return false }
	defer func() { isTerminalFn = nil }()

	r := &TableReporter{}
	out, err := r.Format(fixtureResult())
	if err != nil {
		t.Fatalf("Format error: %v", err)
	}
	if !strings.Contains(string(out), `{"ok":true}`) {
		t.Error("table output should show response body for failed tests")
	}
}

// --- TAP Reporter tests ---

func TestTAPReporter_PlanLine(t *testing.T) {
	r := &TAPReporter{}
	out, err := r.Format(fixtureResult())
	if err != nil {
		t.Fatalf("Format error: %v", err)
	}

	lines := strings.Split(string(out), "\n")
	if len(lines) == 0 {
		t.Fatal("empty output")
	}
	if lines[0] != "1..4" {
		t.Errorf("first line = %q; want %q", lines[0], "1..4")
	}
}

func TestTAPReporter_OkNotOk(t *testing.T) {
	r := &TAPReporter{}
	out, err := r.Format(fixtureResult())
	if err != nil {
		t.Fatalf("Format error: %v", err)
	}
	s := string(out)

	if !strings.Contains(s, "ok 1 users/list - status is 200") {
		t.Error("missing ok line for passing test")
	}
	if !strings.Contains(s, "not ok 3 users/delete - returns 404") {
		t.Error("missing not ok line for failed test")
	}
}

func TestTAPReporter_SkipDirective(t *testing.T) {
	r := &TAPReporter{}
	out, err := r.Format(fixtureResult())
	if err != nil {
		t.Fatalf("Format error: %v", err)
	}
	if !strings.Contains(string(out), "# SKIP") {
		t.Error("skipped test should have SKIP directive")
	}
}

func TestTAPReporter_DiagnosticBlock(t *testing.T) {
	r := &TAPReporter{}
	out, err := r.Format(fixtureResult())
	if err != nil {
		t.Fatalf("Format error: %v", err)
	}
	s := string(out)

	if !strings.Contains(s, "  ---") {
		t.Error("missing YAML diagnostic start marker")
	}
	if !strings.Contains(s, "  ...") {
		t.Error("missing YAML diagnostic end marker")
	}
	if !strings.Contains(s, "message:") {
		t.Error("diagnostic missing message field")
	}
	if !strings.Contains(s, "expected:") {
		t.Error("diagnostic missing expected field")
	}
	if !strings.Contains(s, "response:") {
		t.Error("diagnostic missing response block")
	}
}

func TestTAPReporter_AllPassing(t *testing.T) {
	r := &TAPReporter{}
	out, err := r.Format(fixtureAllPassing())
	if err != nil {
		t.Fatalf("Format error: %v", err)
	}
	s := string(out)

	if strings.Contains(s, "not ok") {
		t.Error("all-passing result should not contain 'not ok'")
	}
	if strings.Contains(s, "---") {
		t.Error("all-passing result should not contain diagnostic blocks")
	}
}

// --- Minimal Reporter tests ---

func TestMinimalReporter_AllPassing(t *testing.T) {
	r := &MinimalReporter{}
	out, err := r.Format(fixtureAllPassing())
	if err != nil {
		t.Fatalf("Format error: %v", err)
	}
	expected := "✓ 2/2 passed (50ms)\n"
	if string(out) != expected {
		t.Errorf("output = %q; want %q", string(out), expected)
	}
}

func TestMinimalReporter_WithFailures(t *testing.T) {
	r := &MinimalReporter{}
	out, err := r.Format(fixtureResult())
	if err != nil {
		t.Fatalf("Format error: %v", err)
	}
	expected := "✗ 2/4 passed, 1 failed (1245ms)\n"
	if string(out) != expected {
		t.Errorf("output = %q; want %q", string(out), expected)
	}
}

// --- Reporter interface compliance ---

func TestAllReporters_ImplementInterface(t *testing.T) {
	var _ Reporter = &JSONReporter{}
	var _ Reporter = &TableReporter{}
	var _ Reporter = &TAPReporter{}
	var _ Reporter = &MinimalReporter{}
}

func TestAllReporters_NoError(t *testing.T) {
	reporters := []struct {
		name string
		r    Reporter
	}{
		{"json", &JSONReporter{}},
		{"table", &TableReporter{}},
		{"tap", &TAPReporter{}},
		{"minimal", &MinimalReporter{}},
	}

	// Override terminal detection.
	isTerminalFn = func() bool { return false }
	defer func() { isTerminalFn = nil }()

	for _, rr := range reporters {
		t.Run(rr.name, func(t *testing.T) {
			out, err := rr.r.Format(fixtureResult())
			if err != nil {
				t.Fatalf("%s Format error: %v", rr.name, err)
			}
			if len(out) == 0 {
				t.Errorf("%s Format returned empty output", rr.name)
			}
		})
	}
}
