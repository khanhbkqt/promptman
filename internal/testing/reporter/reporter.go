package reporter

import (
	"fmt"

	testing "github.com/khanhnguyen/promptman/internal/testing"
)

// Output format constants used by ForFormat and callers.
const (
	FormatJSON    = "json"
	FormatTable   = "table"
	FormatTAP     = "tap"
	FormatMinimal = "minimal"
)

// Reporter formats a TestResult into a byte slice suitable for output.
type Reporter interface {
	// Format converts the TestResult into the reporter's output format.
	Format(result *testing.TestResult) ([]byte, error)
}

// ForFormat returns a Reporter for the named format.
// Supported formats: "json", "table", "tap", "minimal".
// Returns an error for unknown format names.
func ForFormat(name string) (Reporter, error) {
	switch name {
	case FormatJSON:
		return &JSONReporter{}, nil
	case FormatTable:
		return &TableReporter{}, nil
	case FormatTAP:
		return &TAPReporter{}, nil
	case FormatMinimal:
		return &MinimalReporter{}, nil
	default:
		return nil, fmt.Errorf("unknown report format: %q (supported: json, table, tap, minimal)", name)
	}
}
