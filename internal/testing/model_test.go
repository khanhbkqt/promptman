package testing

import (
	"encoding/json"
	"testing"
)

func TestTestResult_JSONRoundTrip(t *testing.T) {
	result := TestResult{
		RunID:      "run-001",
		Collection: "users",
		Env:        "staging",
		Summary: TestSummary{
			Total:    5,
			Passed:   3,
			Failed:   1,
			Skipped:  1,
			Duration: 1500,
		},
		Tests: []TestCase{
			{
				Request:  "get-user",
				Name:     "should return 200",
				Status:   "passed",
				Duration: 120,
			},
			{
				Request:  "create-user",
				Name:     "should validate email",
				Status:   "failed",
				Duration: 80,
				Error: &TestError{
					Expected: 201,
					Actual:   400,
					Message:  "expected status 201 but got 400",
				},
			},
		},
		Console: []string{"debug: user fetched"},
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var decoded TestResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	if decoded.RunID != "run-001" {
		t.Errorf("RunID = %q, want %q", decoded.RunID, "run-001")
	}
	if decoded.Summary.Total != 5 {
		t.Errorf("Summary.Total = %d, want 5", decoded.Summary.Total)
	}
	if len(decoded.Tests) != 2 {
		t.Fatalf("len(Tests) = %d, want 2", len(decoded.Tests))
	}
	if decoded.Tests[1].Error == nil {
		t.Fatal("Tests[1].Error should not be nil")
	}
	if decoded.Tests[1].Error.Message != "expected status 201 but got 400" {
		t.Errorf("Error.Message = %q, want %q", decoded.Tests[1].Error.Message, "expected status 201 but got 400")
	}
}

func TestTestResult_JSONTags(t *testing.T) {
	result := TestResult{
		RunID:      "r1",
		Collection: "c1",
		Env:        "dev",
		Summary:    TestSummary{Total: 1, Passed: 1},
		Tests:      []TestCase{{Request: "req1", Name: "t1", Status: "passed"}},
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	// Verify top-level JSON keys match M4 spec
	expectedKeys := []string{"runId", "collection", "env", "summary", "tests"}
	for _, key := range expectedKeys {
		if _, ok := raw[key]; !ok {
			t.Errorf("missing expected JSON key %q", key)
		}
	}

	// console should be omitted when nil
	if _, ok := raw["console"]; ok {
		t.Error("expected console to be omitted when nil")
	}
}

func TestTestCase_OmitsErrorWhenNil(t *testing.T) {
	tc := TestCase{
		Request:  "get-health",
		Name:     "health check",
		Status:   "passed",
		Duration: 10,
	}

	data, err := json.Marshal(tc)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	if _, ok := raw["error"]; ok {
		t.Error("expected error to be omitted when nil")
	}
}

func TestTestSummary_JSONRoundTrip(t *testing.T) {
	summary := TestSummary{
		Total:    10,
		Passed:   7,
		Failed:   2,
		Skipped:  1,
		Duration: 3200,
	}

	data, err := json.Marshal(summary)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var decoded TestSummary
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	if decoded != summary {
		t.Errorf("decoded = %+v, want %+v", decoded, summary)
	}
}

func TestTestError_JSONRoundTrip(t *testing.T) {
	te := TestError{
		Expected: "hello",
		Actual:   "world",
		Message:  "values do not match",
	}

	data, err := json.Marshal(te)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var decoded TestError
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	if decoded.Message != "values do not match" {
		t.Errorf("Message = %q, want %q", decoded.Message, "values do not match")
	}
}
