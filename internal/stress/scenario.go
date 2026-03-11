package stress

import (
	"fmt"
	"strings"
	"time"

	"github.com/khanhnguyen/promptman/pkg/fsutil"
)

// resolvedScenario is an internal representation of a scenario with parsed
// fields ready for execution by workers.
type resolvedScenario struct {
	Name         string        // human-readable scenario name
	CollectionID string        // collection identifier (before the "/")
	RequestID    string        // request path within the collection (after the "/")
	Weight       int           // traffic weight percentage (1–100)
	ThinkTime    time.Duration // pause between requests for this scenario
}

// ParseOpts converts CLI StressOpts into a single resolved scenario and
// execution params. The resulting scenario has weight=100 since it is the
// only scenario.
func ParseOpts(opts *StressOpts) ([]resolvedScenario, StressParams, error) {
	if opts.Collection == "" {
		return nil, StressParams{}, ErrInvalidConfig.Wrap("collection is required")
	}
	if opts.RequestID == "" {
		return nil, StressParams{}, ErrInvalidConfig.Wrap("requestId is required")
	}
	if opts.Users <= 0 {
		return nil, StressParams{}, ErrInvalidConfig.Wrap("users must be > 0")
	}
	if opts.Duration == "" {
		return nil, StressParams{}, ErrInvalidConfig.Wrap("duration is required")
	}
	if _, err := time.ParseDuration(opts.Duration); err != nil {
		return nil, StressParams{}, ErrInvalidConfig.Wrapf("invalid duration %q: %v", opts.Duration, err)
	}

	rampUp := opts.RampUp
	if rampUp == "" {
		rampUp = "0s"
	}
	if _, err := time.ParseDuration(rampUp); err != nil {
		return nil, StressParams{}, ErrInvalidConfig.Wrapf("invalid rampUp %q: %v", rampUp, err)
	}

	scenario := resolvedScenario{
		Name:         opts.RequestID,
		CollectionID: opts.Collection,
		RequestID:    opts.RequestID,
		Weight:       100,
	}

	params := StressParams{
		Users:    opts.Users,
		RampUp:   rampUp,
		Duration: opts.Duration,
	}

	return []resolvedScenario{scenario}, params, nil
}

// ParseConfig loads a YAML config file and returns the parsed config,
// resolved scenarios, and any validation error.
func ParseConfig(path string) (*StressConfig, []resolvedScenario, error) {
	var cfg StressConfig
	if err := fsutil.ReadYAML(path, &cfg); err != nil {
		return nil, nil, ErrInvalidConfig.Wrapf("loading config: %v", err)
	}

	if err := validateConfig(&cfg); err != nil {
		return nil, nil, err
	}

	scenarios, err := resolveScenarios(cfg.Scenarios)
	if err != nil {
		return nil, nil, err
	}

	if err := validateScenarios(scenarios); err != nil {
		return nil, nil, err
	}

	return &cfg, scenarios, nil
}

// validateConfig checks the top-level config params for correctness.
func validateConfig(cfg *StressConfig) error {
	if cfg.Config.Users <= 0 {
		return ErrInvalidConfig.Wrap("users must be > 0")
	}
	if cfg.Config.Duration == "" {
		return ErrInvalidConfig.Wrap("duration is required")
	}
	if _, err := time.ParseDuration(cfg.Config.Duration); err != nil {
		return ErrInvalidConfig.Wrapf("invalid duration %q: %v", cfg.Config.Duration, err)
	}
	if cfg.Config.RampUp != "" {
		if _, err := time.ParseDuration(cfg.Config.RampUp); err != nil {
			return ErrInvalidConfig.Wrapf("invalid rampUp %q: %v", cfg.Config.RampUp, err)
		}
	}
	if len(cfg.Scenarios) == 0 {
		return ErrInvalidConfig.Wrap("at least one scenario is required")
	}
	return nil
}

// resolveScenarios converts raw ScenarioItems into resolvedScenarios by
// parsing request references and think times.
func resolveScenarios(items []ScenarioItem) ([]resolvedScenario, error) {
	resolved := make([]resolvedScenario, 0, len(items))

	for i, item := range items {
		collID, reqID, err := parseRequestRef(item.Request)
		if err != nil {
			return nil, ErrInvalidScenario.Wrapf("scenario[%d] %q: %v", i, item.Name, err)
		}

		var thinkTime time.Duration
		if item.ThinkTime != "" {
			thinkTime, err = time.ParseDuration(item.ThinkTime)
			if err != nil {
				return nil, ErrInvalidScenario.Wrapf("scenario[%d] %q: invalid thinkTime %q: %v",
					i, item.Name, item.ThinkTime, err)
			}
		}

		name := item.Name
		if name == "" {
			name = item.Request
		}

		resolved = append(resolved, resolvedScenario{
			Name:         name,
			CollectionID: collID,
			RequestID:    reqID,
			Weight:       item.Weight,
			ThinkTime:    thinkTime,
		})
	}

	return resolved, nil
}

// validateScenarios checks that the resolved scenarios have valid weights
// (must sum to exactly 100) and non-empty identifiers.
func validateScenarios(scenarios []resolvedScenario) error {
	totalWeight := 0
	for i, s := range scenarios {
		if s.CollectionID == "" || s.RequestID == "" {
			return ErrInvalidScenario.Wrapf("scenario[%d]: collectionID and requestID are required", i)
		}
		if s.Weight <= 0 {
			return ErrInvalidScenario.Wrapf("scenario[%d] %q: weight must be > 0", i, s.Name)
		}
		totalWeight += s.Weight
	}

	if totalWeight != 100 {
		return ErrInvalidScenario.Wrapf("total weight must equal 100, got %d", totalWeight)
	}

	return nil
}

// parseRequestRef splits a "collectionID/requestPath" reference into its
// two components. The first path segment is the collection ID; the rest
// is the request path.
func parseRequestRef(ref string) (collectionID, requestID string, err error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", "", fmt.Errorf("request reference is empty")
	}

	idx := strings.Index(ref, "/")
	if idx < 0 {
		return "", "", fmt.Errorf("request reference %q must contain '/' (format: collectionID/requestPath)", ref)
	}

	collectionID = ref[:idx]
	requestID = ref[idx+1:]

	if collectionID == "" {
		return "", "", fmt.Errorf("collection ID is empty in %q", ref)
	}
	if requestID == "" {
		return "", "", fmt.Errorf("request ID is empty in %q", ref)
	}

	return collectionID, requestID, nil
}
