package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/khanhnguyen/promptman/internal/stress"
	"github.com/khanhnguyen/promptman/pkg/envelope"
	"github.com/spf13/cobra"
)

// ─── Mock executor for CLI stress tests ───────────────────────────────────────

// stressTestExecutor implements stress.RequestExecutor for unit tests.
// It returns pre-configured values without touching the daemon or network.
type stressTestExecutor struct {
	statusCode int
	latencyUs  int64
	bodySize   int64
}

func (e *stressTestExecutor) Execute(
	_ context.Context,
	_, _ string,
) (statusCode int, latencyUs int64, bodySize int64, err error) {
	return e.statusCode, e.latencyUs, e.bodySize, nil
}

// ─── Test-local core helper ───────────────────────────────────────────────────

// stressWithExecutor is the testable core of executeStress that accepts an
// injected RequestExecutor instead of creating a daemonRequestExecutor.
func stressWithExecutor(
	cmd *cobra.Command,
	args []string,
	globals *GlobalFlags,
	sf *stressFlags,
	executor stress.RequestExecutor,
) error {
	if sf.config != "" && len(args) > 0 {
		return writeErrorEnvelope(cmd, globals, "STRESS_ERROR", "cannot use both --config and positional arg")
	}
	if sf.config == "" && len(args) == 0 {
		return writeErrorEnvelope(cmd, globals, "STRESS_ERROR", "requires collection/request or --config")
	}

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

	allPassed := true
	for _, r := range report.Thresholds {
		if !r.Passed {
			allPassed = false
			break
		}
	}

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

// ─── Static buildStressOutput tests ──────────────────────────────────────────

func TestBuildStressOutput_Fields(t *testing.T) {
	report := &stress.StressReport{
		Scenario: "users/list",
		Duration: 1000,
		Summary: stress.StressSummary{
			TotalRequests: 500,
			RPS:           100.0,
			ErrorRate:     2.5,
			Throughput:    65536,
			Latency: stress.LatencyMetrics{
				P50: 12,
				P95: 45,
				P99: 88,
			},
		},
	}

	out := buildStressOutput(report)

	assertStressField(t, out, "scenario", "users/list")
	assertStressField(t, out, "total_requests", 500)
	assertStressField(t, out, "error_rate", "2.50%")
	assertStressField(t, out, "rps", "100.0")

	latency, ok := out["latency"].(map[string]any)
	if !ok {
		t.Fatal("latency field missing or wrong type")
	}
	if latency["p50_ms"] != int64(12) {
		t.Errorf("p50_ms = %v, want 12", latency["p50_ms"])
	}
	if latency["p95_ms"] != int64(45) {
		t.Errorf("p95_ms = %v, want 45", latency["p95_ms"])
	}
}

func TestBuildStressOutput_WithThresholds(t *testing.T) {
	report := &stress.StressReport{
		Scenario: "test",
		Thresholds: []stress.ThresholdResult{
			{Name: "error_rate", Operator: "<", Expected: 5, Actual: 2, Passed: true},
			{Name: "p95", Operator: "<", Expected: 500, Actual: 600, Passed: false},
		},
	}

	out := buildStressOutput(report)
	thresholds, ok := out["thresholds"].([]map[string]any)
	if !ok {
		t.Fatalf("thresholds field missing or wrong type, got %T", out["thresholds"])
	}
	if len(thresholds) != 2 {
		t.Errorf("len(thresholds) = %d, want 2", len(thresholds))
	}
	if thresholds[0]["status"] != "pass" {
		t.Errorf("thresholds[0].status = %v, want pass", thresholds[0]["status"])
	}
	if thresholds[1]["status"] != "FAIL" {
		t.Errorf("thresholds[1].status = %v, want FAIL", thresholds[1]["status"])
	}
}

func TestBuildStressOutput_NoThresholds(t *testing.T) {
	report := &stress.StressReport{Scenario: "x"}
	out := buildStressOutput(report)
	if _, found := out["thresholds"]; found {
		t.Error("thresholds key should be absent when no thresholds defined")
	}
}

// ─── YAML config-file integration test ───────────────────────────────────────

func TestStressCommand_ConfigFile_Pass(t *testing.T) {
	exec := &stressTestExecutor{statusCode: 200, latencyUs: 500, bodySize: 64}

	yamlContent := `
name: CLI Config Test
scenarios:
  - name: list
    request: test/list
    weight: 100
config:
  users: 3
  duration: 1s
thresholds:
  error_rate: "50%"
`
	dir := t.TempDir()
	configPath := filepath.Join(dir, "test.stress.yaml")
	if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("writing config: %v", err)
	}

	globals := &GlobalFlags{Format: FormatJSON}
	sf := &stressFlags{config: configPath}
	out := &bytes.Buffer{}

	cmd := &cobra.Command{
		Use:           "stress",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return stressWithExecutor(cmd, args, globals, sf, exec)
		},
	}
	f := cmd.Flags()
	f.IntVar(&sf.users, "users", 10, "")
	f.StringVar(&sf.duration, "duration", "30s", "")
	f.StringVar(&sf.rampUp, "ramp-up", "0s", "")
	f.StringArrayVar(&sf.thresholds, "threshold", nil, "")
	f.StringVar(&sf.config, "config", configPath, "")
	cmd.SetOut(out)

	root := &cobra.Command{Use: "promptman", SilenceUsage: true, SilenceErrors: true}
	root.SetOut(out)
	root.AddCommand(cmd)
	root.SetArgs([]string{"stress"})

	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out.String())
	}

	var env map[string]any
	if err := json.NewDecoder(out).Decode(&env); err != nil {
		t.Fatalf("parsing output JSON: %v\noutput: %s", err, out.String())
	}
	if env["ok"] != true {
		t.Errorf("envelope ok = %v, want true", env["ok"])
	}
}

// ─── Single-request mode integration test ────────────────────────────────────

func TestStressCommand_SingleRequest_Pass(t *testing.T) {
	exec := &stressTestExecutor{statusCode: 200, latencyUs: 1000, bodySize: 128}

	globals := &GlobalFlags{Format: FormatJSON}
	sf := &stressFlags{
		users:    2,
		duration: "1s",
		rampUp:   "0s",
	}
	out := &bytes.Buffer{}

	cmd := &cobra.Command{
		Use:           "stress",
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return stressWithExecutor(cmd, args, globals, sf, exec)
		},
	}
	f := cmd.Flags()
	f.IntVar(&sf.users, "users", 2, "")
	f.StringVar(&sf.duration, "duration", "1s", "")
	f.StringVar(&sf.rampUp, "ramp-up", "0s", "")
	f.StringArrayVar(&sf.thresholds, "threshold", nil, "")
	f.StringVar(&sf.config, "config", "", "")
	cmd.SetOut(out)

	root := &cobra.Command{Use: "promptman", SilenceUsage: true, SilenceErrors: true}
	root.SetOut(out)
	root.AddCommand(cmd)
	root.SetArgs([]string{"stress", "users/list"})

	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out.String())
	}

	var env map[string]any
	if err := json.NewDecoder(out).Decode(&env); err != nil {
		t.Fatalf("parsing output JSON: %v\noutput: %s", err, out.String())
	}
	if env["ok"] != true {
		t.Errorf("envelope ok = %v, want true", env["ok"])
	}
	data, _ := env["data"].(map[string]any)
	if scenario, _ := data["scenario"].(string); scenario == "" {
		t.Errorf("scenario is empty, want a non-empty value from collection/request")
	}
}

// ─── Helper ───────────────────────────────────────────────────────────────────

// assertStressField verifies a key in the stress output map.
func assertStressField(t *testing.T, m map[string]any, key string, want any) {
	t.Helper()
	got, found := m[key]
	if !found {
		t.Errorf("field %q not found in output", key)
		return
	}
	if got != want {
		t.Errorf("field %q = %v (%T), want %v (%T)", key, got, got, want, want)
	}
}
