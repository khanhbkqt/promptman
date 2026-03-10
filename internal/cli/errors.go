package cli

import (
	"fmt"

	"github.com/khanhnguyen/promptman/pkg/envelope"
)

// CLI-specific error codes returned as DomainErrors from the daemon client.
const (
	// CodeDaemonNotRunning indicates the daemon is not running or the lock file is absent.
	CodeDaemonNotRunning = "CLI_DAEMON_NOT_RUNNING"

	// CodeDaemonUnreachable indicates the daemon is registered in the lock file
	// but does not respond to HTTP requests.
	CodeDaemonUnreachable = "CLI_DAEMON_UNREACHABLE"

	// CodeHTTPError indicates an unexpected HTTP error when calling the daemon.
	CodeHTTPError = "CLI_HTTP_ERROR"

	// CodeResponseDecodeError indicates the daemon returned a response that
	// could not be decoded as an envelope.Envelope.
	CodeResponseDecodeError = "CLI_RESPONSE_DECODE_ERROR"
)

// CLIError is a structured error carrying a CLI-specific error code.
// It mirrors the DomainError pattern used in internal/daemon/errors.go.
type CLIError struct {
	// Code is one of the Code* constants defined in this file.
	Code string

	// Message is a human-readable description of the error.
	Message string

	// Unwrapped is an optional underlying error for wrapping.
	Unwrapped error
}

// Error implements the error interface.
func (e *CLIError) Error() string {
	if e.Unwrapped != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Unwrapped)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap supports errors.Is/As traversal.
func (e *CLIError) Unwrap() error {
	return e.Unwrapped
}

// Wrap returns a new CLIError preserving the code but replacing the message.
func (e *CLIError) Wrap(msg string, cause error) *CLIError {
	return &CLIError{Code: e.Code, Message: msg, Unwrapped: cause}
}

// Sentinel CLI errors.
var (
	// ErrDaemonNotRunning is returned when the lock file is missing or the PID is dead.
	ErrDaemonNotRunning = &CLIError{Code: CodeDaemonNotRunning, Message: "daemon not running"}

	// ErrDaemonUnreachable is returned when the HTTP request to the daemon fails.
	ErrDaemonUnreachable = &CLIError{Code: CodeDaemonUnreachable, Message: "daemon unreachable"}

	// ErrHTTPError is returned when the daemon returns an unexpected HTTP status.
	ErrHTTPError = &CLIError{Code: CodeHTTPError, Message: "HTTP error"}

	// ErrResponseDecodeError is returned when the daemon response cannot be decoded.
	ErrResponseDecodeError = &CLIError{Code: CodeResponseDecodeError, Message: "response decode error"}
)

// IsCLIError reports whether err is a *CLIError with the given code.
func IsCLIError(err error, code string) bool {
	ce, ok := err.(*CLIError)
	return ok && ce.Code == code
}

// ToEnvelope converts a *CLIError into an envelope.Envelope for consistent formatting.
func (e *CLIError) ToEnvelope() *envelope.Envelope {
	return envelope.Fail(e.Code, e.Message)
}
