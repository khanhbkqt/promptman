package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const (
	// FormatJSON is the machine-readable JSON output format.
	FormatJSON = "json"

	// FormatTable is the human-readable aligned table output format.
	FormatTable = "table"

	// FormatMinimal is the compact output format showing only essential data.
	FormatMinimal = "minimal"

	// DefaultFormat is the default output format for CLI commands.
	DefaultFormat = FormatJSON
)

// GlobalFlags holds the parsed global flag values accessible to all subcommands.
type GlobalFlags struct {
	// Format specifies the output format: json, table, or minimal.
	Format string

	// Yes suppresses interactive confirmation prompts.
	Yes bool

	// DryRun prints what would be done without executing.
	DryRun bool

	// ProjectDir is the path to the promptman project directory.
	// Defaults to the current working directory.
	ProjectDir string
}

// NewRootCommand builds and returns the Cobra root command with all global flags.
// It wires the Execute entry point used by cmd/cli/main.go.
func NewRootCommand() *cobra.Command {
	flags := &GlobalFlags{}

	root := &cobra.Command{
		Use:   "promptman",
		Short: "Promptman — CLI-first API development and testing tool",
		Long: `Promptman is a CLI-first HTTP API client with a local daemon.
Run HTTP requests, manage environments, and automate API workflows.`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	// Register global (persistent) flags available to all subcommands.
	pf := root.PersistentFlags()

	pf.StringVar(&flags.Format, "format", DefaultFormat,
		`Output format: json, table, or minimal`)

	pf.BoolVar(&flags.Yes, "yes", false,
		`Skip interactive confirmation prompts`)

	pf.BoolVar(&flags.DryRun, "dry-run", false,
		`Print what would be done without executing`)

	pf.StringVar(&flags.ProjectDir, "project-dir", "",
		`Path to promptman project directory (default: current directory)`)

	// PersistentPreRunE validates flags and injects GlobalFlags into the context.
	// It runs before every subcommand's RunE, giving subcommands access to
	// parsed flag values via GlobalFlagsFrom(cmd.Context()).
	root.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		// Resolve project-dir default.
		if flags.ProjectDir == "" {
			wd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("resolving working directory: %w", err)
			}
			flags.ProjectDir = wd
		}

		// Validate format flag.
		switch flags.Format {
		case FormatJSON, FormatTable, FormatMinimal:
			// valid
		default:
			return fmt.Errorf("invalid --format %q: must be json, table, or minimal", flags.Format)
		}

		// Inject parsed flags into the command context so subcommands can access them.
		cmd.SetContext(withGlobalFlags(cmd.Context(), flags))

		return nil
	}

	// Register subcommands.
	root.AddCommand(newVersionCommand(flags))

	return root
}

// Execute is the entry point called from cmd/cli/main.go.
// It builds the root command and executes it, exiting with code 1 on error.
func Execute() {
	root := NewRootCommand()
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
