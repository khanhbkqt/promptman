package testing

// TestResult holds the complete result of a test suite or single test execution.
type TestResult struct {
	RunID      string      `json:"runId"`             // unique identifier for this test run
	Collection string      `json:"collection"`        // collection that was tested
	Env        string      `json:"env"`               // environment used
	Summary    TestSummary `json:"summary"`           // aggregate pass/fail/skip counts
	Tests      []TestCase  `json:"tests"`             // individual test results
	Console    []string    `json:"console,omitempty"` // captured console.log output
}

// TestSummary holds aggregate counts for a test run.
type TestSummary struct {
	Total    int `json:"total"`    // total number of tests
	Passed   int `json:"passed"`   // tests that passed
	Failed   int `json:"failed"`   // tests that failed
	Skipped  int `json:"skipped"`  // tests that were skipped
	Duration int `json:"duration"` // total duration in milliseconds
}

// TestCase holds the result of a single test case.
type TestCase struct {
	Request  string            `json:"request"`            // request ID that was tested
	Name     string            `json:"name"`               // test case name from pm.test()
	Status   string            `json:"status"`             // passed | failed | timeout | error | skipped
	Duration int               `json:"duration"`           // duration in milliseconds
	Error    *TestError        `json:"error,omitempty"`    // assertion/execution error details
	Response *ResponseSnapshot `json:"response,omitempty"` // response snapshot (failed tests only)
	Console  []string          `json:"console,omitempty"`  // console output captured during this test
}

// TestError holds details about a test failure.
type TestError struct {
	Expected any    `json:"expected"` // expected value from assertion
	Actual   any    `json:"actual"`   // actual value from assertion
	Message  string `json:"message"`  // human-readable error description
}

// ResponseSnapshot is a lightweight copy of a request response
// attached to failed test cases for debugging.
type ResponseSnapshot struct {
	Status  int               `json:"status"`            // HTTP status code
	Headers map[string]string `json:"headers,omitempty"` // response headers
	Body    string            `json:"body,omitempty"`    // response body
	Time    int               `json:"time"`              // total request time in ms
}
