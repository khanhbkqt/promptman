package cli

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/spf13/cobra"
)

// historyFlags holds the flags specific to the history subcommand.
type historyFlags struct {
	collection string
	env        string
	source     string
	limit      int
}

// newHistoryCommand creates the "history" subcommand for querying and managing
// request execution history.
func newHistoryCommand(globals *GlobalFlags) *cobra.Command {
	hf := &historyFlags{}

	cmd := &cobra.Command{
		Use:   "history",
		Short: "View and manage request execution history",
		Long: `View request execution history or clear history entries.

List recent history:
  promptman history
  promptman history --collection users --env dev
  promptman history --limit 50 --format table

Clear history:
  promptman history clear
  promptman history clear --before 2026-03-01`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return executeHistoryList(cmd, globals, hf)
		},
	}

	f := cmd.Flags()
	f.StringVar(&hf.collection, "collection", "", "Filter by collection name")
	f.StringVar(&hf.env, "env", "", "Filter by environment name")
	f.StringVar(&hf.source, "source", "", "Filter by source (cli, gui, test)")
	f.IntVar(&hf.limit, "limit", 20, "Maximum number of entries to display")

	// Add clear subcommand.
	cmd.AddCommand(newHistoryClearCommand(globals))

	return cmd
}

// executeHistoryList queries history from the daemon and displays results.
func executeHistoryList(cmd *cobra.Command, globals *GlobalFlags, hf *historyFlags) error {
	// Ensure daemon is running.
	if err := EnsureDaemon(globals.ProjectDir); err != nil {
		return writeErrorEnvelope(cmd, globals, CodeDaemonNotRunning, err.Error())
	}

	client, err := NewClient(globals.ProjectDir)
	if err != nil {
		return writeClientError(cmd, globals, err)
	}

	// Build query string.
	params := url.Values{}
	if hf.collection != "" {
		params.Set("collection", hf.collection)
	}
	if hf.env != "" {
		params.Set("env", hf.env)
	}
	if hf.source != "" {
		params.Set("source", hf.source)
	}
	if hf.limit > 0 {
		params.Set("limit", strconv.Itoa(hf.limit))
	}

	path := "/history"
	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	env, clientErr := client.Get(path)
	if clientErr != nil {
		return writeClientError(cmd, globals, clientErr)
	}

	// Format and output.
	formatter, fmtErr := NewFormatter(globals.Format)
	if fmtErr != nil {
		return fmt.Errorf("formatter: %w", fmtErr)
	}
	if err := formatter.Format(cmd.OutOrStdout(), env); err != nil {
		return fmt.Errorf("formatting output: %w", err)
	}

	if !env.OK {
		return &ExitError{Code: 1}
	}
	return nil
}

// historyClearFlags holds the flags for the history clear subcommand.
type historyClearFlags struct {
	before string
}

// newHistoryClearCommand creates the "history clear" subcommand.
func newHistoryClearCommand(globals *GlobalFlags) *cobra.Command {
	hcf := &historyClearFlags{}

	cmd := &cobra.Command{
		Use:   "clear",
		Short: "Clear request execution history",
		Long: `Clear all history entries or entries before a specific date.

Clear all:
  promptman history clear

Clear before a date:
  promptman history clear --before 2026-03-01`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return executeHistoryClear(cmd, globals, hcf)
		},
	}

	cmd.Flags().StringVar(&hcf.before, "before", "", "Clear entries before this date (YYYY-MM-DD)")

	return cmd
}

// executeHistoryClear sends a DELETE request to the daemon to clear history.
func executeHistoryClear(cmd *cobra.Command, globals *GlobalFlags, hcf *historyClearFlags) error {
	// Ensure daemon is running.
	if err := EnsureDaemon(globals.ProjectDir); err != nil {
		return writeErrorEnvelope(cmd, globals, CodeDaemonNotRunning, err.Error())
	}

	client, err := NewClient(globals.ProjectDir)
	if err != nil {
		return writeClientError(cmd, globals, err)
	}

	path := "/history"
	if hcf.before != "" {
		params := url.Values{}
		params.Set("before", hcf.before)
		path += "?" + params.Encode()
	}

	env, clientErr := client.Delete(path)
	if clientErr != nil {
		return writeClientError(cmd, globals, clientErr)
	}

	formatter, fmtErr := NewFormatter(globals.Format)
	if fmtErr != nil {
		return fmt.Errorf("formatter: %w", fmtErr)
	}
	if err := formatter.Format(cmd.OutOrStdout(), env); err != nil {
		return fmt.Errorf("formatting output: %w", err)
	}

	if !env.OK {
		return &ExitError{Code: 1}
	}
	return nil
}
