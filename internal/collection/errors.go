package collection

import (
	"fmt"

	"github.com/khanhnguyen/promptman/pkg/envelope"
)

// Domain-specific sentinel errors for the collection module.
// These wrap the canonical error codes defined in pkg/envelope.
var (
	// ErrCollectionNotFound is returned when a collection ID does not exist on disk.
	ErrCollectionNotFound = &DomainError{Code: envelope.CodeCollectionNotFound, Message: "collection not found"}

	// ErrInvalidYAML is returned when a YAML file cannot be parsed.
	ErrInvalidYAML = &DomainError{Code: envelope.CodeInvalidYAML, Message: "invalid YAML"}

	// ErrInvalidRequest is returned when a request or collection fails validation.
	ErrInvalidRequest = &DomainError{Code: envelope.CodeInvalidRequest, Message: "invalid request"}

	// ErrRequestNotFound is returned when a request path does not exist in the collection.
	ErrRequestNotFound = &DomainError{Code: envelope.CodeRequestNotFound, Message: "request not found"}
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
