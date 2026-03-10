package cli_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/khanhnguyen/promptman/internal/cli"
)

func TestRootCommand_Help(t *testing.T) {
	root := cli.NewRootCommand()
	root.SetArgs([]string{"--help"})

	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)

	// --help exits with nil error when SilenceUsage is set.
	_ = root.Execute()

	output := buf.String()
	if !strings.Contains(output, "promptman") {
		t.Errorf("expected --help to mention 'promptman', got: %s", output)
	}
	if !strings.Contains(output, "--format") {
		t.Errorf("expected --help to mention '--format' flag, got: %s", output)
	}
	if !strings.Contains(output, "--yes") {
		t.Errorf("expected --help to mention '--yes' flag, got: %s", output)
	}
	if !strings.Contains(output, "--dry-run") {
		t.Errorf("expected --help to mention '--dry-run' flag, got: %s", output)
	}
}

func TestRootCommand_FormatFlag(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "default format is json",
			args:    []string{"version"},
			wantErr: false,
		},
		{
			name:    "explicit json format",
			args:    []string{"--format", "json", "version"},
			wantErr: false,
		},
		{
			name:    "table format",
			args:    []string{"--format", "table", "version"},
			wantErr: false,
		},
		{
			name:    "minimal format",
			args:    []string{"--format", "minimal", "version"},
			wantErr: false,
		},
		{
			name:    "invalid format",
			args:    []string{"--format", "xml", "version"},
			wantErr: true,
			errMsg:  "invalid --format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := cli.NewRootCommand()
			root.SetArgs(tt.args)

			var buf bytes.Buffer
			root.SetOut(&buf)
			root.SetErr(&buf)

			err := root.Execute()
			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.errMsg != "" {
				if err == nil || !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error containing %q, got: %v", tt.errMsg, err)
				}
			}
		})
	}
}

func TestRootCommand_BoolFlags(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantYes    bool
		wantDryRun bool
	}{
		{
			name:       "defaults: yes=false, dry-run=false",
			args:       []string{"capture"},
			wantYes:    false,
			wantDryRun: false,
		},
		{
			name:       "yes flag set",
			args:       []string{"--yes", "capture"},
			wantYes:    true,
			wantDryRun: false,
		},
		{
			name:       "dry-run flag set",
			args:       []string{"--dry-run", "capture"},
			wantYes:    false,
			wantDryRun: true,
		},
		{
			name:       "both flags set",
			args:       []string{"--yes", "--dry-run", "capture"},
			wantYes:    true,
			wantDryRun: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedFlags *cli.GlobalFlags

			root := cli.NewRootCommand()
			// Add a test-only capture subcommand.
			root.AddCommand(&cobra.Command{
				Use: "capture",
				RunE: func(cmd *cobra.Command, args []string) error {
					capturedFlags = cli.GlobalFlagsFrom(cmd.Context())
					return nil
				},
			})
			root.SetArgs(tt.args)

			var buf bytes.Buffer
			root.SetOut(&buf)
			root.SetErr(&buf)

			if err := root.Execute(); err != nil {
				t.Fatalf("Execute() error = %v", err)
			}

			if capturedFlags == nil {
				t.Fatal("expected GlobalFlags in context, got nil")
			}
			if capturedFlags.Yes != tt.wantYes {
				t.Errorf("Yes = %v, want %v", capturedFlags.Yes, tt.wantYes)
			}
			if capturedFlags.DryRun != tt.wantDryRun {
				t.Errorf("DryRun = %v, want %v", capturedFlags.DryRun, tt.wantDryRun)
			}
		})
	}
}

func TestRootCommand_ProjectDir(t *testing.T) {
	root := cli.NewRootCommand()
	root.SetArgs([]string{"--project-dir", "/tmp/myproject", "version"})

	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
}

func TestGlobalFlagsFrom_NilContext(t *testing.T) {
	// Passing a context with no GlobalFlags stored should return nil.
	flags := cli.GlobalFlagsFrom(context.Background())
	if flags != nil {
		t.Errorf("expected nil from empty context, got %+v", flags)
	}
}
