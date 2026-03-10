// Package reporter provides output formatters for test results.
//
// Four built-in formats are available:
//   - json:    machine-readable indented JSON (default)
//   - table:   human-readable table with color support
//   - tap:     TAP (Test Anything Protocol) for CI/CD
//   - minimal: single-line pass/fail summary
//
// Each format implements the Reporter interface. Use ForFormat to
// obtain a reporter by name.
package reporter
