package testing

import (
	"fmt"

	"github.com/khanhnguyen/promptman/pkg/envelope"
)

// Domain-specific sentinel errors for the testing module.
// These wrap the canonical error codes defined in pkg/envelope.
var (
	// ErrSandboxViolation is returned when a test script attempts to use a blocked API.
	ErrSandboxViolation = &DomainError{Code: envelope.CodeSandboxViolation, Message: "sandbox violation"}

	// ErrTestTimeout is returned when a test exceeds its allowed execution time.
	ErrTestTimeout = &DomainError{Code: envelope.CodeTestTimeout, Message: "test timeout"}

	// ErrScriptParse is returned when a test script contains a JavaScript syntax error.
	ErrScriptParse = &DomainError{Code: envelope.CodeScriptParse, Message: "script parse error"}
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
