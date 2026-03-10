package reporter

import (
	"encoding/json"

	testing "github.com/khanhnguyen/promptman/internal/testing"
)

// JSONReporter formats TestResult as indented JSON.
// This is the default output format, optimised for machine consumption
// and AI agent parsing.
type JSONReporter struct{}

// Format marshals the TestResult to pretty-printed JSON.
func (r *JSONReporter) Format(result *testing.TestResult) ([]byte, error) {
	return json.MarshalIndent(result, "", "  ")
}
