package cli

import (
	"encoding/json"
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/khanhnguyen/promptman/pkg/envelope"
	"github.com/spf13/cobra"
)

// envSetFlags holds the flags specific to the env set subcommand.
type envSetFlags struct {
	// envName targets a specific environment instead of the active one.
	envName string
}

// newEnvCommand creates the "env" parent command with list, use, get, and set subcommands.
func newEnvCommand(globals *GlobalFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "env",
		Short: "Manage environments",
		Long:  `View, switch, and modify environments. Subcommands: list, use, get, set.`,
	}

	cmd.AddCommand(newEnvListCommand(globals))
	cmd.AddCommand(newEnvUseCommand(globals))
	cmd.AddCommand(newEnvGetCommand(globals))
	cmd.AddCommand(newEnvSetCommand(globals))

	return cmd
}

// newEnvListCommand creates the "env list" subcommand.
func newEnvListCommand(globals *GlobalFlags) *cobra.Command {
	return &cobra.Command{
		Use:           "list",
		Short:         "List all environments",
		Long:          `List all environments. The active environment is marked with an asterisk (*).`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return executeEnvList(cmd, globals)
		},
	}
}

// executeEnvList lists all environments via the daemon API.
func executeEnvList(cmd *cobra.Command, globals *GlobalFlags) error {
	if err := EnsureDaemon(globals.ProjectDir); err != nil {
		return writeErrorEnvelope(cmd, globals, CodeDaemonNotRunning, err.Error())
	}

	client, err := NewClient(globals.ProjectDir)
	if err != nil {
		return writeClientError(cmd, globals, err)
	}

	env, clientErr := client.Get("/environments")
	if clientErr != nil {
		return writeClientError(cmd, globals, clientErr)
	}

	if !env.OK {
		return renderEnvResponse(cmd, globals, env)
	}

	// For table/minimal, render a custom environment list view.
	if globals.Format == FormatTable {
		return renderEnvListTable(cmd, env)
	}
	if globals.Format == FormatMinimal {
		return renderEnvListMinimal(cmd, env)
	}

	// JSON: use the default formatter.
	formatter, fmtErr := NewFormatter(globals.Format)
	if fmtErr != nil {
		return fmtErr
	}
	return formatter.Format(cmd.OutOrStdout(), env)
}

// envListItem mirrors the daemon response shape for env listing.
type envListItem struct {
	Name          string `json:"name"`
	VariableCount int    `json:"variableCount"`
	SecretCount   int    `json:"secretCount"`
	Active        bool   `json:"active"`
}

// parseEnvListItems extracts envListItem slice from envelope data.
func parseEnvListItems(data any) ([]envListItem, error) {
	raw, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("serialising env list data: %w", err)
	}
	var items []envListItem
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, fmt.Errorf("parsing env list data: %w", err)
	}
	return items, nil
}

// renderEnvListTable renders environments as an aligned table with active marker.
func renderEnvListTable(cmd *cobra.Command, env *envelope.Envelope) error {
	items, err := parseEnvListItems(env.Data)
	if err != nil {
		return err
	}

	if len(items) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "(no environments)")
		return nil
	}

	tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	defer func() { _ = tw.Flush() }()

	fmt.Fprintln(tw, "NAME\tVARIABLES\tSECRETS\tACTIVE")
	fmt.Fprintln(tw, "----\t---------\t-------\t------")

	for _, item := range items {
		active := ""
		if item.Active {
			active = "*"
		}
		fmt.Fprintf(tw, "%s\t%d\t%d\t%s\n", item.Name, item.VariableCount, item.SecretCount, active)
	}
	return nil
}

// renderEnvListMinimal renders environments as a simple list with active marker.
func renderEnvListMinimal(cmd *cobra.Command, env *envelope.Envelope) error {
	items, err := parseEnvListItems(env.Data)
	if err != nil {
		return err
	}

	for _, item := range items {
		prefix := "  "
		if item.Active {
			prefix = "* "
		}
		fmt.Fprintf(cmd.OutOrStdout(), "%s%s\n", prefix, item.Name)
	}
	return nil
}

// newEnvUseCommand creates the "env use" subcommand.
func newEnvUseCommand(globals *GlobalFlags) *cobra.Command {
	return &cobra.Command{
		Use:           "use <name>",
		Short:         "Set the active environment",
		Long:          `Set the active environment. Subsequent commands will use this environment for variable resolution.`,
		Args:          cobra.ExactArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return executeEnvUse(cmd, args[0], globals)
		},
	}
}

// executeEnvUse sets the active environment via the daemon API.
func executeEnvUse(cmd *cobra.Command, name string, globals *GlobalFlags) error {
	if err := EnsureDaemon(globals.ProjectDir); err != nil {
		return writeErrorEnvelope(cmd, globals, CodeDaemonNotRunning, err.Error())
	}

	client, err := NewClient(globals.ProjectDir)
	if err != nil {
		return writeClientError(cmd, globals, err)
	}

	env, clientErr := client.Post("/environments/active", map[string]string{"name": name})
	if clientErr != nil {
		return writeClientError(cmd, globals, clientErr)
	}

	return renderEnvResponse(cmd, globals, env)
}

// newEnvGetCommand creates the "env get" subcommand.
func newEnvGetCommand(globals *GlobalFlags) *cobra.Command {
	return &cobra.Command{
		Use:           "get <name>",
		Short:         "Show environment variables",
		Long:          `Show all variables in an environment. Secrets are displayed as ***.`,
		Args:          cobra.ExactArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return executeEnvGet(cmd, args[0], globals)
		},
	}
}

// executeEnvGet retrieves an environment by name via the daemon API.
func executeEnvGet(cmd *cobra.Command, name string, globals *GlobalFlags) error {
	if err := EnsureDaemon(globals.ProjectDir); err != nil {
		return writeErrorEnvelope(cmd, globals, CodeDaemonNotRunning, err.Error())
	}

	client, err := NewClient(globals.ProjectDir)
	if err != nil {
		return writeClientError(cmd, globals, err)
	}

	env, clientErr := client.Get("/environments/" + name)
	if clientErr != nil {
		return writeClientError(cmd, globals, clientErr)
	}

	return renderEnvResponse(cmd, globals, env)
}

// newEnvSetCommand creates the "env set" subcommand.
func newEnvSetCommand(globals *GlobalFlags) *cobra.Command {
	sf := &envSetFlags{}

	cmd := &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set an environment variable",
		Long: `Set a variable in the active environment (or a specific one with --env).

Examples:
  promptman env set host localhost
  promptman env set port 8080 --env dev`,
		Args:          cobra.ExactArgs(2),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return executeEnvSet(cmd, args[0], args[1], globals, sf)
		},
	}

	cmd.Flags().StringVar(&sf.envName, "env", "", "Target environment (default: active environment)")

	return cmd
}

// executeEnvSet updates a variable in an environment via the daemon API.
// If --env is not specified, it gets the active environment first.
func executeEnvSet(cmd *cobra.Command, key, value string, globals *GlobalFlags, sf *envSetFlags) error {
	if err := EnsureDaemon(globals.ProjectDir); err != nil {
		return writeErrorEnvelope(cmd, globals, CodeDaemonNotRunning, err.Error())
	}

	client, err := NewClient(globals.ProjectDir)
	if err != nil {
		return writeClientError(cmd, globals, err)
	}

	// Determine which environment to update.
	envName := sf.envName
	if envName == "" {
		// Get the active environment name from the list endpoint.
		envName, err = getActiveEnvName(client)
		if err != nil {
			return writeErrorEnvelope(cmd, globals, "ENV_NOT_SET", "no active environment set; use --env to specify")
		}
	}

	// First, get the current environment to retrieve its variables.
	getEnv, getErr := client.Get("/environments/" + envName)
	if getErr != nil {
		return writeClientError(cmd, globals, getErr)
	}
	if !getEnv.OK {
		return renderEnvResponse(cmd, globals, getEnv)
	}

	// Extract current variables and merge the new key-value.
	currentVars := extractVariables(getEnv.Data)
	currentVars[key] = value

	// PUT the updated environment back.
	updateInput := map[string]any{"variables": currentVars}
	putEnv, putErr := client.Put("/environments/"+envName, updateInput)
	if putErr != nil {
		return writeClientError(cmd, globals, putErr)
	}

	return renderEnvResponse(cmd, globals, putEnv)
}

// getActiveEnvName finds the active environment name from the list endpoint.
func getActiveEnvName(client *Client) (string, error) {
	env, err := client.Get("/environments")
	if err != nil {
		return "", err
	}
	if !env.OK {
		return "", fmt.Errorf("failed to list environments")
	}

	items, parseErr := parseEnvListItems(env.Data)
	if parseErr != nil {
		return "", parseErr
	}

	for _, item := range items {
		if item.Active {
			return item.Name, nil
		}
	}
	return "", fmt.Errorf("no active environment")
}

// extractVariables extracts the variables map from the envelope data.
func extractVariables(data any) map[string]any {
	raw, err := json.Marshal(data)
	if err != nil {
		return make(map[string]any)
	}

	var envData struct {
		Variables map[string]any `json:"variables"`
	}
	if err := json.Unmarshal(raw, &envData); err != nil {
		return make(map[string]any)
	}

	if envData.Variables == nil {
		return make(map[string]any)
	}
	return envData.Variables
}

// renderEnvResponse formats an envelope response and returns ExitError if needed.
func renderEnvResponse(cmd *cobra.Command, globals *GlobalFlags, env *envelope.Envelope) error {
	formatter, err := NewFormatter(globals.Format)
	if err != nil {
		return err
	}

	if fmtErr := formatter.Format(cmd.OutOrStdout(), env); fmtErr != nil {
		return fmt.Errorf("formatting output: %w", fmtErr)
	}

	if !env.OK {
		return &ExitError{Code: 1}
	}
	return nil
}

// renderEnvUseConfirmation outputs a user-friendly confirmation for env use command.
func renderEnvUseConfirmation(cmd *cobra.Command, globals *GlobalFlags, name string) error {
	switch globals.Format {
	case FormatMinimal:
		fmt.Fprintf(cmd.OutOrStdout(), "Active environment set to: %s\n", name)
		return nil
	case FormatTable:
		fmt.Fprintf(cmd.OutOrStdout(), "Active environment set to: %s\n", name)
		return nil
	default:
		// JSON — let the default envelope render handle it.
		return nil
	}
}

// renderEnvGetTable renders environment details as a table.
func renderEnvGetTable(cmd *cobra.Command, env *envelope.Envelope) error {
	raw, err := json.Marshal(env.Data)
	if err != nil {
		return fmt.Errorf("serialising env data: %w", err)
	}

	var envData struct {
		Name      string            `json:"name"`
		Variables map[string]any    `json:"variables"`
		Secrets   map[string]string `json:"secrets"`
	}
	if err := json.Unmarshal(raw, &envData); err != nil {
		return fmt.Errorf("parsing env data: %w", err)
	}

	w := cmd.OutOrStdout()
	fmt.Fprintf(w, "Environment: %s\n\n", envData.Name)

	if len(envData.Variables) > 0 {
		fmt.Fprintln(w, "Variables:")
		tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
		for k, v := range envData.Variables {
			fmt.Fprintf(tw, "  %s\t%v\n", k, v)
		}
		_ = tw.Flush()
	}

	if len(envData.Secrets) > 0 {
		fmt.Fprintln(w, "\nSecrets:")
		tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
		for k, v := range envData.Secrets {
			val := v
			if val != "" && !strings.HasPrefix(val, "***") {
				val = "***"
			}
			fmt.Fprintf(tw, "  %s\t%s\n", k, val)
		}
		_ = tw.Flush()
	}

	return nil
}
