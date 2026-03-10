package cli

import (
	"encoding/json"
	"fmt"
	"text/tabwriter"
	"time"

	"github.com/khanhnguyen/promptman/pkg/envelope"
	"github.com/spf13/cobra"
)

// newStatusCommand creates the "status" subcommand for querying daemon health.
func newStatusCommand(globals *GlobalFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show daemon status and health information",
		Long: `Query the running daemon for health information including
PID, port, uptime, project directory, and active environment.

This command does NOT auto-start the daemon. If the daemon is not running,
it prints a friendly message and exits with code 0.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return executeStatus(cmd, globals)
		},
	}
	return cmd
}

// executeStatus queries the daemon for status information and formats the output.
// If the daemon is not running, it prints a friendly message instead of an error.
func executeStatus(cmd *cobra.Command, globals *GlobalFlags) error {
	client, err := NewClient(globals.ProjectDir)
	if err != nil {
		// Daemon not running — print friendly message, not an error.
		if IsCLIError(err, CodeDaemonNotRunning) {
			return renderDaemonNotRunning(cmd, globals)
		}
		return writeClientError(cmd, globals, err)
	}

	env, err := client.Get("/status")
	if err != nil {
		return writeClientError(cmd, globals, err)
	}

	return renderStatus(cmd, globals, env)
}

// renderDaemonNotRunning outputs a friendly message when the daemon is not running.
func renderDaemonNotRunning(cmd *cobra.Command, globals *GlobalFlags) error {
	message := map[string]string{
		"status":  "not_running",
		"message": "Daemon is not running. Use 'promptman run' to auto-start.",
	}

	formatter, err := NewFormatter(globals.Format)
	if err != nil {
		return err
	}

	return formatter.Format(cmd.OutOrStdout(), localSuccess(message))
}

// renderStatus formats the daemon status response.
func renderStatus(cmd *cobra.Command, globals *GlobalFlags, env *envelope.Envelope) error {
	switch globals.Format {
	case FormatTable:
		return renderStatusTable(cmd, env)
	case FormatMinimal:
		return renderStatusMinimal(cmd, env)
	default:
		formatter, err := NewFormatter(globals.Format)
		if err != nil {
			return err
		}
		return formatter.Format(cmd.OutOrStdout(), env)
	}
}

// statusFields is the parsed status response from the daemon.
type statusFields struct {
	PID        int    `json:"pid"`
	Port       int    `json:"port"`
	ProjectDir string `json:"projectDir"`
	StartedAt  string `json:"startedAt"`
	Uptime     string `json:"uptime"`
	ActiveEnv  string `json:"activeEnv"`
}

// parseStatusFields extracts status fields from envelope data.
func parseStatusFields(data any) (*statusFields, error) {
	raw, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("marshalling status data: %w", err)
	}
	var sf statusFields
	if err := json.Unmarshal(raw, &sf); err != nil {
		return nil, fmt.Errorf("unmarshalling status fields: %w", err)
	}
	return &sf, nil
}

// renderStatusTable renders daemon status as an aligned table.
func renderStatusTable(cmd *cobra.Command, env *envelope.Envelope) error {
	if !env.OK {
		fmt.Fprintf(cmd.OutOrStdout(), "ERROR\t%s: %s\n", env.Error.Code, env.Error.Message)
		return nil
	}

	sf, err := parseStatusFields(env.Data)
	if err != nil {
		return err
	}

	tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	defer func() { _ = tw.Flush() }()

	fmt.Fprintln(tw, "KEY\tVALUE")
	fmt.Fprintln(tw, "---\t-----")
	fmt.Fprintf(tw, "PID\t%d\n", sf.PID)
	fmt.Fprintf(tw, "Port\t%d\n", sf.Port)
	fmt.Fprintf(tw, "Uptime\t%s\n", formatUptime(sf.Uptime, sf.StartedAt))
	fmt.Fprintf(tw, "Project Dir\t%s\n", sf.ProjectDir)
	fmt.Fprintf(tw, "Active Env\t%s\n", formatActiveEnv(sf.ActiveEnv))

	return nil
}

// renderStatusMinimal renders daemon status as a one-line summary.
func renderStatusMinimal(cmd *cobra.Command, env *envelope.Envelope) error {
	if !env.OK {
		fmt.Fprintf(cmd.OutOrStdout(), "error: %s — %s\n", env.Error.Code, env.Error.Message)
		return nil
	}

	sf, err := parseStatusFields(env.Data)
	if err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "running: pid=%d port=%d uptime=%s\n",
		sf.PID, sf.Port, formatUptime(sf.Uptime, sf.StartedAt))

	return nil
}

// formatUptime returns the uptime string from the status response.
// If uptime is empty, it calculates it from startedAt.
func formatUptime(uptime, startedAt string) string {
	if uptime != "" {
		return uptime
	}
	if startedAt == "" {
		return "unknown"
	}
	t, err := time.Parse(time.RFC3339, startedAt)
	if err != nil {
		return startedAt
	}
	d := time.Since(t)
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
}

// formatActiveEnv returns the active environment name or "(none)" if empty.
func formatActiveEnv(name string) string {
	if name == "" {
		return "(none)"
	}
	return name
}
