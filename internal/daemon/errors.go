package daemon

import (
	"fmt"

	"github.com/khanhnguyen/promptman/pkg/envelope"
)

// Domain-specific sentinel errors for the daemon module.
// These wrap the canonical error codes defined in pkg/envelope.
var (
	// ErrDaemonAlreadyRunning is returned when Start() is called
	// but the daemon is already active.
	ErrDaemonAlreadyRunning = &DomainError{Code: envelope.CodeDaemonAlreadyRunning, Message: "daemon already running"}

	// ErrDaemonNotRunning is returned when Stop/Shutdown/Status is called
	// but the daemon is not active.
	ErrDaemonNotRunning = &DomainError{Code: envelope.CodeDaemonNotRunning, Message: "daemon not running"}

	// ErrLockFileCorrupt is returned when the .daemon.lock file exists
	// but cannot be parsed as valid JSON.
	ErrLockFileCorrupt = &DomainError{Code: envelope.CodeLockFileCorrupt, Message: "lock file corrupt"}

	// ErrPortUnavailable is returned when no available port can be found
	// on 127.0.0.1.
	ErrPortUnavailable = &DomainError{Code: envelope.CodePortUnavailable, Message: "port unavailable"}
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
