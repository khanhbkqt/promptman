package stress

import (
	"context"
	"fmt"
	"time"
)

// StressRunner orchestrates a stress test by coordinating the scheduler,
// workers, and metrics collector. It exposes two entry points:
//   - Run: for quick CLI-driven tests using StressOpts
//   - RunFromConfig: for complex multi-scenario tests from YAML files
type StressRunner struct {
	executor RequestExecutor
}

// NewStressRunner creates a StressRunner with the given request executor.
func NewStressRunner(executor RequestExecutor) *StressRunner {
	return &StressRunner{executor: executor}
}

// Run executes a single-scenario stress test from CLI flags.
// If opts.Thresholds are provided, they are evaluated against the final report
// and attached to report.Thresholds.
func (sr *StressRunner) Run(opts *StressOpts) (*StressReport, error) {
	scenarios, params, err := ParseOpts(opts)
	if err != nil {
		return nil, err
	}

	thresholds, err := ParseThresholds(opts.Thresholds)
	if err != nil {
		return nil, err
	}

	return sr.execute(context.Background(), opts.RequestID, scenarios, params, thresholds)
}

// RunFromConfig loads a YAML config file and executes a multi-scenario
// stress test. Thresholds defined in the YAML ThresholdConfig are evaluated
// against the final report.
func (sr *StressRunner) RunFromConfig(path string) (*StressReport, error) {
	cfg, scenarios, err := ParseConfig(path)
	if err != nil {
		return nil, err
	}

	thresholds, err := thresholdsFromConfig(cfg.Thresholds)
	if err != nil {
		return nil, err
	}

	return sr.execute(context.Background(), cfg.Name, scenarios, cfg.Config, thresholds)
}

// execute runs the core stress test loop:
//  1. Parse duration and ramp-up from params
//  2. Create a MetricsCollector
//  3. Start the scheduler with ramp-up
//  4. Run a ticker for per-second timeline snapshots
//  5. Wait for duration to expire
//  6. Collect final metrics and build StressReport
func (sr *StressRunner) execute(
	parent context.Context,
	name string,
	scenarios []resolvedScenario,
	params StressParams,
	thresholds []Threshold,
) (*StressReport, error) {
	duration, err := time.ParseDuration(params.Duration)
	if err != nil {
		return nil, ErrInvalidConfig.Wrapf("invalid duration: %v", err)
	}

	var rampUp time.Duration
	if params.RampUp != "" {
		rampUp, err = time.ParseDuration(params.RampUp)
		if err != nil {
			return nil, ErrInvalidConfig.Wrapf("invalid rampUp: %v", err)
		}
	}

	metrics := NewMetricsCollector()
	sched := newScheduler(params.Users, rampUp)
	var connections int64

	// Create duration-limited context.
	ctx, cancel := context.WithTimeout(parent, duration)
	defer cancel()

	// Start scheduler in background — spawns workers gradually.
	schedulerDone := make(chan struct{})
	go func() {
		defer close(schedulerDone)
		sched.Start(ctx, func(wCtx context.Context) {
			w := newWorker(scenarios, sr.executor, metrics, &connections)
			w.run(wCtx)
		})
	}()

	// Timeline ticker: capture snapshots every second.
	var timeline []TimelinePoint
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	start := time.Now()

loop:
	for {
		select {
		case <-ctx.Done():
			break loop
		case <-ticker.C:
			elapsed := time.Since(start).Seconds()
			point := metrics.Snapshot(elapsed, sched.ActiveUsers())
			timeline = append(timeline, point)
		}
	}

	// Wait for scheduler to finish spawning (it may already be done).
	<-schedulerDone

	// Wait for all workers to finish their current requests.
	sched.Wait()

	// Build final report.
	summary := metrics.Summary()
	report := &StressReport{
		Scenario: name,
		Duration: time.Since(start).Milliseconds(),
		Summary:  summary,
		Timeline: timeline,
	}

	// Evaluate thresholds and attach results to the report.
	if len(thresholds) > 0 {
		results, _ := EvaluateThresholds(thresholds, summary)
		report.Thresholds = results
	}

	return report, nil
}

// thresholdsFromConfig converts a ThresholdConfig (from YAML) into a slice of
// Threshold values ready for evaluation.
//
// Conversion rules:
//   - P95Latency: "500ms" → "p95<500ms"
//   - ErrorRate:  "5%"    → "error_rate<5%"
//   - RPS:        100.0   → "rps>100"
func thresholdsFromConfig(cfg ThresholdConfig) ([]Threshold, error) {
	var exprs []string

	if cfg.P95Latency != "" {
		exprs = append(exprs, fmt.Sprintf("p95<%s", cfg.P95Latency))
	}
	if cfg.ErrorRate != "" {
		exprs = append(exprs, fmt.Sprintf("error_rate<%s", cfg.ErrorRate))
	}
	if cfg.RPS > 0 {
		exprs = append(exprs, fmt.Sprintf("rps>%g", cfg.RPS))
	}

	return ParseThresholds(exprs)
}
