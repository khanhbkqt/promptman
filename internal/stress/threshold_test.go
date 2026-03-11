package stress

import (
	"testing"
)

// ─── ParseThreshold ──────────────────────────────────────────────────────────

func TestParseThreshold_ValidExpressions(t *testing.T) {
	tests := []struct {
		name       string
		expr       string
		wantMetric string
		wantOp     string
		wantVal    float64
	}{
		// Latency with milliseconds
		{"p95 ms", "p95<500ms", "p95", "<", 500.0},
		{"p99 ms", "p99<=750ms", "p99", "<=", 750.0},
		{"p50 ms", "p50<100ms", "p50", "<", 100.0},
		// Latency with seconds
		{"p95 1s", "p95<1s", "p95", "<", 1000.0},
		{"p99 2.5s", "p99<=2.5s", "p99", "<=", 2500.0},
		// Error rate with percentage
		{"error_rate lt", "error_rate<5%", "error_rate", "<", 5.0},
		{"error_rate lte", "error_rate<=10%", "error_rate", "<=", 10.0},
		// RPS plain number
		{"rps gt", "rps>100", "rps", ">", 100.0},
		{"rps gte", "rps>=200", "rps", ">=", 200.0},
		// Throughput
		{"throughput gt", "throughput>1000", "throughput", ">", 1000.0},
		// Whitespace tolerance
		{"spaces", "p95 < 500ms", "p95", "<", 500.0},
		// Float milliseconds
		{"float ms", "p95<500.5ms", "p95", "<", 500.5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseThreshold(tt.expr)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Metric != tt.wantMetric {
				t.Errorf("metric = %q, want %q", got.Metric, tt.wantMetric)
			}
			if got.Operator != tt.wantOp {
				t.Errorf("operator = %q, want %q", got.Operator, tt.wantOp)
			}
			if got.Value != tt.wantVal {
				t.Errorf("value = %f, want %f", got.Value, tt.wantVal)
			}
		})
	}
}

func TestParseThreshold_Errors(t *testing.T) {
	tests := []struct {
		name string
		expr string
	}{
		{"empty", ""},
		{"whitespace only", "   "},
		{"no operator", "p95500ms"},
		{"unknown metric", "latency<500ms"},
		{"bad millisecond value", "p95<abcms"},
		{"bad second value", "p95<xyzs"},
		{"bad percentage", "error_rate<abc%"},
		{"bad plain number", "rps>abc"},
		{"empty metric", "<500ms"},
		{"empty value", "p95<"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseThreshold(tt.expr)
			if err == nil {
				t.Fatalf("expected error for expr %q, got nil", tt.expr)
			}
		})
	}
}

// ─── ParseThresholds ──────────────────────────────────────────────────────────

func TestParseThresholds_ValidSlice(t *testing.T) {
	exprs := []string{"p95<500ms", "error_rate<5%", "rps>100"}
	got, err := ParseThresholds(exprs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 3 {
		t.Errorf("len = %d, want 3", len(got))
	}
}

func TestParseThresholds_EmptySlice(t *testing.T) {
	got, err := ParseThresholds(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("len = %d, want 0", len(got))
	}
}

func TestParseThresholds_StopOnFirstError(t *testing.T) {
	exprs := []string{"p95<500ms", "bad expression", "rps>100"}
	_, err := ParseThresholds(exprs)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ─── EvaluateThresholds ───────────────────────────────────────────────────────

func makeSummary() StressSummary {
	return StressSummary{
		TotalRequests: 1000,
		RPS:           150.0,
		Latency: LatencyMetrics{
			P50: 120,
			P95: 450,
			P99: 800,
		},
		ErrorRate:  2.5,
		Throughput: 5000,
	}
}

func TestEvaluateThresholds_AllPass(t *testing.T) {
	summary := makeSummary()
	thresholds, err := ParseThresholds([]string{
		"p95<500ms",     // actual=450, limit=500 → pass
		"error_rate<5%", // actual=2.5, limit=5 → pass
		"rps>100",       // actual=150, limit=100 → pass
	})
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	results, allPassed := EvaluateThresholds(thresholds, summary)
	if !allPassed {
		t.Error("allPassed = false, want true")
	}
	if len(results) != 3 {
		t.Errorf("len(results) = %d, want 3", len(results))
	}
	for _, r := range results {
		if !r.Passed {
			t.Errorf("threshold %q not passed: actual=%.1f, expected=%.1f", r.Name, r.Actual, r.Expected)
		}
	}
}

func TestEvaluateThresholds_SomeFail(t *testing.T) {
	summary := makeSummary()
	thresholds, err := ParseThresholds([]string{
		"p95<500ms",     // actual=450 → pass
		"p99<500ms",     // actual=800, limit=500 → FAIL
		"error_rate<5%", // actual=2.5 → pass
	})
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	results, allPassed := EvaluateThresholds(thresholds, summary)
	if allPassed {
		t.Error("allPassed = true, want false")
	}
	if len(results) != 3 {
		t.Errorf("len(results) = %d, want 3", len(results))
	}
	// p99 result should be failed
	if results[1].Passed {
		t.Errorf("p99 result: Passed = true, want false")
	}
	if !results[0].Passed {
		t.Errorf("p95 result: Passed = false, want true")
	}
}

func TestEvaluateThresholds_Empty(t *testing.T) {
	summary := makeSummary()
	results, allPassed := EvaluateThresholds(nil, summary)

	if !allPassed {
		t.Error("allPassed = false, want true for empty thresholds")
	}
	if len(results) != 0 {
		t.Errorf("len(results) = %d, want 0", len(results))
	}
}

func TestEvaluateThresholds_Operators(t *testing.T) {
	summary := makeSummary() // ErrorRate=2.5, RPS=150

	tests := []struct {
		name     string
		expr     string
		wantPass bool
	}{
		{"error_rate lt pass", "error_rate<5%", true},
		{"error_rate lt fail", "error_rate<1%", false},
		{"rps gt pass", "rps>100", true},
		{"rps gt fail", "rps>200", false},
		{"p95 lte pass", "p95<=450ms", true},
		{"p95 lte fail", "p95<=449ms", false},
		{"rps gte pass", "rps>=150", true},
		{"rps gte fail", "rps>=151", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			th, err := ParseThreshold(tt.expr)
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}
			results, allPassed := EvaluateThresholds([]Threshold{th}, summary)
			if len(results) != 1 {
				t.Fatalf("expected 1 result, got %d", len(results))
			}
			if allPassed != tt.wantPass {
				t.Errorf("allPassed = %v, want %v (actual=%.1f, op=%s, expected=%.1f)",
					allPassed, tt.wantPass, results[0].Actual, results[0].Operator, results[0].Expected)
			}
		})
	}
}

func TestEvaluateThresholds_ThresholdResultFields(t *testing.T) {
	summary := makeSummary() // P95=450ms
	th, _ := ParseThreshold("p95<500ms")

	results, _ := EvaluateThresholds([]Threshold{th}, summary)
	r := results[0]

	if r.Name != "p95" {
		t.Errorf("Name = %q, want %q", r.Name, "p95")
	}
	if r.Operator != "<" {
		t.Errorf("Operator = %q, want %q", r.Operator, "<")
	}
	if r.Expected != 500.0 {
		t.Errorf("Expected = %.1f, want 500.0", r.Expected)
	}
	if r.Actual != 450.0 {
		t.Errorf("Actual = %.1f, want 450.0", r.Actual)
	}
	if !r.Passed {
		t.Error("Passed = false, want true")
	}
}
