package daemon

import (
	"fmt"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/khanhnguyen/promptman/pkg/envelope"
)

func TestManager_Start_Success(t *testing.T) {
	dir := t.TempDir()
	m := NewManager(WithIdleTimeout(0)) // disable idle timer for tests

	info, err := m.Start(dir)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer func() { _ = m.Stop() }()

	if info.PID != os.Getpid() {
		t.Errorf("PID = %d, want %d", info.PID, os.Getpid())
	}
	if info.Port <= 0 {
		t.Errorf("Port = %d, want positive", info.Port)
	}
	if len(info.Token) != 64 {
		t.Errorf("Token length = %d, want 64", len(info.Token))
	}
	if info.ProjectDir != dir {
		t.Errorf("ProjectDir = %q, want %q", info.ProjectDir, dir)
	}
	if !m.IsRunning() {
		t.Error("IsRunning() = false after Start")
	}

	// Lock file should exist.
	if _, err := ReadLockFile(dir); err != nil {
		t.Errorf("lock file not readable after Start: %v", err)
	}
}

func TestManager_Start_DoubleStart(t *testing.T) {
	dir := t.TempDir()
	m := NewManager(WithIdleTimeout(0))

	if _, err := m.Start(dir); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer func() { _ = m.Stop() }()

	_, err := m.Start(dir)
	if !IsDomainError(err, envelope.CodeDaemonAlreadyRunning) {
		t.Errorf("expected ErrDaemonAlreadyRunning, got %v", err)
	}
}

func TestManager_Stop_Success(t *testing.T) {
	dir := t.TempDir()
	m := NewManager(WithIdleTimeout(0))

	if _, err := m.Start(dir); err != nil {
		t.Fatalf("Start: %v", err)
	}

	if err := m.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	if m.IsRunning() {
		t.Error("IsRunning() = true after Stop")
	}

	// Lock file should be deleted.
	_, err := ReadLockFile(dir)
	if !IsDomainError(err, envelope.CodeDaemonNotRunning) {
		t.Errorf("lock file should not exist after Stop, got %v", err)
	}
}

func TestManager_Stop_NotRunning(t *testing.T) {
	m := NewManager(WithIdleTimeout(0))

	err := m.Stop()
	if !IsDomainError(err, envelope.CodeDaemonNotRunning) {
		t.Errorf("expected ErrDaemonNotRunning, got %v", err)
	}
}

func TestManager_Shutdown_Success(t *testing.T) {
	dir := t.TempDir()
	m := NewManager(WithIdleTimeout(0))

	if _, err := m.Start(dir); err != nil {
		t.Fatalf("Start: %v", err)
	}

	if err := m.Shutdown(); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}

	if m.IsRunning() {
		t.Error("IsRunning() = true after Shutdown")
	}
}

func TestManager_Shutdown_NotRunning(t *testing.T) {
	m := NewManager(WithIdleTimeout(0))

	err := m.Shutdown()
	if !IsDomainError(err, envelope.CodeDaemonNotRunning) {
		t.Errorf("expected ErrDaemonNotRunning, got %v", err)
	}
}

func TestManager_Status_Running(t *testing.T) {
	dir := t.TempDir()
	m := NewManager(WithIdleTimeout(0))

	startInfo, err := m.Start(dir)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer func() { _ = m.Stop() }()

	status, err := m.Status()
	if err != nil {
		t.Fatalf("Status: %v", err)
	}

	if status.PID != startInfo.PID {
		t.Errorf("PID mismatch: %d vs %d", status.PID, startInfo.PID)
	}
	if status.Port != startInfo.Port {
		t.Errorf("Port mismatch: %d vs %d", status.Port, startInfo.Port)
	}
	if status.Uptime == "" {
		t.Error("Uptime should be populated")
	}
}

func TestManager_Status_NotRunning(t *testing.T) {
	m := NewManager(WithIdleTimeout(0))

	_, err := m.Status()
	if !IsDomainError(err, envelope.CodeDaemonNotRunning) {
		t.Errorf("expected ErrDaemonNotRunning, got %v", err)
	}
}

func TestManager_IsRunning_InitialState(t *testing.T) {
	m := NewManager()
	if m.IsRunning() {
		t.Error("new Manager should not be running")
	}
}

func TestManager_IdleTimeout(t *testing.T) {
	dir := t.TempDir()
	var shutdownCalled atomic.Int32

	m := NewManager(
		WithIdleTimeout(50*time.Millisecond),
		WithShutdownCallback(func() {
			shutdownCalled.Add(1)
		}),
	)

	if _, err := m.Start(dir); err != nil {
		t.Fatalf("Start: %v", err)
	}

	// Wait for the idle timer to fire.
	time.Sleep(200 * time.Millisecond)

	if m.IsRunning() {
		t.Error("daemon should have auto-stopped after idle timeout")
	}

	if shutdownCalled.Load() != 1 {
		t.Errorf("shutdown callback called %d times, want 1", shutdownCalled.Load())
	}

	// Lock file should be deleted.
	_, err := ReadLockFile(dir)
	if !IsDomainError(err, envelope.CodeDaemonNotRunning) {
		t.Errorf("lock file should not exist after idle shutdown, got %v", err)
	}
}

func TestManager_ResetIdleTimer(t *testing.T) {
	dir := t.TempDir()

	m := NewManager(WithIdleTimeout(100 * time.Millisecond))

	if _, err := m.Start(dir); err != nil {
		t.Fatalf("Start: %v", err)
	}

	// Reset timer before it fires.
	time.Sleep(60 * time.Millisecond)
	m.ResetIdleTimer()

	// Wait less than the idle timeout after reset.
	time.Sleep(60 * time.Millisecond)
	if !m.IsRunning() {
		t.Error("daemon should still be running after timer reset")
	}

	// Wait for timeout to fire after reset.
	time.Sleep(100 * time.Millisecond)
	if m.IsRunning() {
		t.Error("daemon should have stopped after idle timeout")
	}
}

func TestManager_StartAfterStop(t *testing.T) {
	dir := t.TempDir()
	m := NewManager(WithIdleTimeout(0))

	// Start → Stop → Start again should work.
	if _, err := m.Start(dir); err != nil {
		t.Fatalf("first Start: %v", err)
	}
	if err := m.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	info, err := m.Start(dir)
	if err != nil {
		t.Fatalf("second Start: %v", err)
	}
	defer func() { _ = m.Stop() }()

	if !m.IsRunning() {
		t.Error("not running after second Start")
	}
	if info.Port <= 0 {
		t.Error("port should be positive after restart")
	}
}

func TestManager_StaleLockCleanup(t *testing.T) {
	dir := setupTestDir(t)

	// Write a stale lock file with a dead PID.
	stale := &DaemonInfo{
		PID:        999999,
		Port:       12345,
		Token:      "stale",
		ProjectDir: dir,
		StartedAt:  time.Now().Add(-time.Hour),
	}
	if err := WriteLockFile(dir, stale); err != nil {
		t.Fatalf("writing stale lock file: %v", err)
	}

	m := NewManager(WithIdleTimeout(0))

	// Start should clean up the stale lock and succeed.
	info, err := m.Start(dir)
	if err != nil {
		t.Fatalf("Start with stale lock: %v", err)
	}
	defer func() { _ = m.Stop() }()

	if info.Port == 12345 {
		t.Error("should have picked a new port, not reused stale one")
	}
}

func TestManager_WithDefaults(t *testing.T) {
	m := NewManager()
	if m.idleTimeout != DefaultIdleTimeout {
		t.Errorf("default idle timeout = %v, want %v", m.idleTimeout, DefaultIdleTimeout)
	}
}

// --- Integration tests ---

func TestManager_FullCycle(t *testing.T) {
	dir := t.TempDir()
	m := NewManager(WithIdleTimeout(0))

	// Start.
	info, err := m.Start(dir)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if !m.IsRunning() {
		t.Fatal("not running after Start")
	}

	// Status.
	status, err := m.Status()
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if status.Port != info.Port {
		t.Errorf("port mismatch in status")
	}

	// Shutdown.
	if err := m.Shutdown(); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}
	if m.IsRunning() {
		t.Fatal("running after Shutdown")
	}

	// Restart.
	info2, err := m.Start(dir)
	if err != nil {
		t.Fatalf("Restart: %v", err)
	}
	defer func() { _ = m.Stop() }()

	if info2.Token == info.Token {
		t.Error("token should differ after restart")
	}
}

func TestManager_ConcurrentStatusAccess(t *testing.T) {
	dir := t.TempDir()
	m := NewManager(WithIdleTimeout(0))

	if _, err := m.Start(dir); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer func() { _ = m.Stop() }()

	// Hammer Status and IsRunning from multiple goroutines.
	const n = 50
	errs := make(chan error, n)

	for i := 0; i < n; i++ {
		go func() {
			if !m.IsRunning() {
				errs <- fmt.Errorf("IsRunning returned false")
				return
			}
			if _, err := m.Status(); err != nil {
				errs <- err
				return
			}
			errs <- nil
		}()
	}

	for i := 0; i < n; i++ {
		if err := <-errs; err != nil {
			t.Fatalf("concurrent access error: %v", err)
		}
	}
}

func TestManager_StartupSpeed(t *testing.T) {
	dir := t.TempDir()
	m := NewManager(WithIdleTimeout(0))

	start := time.Now()
	if _, err := m.Start(dir); err != nil {
		t.Fatalf("Start: %v", err)
	}
	elapsed := time.Since(start)
	defer func() { _ = m.Stop() }()

	// Startup must complete in < 500ms (acceptance criterion).
	if elapsed > 500*time.Millisecond {
		t.Errorf("startup took %v, want < 500ms", elapsed)
	}
}
