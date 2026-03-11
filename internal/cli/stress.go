package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/khanhnguyen/promptman/internal/daemon"
	"github.com/khanhnguyen/promptman/internal/stress"
	"github.com/khanhnguyen/promptman/pkg/envelope"
	"github.com/spf13/cobra"
)

// stressFlags holds the flags specific to the stress subcommand.
type stressFlags struct {
	// users is the number of concurrent virtual users.
	users int

	// duration is how long to run the test (e.g. "30s", "2m").
	duration string

	// rampUp is the period over which users are gradually spawned.
	rampUp string

	// thresholds are expressions evaluated against the final report
	// (e.g. "p95<500ms", "error_rate<5%", "rps>100").
	thresholds []string

	// config is the path to a YAML stress-test config file.
	// When set, positional arguments are ignored.
	config string
}

// newStressCommand creates the "stress" subcommand for running load tests.
//
// The stress command runs entirely in-process — it does not call the daemon.
// HTTP requests are executed through a [daemonRequestExecutor] that forwards
// each request to the daemon's /run endpoint, allowing prompt resolution and
// environment injection while the scheduling and metrics collection happen
// locally.
//
// Usage:
//
//	promptman stress <collection/request> [flags]
//	promptman stress --config ./load.stress.yaml [flags]
func newStressCommand(globals *GlobalFlags) *cobra.Command {
	sf := &stressFlags{}

	cmd := &cobra.Command{
		Use:   "stress [collection/request]",
		Short: "Run a load test against an HTTP request or a YAML scenario file",
		Long: `Run a stress test (load test) against a single HTTP request or a YAML
scenario file. The load generation runs in-process; the daemon is used only for
request resolution (collection lookup, environment injection).

Single-request mode:
  promptman stress users/list --users 50 --duration 30s
  promptman stress users/list --threshold "p95<500ms" --threshold "error_rate<5%"

Config-file mode:
  promptman stress --config ./httpbin.stress.yaml

Thresholds:
  Supported metrics: p50, p95, p99, error_rate, rps, throughput
  Operators:         <, <=, >, >=
  Value units:       ms (milliseconds), s (seconds), % (for error_rate)
  Examples:          p95<500ms   error_rate<5%   rps>=100   throughput>1024

Exit codes:
  0  All thresholds passed (or no thresholds defined)
  1  One or more thresholds failed, or a runtime error occurred`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return executeStress(cmd, args, globals, sf)
		},
	}

	f := cmd.Flags()
	f.IntVar(&sf.users, "users", 10, "Number of concurrent virtual users")
	f.StringVar(&sf.duration, "duration", "30s", "Test duration (e.g. 30s, 2m)")
	f.StringVar(&sf.rampUp, "ramp-up", "0s", "Ramp-up period to gradually spawn users (e.g. 10s)")
	f.StringArrayVar(&sf.thresholds, "threshold", nil,
		`Pass/fail threshold expression (repeatable). E.g. --threshold "p95<500ms"`)
	f.StringVar(&sf.config, "config", "", "Path to a YAML stress-test config file")

	return cmd
}

// executeStress dispatches to config-file or single-request mode.
func executeStress(cmd *cobra.Command, args []string, globals *GlobalFlags, sf *stressFlags) error {
	if sf.config != "" && len(args) > 0 {
		return fmt.Errorf("cannot use both --config and a positional <collection/request> argument")
	}
	if sf.config == "" && len(args) == 0 {
		return fmt.Errorf("requires either a <collection/request> argument or --config flag")
	}

	// Build the in-process executor. The stress runner talks to the daemon for
	// request resolution (env vars, collection metadata), so we only need the
	// daemon running when a request ID is involved. For config-file mode the
	// YAML contains full scenario definitions.
	executor := newDaemonRequestExecutor(globals.ProjectDir, globals.Format, cmd)

	runner := stress.NewStressRunner(executor)

	var (
		report *stress.StressReport
		err    error
	)

	if sf.config != "" {
		report, err = runner.RunFromConfig(sf.config)
	} else {
		collectionID, requestID, parseErr := parsePath(args[0])
		if parseErr != nil {
			return parseErr
		}
		report, err = runner.Run(&stress.StressOpts{
			Collection: collectionID,
			RequestID:  requestID,
			Users:      sf.users,
			Duration:   sf.duration,
			RampUp:     sf.rampUp,
			Thresholds: sf.thresholds,
		})
	}
	if err != nil {
		return writeErrorEnvelope(cmd, globals, "STRESS_ERROR", err.Error())
	}

	// Determine overall pass/fail from threshold results.
	allPassed := true
	for _, r := range report.Thresholds {
		if !r.Passed {
			allPassed = false
			break
		}
	}

	// Render the report as a structured JSON envelope.
	formatter, fErr := NewFormatter(globals.Format)
	if fErr != nil {
		return fErr
	}

	env := envelope.Success(buildStressOutput(report))
	if fmtErr := formatter.Format(cmd.OutOrStdout(), env); fmtErr != nil {
		return fmtErr
	}

	if len(report.Thresholds) > 0 && !allPassed {
		return &ExitError{Code: 1}
	}
	return nil
}

// buildStressOutput converts a StressReport into the structured map written
// to stdout. The shape matches the envelope data field expected by formatters.
func buildStressOutput(r *stress.StressReport) map[string]any {
	summary := map[string]any{
		"scenario":       r.Scenario,
		"duration_ms":    r.Duration,
		"total_requests": r.Summary.TotalRequests,
		"error_rate":     fmt.Sprintf("%.2f%%", r.Summary.ErrorRate),
		"rps":            fmt.Sprintf("%.1f", r.Summary.RPS),
		"throughput_bps": fmt.Sprintf("%.0f", float64(r.Summary.Throughput)),
		"latency": map[string]any{
			"p50_ms": r.Summary.Latency.P50,
			"p95_ms": r.Summary.Latency.P95,
			"p99_ms": r.Summary.Latency.P99,
		},
	}

	if len(r.Thresholds) > 0 {
		thresholds := make([]map[string]any, 0, len(r.Thresholds))
		for _, t := range r.Thresholds {
			status := "pass"
			if !t.Passed {
				status = "FAIL"
			}
			thresholds = append(thresholds, map[string]any{
				"metric":   t.Name,
				"operator": t.Operator,
				"expected": t.Expected,
				"actual":   t.Actual,
				"status":   status,
			})
		}
		summary["thresholds"] = thresholds
	}

	return summary
}

// ─── In-process RequestExecutor adapter ──────────────────────────────────────

// daemonRequestExecutor implements stress.RequestExecutor by forwarding each
// request to the daemon's POST /run endpoint over raw HTTP. This design:
//   - Measures latency around the bare HTTP call (before envelope decode).
//   - Lets the daemon handle env-var substitution and collection resolution.
//   - Keeps a reusable *http.Client per executor instance for connection pooling.
type daemonRequestExecutor struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

// newDaemonRequestExecutor reads the daemon lock file and builds the executor.
// Returns ErrDaemonNotRunning if the lock file is absent or stale.
func newDaemonRequestExecutor(projectDir, _ string, _ *cobra.Command) *daemonRequestExecutor {
	info, err := daemon.ReadLockFile(projectDir)
	if err != nil || !daemon.IsPIDAlive(info.PID) {
		// Return a "broken" executor; execute() will propagate the error gracefully.
		return &daemonRequestExecutor{}
	}
	return &daemonRequestExecutor{
		baseURL: fmt.Sprintf("http://127.0.0.1:%d/api/v1", info.Port),
		token:   info.Token,
		httpClient: &http.Client{
			Timeout: 60 * time.Second, // generous timeout for load tests
		},
	}
}

// Execute sends a single HTTP request to the daemon's /run endpoint and returns
// performance metrics. It never returns a non-nil err for failed HTTP responses;
// those are captured as non-2xx status codes so the metrics collector can tally
// error rates correctly.
func (d *daemonRequestExecutor) Execute(
	ctx context.Context,
	collectionID, requestID string,
) (statusCode int, latencyUs int64, bodySize int64, err error) {
	if d.httpClient == nil {
		return 0, 0, 0, ErrDaemonNotRunning
	}

	payload, _ := json.Marshal(map[string]any{
		"collection": collectionID,
		"requestId":  requestID,
	})

	req, reqErr := http.NewRequestWithContext(ctx, http.MethodPost,
		d.baseURL+"/run", bytes.NewReader(payload))
	if reqErr != nil {
		return 0, 0, 0, reqErr
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+d.token)

	start := time.Now()
	resp, doErr := d.httpClient.Do(req)
	latencyUs = time.Since(start).Microseconds()

	if doErr != nil {
		// Network error: count it as a request but don't propagate — the
		// metrics collector will see statusCode=0 and count it as an error.
		return 0, latencyUs, 0, nil //nolint:nilerr
	}
	defer func() { _ = resp.Body.Close() }()

	body, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, latencyUs, int64(len(body)), nil
}

