package daemon

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/khanhnguyen/promptman/pkg/envelope"
)

// --- DomainError tests ---

func TestDomainError_Error(t *testing.T) {
	err := ErrDaemonAlreadyRunning
	expected := "DAEMON_ALREADY_RUNNING: daemon already running"
	if err.Error() != expected {
		t.Errorf("Error() = %q, want %q", err.Error(), expected)
	}
}

func TestDomainError_Wrap(t *testing.T) {
	wrapped := ErrDaemonNotRunning.Wrap("custom message")
	if wrapped.Code != envelope.CodeDaemonNotRunning {
		t.Errorf("Code = %q, want %q", wrapped.Code, envelope.CodeDaemonNotRunning)
	}
	if wrapped.Message != "custom message" {
		t.Errorf("Message = %q, want %q", wrapped.Message, "custom message")
	}
}

func TestDomainError_Wrapf(t *testing.T) {
	wrapped := ErrPortUnavailable.Wrapf("port %d is taken", 8080)
	if wrapped.Code != envelope.CodePortUnavailable {
		t.Errorf("Code = %q, want %q", wrapped.Code, envelope.CodePortUnavailable)
	}
	expected := "port 8080 is taken"
	if wrapped.Message != expected {
		t.Errorf("Message = %q, want %q", wrapped.Message, expected)
	}
}

func TestIsDomainError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		code string
		want bool
	}{
		{"matching code", ErrDaemonAlreadyRunning, envelope.CodeDaemonAlreadyRunning, true},
		{"wrong code", ErrDaemonAlreadyRunning, envelope.CodeDaemonNotRunning, false},
		{"nil error type", nil, envelope.CodeDaemonAlreadyRunning, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsDomainError(tt.err, tt.code); got != tt.want {
				t.Errorf("IsDomainError() = %v, want %v", got, tt.want)
			}
		})
	}
}

// --- Lock file tests ---

func setupTestDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	// Create the .promptman subdirectory.
	if err := os.MkdirAll(filepath.Join(dir, promptmanDir), 0o755); err != nil {
		t.Fatalf("creating .promptman dir: %v", err)
	}
	return dir
}

func TestWriteAndReadLockFile(t *testing.T) {
	dir := setupTestDir(t)

	info := &DaemonInfo{
		PID:        12345,
		Port:       48721,
		Token:      "abc123",
		ProjectDir: dir,
		StartedAt:  time.Date(2026, 3, 10, 10, 30, 0, 0, time.UTC),
	}

	// Write.
	if err := WriteLockFile(dir, info); err != nil {
		t.Fatalf("WriteLockFile: %v", err)
	}

	// Read back.
	got, err := ReadLockFile(dir)
	if err != nil {
		t.Fatalf("ReadLockFile: %v", err)
	}

	if got.PID != info.PID {
		t.Errorf("PID = %d, want %d", got.PID, info.PID)
	}
	if got.Port != info.Port {
		t.Errorf("Port = %d, want %d", got.Port, info.Port)
	}
	if got.Token != info.Token {
		t.Errorf("Token = %q, want %q", got.Token, info.Token)
	}
	if got.ProjectDir != info.ProjectDir {
		t.Errorf("ProjectDir = %q, want %q", got.ProjectDir, info.ProjectDir)
	}
}

func TestWriteLockFile_CreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	// Don't create .promptman — WriteLockFile should create it.

	info := &DaemonInfo{PID: 1, Port: 8080, Token: "t", ProjectDir: dir, StartedAt: time.Now()}
	if err := WriteLockFile(dir, info); err != nil {
		t.Fatalf("WriteLockFile: %v", err)
	}

	// Verify the file exists.
	path := lockFilePath(dir)
	if _, err := os.Stat(path); err != nil {
		t.Errorf("lock file not created: %v", err)
	}
}

func TestReadLockFile_NotFound(t *testing.T) {
	dir := t.TempDir()

	_, err := ReadLockFile(dir)
	if !IsDomainError(err, envelope.CodeDaemonNotRunning) {
		t.Errorf("expected ErrDaemonNotRunning, got %v", err)
	}
}

func TestReadLockFile_CorruptJSON(t *testing.T) {
	dir := setupTestDir(t)

	// Write invalid JSON.
	path := lockFilePath(dir)
	if err := os.WriteFile(path, []byte("not json"), 0o600); err != nil {
		t.Fatalf("writing corrupt file: %v", err)
	}

	_, err := ReadLockFile(dir)
	if !IsDomainError(err, envelope.CodeLockFileCorrupt) {
		t.Errorf("expected ErrLockFileCorrupt, got %v", err)
	}
}

func TestReadLockFile_InvalidPID(t *testing.T) {
	dir := setupTestDir(t)

	// Write valid JSON but with invalid PID.
	info := map[string]any{
		"pid":        0,
		"port":       8080,
		"token":      "t",
		"projectDir": dir,
		"startedAt":  time.Now().UTC().Format(time.RFC3339),
	}
	data, _ := json.Marshal(info)
	path := lockFilePath(dir)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("writing file: %v", err)
	}

	_, err := ReadLockFile(dir)
	if !IsDomainError(err, envelope.CodeLockFileCorrupt) {
		t.Errorf("expected ErrLockFileCorrupt for invalid PID, got %v", err)
	}
}

func TestDeleteLockFile(t *testing.T) {
	dir := setupTestDir(t)

	// Write a lock file.
	info := &DaemonInfo{PID: 1, Port: 8080, Token: "t", ProjectDir: dir, StartedAt: time.Now()}
	if err := WriteLockFile(dir, info); err != nil {
		t.Fatalf("WriteLockFile: %v", err)
	}

	// Delete it.
	if err := DeleteLockFile(dir); err != nil {
		t.Fatalf("DeleteLockFile: %v", err)
	}

	// Verify it's gone.
	path := lockFilePath(dir)
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("lock file still exists after delete")
	}
}

func TestDeleteLockFile_NoOp(t *testing.T) {
	dir := t.TempDir()

	// Deleting a non-existent lock file should not error.
	if err := DeleteLockFile(dir); err != nil {
		t.Errorf("DeleteLockFile on non-existent file: %v", err)
	}
}

// --- PID validation tests ---

func TestIsPIDAlive_CurrentProcess(t *testing.T) {
	// Current process should be alive.
	if !IsPIDAlive(os.Getpid()) {
		t.Error("IsPIDAlive(os.Getpid()) = false, want true")
	}
}

func TestIsPIDAlive_DeadProcess(t *testing.T) {
	// PID -1 should not be alive.
	if IsPIDAlive(-1) {
		t.Error("IsPIDAlive(-1) = true, want false")
	}
}

// --- Stale lock file cleanup ---

func TestCleanStaleLockFile_NoLockFile(t *testing.T) {
	dir := t.TempDir()

	info, err := CleanStaleLockFile(dir)
	if err != nil {
		t.Fatalf("CleanStaleLockFile: %v", err)
	}
	if info != nil {
		t.Errorf("expected nil info when no lock file, got %+v", info)
	}
}

func TestCleanStaleLockFile_AlivePID(t *testing.T) {
	dir := setupTestDir(t)

	// Write a lock file with our own PID (alive).
	info := NewDaemonInfo(8080, "token", dir)
	if err := WriteLockFile(dir, info); err != nil {
		t.Fatalf("WriteLockFile: %v", err)
	}

	got, err := CleanStaleLockFile(dir)
	if err != nil {
		t.Fatalf("CleanStaleLockFile: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil info for alive PID")
	}
	if got.PID != os.Getpid() {
		t.Errorf("PID = %d, want %d", got.PID, os.Getpid())
	}
}

func TestCleanStaleLockFile_DeadPID(t *testing.T) {
	dir := setupTestDir(t)

	// Write a lock file with a fake dead PID.
	info := &DaemonInfo{
		PID:        999999, // very unlikely to be alive
		Port:       8080,
		Token:      "t",
		ProjectDir: dir,
		StartedAt:  time.Now(),
	}
	if err := WriteLockFile(dir, info); err != nil {
		t.Fatalf("WriteLockFile: %v", err)
	}

	got, err := CleanStaleLockFile(dir)
	if err != nil {
		t.Fatalf("CleanStaleLockFile: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil info for dead PID, got %+v", got)
	}

	// Lock file should be removed.
	if _, statErr := os.Stat(lockFilePath(dir)); !os.IsNotExist(statErr) {
		t.Error("lock file not cleaned up for dead PID")
	}
}

func TestCleanStaleLockFile_CorruptFile(t *testing.T) {
	dir := setupTestDir(t)

	// Write corrupt content.
	path := lockFilePath(dir)
	if err := os.WriteFile(path, []byte("{bad"), 0o600); err != nil {
		t.Fatalf("writing corrupt file: %v", err)
	}

	got, err := CleanStaleLockFile(dir)
	if err != nil {
		t.Fatalf("CleanStaleLockFile: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil info for corrupt file, got %+v", got)
	}

	// Corrupt file should be cleaned up.
	if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
		t.Error("corrupt lock file not cleaned up")
	}
}

// --- Port selection tests ---

func TestPickRandomPort(t *testing.T) {
	port, err := PickRandomPort()
	if err != nil {
		t.Fatalf("PickRandomPort: %v", err)
	}
	if port <= 0 || port > 65535 {
		t.Errorf("port %d out of valid range", port)
	}
}

func TestPickRandomPort_ReturnsDifferentPorts(t *testing.T) {
	port1, err := PickRandomPort()
	if err != nil {
		t.Fatalf("PickRandomPort 1: %v", err)
	}

	port2, err := PickRandomPort()
	if err != nil {
		t.Fatalf("PickRandomPort 2: %v", err)
	}

	// It's theoretically possible for the same port to be picked twice,
	// but extremely unlikely. We only log a warning.
	if port1 == port2 {
		t.Logf("WARNING: same port picked twice: %d (unlikely but possible)", port1)
	}
}

// --- Token generation tests ---

func TestGenerateToken(t *testing.T) {
	token, err := GenerateToken()
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}

	// 32 bytes → 64 hex characters.
	if len(token) != 64 {
		t.Errorf("token length = %d, want 64", len(token))
	}
}

func TestGenerateToken_Unique(t *testing.T) {
	token1, _ := GenerateToken()
	token2, _ := GenerateToken()

	if token1 == token2 {
		t.Errorf("two tokens are identical: %s", token1)
	}
}

// --- DaemonInfo construction ---

func TestNewDaemonInfo(t *testing.T) {
	before := time.Now().UTC()
	info := NewDaemonInfo(8080, "mytoken", "/tmp/project")
	after := time.Now().UTC()

	if info.PID != os.Getpid() {
		t.Errorf("PID = %d, want %d", info.PID, os.Getpid())
	}
	if info.Port != 8080 {
		t.Errorf("Port = %d, want 8080", info.Port)
	}
	if info.Token != "mytoken" {
		t.Errorf("Token = %q, want %q", info.Token, "mytoken")
	}
	if info.ProjectDir != "/tmp/project" {
		t.Errorf("ProjectDir = %q, want %q", info.ProjectDir, "/tmp/project")
	}
	if info.StartedAt.Before(before) || info.StartedAt.After(after) {
		t.Errorf("StartedAt = %v, want between %v and %v", info.StartedAt, before, after)
	}
}

// --- Integration tests ---

func TestLockFile_JSONFormat(t *testing.T) {
	dir := setupTestDir(t)

	info := &DaemonInfo{
		PID:        12345,
		Port:       48721,
		Token:      "a1b2c3d4e5f6",
		ProjectDir: "/path/to/project",
		StartedAt:  time.Date(2026, 3, 10, 10, 30, 0, 0, time.UTC),
	}

	if err := WriteLockFile(dir, info); err != nil {
		t.Fatalf("WriteLockFile: %v", err)
	}

	// Read raw JSON and verify field names match spec.
	data, err := os.ReadFile(lockFilePath(dir))
	if err != nil {
		t.Fatalf("reading lock file: %v", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("parsing lock file JSON: %v", err)
	}

	expectedFields := []string{"pid", "port", "token", "projectDir", "startedAt"}
	for _, field := range expectedFields {
		if _, ok := raw[field]; !ok {
			t.Errorf("missing JSON field %q in lock file", field)
		}
	}
}

func TestLockFile_FilePermissions(t *testing.T) {
	dir := setupTestDir(t)

	info := &DaemonInfo{PID: 1, Port: 8080, Token: "t", ProjectDir: dir, StartedAt: time.Now()}
	if err := WriteLockFile(dir, info); err != nil {
		t.Fatalf("WriteLockFile: %v", err)
	}

	path := lockFilePath(dir)
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat lock file: %v", err)
	}

	// File should be 0600 (owner read/write only).
	perm := fi.Mode().Perm()
	if perm != 0o600 {
		t.Errorf("lock file permissions = %o, want 0600", perm)
	}
}

func TestConcurrentPortPicks(t *testing.T) {
	const n = 10
	ports := make(chan int, n)
	errs := make(chan error, n)

	for i := 0; i < n; i++ {
		go func() {
			port, err := PickRandomPort()
			if err != nil {
				errs <- err
				return
			}
			ports <- port
		}()
	}

	seen := make(map[int]bool)
	for i := 0; i < n; i++ {
		select {
		case err := <-errs:
			t.Fatalf("concurrent port pick failed: %v", err)
		case port := <-ports:
			if port <= 0 {
				t.Errorf("invalid port %d", port)
			}
			if seen[port] {
				// Log but don't fail — duplicates are possible but unlikely.
				t.Logf("WARNING: duplicate port %d", port)
			}
			seen[port] = true
		}
	}
}
