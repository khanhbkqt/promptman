// Package ws provides the WebSocket hub for real-time event broadcasting
// from the daemon to connected CLI/GUI clients. The design is
// notification-only — clients receive events and reload data via the REST
// API rather than receiving full state synchronisation over the socket.
//
// Key components:
//   - Event: typed event payloads for 6 event categories
//   - Hub:   goroutine-based connection manager and broadcaster
//   - Client: per-connection write pump with ping/pong keepalive
package ws
