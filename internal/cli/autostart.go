package cli

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/khanhnguyen/promptman/internal/daemon"
)

const (
	// autoStartMaxRetries is the maximum number of times to retry reading
	// the lock file after spawning the daemon process.
	autoStartMaxRetries = 6

	// autoStartRetryDelay is the wait between each lock file retry attempt.
	autoStartRetryDelay = 500 * time.Millisecond
)

// EnsureDaemon checks whether the daemon is running for the given project directory.
// If the lock file is missing or the recorded PID is dead, it spawns a new daemon
// process in the background using the promptman binary and waits for it to write
// the lock file. Returns nil when the daemon is confirmed running.
func EnsureDaemon(projectDir string) error {
	// Use CleanStaleLockFile to determine current state.
	info, err := daemon.CleanStaleLockFile(projectDir)
	if err != nil {
		return fmt.Errorf("checking daemon state: %w", err)
	}
	if info != nil {
		// Daemon is already running.
		return nil
	}

	// Daemon is not running — spawn it as a background process.
	if err := spawnDaemon(projectDir); err != nil {
		return fmt.Errorf("spawning daemon: %w", err)
	}

	// Poll for the lock file to appear (daemon writes it during startup).
	return waitForDaemon(projectDir)
}

// spawnDaemon launches the daemon binary as a background process.
// It uses the same executable path as the current process and passes
// the daemon subcommand with the project directory.
func spawnDaemon(projectDir string) error {
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolving executable path: %w", err)
	}

	cmd := exec.Command(execPath, "daemon", "start", "--project-dir", projectDir)
	cmd.Stdout = nil
	cmd.Stderr = nil

	// Start() spawns the process and returns immediately (non-blocking).
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting daemon process: %w", err)
	}

	// Detach from the child process — we don't wait for it.
	// The daemon manages its own lifecycle via the lock file.
	return nil
}

// waitForDaemon polls for the daemon lock file up to autoStartMaxRetries times.
// Returns nil when the lock file exists and the PID is alive, or an error
// if the daemon does not start within the polling window.
func waitForDaemon(projectDir string) error {
	for i := 0; i < autoStartMaxRetries; i++ {
		time.Sleep(autoStartRetryDelay)

		info, err := daemon.CleanStaleLockFile(projectDir)
		if err != nil {
			// CleanStaleLockFile returned an unexpected error — abort.
			return fmt.Errorf("polling daemon state: %w", err)
		}
		if info != nil {
			// Lock file present and PID alive — daemon is up.
			return nil
		}
	}

	return fmt.Errorf("daemon did not start within %.1f seconds",
		float64(autoStartMaxRetries)*autoStartRetryDelay.Seconds())
}
