package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Build-time variables injected via -ldflags.
// Example: go build -ldflags "-X github.com/khanhnguyen/promptman/internal/cli.Version=1.0.0"
var (
	// Version is the semantic version string (e.g. "1.0.0" or "dev").
	Version = "dev"

	// Commit is the git commit hash at build time.
	Commit = "unknown"

	// Date is the build date in RFC3339 format.
	Date = "unknown"
)

// newVersionCommand returns the "version" subcommand that prints build info.
func newVersionCommand(_ *GlobalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Long:  `Print the Promptman CLI version, git commit, and build date.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintf(cmd.OutOrStdout(), "promptman %s (commit: %s, built: %s)\n",
				Version, Commit, Date)
			return nil
		},
	}
}
