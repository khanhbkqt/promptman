package cli_test

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/khanhnguyen/promptman/internal/cli"
	"github.com/khanhnguyen/promptman/internal/daemon"
	"github.com/khanhnguyen/promptman/pkg/envelope"
)

// writeTestLockFile writes a .daemon.lock file pointing at the given port.
// It uses the current process PID so IsPIDAlive returns true during tests.
func writeTestLockFile(t *testing.T, dir string, port int, token string) {
	t.Helper()
	info := &daemon.DaemonInfo{
		PID:        os.Getpid(),
		Port:       port,
		Token:      token,
		ProjectDir: dir,
		StartedAt:  time.Now(),
	}
	if err := daemon.WriteLockFile(dir, info); err != nil {
		t.Fatalf("writeTestLockFile: %v", err)
	}
	t.Cleanup(func() { _ = daemon.DeleteLockFile(dir) })
}

// newEnvelopeServer starts an httptest server that responds with the given envelope.
// It validates the Authorization Bearer token on every request.
func newEnvelopeServer(t *testing.T, env *envelope.Envelope, token string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer "+token {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(env)
	}))
}

// extractPort parses the port number from an httptest server URL.
func extractPort(t *testing.T, rawURL string) int {
	t.Helper()
	// URL has the form "http://127.0.0.1:PORT".
	addr := rawURL[len("http://"):]
	_, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		t.Fatalf("extractPort: SplitHostPort(%q): %v", addr, err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatalf("extractPort: Atoi(%q): %v", portStr, err)
	}
	return port
}

func TestNewClient_NoLockFile(t *testing.T) {
	dir := t.TempDir()
	_, err := cli.NewClient(dir)
	if err == nil {
		t.Fatal("expected error when no lock file, got nil")
	}
	if !cli.IsCLIError(err, cli.CodeDaemonNotRunning) {
		t.Errorf("expected CLI_DAEMON_NOT_RUNNING, got: %v", err)
	}
}

func TestNewClient_DeadPID(t *testing.T) {
	dir := t.TempDir()
	info := &daemon.DaemonInfo{
		PID:        99999999,
		Port:       12345,
		Token:      "tok",
		ProjectDir: dir,
		StartedAt:  time.Now(),
	}
	if err := daemon.WriteLockFile(dir, info); err != nil {
		t.Fatalf("setup: %v", err)
	}

	_, err := cli.NewClient(dir)
	if err == nil {
		t.Fatal("expected error when PID is dead, got nil")
	}
	if !cli.IsCLIError(err, cli.CodeDaemonNotRunning) {
		t.Errorf("expected CLI_DAEMON_NOT_RUNNING, got: %v", err)
	}
}

func TestClient_Get_Success(t *testing.T) {
	const token = "test-token-abc"
	expected := envelope.Success(map[string]string{"status": "ok"})
	srv := newEnvelopeServer(t, expected, token)
	defer srv.Close()

	port := extractPort(t, srv.URL)
	dir := t.TempDir()
	writeTestLockFile(t, dir, port, token)

	c, err := cli.NewClient(dir)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	env, err := c.Get("/status")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !env.OK {
		t.Errorf("expected OK=true, got false (error: %v)", env.Error)
	}
}

func TestClient_Post_Success(t *testing.T) {
	const token = "post-token-xyz"
	expected := envelope.Success("created")
	srv := newEnvelopeServer(t, expected, token)
	defer srv.Close()

	port := extractPort(t, srv.URL)
	dir := t.TempDir()
	writeTestLockFile(t, dir, port, token)

	c, err := cli.NewClient(dir)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	env, err := c.Post("/run", map[string]string{"request": "users/health"})
	if err != nil {
		t.Fatalf("Post: %v", err)
	}
	if !env.OK {
		t.Errorf("expected OK=true, got false")
	}
}

func TestClient_Get_WrongToken(t *testing.T) {
	// Server expects "real-token" but lock file has "wrong-token".
	srv := newEnvelopeServer(t, envelope.Success("ok"), "real-token")
	defer srv.Close()

	port := extractPort(t, srv.URL)
	dir := t.TempDir()
	writeTestLockFile(t, dir, port, "wrong-token")

	c, err := cli.NewClient(dir)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	// Server returns 401 with plain text → cannot decode as envelope → decode error.
	_, err = c.Get("/status")
	if err == nil {
		t.Fatal("expected error on wrong token, got nil")
	}
	if !cli.IsCLIError(err, cli.CodeResponseDecodeError) {
		t.Errorf("expected CLI_RESPONSE_DECODE_ERROR, got: %v", err)
	}
}

func TestClient_Get_DaemonUnreachable(t *testing.T) {
	dir := t.TempDir()
	// Write lock using current PID (alive) but port 1 (nothing listening).
	info := &daemon.DaemonInfo{
		PID:        os.Getpid(),
		Port:       1,
		Token:      "tok",
		ProjectDir: dir,
		StartedAt:  time.Now(),
	}
	if err := daemon.WriteLockFile(dir, info); err != nil {
		t.Fatalf("setup: %v", err)
	}
	t.Cleanup(func() { _ = daemon.DeleteLockFile(dir) })

	c, err := cli.NewClient(dir)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = c.Get("/status")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !cli.IsCLIError(err, cli.CodeDaemonUnreachable) {
		t.Errorf("expected CLI_DAEMON_UNREACHABLE, got %v", err)
	}
}

func TestIsCLIError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		code     string
		expected bool
	}{
		{"matching CLIError", cli.ErrDaemonNotRunning, cli.CodeDaemonNotRunning, true},
		{"non-matching CLIError", cli.ErrDaemonNotRunning, cli.CodeHTTPError, false},
		{"non-CLIError", fmt.Errorf("plain error"), cli.CodeDaemonNotRunning, false},
		{"nil error", nil, cli.CodeDaemonNotRunning, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cli.IsCLIError(tt.err, tt.code)
			if got != tt.expected {
				t.Errorf("IsCLIError(%v, %q) = %v, want %v", tt.err, tt.code, got, tt.expected)
			}
		})
	}
}
