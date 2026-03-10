package ws

import "fmt"

// Domain-specific error sentinels for the WebSocket module.
var (
	// ErrHubStopped is returned when an operation is attempted on a
	// hub that has already been shut down.
	ErrHubStopped = &WSError{Message: "hub stopped"}

	// ErrClientClosed is returned when writing to a client whose
	// send channel has been closed.
	ErrClientClosed = &WSError{Message: "client connection closed"}
)

// WSError is a structured error for the WebSocket module.
type WSError struct {
	Message string
}

// Error implements the error interface.
func (e *WSError) Error() string {
	return fmt.Sprintf("ws: %s", e.Message)
}
