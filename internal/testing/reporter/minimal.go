package reporter

import (
	"fmt"

	testing "github.com/khanhnguyen/promptman/internal/testing"
)

// MinimalReporter formats TestResult as a single summary line.
// Output: "✓ 6/8 passed (1245ms)" or "✗ 6/8 passed, 2 failed (1245ms)".
type MinimalReporter struct{}

// Format produces a one-liner summary of the test run.
func (r *MinimalReporter) Format(result *testing.TestResult) ([]byte, error) {
	s := result.Summary
	var line string
	if s.Failed == 0 {
		line = fmt.Sprintf("✓ %d/%d passed (%dms)\n", s.Passed, s.Total, s.Duration)
	} else {
		line = fmt.Sprintf("✗ %d/%d passed, %d failed (%dms)\n",
			s.Passed, s.Total, s.Failed, s.Duration)
	}
	return []byte(line), nil
}
