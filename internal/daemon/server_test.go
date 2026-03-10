package daemon

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/khanhnguyen/promptman/pkg/envelope"
)

// TestServer_StartAndShutdown verifies the full server lifecycle:
// start listening, make a request, shut down gracefully.
func TestServer_StartAndShutdown(t *testing.T) {
	mgr := NewManager(WithIdleTimeout(0)) // disable idle timeout for test
	dir := t.TempDir()

	info, err := mgr.Start(dir)
	if err != nil {
		t.Fatalf("starting manager: %v", err)
	}

	srv := NewServer(mgr)
	addr := fmt.Sprintf("127.0.0.1:%d", info.Port)

	if err := srv.Start(addr, info.Token); err != nil {
		t.Fatalf("starting server: %v", err)
	}

	// Verify server is running.
	if !srv.IsRunning() {
		t.Fatal("server should be running after Start")
	}

	// Make an authenticated request to /api/v1/status.
	req, _ := http.NewRequest(http.MethodGet, "http://"+addr+apiPrefix+"status", nil)
	req.Header.Set("Authorization", "Bearer "+info.Token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /api/v1/status: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	// Shut down the server.
	if err := srv.Shutdown(); err != nil {
		t.Fatalf("shutting down server: %v", err)
	}

	if srv.IsRunning() {
		t.Error("server should not be running after Shutdown")
	}

	// Clean up manager.
	_ = mgr.Stop()
}

// TestServer_AuthRequired verifies that unauthenticated requests are rejected.
func TestServer_AuthRequired(t *testing.T) {
	mgr := NewManager(WithIdleTimeout(0))
	dir := t.TempDir()

	info, err := mgr.Start(dir)
	if err != nil {
		t.Fatalf("starting manager: %v", err)
	}

	srv := NewServer(mgr)
	addr := fmt.Sprintf("127.0.0.1:%d", info.Port)

	if err := srv.Start(addr, info.Token); err != nil {
		t.Fatalf("starting server: %v", err)
	}
	defer func() {
		_ = srv.Shutdown()
		_ = mgr.Stop()
	}()

	tests := []struct {
		name       string
		authHeader string
		wantStatus int
	}{
		{"no auth header", "", http.StatusUnauthorized},
		{"wrong token", "Bearer wrong-token", http.StatusUnauthorized},
		{"valid token", "Bearer " + info.Token, http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodGet, "http://"+addr+apiPrefix+"status", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.wantStatus {
				t.Errorf("status = %d, want %d", resp.StatusCode, tt.wantStatus)
			}
		})
	}
}

// TestServer_StatusEndpoint verifies the /api/v1/status response shape.
func TestServer_StatusEndpoint(t *testing.T) {
	mgr := NewManager(WithIdleTimeout(0))
	dir := t.TempDir()

	info, err := mgr.Start(dir)
	if err != nil {
		t.Fatalf("starting manager: %v", err)
	}

	srv := NewServer(mgr)
	addr := fmt.Sprintf("127.0.0.1:%d", info.Port)

	if err := srv.Start(addr, info.Token); err != nil {
		t.Fatalf("starting server: %v", err)
	}
	defer func() {
		_ = srv.Shutdown()
		_ = mgr.Stop()
	}()

	req, _ := http.NewRequest(http.MethodGet, "http://"+addr+apiPrefix+"status", nil)
	req.Header.Set("Authorization", "Bearer "+info.Token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /api/v1/status: %v", err)
	}
	defer resp.Body.Close()

	var env envelope.Envelope
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		t.Fatalf("decoding response: %v", err)
	}

	if !env.OK {
		t.Fatal("expected ok=true")
	}

	// Parse the data as DaemonInfo (it comes as map from JSON).
	dataBytes, err := json.Marshal(env.Data)
	if err != nil {
		t.Fatalf("marshalling data: %v", err)
	}

	var statusInfo DaemonInfo
	if err := json.Unmarshal(dataBytes, &statusInfo); err != nil {
		t.Fatalf("unmarshalling DaemonInfo: %v", err)
	}

	if statusInfo.PID <= 0 {
		t.Errorf("expected positive PID, got %d", statusInfo.PID)
	}
	if statusInfo.Port != info.Port {
		t.Errorf("port = %d, want %d", statusInfo.Port, info.Port)
	}
	if statusInfo.Token != info.Token {
		t.Errorf("token mismatch")
	}
	if statusInfo.ProjectDir == "" {
		t.Error("expected non-empty projectDir")
	}
	if statusInfo.Uptime == "" {
		t.Error("expected non-empty uptime")
	}
}

// TestServer_ShutdownEndpoint verifies POST /api/v1/shutdown triggers
// graceful server shutdown.
func TestServer_ShutdownEndpoint(t *testing.T) {
	mgr := NewManager(WithIdleTimeout(0))
	dir := t.TempDir()

	info, err := mgr.Start(dir)
	if err != nil {
		t.Fatalf("starting manager: %v", err)
	}

	srv := NewServer(mgr)
	addr := fmt.Sprintf("127.0.0.1:%d", info.Port)

	if err := srv.Start(addr, info.Token); err != nil {
		t.Fatalf("starting server: %v", err)
	}
	defer func() { _ = mgr.Stop() }()

	req, _ := http.NewRequest(http.MethodPost, "http://"+addr+apiPrefix+"shutdown", nil)
	req.Header.Set("Authorization", "Bearer "+info.Token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST /api/v1/shutdown: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var env envelope.Envelope
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		t.Fatalf("decoding response: %v", err)
	}

	if !env.OK {
		t.Fatal("expected ok=true")
	}

	// Wait a bit for the shutdown goroutine to complete.
	time.Sleep(200 * time.Millisecond)

	if srv.IsRunning() {
		t.Error("server should not be running after shutdown endpoint")
	}
}

// TestServer_BindsLocalhost verifies the server only accepts
// connections via 127.0.0.1 by checking the address format.
func TestServer_BindsLocalhost(t *testing.T) {
	mgr := NewManager(WithIdleTimeout(0))
	dir := t.TempDir()

	info, err := mgr.Start(dir)
	if err != nil {
		t.Fatalf("starting manager: %v", err)
	}

	srv := NewServer(mgr)
	addr := fmt.Sprintf("127.0.0.1:%d", info.Port)

	if err := srv.Start(addr, info.Token); err != nil {
		t.Fatalf("starting server: %v", err)
	}
	defer func() {
		_ = srv.Shutdown()
		_ = mgr.Stop()
	}()

	// Verify we can connect via 127.0.0.1.
	req, _ := http.NewRequest(http.MethodGet, "http://127.0.0.1:"+fmt.Sprintf("%d", info.Port)+apiPrefix+"status", nil)
	req.Header.Set("Authorization", "Bearer "+info.Token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request to 127.0.0.1 failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}
