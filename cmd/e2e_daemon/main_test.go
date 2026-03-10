package e2e

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/khanhnguyen/promptman/internal/daemon"
	"github.com/khanhnguyen/promptman/pkg/envelope"
)

func TestDaemonLifecycleE2E(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	// 1. Build the daemon binary
	binPath := filepath.Join(t.TempDir(), "promptman-daemon")
	buildCmd := exec.Command("go", "build", "-o", binPath, "../daemon")
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build daemon: %v\n%s", err, string(out))
	}

	// 2. Create a temporary project directory
	projectDir := t.TempDir()

	// 3. Start the daemon
	cmd := exec.Command(binPath, "start", "--project-dir", projectDir)

	// Capture stderr for debugging
	cmd.Stderr = os.Stderr

	// We want to kill it if the test fails or hangs
	err := cmd.Start()
	if err != nil {
		t.Fatalf("failed to start daemon: %v", err)
	}
	defer func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
	}()

	// 4. Wait for the lock file to be created
	var info *daemon.DaemonInfo
	lockFile := filepath.Join(projectDir, ".promptman", ".daemon.lock")

	// Wait up to 5 seconds for lock file
	for i := 0; i < 50; i++ {
		if _, err := os.Stat(lockFile); err == nil {
			// Try reading it
			info, err = daemon.ReadLockFile(projectDir)
			if err == nil {
				break
			}
		}
		time.Sleep(100 * time.Millisecond)
	}

	if info == nil {
		t.Fatalf("timed out waiting for .daemon.lock file or invalid format")
	}

	// Verify PID
	if info.PID != cmd.Process.Pid {
		t.Errorf("expected PID %d, got %d", cmd.Process.Pid, info.PID)
	}

	// 5. Query /api/v1/status
	addr := fmt.Sprintf("127.0.0.1:%d", info.Port)
	client := &http.Client{Timeout: 5 * time.Second}
	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("http://%s/api/v1/status", addr), nil)
	req.Header.Set("Authorization", "Bearer "+info.Token)

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("failed to call /api/v1/status: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var env envelope.Envelope
	if err := json.Unmarshal(body, &env); err != nil {
		t.Fatalf("failed to parse envelope: %v", err)
	}

	// Ensure the response data has the right pid
	dataMap, ok := env.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("expected data object, got %T", env.Data)
	}
	if pid, ok := dataMap["pid"].(float64); !ok || int(pid) != cmd.Process.Pid {
		t.Errorf("expected JSON response PID %d, got %v", cmd.Process.Pid, dataMap["pid"])
	}

	// 6. Test SIGTERM + Graceful Shutdown
	if err := cmd.Process.Signal(syscall.SIGINT); err != nil {
		t.Fatalf("failed to send SIGINT: %v", err)
	}

	// Wait for process to exit
	errCh := make(chan error, 1)
	go func() {
		errCh <- cmd.Wait()
	}()

	select {
	case <-time.After(5 * time.Second):
		t.Fatalf("daemon did not shut down gracefully after 5s")
	case err := <-errCh:
		// Expected to exit with code 0
		if err != nil {
			t.Errorf("daemon exit error: %v", err)
		}
	}

	// 7. Verify lock file is deleted
	if _, err := os.Stat(lockFile); !os.IsNotExist(err) {
		t.Errorf("expected lock file to be deleted, but it exists or stat error: %v", err)
	}
}
