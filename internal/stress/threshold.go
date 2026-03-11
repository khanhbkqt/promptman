package stress

import (
	"fmt"
	"strconv"
	"strings"
)

// Threshold represents a parsed threshold expression, e.g. "p95<500ms".
type Threshold struct {
	// Metric is the metric name: p50, p95, p99, error_rate, rps, throughput.
	Metric string

	// Operator is the comparison operator: <, >, <=, >=.
	Operator string

	// Value is the threshold limit in canonical units:
	//   - latency metrics (p50/p95/p99): milliseconds (float64)
	//   - error_rate: percentage 0–100
	//   - rps, throughput: raw float64
	Value float64
}

// supportedMetrics lists all valid metric names for threshold expressions.
var supportedMetrics = map[string]bool{
	"p50":        true,
	"p95":        true,
	"p99":        true,
	"error_rate": true,
	"rps":        true,
	"throughput": true,
}

// ParseThreshold parses a threshold expression string into a Threshold.
// Expressions take the form: <metric><operator><value>
//
// Examples:
//
//	"p95<500ms"    → {Metric: "p95", Operator: "<", Value: 500.0}
//	"error_rate<5%" → {Metric: "error_rate", Operator: "<", Value: 5.0}
//	"rps>100"      → {Metric: "rps", Operator: ">", Value: 100.0}
//	"p99<=2s"      → {Metric: "p99", Operator: "<=", Value: 2000.0}
func ParseThreshold(expr string) (Threshold, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return Threshold{}, ErrInvalidConfig.Wrap("threshold expression is empty")
	}

	// Find the operator — check two-char operators before single-char.
	var metric, op, rawVal string
	for _, candidate := range []string{"<=", ">=", "<", ">"} {
		idx := strings.Index(expr, candidate)
		if idx < 0 {
			continue
		}
		metric = strings.TrimSpace(expr[:idx])
		op = candidate
		rawVal = strings.TrimSpace(expr[idx+len(candidate):])
		break
	}

	if op == "" {
		return Threshold{}, ErrInvalidConfig.Wrapf("threshold %q: missing operator (use <, >, <=, >=)", expr)
	}
	if metric == "" {
		return Threshold{}, ErrInvalidConfig.Wrapf("threshold %q: metric name is empty", expr)
	}
	if rawVal == "" {
		return Threshold{}, ErrInvalidConfig.Wrapf("threshold %q: value is empty", expr)
	}

	if !supportedMetrics[metric] {
		return Threshold{}, ErrInvalidConfig.Wrapf("threshold %q: unknown metric %q (supported: p50, p95, p99, error_rate, rps, throughput)", expr, metric)
	}

	val, err := parseThresholdValue(rawVal)
	if err != nil {
		return Threshold{}, ErrInvalidConfig.Wrapf("threshold %q: %v", expr, err)
	}

	return Threshold{Metric: metric, Operator: op, Value: val}, nil
}

// ParseThresholds parses a slice of threshold expression strings.
// Returns on the first parse error.
func ParseThresholds(exprs []string) ([]Threshold, error) {
	result := make([]Threshold, 0, len(exprs))
	for _, expr := range exprs {
		t, err := ParseThreshold(expr)
		if err != nil {
			return nil, err
		}
		result = append(result, t)
	}
	return result, nil
}

// EvaluateThresholds compares each threshold against the given StressSummary.
// It returns the individual ThresholdResult for each threshold and an allPassed
// boolean that is true only when every threshold is met.
func EvaluateThresholds(thresholds []Threshold, summary StressSummary) ([]ThresholdResult, bool) {
	if len(thresholds) == 0 {
		return nil, true
	}

	results := make([]ThresholdResult, 0, len(thresholds))
	allPassed := true

	for _, t := range thresholds {
		actual := metricValue(t.Metric, summary)
		passed := compare(actual, t.Operator, t.Value)

		results = append(results, ThresholdResult{
			Name:     t.Metric,
			Operator: t.Operator,
			Expected: t.Value,
			Actual:   actual,
			Passed:   passed,
		})

		if !passed {
			allPassed = false
		}
	}

	return results, allPassed
}

// metricValue extracts the numeric value for the given metric name from the summary.
// Latency values are returned as milliseconds (float64).
func metricValue(metric string, s StressSummary) float64 {
	switch metric {
	case "p50":
		return float64(s.Latency.P50)
	case "p95":
		return float64(s.Latency.P95)
	case "p99":
		return float64(s.Latency.P99)
	case "error_rate":
		return s.ErrorRate
	case "rps":
		return s.RPS
	case "throughput":
		return float64(s.Throughput)
	default:
		return 0
	}
}

// compare evaluates: actual <op> threshold.
func compare(actual float64, op string, threshold float64) bool {
	switch op {
	case "<":
		return actual < threshold
	case ">":
		return actual > threshold
	case "<=":
		return actual <= threshold
	case ">=":
		return actual >= threshold
	default:
		return false
	}
}

// parseThresholdValue converts a raw value string to a canonical float64.
//
// Conversion rules:
//   - "500ms" → 500.0 (milliseconds)
//   - "1s"    → 1000.0
//   - "2.5s"  → 2500.0
//   - "5%"    → 5.0  (0–100 scale, matches StressSummary.ErrorRate)
//   - "100"   → 100.0 (plain number)
func parseThresholdValue(raw string) (float64, error) {
	raw = strings.TrimSpace(raw)

	// Duration: ends with "ms"
	if strings.HasSuffix(raw, "ms") {
		n, err := strconv.ParseFloat(strings.TrimSuffix(raw, "ms"), 64)
		if err != nil {
			return 0, fmt.Errorf("invalid millisecond value %q: %v", raw, err)
		}
		return n, nil
	}

	// Duration: ends with "s" (seconds → milliseconds)
	if strings.HasSuffix(raw, "s") {
		n, err := strconv.ParseFloat(strings.TrimSuffix(raw, "s"), 64)
		if err != nil {
			return 0, fmt.Errorf("invalid second value %q: %v", raw, err)
		}
		return n * 1000, nil
	}

	// Percentage: ends with "%"
	if strings.HasSuffix(raw, "%") {
		n, err := strconv.ParseFloat(strings.TrimSuffix(raw, "%"), 64)
		if err != nil {
			return 0, fmt.Errorf("invalid percentage value %q: %v", raw, err)
		}
		return n, nil
	}

	// Plain number.
	n, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid numeric value %q: %v", raw, err)
	}
	return n, nil
}
