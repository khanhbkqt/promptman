package main

import (
	"fmt"
	"os"

	"github.com/khanhnguyen/promptman/internal/collection"
	"github.com/khanhnguyen/promptman/internal/daemon"
	"github.com/khanhnguyen/promptman/internal/environment"
	"github.com/khanhnguyen/promptman/internal/request"
	"github.com/khanhnguyen/promptman/internal/ws"
	"github.com/khanhnguyen/promptman/pkg/fsutil"
	"github.com/spf13/cobra"
)

func main() {
	if err := rootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func rootCmd() *cobra.Command {
	var projectDir string

	root := &cobra.Command{
		Use:   "daemon",
		Short: "Promptman Daemon",
		Long:  "The Promptman daemon handles local backend duties for project execution.",
	}

	startCmd := &cobra.Command{
		Use:   "start",
		Short: "Start the daemon",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStart(projectDir)
		},
	}

	startCmd.Flags().StringVar(&projectDir, "project-dir", ".", "Project directory")
	root.AddCommand(startCmd)

	return root
}

func runStart(projectDir string) error {
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

	// 3. Engine
	engine := request.NewEngine(collSvc, envSvc, request.WithCollectionGetter(collSvc))

	// 4. Registrars
	reqReg := daemon.NewRequestRegistrar(engine)
	envReg := daemon.NewEnvironmentRegistrar(envSvc)

	// 5. Infra
	hub := ws.NewHub()

	mgr := daemon.NewManager()

	// 6. Server
	srv := daemon.NewServer(mgr, reqReg, envReg)
	srv.WithHub(hub)

	// Start manager
	info, err := mgr.Start(pDir)
	if err != nil {
		return fmt.Errorf("failed to start manager: %w", err)
	}

	// Start server on allocated port
	addr := fmt.Sprintf("127.0.0.1:%d", info.Port)
	if err := srv.Start(addr, info.Token); err != nil {
		_ = mgr.Stop()
		return fmt.Errorf("failed to start server: %w", err)
	}

	fmt.Fprintf(os.Stdout, "Daemon started on %s (pid: %d) inside %s\n", addr, info.PID, info.ProjectDir)

	// Temporarily block to keep active (will add signal handling in next task)
	select {}
}
