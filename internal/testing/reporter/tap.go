package reporter

import (
	"bytes"
	"fmt"
	"strings"

	testing "github.com/khanhnguyen/promptman/internal/testing"
)

// TAPReporter formats TestResult as TAP (Test Anything Protocol)
// output suitable for CI/CD integration.
//
// TAP spec reference: https://testanything.org/tap-specification.html
type TAPReporter struct{}

// Format produces TAP-compliant output with YAML diagnostic blocks
// for failures.
func (r *TAPReporter) Format(result *testing.TestResult) ([]byte, error) {
	var buf bytes.Buffer

	total := len(result.Tests)
	// TAP plan line.
	fmt.Fprintf(&buf, "1..%d\n", total)

	for i, tc := range result.Tests {
		num := i + 1
		name := tc.Request
		if tc.Name != "" {
			name = tc.Request + " - " + tc.Name
		}

		switch tc.Status {
		case "passed":
			fmt.Fprintf(&buf, "ok %d %s\n", num, name)
		case "skipped":
			fmt.Fprintf(&buf, "ok %d %s # SKIP no matching test\n", num, name)
		default:
			// failed, timeout, error
			fmt.Fprintf(&buf, "not ok %d %s\n", num, name)
			writeDiagnostic(&buf, tc)
		}
	}

	return buf.Bytes(), nil
}

// writeDiagnostic writes a YAML diagnostic block for a failed test.
// TAP diagnostic blocks are enclosed in --- / ... markers.
func writeDiagnostic(buf *bytes.Buffer, tc testing.TestCase) {
	buf.WriteString("  ---\n")
	fmt.Fprintf(buf, "  status: %s\n", tc.Status)
	fmt.Fprintf(buf, "  duration_ms: %d\n", tc.Duration)

	if tc.Error != nil {
		fmt.Fprintf(buf, "  message: %s\n", yamlEscape(tc.Error.Message))
		if tc.Error.Expected != nil {
			fmt.Fprintf(buf, "  expected: %v\n", tc.Error.Expected)
		}
		if tc.Error.Actual != nil {
			fmt.Fprintf(buf, "  actual: %v\n", tc.Error.Actual)
		}
	}

	if tc.Response != nil {
		buf.WriteString("  response:\n")
		fmt.Fprintf(buf, "    status: %d\n", tc.Response.Status)
		if tc.Response.Body != "" {
			body := tc.Response.Body
			if len(body) > 200 {
				body = body[:200] + "..."
			}
			fmt.Fprintf(buf, "    body: %s\n", yamlEscape(body))
		}
	}

	buf.WriteString("  ...\n")
}

// yamlEscape wraps a string in quotes if it contains characters that
// could be ambiguous in YAML.
func yamlEscape(s string) string {
	if strings.ContainsAny(s, ":\n\"'{}[]#&*!|>%@`") {
		return fmt.Sprintf("%q", s)
	}
	return s
}
