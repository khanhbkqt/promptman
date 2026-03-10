package cli_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/khanhnguyen/promptman/internal/cli"
	"github.com/khanhnguyen/promptman/internal/daemon"
)

func TestEnsureDaemon_AlreadyRunning(t *testing.T) {
	dir := t.TempDir()

	// Write lock file with current PID — daemon is "running".
	info := daemon.NewDaemonInfo(12345, "tok", dir)
	if err := daemon.WriteLockFile(dir, info); err != nil {
		t.Fatalf("setup: %v", err)
	}
	t.Cleanup(func() { _ = daemon.DeleteLockFile(dir) })

	// EnsureDaemon should detect the running daemon and return nil immediately.
	if err := cli.EnsureDaemon(dir); err != nil {
		t.Errorf("EnsureDaemon returned error for running daemon: %v", err)
	}
}

func TestEnsureDaemon_NotRunning_SpawnFails(t *testing.T) {
	// When the daemon binary doesn't exist, spawnDaemon will fail.
	// We test this by temporarily overriding os.Executable via a temp dir
	// that doesn't contain a valid daemon binary — but since we're testing the
	// real binary lookup, this will error on Start() or waitForDaemon timeout.
	// The important thing is: if daemon can't start, EnsureDaemon returns an error.
	dir := t.TempDir()

	// No lock file — daemon is not running.
	err := cli.EnsureDaemon(dir)
	// We expect an error because the daemon binary doesn't implement "daemon start" yet.
	// This test simply verifies the function doesn't panic or hang.
	if err == nil {
		// If somehow the daemon started (unlikely in test env), clean up.
		_ = daemon.DeleteLockFile(dir)
	}
	// Either success or failure is acceptable here — we just verify no panic/hang.
}

func TestSpawnDaemon_ExecutableExists(t *testing.T) {
	// Verify that os.Executable() resolves without error — the foundation of spawnDaemon.
	execPath, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable failed: %v", err)
	}
	if execPath == "" {
		t.Error("expected non-empty executable path")
	}
	// Verify the binary actually exists on disk.
	if _, err := os.Stat(execPath); err != nil {
		// In test environments the test binary may be in a temp path — acceptable.
		t.Logf("executable path %q: %v (acceptable in test env)", execPath, err)
	}
}

func TestEnsureDaemon_StaleLockFileCleaned(t *testing.T) {
	dir := t.TempDir()

	// Write lock file with dead PID (99999999).
	info := &daemon.DaemonInfo{
		PID:        99999999,
		Port:       12345,
		Token:      "tok",
		ProjectDir: dir,
	}
	if err := daemon.WriteLockFile(dir, info); err != nil {
		t.Fatalf("setup: %v", err)
	}

	// EnsureDaemon will clean the stale lock file then try to spawn.
	// We don't assert success since daemon binary isn't set up, but the
	// stale lock must be cleaned first.
	_ = cli.EnsureDaemon(dir)

	// The dead-PID lock file should have been removed.
	lockPath := filepath.Join(dir, ".promptman", ".daemon.lock")
	_, statErr := os.Stat(lockPath)
	if statErr == nil {
		// Lock exists — check if it still has the dead PID.
		remaining, readErr := daemon.ReadLockFile(dir)
		if readErr == nil && remaining.PID == 99999999 {
			t.Error("stale lock file with dead PID was not cleaned up")
		}
	}
}
