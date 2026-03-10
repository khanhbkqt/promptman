package daemon

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

const (
	// lockFileName is the name of the daemon lock file within the .promptman directory.
	lockFileName = ".daemon.lock"

	// promptmanDir is the hidden directory for promptman metadata.
	promptmanDir = ".promptman"

	// tokenBytes is the number of random bytes used to generate the auth token.
	// Produces a 64-character hex string.
	tokenBytes = 32
)

// lockFilePath returns the full path to the .daemon.lock file for the given project directory.
func lockFilePath(projectDir string) string {
	return filepath.Join(projectDir, promptmanDir, lockFileName)
}

// WriteLockFile writes the daemon info as JSON to the .daemon.lock file.
// It creates the .promptman directory if it does not exist.
func WriteLockFile(projectDir string, info *DaemonInfo) error {
	dir := filepath.Join(projectDir, promptmanDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating .promptman directory: %w", err)
	}

	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling lock file: %w", err)
	}

	path := lockFilePath(projectDir)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("writing lock file: %w", err)
	}

	return nil
}

// ReadLockFile reads and parses the .daemon.lock file for the given project directory.
// Returns ErrDaemonNotRunning if the file does not exist.
// Returns ErrLockFileCorrupt if the file cannot be parsed.
func ReadLockFile(projectDir string) (*DaemonInfo, error) {
	path := lockFilePath(projectDir)

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrDaemonNotRunning
		}
		return nil, fmt.Errorf("reading lock file: %w", err)
	}

	var info DaemonInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, ErrLockFileCorrupt.Wrapf("cannot parse lock file: %v", err)
	}

	// Basic validation: PID and Port must be positive.
	if info.PID <= 0 || info.Port <= 0 {
		return nil, ErrLockFileCorrupt.Wrap("lock file contains invalid PID or port")
	}

	return &info, nil
}

// DeleteLockFile removes the .daemon.lock file.
// It is a no-op if the file does not exist.
func DeleteLockFile(projectDir string) error {
	path := lockFilePath(projectDir)
	err := os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("deleting lock file: %w", err)
	}
	return nil
}

// IsPIDAlive checks whether the given process ID is still alive.
// On Unix, this sends signal 0 which checks for process existence
// without actually sending a signal.
func IsPIDAlive(pid int) bool {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// Signal 0 tests for process existence without sending a signal.
	err = proc.Signal(syscall.Signal(0))
	return err == nil
}

// CleanStaleLockFile checks if a lock file exists with a dead PID
// and removes it if the process is no longer alive.
// Returns the DaemonInfo if the daemon is still running, nil if cleaned up or no lock file.
func CleanStaleLockFile(projectDir string) (*DaemonInfo, error) {
	info, err := ReadLockFile(projectDir)
	if err != nil {
		// No lock file or corrupt — clean up corrupt ones.
		if IsDomainError(err, "DAEMON_NOT_RUNNING") {
			return nil, nil
		}
		if IsDomainError(err, "LOCK_FILE_CORRUPT") {
			// Remove corrupt lock file.
			if delErr := DeleteLockFile(projectDir); delErr != nil {
				return nil, fmt.Errorf("cleaning corrupt lock file: %w", delErr)
			}
			return nil, nil
		}
		return nil, err
	}

	// Lock file exists — check if PID is alive.
	if IsPIDAlive(info.PID) {
		return info, nil
	}

	// PID is dead — stale lock file, clean up.
	if err := DeleteLockFile(projectDir); err != nil {
		return nil, fmt.Errorf("cleaning stale lock file: %w", err)
	}

	return nil, nil
}

// PickRandomPort selects a random available TCP port on 127.0.0.1.
// It binds to port 0 (OS assigns a random port), records the port,
// and closes the listener so the port can be reused.
func PickRandomPort() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, ErrPortUnavailable.Wrapf("cannot bind to 127.0.0.1:0: %v", err)
	}

	addr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		_ = listener.Close()
		return 0, ErrPortUnavailable.Wrap("unexpected listener address type")
	}

	port := addr.Port
	if err := listener.Close(); err != nil {
		return 0, fmt.Errorf("closing temporary listener: %w", err)
	}

	return port, nil
}

// GenerateToken creates a cryptographically secure random token
// as a hex-encoded string. The token is tokenBytes (32) random bytes,
// producing a 64-character hex string.
func GenerateToken() (string, error) {
	b := make([]byte, tokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generating auth token: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// NewDaemonInfo creates a DaemonInfo populated with the current process ID
// and the given port, token, and project directory.
func NewDaemonInfo(port int, token, projectDir string) *DaemonInfo {
	return &DaemonInfo{
		PID:        os.Getpid(),
		Port:       port,
		Token:      token,
		ProjectDir: projectDir,
		StartedAt:  time.Now().UTC(),
	}
}
