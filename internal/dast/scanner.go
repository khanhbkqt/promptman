package dast

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/khanhnguyen/promptman/internal/request"
)

// RequestExecutor abstracts the request engine's collection/single-request
// execution capabilities. This allows the scanner to remain testable
// without depending on a concrete engine implementation.
type RequestExecutor interface {
	Execute(ctx context.Context, input request.ExecuteInput) (*request.Response, error)
	ExecuteCollection(ctx context.Context, opts request.CollectionRunOpts) ([]*request.Response, error)
}

// PassiveChecker evaluates passive rules against a response.
// This decouples the scanner from the rules package to avoid circular imports.
type PassiveChecker func(rules []Rule, reqID string, resp *request.Response) []Finding

// Scanner orchestrates DAST scans by executing HTTP requests
// and applying passive (and eventually active) rules to responses.
type Scanner struct {
	exec           RequestExecutor
	checker        PassiveChecker
	customRulesDir string
}

// ScannerOption configures the Scanner via functional options.
type ScannerOption func(*Scanner)

// WithCustomRulesDir sets the directory for custom YAML rule files.
// Defaults to ".promptman/dast/rules" relative to the working directory.
func WithCustomRulesDir(dir string) ScannerOption {
	return func(s *Scanner) {
		s.customRulesDir = dir
	}
}

// NewScanner creates a Scanner with the given request executor and
// passive rule checker.
func NewScanner(exec RequestExecutor, checker PassiveChecker, opts ...ScannerOption) *Scanner {
	s := &Scanner{
		exec:           exec,
		checker:        checker,
		customRulesDir: filepath.Join(".promptman", "dast", "rules"),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Scan executes all requests in a collection and runs passive rules
// on each response. It returns a DASTReport summarizing findings.
func (s *Scanner) Scan(ctx context.Context, collID string, opts ScanOpts) (*DASTReport, error) {
	rules, err := s.loadRules(opts)
	if err != nil {
		return nil, err
	}

	// Execute all requests in the collection.
	responses, err := s.exec.ExecuteCollection(ctx, request.CollectionRunOpts{
		CollectionID: collID,
	})
	if err != nil {
		return nil, fmt.Errorf("executing collection: %w", err)
	}

	return s.buildReport(collID, opts, rules, responses), nil
}

// ScanRequest executes a single request and runs passive rules on the response.
func (s *Scanner) ScanRequest(ctx context.Context, collID, reqID string, opts ScanOpts) (*DASTReport, error) {
	rules, err := s.loadRules(opts)
	if err != nil {
		return nil, err
	}

	// Execute the single request.
	resp, err := s.exec.Execute(ctx, request.ExecuteInput{
		CollectionID: collID,
		RequestID:    reqID,
		Source:       "dast",
	})
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}

	return s.buildReport(collID, opts, rules, []*request.Response{resp}), nil
}

// --------------------------------------------------------------------------
// Internal helpers
// --------------------------------------------------------------------------

// RuleLoader loads and prepares rules for scanning. This is injected
// to break the dependency between scanner and rules/profiles packages.
type RuleLoader interface {
	LoadRules(profile, customDir string) ([]Rule, error)
}

// DefaultRuleLoaderFunc is a function type implementing RuleLoader.
type DefaultRuleLoaderFunc func(profile, customDir string) ([]Rule, error)

// LoadRules implements RuleLoader.
func (f DefaultRuleLoaderFunc) LoadRules(profile, customDir string) ([]Rule, error) {
	return f(profile, customDir)
}

// ruleLoaderField is set by the caller to allow dependency injection
// for rule loading. When nil, loadRules returns an error.
var defaultRuleLoader RuleLoader

// SetRuleLoader sets the global rule loader. Call this from main or init
// to wire the profiles/rules packages without circular imports.
func SetRuleLoader(l RuleLoader) {
	defaultRuleLoader = l
}

func (s *Scanner) loadRules(opts ScanOpts) ([]Rule, error) {
	if defaultRuleLoader == nil {
		return nil, fmt.Errorf("no rule loader configured; call dast.SetRuleLoader()")
	}

	profile := opts.Profile
	if profile == "" {
		profile = "standard"
	}

	rules, err := defaultRuleLoader.LoadRules(profile, s.customRulesDir)
	if err != nil {
		return nil, err
	}

	// Filter to passive-only rules for passive scanning mode.
	mode := opts.Mode
	if mode == "" {
		mode = "passive"
	}
	if mode == "passive" {
		rules = filterByType(rules, RuleTypePassive)
	}

	return rules, nil
}

// filterByType returns only rules matching the given type.
func filterByType(rules []Rule, ruleType RuleType) []Rule {
	filtered := make([]Rule, 0, len(rules))
	for _, r := range rules {
		if r.Type == ruleType {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

func (s *Scanner) buildReport(collID string, opts ScanOpts, rules []Rule, responses []*request.Response) *DASTReport {
	profile := opts.Profile
	if profile == "" {
		profile = "standard"
	}
	mode := opts.Mode
	if mode == "" {
		mode = "passive"
	}

	report := &DASTReport{
		Collection: collID,
		Profile:    profile,
		Mode:       mode,
		Findings:   make([]Finding, 0),
	}

	for _, resp := range responses {
		// Skip responses that had execution errors.
		if resp.Error != "" {
			continue
		}
		findings := s.checker(rules, resp.RequestID, resp)
		report.Findings = append(report.Findings, findings...)
	}

	// Compute severity counts.
	for _, f := range report.Findings {
		switch Severity(f.Severity) {
		case SeverityCritical:
			report.Summary.Critical++
		case SeverityHigh:
			report.Summary.High++
		case SeverityMedium:
			report.Summary.Medium++
		case SeverityLow:
			report.Summary.Low++
		case SeverityInfo:
			report.Summary.Info++
		}
	}

	return report
}
