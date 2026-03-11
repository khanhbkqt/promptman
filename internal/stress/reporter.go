package stress

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"
)

// FormatTable formats a StressReport as an aligned CLI table suitable
// for terminal output. The output includes scenario name, duration,
// request summary, latency percentiles, and threshold results.
func FormatTable(report *StressReport) string {
	var b strings.Builder

	// Header.
	fmt.Fprintf(&b, "\n══════════════════════════════════════════════════\n")
	fmt.Fprintf(&b, "  Stress Test Report: %s\n", report.Scenario)
	fmt.Fprintf(&b, "══════════════════════════════════════════════════\n\n")

	// Summary table.
	tw := tabwriter.NewWriter(&b, 0, 0, 2, ' ', 0)
	fmt.Fprintf(tw, "  Duration:\t%s\n", formatDuration(report.Duration))
	fmt.Fprintf(tw, "  Total Requests:\t%d\n", report.Summary.TotalRequests)
	fmt.Fprintf(tw, "  RPS:\t%.2f req/s\n", report.Summary.RPS)
	fmt.Fprintf(tw, "  Error Rate:\t%.2f%%\n", report.Summary.ErrorRate)
	fmt.Fprintf(tw, "  Throughput:\t%s/s\n", formatBytes(report.Summary.Throughput))
	fmt.Fprintf(tw, "  Peak Connections:\t%d\n", report.Summary.PeakConnections)
	_ = tw.Flush()

	// Latency table.
	fmt.Fprintf(&b, "\n  Latency Percentiles\n")
	fmt.Fprintf(&b, "  ──────────────────\n")
	tw = tabwriter.NewWriter(&b, 0, 0, 2, ' ', 0)
	fmt.Fprintf(tw, "  p50:\t%d ms\n", report.Summary.Latency.P50)
	fmt.Fprintf(tw, "  p95:\t%d ms\n", report.Summary.Latency.P95)
	fmt.Fprintf(tw, "  p99:\t%d ms\n", report.Summary.Latency.P99)
	_ = tw.Flush()

	// Threshold results (if any).
	if len(report.Thresholds) > 0 {
		fmt.Fprintf(&b, "\n  Thresholds\n")
		fmt.Fprintf(&b, "  ──────────\n")
		tw = tabwriter.NewWriter(&b, 0, 0, 2, ' ', 0)
		for _, t := range report.Thresholds {
			status := "✓ PASS"
			if !t.Passed {
				status = "✗ FAIL"
			}
			fmt.Fprintf(tw, "  %s\t%s %s %g\t(actual: %g)\t%s\n",
				t.Name, t.Name, t.Operator, t.Expected, t.Actual, status)
		}
		_ = tw.Flush()
	}

	fmt.Fprintf(&b, "\n══════════════════════════════════════════════════\n")

	return b.String()
}

// WriteJSON serialises a StressReport to the given file path as
// indented JSON. The file is created with 0644 permissions. If the
// file already exists it is overwritten.
func WriteJSON(report *StressReport, path string) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling report: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing report to %s: %w", path, err)
	}

	return nil
}

// formatDuration converts milliseconds to a human-readable string.
func formatDuration(ms int64) string {
	d := time.Duration(ms) * time.Millisecond
	if d < time.Second {
		return fmt.Sprintf("%dms", ms)
	}
	return d.Truncate(time.Millisecond).String()
}

// formatBytes converts bytes to a human-readable string with appropriate units.
func formatBytes(b int64) string {
	const (
		kb = 1024
		mb = kb * 1024
	)
	switch {
	case b >= mb:
		return fmt.Sprintf("%.2f MB", float64(b)/float64(mb))
	case b >= kb:
		return fmt.Sprintf("%.2f KB", float64(b)/float64(kb))
	default:
		return fmt.Sprintf("%d B", b)
	}
}
