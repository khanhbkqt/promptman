package cli_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/khanhnguyen/promptman/internal/cli"
	"github.com/khanhnguyen/promptman/internal/daemon"
	"github.com/khanhnguyen/promptman/pkg/envelope"
)

// newTestClient creates an httptest.Server and a Client that communicates with
// it via the server's loopback transport.  This avoids the IPv4/IPv6 mismatch
// that occurs when httptest.NewServer binds to [::1] but NewClient hardcodes
// 127.0.0.1 — and also avoids ephemeral port exhaustion when many tests in the
// package run in parallel.
//
// The handler is called for every request; the returned Client is ready to use.
func newTestClient(t *testing.T, handler http.Handler) (*httptest.Server, *cli.Client) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	c := cli.NewClientDirect(srv.URL+"/api/v1", "test-token", srv.Client())
	return srv, c
}

// -----------------------------------------------------------------------
// NewClient tests — validate lock-file / PID-alive checks (no server needed)
// -----------------------------------------------------------------------

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

// -----------------------------------------------------------------------
// Client.Get / Post / Put tests — use loopback transport via NewClientDirect
// -----------------------------------------------------------------------

func TestClient_Get_Success(t *testing.T) {
	expected := envelope.Success(map[string]string{"status": "ok"})

	_, c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(expected)
	}))

	env, err := c.Get("/status")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !env.OK {
		t.Errorf("expected OK=true, got false (error: %v)", env.Error)
	}
}

func TestClient_Post_Success(t *testing.T) {
	expected := envelope.Success("created")

	_, c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(expected)
	}))

	env, err := c.Post("/run", map[string]string{"request": "users/health"})
	if err != nil {
		t.Fatalf("Post: %v", err)
	}
	if !env.OK {
		t.Errorf("expected OK=true, got false")
	}
}

func TestClient_Get_WrongToken(t *testing.T) {
	// Server expects "real-token" but client sends "test-token" (from newTestClient).
	_, c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer real-token" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(envelope.Success("ok"))
	}))

	// Server returns 401 with plain text → cannot decode as envelope → decode error.
	_, err := c.Get("/status")
	if err == nil {
		t.Fatal("expected error on wrong token, got nil")
	}
	if !cli.IsCLIError(err, cli.CodeResponseDecodeError) {
		t.Errorf("expected CLI_RESPONSE_DECODE_ERROR, got: %v", err)
	}
}

func TestClient_Get_DaemonUnreachable(t *testing.T) {
	// Create a client that points to a server that's already closed.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	c := cli.NewClientDirect(srv.URL+"/api/v1", "tok", &http.Client{Timeout: 1 * time.Second})
	srv.Close() // Close immediately so requests fail.

	_, err := c.Get("/status")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !cli.IsCLIError(err, cli.CodeDaemonUnreachable) {
		t.Errorf("expected CLI_DAEMON_UNREACHABLE, got %v", err)
	}
}

// -----------------------------------------------------------------------
// IsCLIError utility tests
// -----------------------------------------------------------------------

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
