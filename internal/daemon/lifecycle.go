package daemon

import (
	"fmt"
	"sync"
	"time"
)

const (
	// DefaultIdleTimeout is the default duration after which the daemon
	// automatically shuts down if no activity occurs.
	DefaultIdleTimeout = 30 * time.Minute
)

// Manager implements the DaemonManager interface.
// It manages daemon lifecycle including start, stop, graceful shutdown,
// status queries, and an idle shutdown timer.
type Manager struct {
	mu sync.RWMutex

	// running indicates whether the daemon is currently active.
	running bool

	// info holds the current daemon's runtime information.
	info *DaemonInfo

	// projectDir is the project directory for which this daemon was started.
	projectDir string

	// idleTimeout is the duration after which the daemon auto-shuts down.
	idleTimeout time.Duration

	// idleTimer fires after the idle timeout to trigger auto-shutdown.
	idleTimer *time.Timer

	// onShutdown is an optional callback invoked when idle shutdown triggers.
	// This allows the caller (e.g., HTTP server) to perform cleanup.
	onShutdown func()
}

// ManagerOption configures optional Manager behaviour.
type ManagerOption func(*Manager)

// WithIdleTimeout sets a custom idle timeout. Zero disables idle shutdown.
func WithIdleTimeout(d time.Duration) ManagerOption {
	return func(m *Manager) {
		m.idleTimeout = d
	}
}

// WithShutdownCallback registers a function to be called when the daemon
// shuts down (either via idle timer or explicit Shutdown call).
func WithShutdownCallback(fn func()) ManagerOption {
	return func(m *Manager) {
		m.onShutdown = fn
	}
}

// NewManager creates a new Manager with the given options.
func NewManager(opts ...ManagerOption) *Manager {
	m := &Manager{
		idleTimeout: DefaultIdleTimeout,
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

// Start initialises the daemon for the given project directory.
// It picks a random available port, generates an auth token,
// writes the .daemon.lock file, and starts the idle shutdown timer.
func (m *Manager) Start(projectDir string) (*DaemonInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return nil, ErrDaemonAlreadyRunning
	}

	// Clean up any stale lock file first.
	existing, err := CleanStaleLockFile(projectDir)
	if err != nil {
		return nil, fmt.Errorf("checking stale lock: %w", err)
	}
	if existing != nil {
		return nil, ErrDaemonAlreadyRunning.Wrapf(
			"daemon already running on port %d (pid %d)", existing.Port, existing.PID,
		)
	}

	// Pick a random available port.
	port, err := PickRandomPort()
	if err != nil {
		return nil, err
	}

	// Generate auth token.
	token, err := GenerateToken()
	if err != nil {
		return nil, err
	}

	// Build the DaemonInfo.
	info := NewDaemonInfo(port, token, projectDir)

	// Write the lock file.
	if err := WriteLockFile(projectDir, info); err != nil {
		return nil, err
	}

	m.running = true
	m.info = info
	m.projectDir = projectDir

	// Start the idle timer.
	m.startIdleTimerLocked()

	return info, nil
}

// Stop immediately terminates the daemon and cleans up resources.
func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return ErrDaemonNotRunning
	}

	return m.shutdownLocked()
}

// Shutdown performs a graceful shutdown. For now, identical to Stop
// since there are no in-flight HTTP requests to drain (that will come
// in the REST API story).
func (m *Manager) Shutdown() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return ErrDaemonNotRunning
	}

	return m.shutdownLocked()
}

// Status returns the current daemon info.
// Returns ErrDaemonNotRunning if the daemon is not active.
func (m *Manager) Status() (*DaemonInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.running {
		return nil, ErrDaemonNotRunning
	}

	// Compute uptime.
	info := *m.info
	info.Uptime = time.Since(info.StartedAt).Truncate(time.Second).String()

	return &info, nil
}

// IsRunning reports whether the daemon is currently active.
func (m *Manager) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.running
}

// ResetIdleTimer resets the idle shutdown timer. Call this on each API
// request to keep the daemon alive.
func (m *Manager) ResetIdleTimer() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return
	}

	m.stopIdleTimerLocked()
	m.startIdleTimerLocked()
}

// shutdownLocked performs the actual shutdown. Must be called with m.mu held.
func (m *Manager) shutdownLocked() error {
	m.stopIdleTimerLocked()

	// Delete lock file.
	if err := DeleteLockFile(m.projectDir); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}

	m.running = false
	m.info = nil

	return nil
}

// startIdleTimerLocked starts the idle timer. Must be called with m.mu held.
func (m *Manager) startIdleTimerLocked() {
	if m.idleTimeout <= 0 {
		return
	}

	m.idleTimer = time.AfterFunc(m.idleTimeout, func() {
		m.mu.Lock()
		if !m.running {
			m.mu.Unlock()
			return
		}
		_ = m.shutdownLocked()
		m.mu.Unlock()

		// Call the shutdown callback outside the lock.
		if m.onShutdown != nil {
			m.onShutdown()
		}
	})
}

// stopIdleTimerLocked cancels the idle timer. Must be called with m.mu held.
func (m *Manager) stopIdleTimerLocked() {
	if m.idleTimer != nil {
		m.idleTimer.Stop()
		m.idleTimer = nil
	}
}
