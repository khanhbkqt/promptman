package cli

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/khanhnguyen/promptman/pkg/envelope"
	"github.com/spf13/cobra"
)

// ExitError is a sentinel error that carries an exit code for the CLI process.
// When returned from a RunE function, the root command handler should call
// os.Exit with the embedded code.
type ExitError struct {
	Code int
}

// Error implements the error interface.
func (e *ExitError) Error() string {
	return fmt.Sprintf("exit status %d", e.Code)
}

// runFlags holds the flags specific to the run subcommand.
type runFlags struct {
	// env overrides the active environment for this run.
	env string

	// timeout sets the request timeout duration.
	timeout time.Duration

	// insecure skips TLS certificate verification.
	insecure bool

	// collection runs all requests in the named collection.
	collection string

	// stopOnError stops collection execution on first failure.
	stopOnError bool
}

// newRunCommand creates the "run" subcommand for executing HTTP requests.
// It supports two modes:
//   - Single request: promptman run <collection>/<request>
//   - Collection run: promptman run --collection <id>
func newRunCommand(globals *GlobalFlags) *cobra.Command {
	rf := &runFlags{}

	cmd := &cobra.Command{
		Use:   "run [collection/request]",
		Short: "Execute HTTP requests via the daemon",
		Long: `Execute a single HTTP request or run all requests in a collection.

Single request:
  promptman run users/health
  promptman run users/admin/list-admins --env dev

Collection run:
  promptman run --collection users
  promptman run --collection users --stop-on-error`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return executeRun(cmd, args, globals, rf)
		},
	}

	f := cmd.Flags()
	f.StringVar(&rf.env, "env", "", "Override the active environment for this run")
	f.DurationVar(&rf.timeout, "timeout", 60*time.Second, "Request timeout duration")
	f.BoolVar(&rf.insecure, "insecure", false, "Skip TLS certificate verification")
	f.StringVar(&rf.collection, "collection", "", "Run all requests in the named collection")
	f.BoolVar(&rf.stopOnError, "stop-on-error", false, "Stop collection execution on first failure")

	return cmd
}

// executeRun dispatches to single request or collection run mode.
func executeRun(cmd *cobra.Command, args []string, globals *GlobalFlags, rf *runFlags) error {
	// Validate mutual exclusion.
	if rf.collection != "" && len(args) > 0 {
		return fmt.Errorf("cannot use both --collection flag and positional argument")
	}
	if rf.collection == "" && len(args) == 0 {
		return fmt.Errorf("requires either a <collection>/<request> argument or --collection flag")
	}

	// Ensure daemon is running (auto-start if needed).
	if err := EnsureDaemon(globals.ProjectDir); err != nil {
		return writeErrorEnvelope(cmd, globals, CodeDaemonNotRunning, err.Error())
	}

	// Create the daemon client.
	client, err := NewClient(globals.ProjectDir)
	if err != nil {
		return writeClientError(cmd, globals, err)
	}

	if rf.collection != "" {
		return runCollection(cmd, globals, rf, client)
	}
	return runSingle(cmd, args, globals, rf, client)
}

// runSingle executes a single request via POST /run.
func runSingle(cmd *cobra.Command, args []string, globals *GlobalFlags, rf *runFlags, client *Client) error {
	collectionID, requestID, err := parsePath(args[0])
	if err != nil {
		return err
	}

	input := map[string]any{
		"collection": collectionID,
		"requestId":  requestID,
	}
	if rf.env != "" {
		input["env"] = rf.env
	}
	if rf.insecure {
		input["skipTlsVerify"] = true
	}

	env, clientErr := client.Post("/run", input)
	if clientErr != nil {
		return writeClientError(cmd, globals, clientErr)
	}

	return renderAndExit(cmd, globals, env)
}

// runCollection executes all requests in a collection via POST /run/collection.
func runCollection(cmd *cobra.Command, globals *GlobalFlags, rf *runFlags, client *Client) error {
	input := map[string]any{
		"collection": rf.collection,
	}
	if rf.env != "" {
		input["env"] = rf.env
	}
	if rf.stopOnError {
		input["stopOnError"] = true
	}
	if rf.insecure {
		input["skipTlsVerify"] = true
	}

	env, clientErr := client.Post("/run/collection", input)
	if clientErr != nil {
		return writeClientError(cmd, globals, clientErr)
	}

	return renderAndExit(cmd, globals, env)
}

// parsePath splits a "collection/request" path into collection ID and request path.
// The first segment before the first "/" is the collection ID.
// Everything after is the request path (which may contain "/" for nested folders).
//
// Examples:
//
//	"users/health"     → ("users", "health")
//	"users/admin/list" → ("users", "admin/list")
func parsePath(path string) (collectionID, requestID string, err error) {
	idx := strings.Index(path, "/")
	if idx < 0 {
		return "", "", fmt.Errorf("invalid path %q: must be <collection>/<request>", path)
	}
	collectionID = path[:idx]
	requestID = path[idx+1:]

	if collectionID == "" {
		return "", "", fmt.Errorf("invalid path %q: collection name cannot be empty", path)
	}
	if requestID == "" {
		return "", "", fmt.Errorf("invalid path %q: request name cannot be empty", path)
	}
	return collectionID, requestID, nil
}

// renderAndExit formats the envelope response and returns an ExitError
// with the appropriate exit code. For table/minimal formats, it uses the
// response-specific formatter that shows timing, headers, and body.
func renderAndExit(cmd *cobra.Command, globals *GlobalFlags, env *envelope.Envelope) error {
	if fmtErr := FormatRunResponse(cmd.OutOrStdout(), globals.Format, env); fmtErr != nil {
		return fmt.Errorf("formatting output: %w", fmtErr)
	}

	if !env.OK {
		return &ExitError{Code: 1}
	}

	// Extract HTTP status code from response data for exit code.
	exitCode := extractExitCode(env.Data)
	if exitCode != 0 {
		return &ExitError{Code: exitCode}
	}

	return nil
}

// extractExitCode extracts the HTTP status code from response data.
// For single responses: reads the "status" field.
// For collection responses (array): returns the first non-2xx status; 0 if all 2xx.
// Returns 0 if unable to extract a status.
func extractExitCode(data any) int {
	if data == nil {
		return 0
	}

	// JSON round-trip to a generic type.
	raw, err := json.Marshal(data)
	if err != nil {
		return 0
	}

	// Try single response (map with "status").
	var single map[string]any
	if err := json.Unmarshal(raw, &single); err == nil {
		return statusToExitCode(single)
	}

	// Try collection response (array of maps).
	var list []map[string]any
	if err := json.Unmarshal(raw, &list); err == nil {
		for _, item := range list {
			code := statusToExitCode(item)
			if code != 0 {
				return code
			}
		}
		return 0
	}

	return 0
}

// statusToExitCode reads the "status" field from a map and returns
// 0 for 2xx responses or the status code itself for non-2xx.
func statusToExitCode(m map[string]any) int {
	raw, ok := m["status"]
	if !ok {
		return 0
	}
	status, ok := raw.(float64) // JSON numbers decode as float64
	if !ok {
		return 0
	}
	code := int(status)
	if code >= 200 && code < 300 {
		return 0
	}
	return code
}

// writeClientError formats a client error and returns an ExitError.
func writeClientError(cmd *cobra.Command, globals *GlobalFlags, err error) error {
	cliErr, ok := err.(*CLIError)
	if ok {
		return writeErrorEnvelope(cmd, globals, cliErr.Code, cliErr.Message)
	}
	return writeErrorEnvelope(cmd, globals, CodeHTTPError, err.Error())
}

// writeErrorEnvelope creates a failure envelope, formats it, and returns an ExitError.
func writeErrorEnvelope(cmd *cobra.Command, globals *GlobalFlags, code, message string) error {
	env := envelope.Fail(code, message)

	formatter, err := NewFormatter(globals.Format)
	if err != nil {
		return fmt.Errorf("%s: %s", code, message)
	}

	if fmtErr := formatter.Format(cmd.OutOrStdout(), env); fmtErr != nil {
		return fmt.Errorf("%s: %s", code, message)
	}

	return &ExitError{Code: 1}
}
