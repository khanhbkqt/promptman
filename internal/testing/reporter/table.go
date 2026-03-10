package reporter

import (
	"bytes"
	"fmt"
	"os"
	"text/tabwriter"

	testing "github.com/khanhnguyen/promptman/internal/testing"
	"golang.org/x/term"
)

const (
	// maxErrorLen is the maximum length of an error message shown in
	// table output before truncation.
	maxErrorLen = 80

	// maxBodyPreview is the maximum length of a response body preview
	// shown in table output for failed tests.
	maxBodyPreview = 120
)

// ANSI colour codes — used only when the output is a terminal.
const (
	colorReset  = "\033[0m"
	colorGreen  = "\033[32m"
	colorRed    = "\033[31m"
	colorYellow = "\033[33m"
	colorDim    = "\033[2m"
)

// isTerminalFn is overridable for testing. When nil, the default
// implementation checks os.Stdout.
var isTerminalFn func() bool

// isTerminal reports whether stdout is a terminal.
func isTerminal() bool {
	if isTerminalFn != nil {
		return isTerminalFn()
	}
	return term.IsTerminal(int(os.Stdout.Fd()))
}

// TableReporter formats TestResult as a human-readable table with
// optional ANSI colour codes when the output is a terminal.
type TableReporter struct{}

// Format produces a human-readable table output.
func (r *TableReporter) Format(result *testing.TestResult) ([]byte, error) {
	var buf bytes.Buffer
	color := isTerminal()

	// Header.
	fmt.Fprintf(&buf, "\n  Test Results: %s", result.Collection)
	if result.Env != "" {
		fmt.Fprintf(&buf, " [%s]", result.Env)
	}
	fmt.Fprintln(&buf)
	fmt.Fprintln(&buf, "  "+repeatStr("─", 60))

	// Test rows via tabwriter for alignment.
	tw := tabwriter.NewWriter(&buf, 0, 0, 2, ' ', 0)
	for _, tc := range result.Tests {
		icon := statusIcon(tc.Status, color)
		name := tc.Request
		if tc.Name != "" {
			name = tc.Request + " › " + tc.Name
		}
		fmt.Fprintf(tw, "  %s\t%s\t%dms\n", icon, name, tc.Duration)

		// Show error details for failed tests.
		if tc.Error != nil {
			msg := tc.Error.Message
			if len(msg) > maxErrorLen {
				msg = msg[:maxErrorLen] + "…"
			}
			if color {
				fmt.Fprintf(tw, "  \t%s%s%s\t\n", colorDim, msg, colorReset)
			} else {
				fmt.Fprintf(tw, "  \t%s\t\n", msg)
			}
		}

		// Show truncated body preview for failed tests with response.
		if tc.Response != nil && tc.Response.Body != "" {
			body := tc.Response.Body
			if len(body) > maxBodyPreview {
				body = body[:maxBodyPreview] + "…"
			}
			if color {
				fmt.Fprintf(tw, "  \t%sBody: %s%s\t\n", colorDim, body, colorReset)
			} else {
				fmt.Fprintf(tw, "  \tBody: %s\t\n", body)
			}
		}
	}
	if err := tw.Flush(); err != nil {
		return nil, fmt.Errorf("flushing tabwriter: %w", err)
	}

	// Summary line.
	fmt.Fprintln(&buf, "  "+repeatStr("─", 60))
	s := result.Summary
	if s.Failed == 0 {
		if color {
			fmt.Fprintf(&buf, "  %s✓ %d/%d passed%s (%dms)\n",
				colorGreen, s.Passed, s.Total, colorReset, s.Duration)
		} else {
			fmt.Fprintf(&buf, "  ✓ %d/%d passed (%dms)\n",
				s.Passed, s.Total, s.Duration)
		}
	} else {
		if color {
			fmt.Fprintf(&buf, "  %s✗ %d/%d passed, %d failed%s (%dms)\n",
				colorRed, s.Passed, s.Total, s.Failed, colorReset, s.Duration)
		} else {
			fmt.Fprintf(&buf, "  ✗ %d/%d passed, %d failed (%dms)\n",
				s.Passed, s.Total, s.Failed, s.Duration)
		}
	}
	if s.Skipped > 0 {
		if color {
			fmt.Fprintf(&buf, "  %s%d skipped%s\n",
				colorYellow, s.Skipped, colorReset)
		} else {
			fmt.Fprintf(&buf, "  %d skipped\n", s.Skipped)
		}
	}

	return buf.Bytes(), nil
}

// statusIcon returns the appropriate icon for a test status.
func statusIcon(status string, color bool) string {
	switch status {
	case "passed":
		if color {
			return colorGreen + "✓" + colorReset
		}
		return "✓"
	case "failed", "error":
		if color {
			return colorRed + "✗" + colorReset
		}
		return "✗"
	case "timeout":
		if color {
			return colorRed + "⏱" + colorReset
		}
		return "⏱"
	case "skipped":
		if color {
			return colorYellow + "○" + colorReset
		}
		return "○"
	default:
		return "?"
	}
}

// repeatStr repeats a string n times.
func repeatStr(s string, n int) string {
	out := make([]byte, 0, len(s)*n)
	for i := 0; i < n; i++ {
		out = append(out, s...)
	}
	return string(out)
}
