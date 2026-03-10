package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/khanhnguyen/promptman/internal/collection"
	"github.com/khanhnguyen/promptman/internal/config"
	"github.com/khanhnguyen/promptman/internal/environment"
	"github.com/khanhnguyen/promptman/pkg/envelope"
	"github.com/khanhnguyen/promptman/pkg/fsutil"
	"github.com/spf13/cobra"
)

const (
	// promptmanDirName is the name of the project metadata directory.
	promptmanDirName = ".promptman"
)

// gitignoreEntries are the entries added to .gitignore during project init.
var gitignoreEntries = []string{
	".promptman/.daemon.lock",
	"*.secrets.yaml",
	".promptman/history/",
}

// initFlags holds the flags specific to the init subcommand.
type initFlags struct {
	// force reinitializes even if .promptman/ already exists.
	force bool
}

// newInitCommand creates the "init" subcommand for scaffolding a new project.
func newInitCommand(globals *GlobalFlags) *cobra.Command {
	iflags := &initFlags{}

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a new promptman project",
		Long: `Create a .promptman/ directory structure with default configuration,
an example collection, and an example environment.

This command is purely local — it does not require the daemon to be running.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return executeInit(cmd, globals, iflags)
		},
	}

	cmd.Flags().BoolVar(&iflags.force, "force", false,
		"Reinitialize even if .promptman/ already exists")

	return cmd
}

// executeInit scaffolds the .promptman/ directory structure.
// It creates config.yaml, an example collection, an example environment,
// and appends entries to .gitignore. This is purely local file I/O —
// no daemon interaction is needed.
func executeInit(cmd *cobra.Command, globals *GlobalFlags, iflags *initFlags) error {
	projectDir := globals.ProjectDir
	pmDir := filepath.Join(projectDir, promptmanDirName)

	// Check if .promptman/ already exists.
	if info, err := os.Stat(pmDir); err == nil && info.IsDir() {
		if !iflags.force {
			return writeErrorEnvelope(cmd, globals, "INIT_ALREADY_EXISTS",
				"Project already initialized. Use --force to reinitialize.")
		}
	}

	// Create directory structure.
	dirs := []string{
		filepath.Join(pmDir, "collections"),
		filepath.Join(pmDir, "environments"),
		filepath.Join(pmDir, "tests"),
		filepath.Join(pmDir, "history"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return writeErrorEnvelope(cmd, globals, "INIT_DIR_ERROR",
				fmt.Sprintf("creating directory %s: %v", dir, err))
		}
	}

	// Write config.yaml with defaults.
	configPath := filepath.Join(pmDir, "config.yaml")
	if err := fsutil.WriteYAML(configPath, config.DefaultConfig()); err != nil {
		return writeErrorEnvelope(cmd, globals, "INIT_CONFIG_ERROR",
			fmt.Sprintf("writing config.yaml: %v", err))
	}

	// Write example collection.
	exampleCollection := &collection.Collection{
		Name:    "Example Collection",
		BaseURL: "https://httpbin.org",
		Requests: []collection.Request{
			{
				ID:     "health",
				Method: "GET",
				Path:   "/get",
			},
		},
	}
	collectionPath := filepath.Join(pmDir, "collections", "example.yaml")
	if err := fsutil.WriteYAML(collectionPath, exampleCollection); err != nil {
		return writeErrorEnvelope(cmd, globals, "INIT_COLLECTION_ERROR",
			fmt.Sprintf("writing example collection: %v", err))
	}

	// Write example environment.
	exampleEnv := &environment.Environment{
		Name: "dev",
		Variables: map[string]any{
			"host": "localhost",
			"port": 8080,
		},
	}
	envPath := filepath.Join(pmDir, "environments", "dev.yaml")
	if err := fsutil.WriteYAML(envPath, exampleEnv); err != nil {
		return writeErrorEnvelope(cmd, globals, "INIT_ENV_ERROR",
			fmt.Sprintf("writing example environment: %v", err))
	}

	// Append .gitignore entries.
	gitignorePath := filepath.Join(projectDir, ".gitignore")
	if err := appendGitignore(gitignorePath, gitignoreEntries); err != nil {
		return writeErrorEnvelope(cmd, globals, "INIT_GITIGNORE_ERROR",
			fmt.Sprintf("updating .gitignore: %v", err))
	}

	// Output success.
	result := map[string]any{
		"message":    "Project initialized successfully",
		"projectDir": pmDir,
		"created": []string{
			"config.yaml",
			"collections/example.yaml",
			"environments/dev.yaml",
			".gitignore entries",
		},
	}

	formatter, fmtErr := NewFormatter(globals.Format)
	if fmtErr != nil {
		return fmtErr
	}
	return formatter.Format(cmd.OutOrStdout(), localSuccess(result))
}

// appendGitignore appends entries to a .gitignore file, creating it if
// it does not exist. Entries that already exist in the file are skipped.
func appendGitignore(path string, entries []string) error {
	existing := make(map[string]bool)

	// Read existing entries if the file already exists.
	if f, err := os.Open(path); err == nil {
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			existing[strings.TrimSpace(scanner.Text())] = true
		}
		_ = f.Close()
	}

	// Collect new entries that are not already present.
	var toAdd []string
	for _, entry := range entries {
		if !existing[entry] {
			toAdd = append(toAdd, entry)
		}
	}

	if len(toAdd) == 0 {
		return nil
	}

	// Append to file (create if not exists).
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("opening .gitignore: %w", err)
	}
	defer func() { _ = f.Close() }()

	// Add a blank line separator before our entries.
	content := "\n# Promptman\n"
	for _, entry := range toAdd {
		content += entry + "\n"
	}

	if _, err := f.WriteString(content); err != nil {
		return fmt.Errorf("writing .gitignore: %w", err)
	}

	return nil
}

// localSuccess creates a success envelope.Envelope for local commands
// that don't go through the daemon.
func localSuccess(data any) *envelope.Envelope {
	return envelope.Success(data)
}
