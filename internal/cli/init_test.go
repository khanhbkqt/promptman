package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/khanhnguyen/promptman/internal/config"
	"github.com/khanhnguyen/promptman/pkg/fsutil"
)

func TestInit_CreatesDirectoryStructure(t *testing.T) {
	tmpDir := t.TempDir()
	globals := &GlobalFlags{Format: FormatJSON, ProjectDir: tmpDir}
	iflags := &initFlags{}

	root := NewRootCommand()
	root.SetArgs([]string{})

	err := executeInit(root, globals, iflags)
	if err != nil {
		t.Fatalf("executeInit failed: %v", err)
	}

	// Verify all directories were created.
	expectedDirs := []string{
		".promptman",
		".promptman/collections",
		".promptman/environments",
		".promptman/tests",
		".promptman/history",
	}
	for _, dir := range expectedDirs {
		path := filepath.Join(tmpDir, dir)
		info, err := os.Stat(path)
		if err != nil {
			t.Errorf("directory %s not created: %v", dir, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("%s is not a directory", dir)
		}
	}
}

func TestInit_ConfigYAML(t *testing.T) {
	tmpDir := t.TempDir()
	globals := &GlobalFlags{Format: FormatJSON, ProjectDir: tmpDir}
	iflags := &initFlags{}

	root := NewRootCommand()
	root.SetArgs([]string{})

	if err := executeInit(root, globals, iflags); err != nil {
		t.Fatalf("executeInit failed: %v", err)
	}

	// Read and verify config.yaml matches defaults.
	configPath := filepath.Join(tmpDir, ".promptman", "config.yaml")
	var cfg config.ProjectConfig
	if err := fsutil.ReadYAML(configPath, &cfg); err != nil {
		t.Fatalf("reading config.yaml: %v", err)
	}

	defaults := config.DefaultConfig()
	if cfg.Daemon.AutoStart != defaults.Daemon.AutoStart {
		t.Errorf("daemon.autoStart = %v, want %v", cfg.Daemon.AutoStart, defaults.Daemon.AutoStart)
	}
	if cfg.Daemon.ShutdownAfter != defaults.Daemon.ShutdownAfter {
		t.Errorf("daemon.shutdownAfter = %q, want %q", cfg.Daemon.ShutdownAfter, defaults.Daemon.ShutdownAfter)
	}
}

func TestInit_ExampleCollection(t *testing.T) {
	tmpDir := t.TempDir()
	globals := &GlobalFlags{Format: FormatJSON, ProjectDir: tmpDir}
	iflags := &initFlags{}

	root := NewRootCommand()
	root.SetArgs([]string{})

	if err := executeInit(root, globals, iflags); err != nil {
		t.Fatalf("executeInit failed: %v", err)
	}

	// Verify example collection is valid YAML.
	collectionPath := filepath.Join(tmpDir, ".promptman", "collections", "example.yaml")
	data, err := os.ReadFile(collectionPath)
	if err != nil {
		t.Fatalf("reading example.yaml: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "Example Collection") {
		t.Error("expected 'Example Collection' in example.yaml")
	}
	if !strings.Contains(content, "httpbin.org") {
		t.Error("expected 'httpbin.org' in example.yaml")
	}
}

func TestInit_ExampleEnvironment(t *testing.T) {
	tmpDir := t.TempDir()
	globals := &GlobalFlags{Format: FormatJSON, ProjectDir: tmpDir}
	iflags := &initFlags{}

	root := NewRootCommand()
	root.SetArgs([]string{})

	if err := executeInit(root, globals, iflags); err != nil {
		t.Fatalf("executeInit failed: %v", err)
	}

	// Verify example environment is valid YAML.
	envPath := filepath.Join(tmpDir, ".promptman", "environments", "dev.yaml")
	data, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("reading dev.yaml: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "dev") {
		t.Error("expected 'dev' in dev.yaml")
	}
	if !strings.Contains(content, "host") {
		t.Error("expected 'host' variable in dev.yaml")
	}
	if !strings.Contains(content, "port") {
		t.Error("expected 'port' variable in dev.yaml")
	}
}

func TestInit_GitignoreEntries(t *testing.T) {
	t.Run("creates new gitignore", func(t *testing.T) {
		tmpDir := t.TempDir()
		globals := &GlobalFlags{Format: FormatJSON, ProjectDir: tmpDir}
		iflags := &initFlags{}

		root := NewRootCommand()
		root.SetArgs([]string{})

		if err := executeInit(root, globals, iflags); err != nil {
			t.Fatalf("executeInit failed: %v", err)
		}

		gitignorePath := filepath.Join(tmpDir, ".gitignore")
		data, err := os.ReadFile(gitignorePath)
		if err != nil {
			t.Fatalf("reading .gitignore: %v", err)
		}
		content := string(data)
		for _, entry := range gitignoreEntries {
			if !strings.Contains(content, entry) {
				t.Errorf(".gitignore missing entry: %s", entry)
			}
		}
	})

	t.Run("appends to existing gitignore", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create existing .gitignore.
		existingContent := "node_modules/\n.env\n"
		gitignorePath := filepath.Join(tmpDir, ".gitignore")
		if err := os.WriteFile(gitignorePath, []byte(existingContent), 0644); err != nil {
			t.Fatalf("creating .gitignore: %v", err)
		}

		globals := &GlobalFlags{Format: FormatJSON, ProjectDir: tmpDir}
		iflags := &initFlags{}

		root := NewRootCommand()
		root.SetArgs([]string{})

		if err := executeInit(root, globals, iflags); err != nil {
			t.Fatalf("executeInit failed: %v", err)
		}

		data, err := os.ReadFile(gitignorePath)
		if err != nil {
			t.Fatalf("reading .gitignore: %v", err)
		}
		content := string(data)

		// Original entries preserved.
		if !strings.Contains(content, "node_modules/") {
			t.Error("original entry 'node_modules/' missing")
		}

		// New entries added.
		for _, entry := range gitignoreEntries {
			if !strings.Contains(content, entry) {
				t.Errorf(".gitignore missing entry: %s", entry)
			}
		}
	})

	t.Run("skips duplicate entries", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Pre-populate with one of the entries.
		existingContent := ".promptman/.daemon.lock\n"
		gitignorePath := filepath.Join(tmpDir, ".gitignore")
		if err := os.WriteFile(gitignorePath, []byte(existingContent), 0644); err != nil {
			t.Fatalf("creating .gitignore: %v", err)
		}

		globals := &GlobalFlags{Format: FormatJSON, ProjectDir: tmpDir}
		iflags := &initFlags{}

		root := NewRootCommand()
		root.SetArgs([]string{})

		if err := executeInit(root, globals, iflags); err != nil {
			t.Fatalf("executeInit failed: %v", err)
		}

		data, err := os.ReadFile(gitignorePath)
		if err != nil {
			t.Fatalf("reading .gitignore: %v", err)
		}

		// Count occurrences of the pre-existing entry.
		count := strings.Count(string(data), ".promptman/.daemon.lock")
		if count != 1 {
			t.Errorf(".promptman/.daemon.lock appears %d times, want 1", count)
		}
	})
}

func TestInit_AlreadyExists(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .promptman/ directory to simulate existing project.
	pmDir := filepath.Join(tmpDir, ".promptman")
	if err := os.MkdirAll(pmDir, 0755); err != nil {
		t.Fatalf("creating .promptman: %v", err)
	}

	globals := &GlobalFlags{Format: FormatJSON, ProjectDir: tmpDir}
	iflags := &initFlags{force: false}

	root := NewRootCommand()
	var buf strings.Builder
	root.SetOut(&buf)
	root.SetArgs([]string{})

	err := executeInit(root, globals, iflags)
	// Should return ExitError (from writeErrorEnvelope).
	if err == nil {
		t.Fatal("expected error for existing project")
	}
	exitErr, ok := err.(*ExitError)
	if !ok {
		t.Fatalf("expected *ExitError, got %T: %v", err, err)
	}
	if exitErr.Code != 1 {
		t.Errorf("exit code = %d, want 1", exitErr.Code)
	}

	// Verify error message mentions "already initialized".
	output := buf.String()
	if !strings.Contains(output, "already initialized") {
		t.Errorf("expected 'already initialized' in output, got: %s", output)
	}
}

func TestInit_ForceReinitializes(t *testing.T) {
	tmpDir := t.TempDir()

	// Create existing .promptman/ directory.
	pmDir := filepath.Join(tmpDir, ".promptman")
	if err := os.MkdirAll(pmDir, 0755); err != nil {
		t.Fatalf("creating .promptman: %v", err)
	}

	globals := &GlobalFlags{Format: FormatJSON, ProjectDir: tmpDir}
	iflags := &initFlags{force: true}

	root := NewRootCommand()
	root.SetArgs([]string{})

	err := executeInit(root, globals, iflags)
	if err != nil {
		t.Fatalf("executeInit with --force failed: %v", err)
	}

	// Verify files were created even though dir existed.
	configPath := filepath.Join(pmDir, "config.yaml")
	if _, err := os.Stat(configPath); err != nil {
		t.Errorf("config.yaml not created with --force: %v", err)
	}
}
