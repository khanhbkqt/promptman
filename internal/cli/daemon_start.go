package cli

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/khanhnguyen/promptman/internal/collection"
	"github.com/khanhnguyen/promptman/internal/daemon"
	"github.com/khanhnguyen/promptman/internal/environment"
	"github.com/khanhnguyen/promptman/internal/history"
	"github.com/khanhnguyen/promptman/internal/request"
	"github.com/khanhnguyen/promptman/internal/ws"
	"github.com/khanhnguyen/promptman/pkg/fsutil"
)

// runDaemonStart starts the daemon process in-process. It constructs the full
// service graph (repositories → services → engine → registrars → server),
// starts the HTTP server, and blocks until SIGINT or SIGTERM is received.
//
// This function mirrors the logic in cmd/daemon/main.go:runStart() and is
// called by the hidden "daemon start" subcommand embedded in the CLI binary.
// Having it here allows spawnDaemon() in autostart.go to use os.Executable()
// (the CLI binary itself) instead of requiring a separate promptman-daemon binary.
func runDaemonStart(projectDir string) error {
	pDir := projectDir
	if pDir == "" || pDir == "." {
		resolved, err := fsutil.ProjectDir()
		if err == nil {
			pDir = resolved
		}
	}

	if err := os.Chdir(pDir); err != nil {
		return fmt.Errorf("chdir to project dir: %w", err)
	}

	// 1. Repositories
	collRepo := collection.NewFileRepository("")
	envRepo := environment.NewFileRepository("")

	// 2. Services
	collSvc := collection.NewService(collRepo)
	envSvc := environment.NewService(envRepo)

	// 2b. History service
	histDir := filepath.Join(pDir, ".promptman", "history")
	historySvc, err := history.NewService(histDir)
	if err != nil {
		return fmt.Errorf("creating history service: %w", err)
	}

	// 3. Engine — with history adapter for async append
	histAdapter := history.NewAdapter(historySvc)
	engine := request.NewEngine(collSvc, envSvc,
		request.WithCollectionGetter(collSvc),
		request.WithHistoryAppender(histAdapter),
	)

	// 4. Registrars
	reqReg := daemon.NewRequestRegistrar(engine)
	envReg := daemon.NewEnvironmentRegistrar(envSvc)
	histReg := daemon.NewHistoryRegistrar(historySvc)

	// 5. Infra
	hub := ws.NewHub()

	// Pre-declare srv so we can use it in the shutdown callback.
	var srv *daemon.Server

	mgr := daemon.NewManager(daemon.WithShutdownCallback(func() {
		if srv != nil {
			_ = srv.Shutdown()
		}
	}))

	// 6. Server
	srv = daemon.NewServer(mgr, reqReg, envReg, histReg)
	srv.WithHub(hub)

	// Start manager
	info, err := mgr.Start(pDir)
	if err != nil {
		historySvc.Close()
		return fmt.Errorf("failed to start manager: %w", err)
	}

	// Start server on allocated port
	addr := fmt.Sprintf("127.0.0.1:%d", info.Port)
	if err := srv.Start(addr, info.Token); err != nil {
		_ = mgr.Stop()
		historySvc.Close()
		return fmt.Errorf("failed to start server: %w", err)
	}

	fmt.Fprintf(os.Stdout, "Daemon started on %s (pid: %d) inside %s\n", addr, info.PID, info.ProjectDir)

	// Wait for shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	fmt.Fprintln(os.Stderr, "\n[daemon] Received shutdown signal. Shutting down...")

	// Gracefully shutdown the HTTP server and WS hub
	if err := srv.Shutdown(); err != nil {
		fmt.Fprintf(os.Stderr, "[daemon] Server shutdown error: %v\n", err)
	}

	// Flush history writer
	if err := historySvc.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "[daemon] History close error: %v\n", err)
	}

	// Terminate the manager (which deletes the lock file)
	if err := mgr.Stop(); err != nil {
		fmt.Fprintf(os.Stderr, "[daemon] Manager stop error: %v\n", err)
	}

	fmt.Fprintln(os.Stderr, "[daemon] Shutdown complete.")
	return nil
}
