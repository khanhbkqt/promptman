package stress

import (
	"context"
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
func (sr *StressRunner) Run(opts *StressOpts) (*StressReport, error) {
	scenarios, params, err := ParseOpts(opts)
	if err != nil {
		return nil, err
	}

	return sr.execute(context.Background(), opts.RequestID, scenarios, params)
}

// RunFromConfig loads a YAML config file and executes a multi-scenario
// stress test.
func (sr *StressRunner) RunFromConfig(path string) (*StressReport, error) {
	cfg, scenarios, err := ParseConfig(path)
	if err != nil {
		return nil, err
	}

	return sr.execute(context.Background(), cfg.Name, scenarios, cfg.Config)
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

	return report, nil
}
