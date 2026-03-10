package daemon

import "time"

// DaemonInfo holds the runtime state of a running daemon instance.
// JSON tags match the .daemon.lock file format specified in the daemon spec.
type DaemonInfo struct {
	PID        int       `json:"pid"`
	Port       int       `json:"port"`
	Token      string    `json:"token"`
	ProjectDir string    `json:"projectDir"`
	StartedAt  time.Time `json:"startedAt"`
	Uptime     string    `json:"uptime,omitempty"`
}

// DaemonManager defines the lifecycle operations for the daemon process.
type DaemonManager interface {
	// Start initialises the daemon for the given project directory.
	// It picks a random available port, generates an auth token,
	// writes the .daemon.lock file, and starts the idle shutdown timer.
	Start(projectDir string) (*DaemonInfo, error)

	// Stop immediately terminates the daemon and cleans up resources.
	Stop() error

	// Shutdown performs a graceful shutdown, allowing in-flight
	// operations to complete before cleaning up.
	Shutdown() error

	// Status returns the current daemon info, or an error if not running.
	Status() (*DaemonInfo, error)

	// IsRunning reports whether the daemon is currently active.
	IsRunning() bool
}
