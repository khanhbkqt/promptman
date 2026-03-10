package cli_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/khanhnguyen/promptman/internal/cli"
)

func TestVersionCommand_Output(t *testing.T) {
	root := cli.NewRootCommand()
	root.SetArgs([]string{"version"})

	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	output := buf.String()
	// Default values when not built with ldflags.
	if !strings.Contains(output, "promptman") {
		t.Errorf("expected 'promptman' in version output, got: %s", output)
	}
	if !strings.Contains(output, "dev") {
		t.Errorf("expected default version 'dev' in output, got: %s", output)
	}
	if !strings.Contains(output, "commit:") {
		t.Errorf("expected 'commit:' label in version output, got: %s", output)
	}
	if !strings.Contains(output, "built:") {
		t.Errorf("expected 'built:' label in version output, got: %s", output)
	}
}

func TestVersionCommand_WithCustomVersion(t *testing.T) {
	// Save originals and restore after test.
	origVersion := cli.Version
	origCommit := cli.Commit
	origDate := cli.Date
	t.Cleanup(func() {
		cli.Version = origVersion
		cli.Commit = origCommit
		cli.Date = origDate
	})

	cli.Version = "1.2.3"
	cli.Commit = "abc1234"
	cli.Date = "2026-03-10"

	root := cli.NewRootCommand()
	root.SetArgs([]string{"version"})

	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "1.2.3") {
		t.Errorf("expected version '1.2.3' in output, got: %s", output)
	}
	if !strings.Contains(output, "abc1234") {
		t.Errorf("expected commit 'abc1234' in output, got: %s", output)
	}
	if !strings.Contains(output, "2026-03-10") {
		t.Errorf("expected date '2026-03-10' in output, got: %s", output)
	}
}

func TestVersionCommand_ExitCode(t *testing.T) {
	root := cli.NewRootCommand()
	root.SetArgs([]string{"version"})

	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)

	err := root.Execute()
	if err != nil {
		t.Errorf("version command should exit with code 0 (nil error), got: %v", err)
	}
}
