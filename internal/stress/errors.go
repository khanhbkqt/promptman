package stress

import (
	"fmt"

	"github.com/khanhnguyen/promptman/pkg/envelope"
)

// Domain-specific sentinel errors for the stress module.
// These wrap the canonical error codes defined in pkg/envelope.
var (
	// ErrInvalidConfig is returned when a stress config file fails validation.
	ErrInvalidConfig = &DomainError{Code: envelope.CodeInvalidConfig, Message: "invalid stress config"}

	// ErrThresholdFailed is returned when one or more thresholds are not met.
	ErrThresholdFailed = &DomainError{Code: envelope.CodeThresholdFailed, Message: "threshold failed"}

	// ErrStressTimeout is returned when a stress test exceeds its time limit.
	ErrStressTimeout = &DomainError{Code: envelope.CodeStressTimeout, Message: "stress test timed out"}

	// ErrInvalidScenario is returned when a scenario definition is invalid.
	ErrInvalidScenario = &DomainError{Code: envelope.CodeInvalidScenario, Message: "invalid scenario"}
)

// DomainError is a structured error carrying an envelope-compatible error code.
type DomainError struct {
	Code    string // one of the envelope.Code* constants
	Message string // human-readable description
}

// Error implements the error interface.
func (e *DomainError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Wrap returns a new DomainError that preserves the original code but
// replaces the message with additional context.
func (e *DomainError) Wrap(msg string) *DomainError {
	return &DomainError{Code: e.Code, Message: msg}
}

// Wrapf is like Wrap but accepts a format string.
func (e *DomainError) Wrapf(format string, args ...any) *DomainError {
	return &DomainError{Code: e.Code, Message: fmt.Sprintf(format, args...)}
}

// IsDomainError checks whether err is a *DomainError with the given code.
func IsDomainError(err error, code string) bool {
	de, ok := err.(*DomainError)
	return ok && de.Code == code
}
