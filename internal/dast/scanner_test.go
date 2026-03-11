package dast

import (
	"context"
	"testing"

	"github.com/khanhnguyen/promptman/internal/request"
)

// --------------------------------------------------------------------------
// Mock types
// --------------------------------------------------------------------------

type mockExecutor struct {
	responses  []*request.Response
	singleResp *request.Response
	execErr    error
}

func (m *mockExecutor) Execute(_ context.Context, _ request.ExecuteInput) (*request.Response, error) {
	if m.execErr != nil {
		return nil, m.execErr
	}
	return m.singleResp, nil
}

func (m *mockExecutor) ExecuteCollection(_ context.Context, _ request.CollectionRunOpts) ([]*request.Response, error) {
	if m.execErr != nil {
		return nil, m.execErr
	}
	return m.responses, nil
}

func mockChecker(rules []Rule, reqID string, resp *request.Response) []Finding {
	var findings []Finding
	for _, r := range rules {
		if r.Type != RuleTypePassive || !r.Enabled {
			continue
		}
		// Simple mock: trigger on missing Content-Security-Policy header.
		if r.ID == "missing-csp-header" {
			if _, ok := resp.Headers["Content-Security-Policy"]; !ok {
				findings = append(findings, Finding{
					Rule:        r.ID,
					Severity:    string(r.Severity),
					Request:     reqID,
					Description: r.Description,
					Evidence:    map[string]string{"missing_header": "Content-Security-Policy"},
					Remediation: r.Remediation,
				})
			}
		}
	}
	return findings
}

func mockRuleLoader(profile, customDir string) ([]Rule, error) {
	return []Rule{
		{
			ID:          "missing-csp-header",
			Name:        "Missing CSP",
			Severity:    SeverityMedium,
			Type:        RuleTypePassive,
			Description: "CSP header is missing",
			Remediation: "Add CSP header",
			Enabled:     true,
		},
		{
			ID:          "active-rule",
			Name:        "SQLi Probe",
			Severity:    SeverityCritical,
			Type:        RuleTypeActive,
			Description: "SQL injection probe",
			Remediation: "Parameterize queries",
			Enabled:     true,
		},
	}, nil
}

// --------------------------------------------------------------------------
// Setup
// --------------------------------------------------------------------------

func init() {
	SetRuleLoader(DefaultRuleLoaderFunc(mockRuleLoader))
}

// --------------------------------------------------------------------------
// Tests
// --------------------------------------------------------------------------

func TestScanner_Scan(t *testing.T) {
	exec := &mockExecutor{
		responses: []*request.Response{
			{
				RequestID: "req-1",
				Method:    "GET",
				URL:       "https://example.com/api/users",
				Status:    200,
				Headers:   map[string]string{"Content-Type": "application/json"},
				Body:      `{"users":[]}`,
			},
			{
				RequestID: "req-2",
				Method:    "POST",
				URL:       "https://example.com/api/login",
				Status:    200,
				Headers:   map[string]string{"Content-Security-Policy": "default-src 'self'"},
				Body:      `{"token":"abc"}`,
			},
		},
	}

	s := NewScanner(exec, mockChecker)
	report, err := s.Scan(context.Background(), "test-collection", ScanOpts{
		Profile: "quick",
	})
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	if report.Collection != "test-collection" {
		t.Errorf("collection: want test-collection, got %s", report.Collection)
	}
	if report.Profile != "quick" {
		t.Errorf("profile: want quick, got %s", report.Profile)
	}
	if report.Mode != "passive" {
		t.Errorf("mode: want passive, got %s", report.Mode)
	}

	// req-1 has no CSP → 1 finding; req-2 has CSP → 0 findings.
	if len(report.Findings) != 1 {
		t.Fatalf("want 1 finding, got %d", len(report.Findings))
	}
	if report.Findings[0].Request != "req-1" {
		t.Errorf("finding request: want req-1, got %s", report.Findings[0].Request)
	}
	if report.Summary.Medium != 1 {
		t.Errorf("summary medium: want 1, got %d", report.Summary.Medium)
	}
}

func TestScanner_ScanRequest(t *testing.T) {
	exec := &mockExecutor{
		singleResp: &request.Response{
			RequestID: "req-1",
			Method:    "GET",
			URL:       "https://example.com/api/users",
			Status:    200,
			Headers:   map[string]string{},
			Body:      "",
		},
	}

	s := NewScanner(exec, mockChecker)
	report, err := s.ScanRequest(context.Background(), "test-coll", "req-1", ScanOpts{
		Profile: "standard",
	})
	if err != nil {
		t.Fatalf("ScanRequest failed: %v", err)
	}

	if len(report.Findings) != 1 {
		t.Fatalf("want 1 finding, got %d", len(report.Findings))
	}
	if report.Findings[0].Rule != "missing-csp-header" {
		t.Errorf("finding rule: want missing-csp-header, got %s", report.Findings[0].Rule)
	}
}

func TestScanner_Scan_SkipsErrorResponses(t *testing.T) {
	exec := &mockExecutor{
		responses: []*request.Response{
			{
				RequestID: "good-req",
				Status:    200,
				Headers:   map[string]string{},
			},
			{
				RequestID: "bad-req",
				Error:     "connection refused",
			},
		},
	}

	s := NewScanner(exec, mockChecker)
	report, err := s.Scan(context.Background(), "coll", ScanOpts{})
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	// Only good-req should produce findings.
	if len(report.Findings) != 1 {
		t.Fatalf("want 1 finding (from good-req only), got %d", len(report.Findings))
	}
	if report.Findings[0].Request != "good-req" {
		t.Errorf("finding from wrong request: %s", report.Findings[0].Request)
	}
}

func TestScanner_Scan_DefaultProfile(t *testing.T) {
	exec := &mockExecutor{
		responses: []*request.Response{
			{RequestID: "r1", Status: 200, Headers: map[string]string{}},
		},
	}

	s := NewScanner(exec, mockChecker)
	report, err := s.Scan(context.Background(), "coll", ScanOpts{}) // no profile specified
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}
	if report.Profile != "standard" {
		t.Errorf("default profile: want standard, got %s", report.Profile)
	}
	if report.Mode != "passive" {
		t.Errorf("default mode: want passive, got %s", report.Mode)
	}
}

func TestScanner_Scan_FiltersActiveRules(t *testing.T) {
	exec := &mockExecutor{
		responses: []*request.Response{
			{RequestID: "r1", Status: 200, Headers: map[string]string{}},
		},
	}

	// Custom checker that counts how many rules it receives.
	var receivedRules int
	checker := func(rules []Rule, reqID string, resp *request.Response) []Finding {
		receivedRules = len(rules)
		return nil
	}

	s := NewScanner(exec, checker)
	_, err := s.Scan(context.Background(), "coll", ScanOpts{Mode: "passive"})
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	// mockRuleLoader returns 2 rules (1 passive, 1 active).
	// In passive mode, active rules should be filtered out.
	if receivedRules != 1 {
		t.Errorf("want 1 passive rule after filtering, got %d", receivedRules)
	}
}

func TestScanner_SeverityCounts(t *testing.T) {
	exec := &mockExecutor{
		responses: []*request.Response{
			{RequestID: "r1", Status: 200, Headers: map[string]string{}},
		},
	}

	checker := func(rules []Rule, reqID string, resp *request.Response) []Finding {
		return []Finding{
			{Rule: "r1", Severity: "critical", Request: reqID},
			{Rule: "r2", Severity: "high", Request: reqID},
			{Rule: "r3", Severity: "high", Request: reqID},
			{Rule: "r4", Severity: "medium", Request: reqID},
			{Rule: "r5", Severity: "low", Request: reqID},
			{Rule: "r6", Severity: "info", Request: reqID},
		}
	}

	s := NewScanner(exec, checker)
	report, err := s.Scan(context.Background(), "coll", ScanOpts{})
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	if report.Summary.Critical != 1 {
		t.Errorf("critical: want 1, got %d", report.Summary.Critical)
	}
	if report.Summary.High != 2 {
		t.Errorf("high: want 2, got %d", report.Summary.High)
	}
	if report.Summary.Medium != 1 {
		t.Errorf("medium: want 1, got %d", report.Summary.Medium)
	}
	if report.Summary.Low != 1 {
		t.Errorf("low: want 1, got %d", report.Summary.Low)
	}
	if report.Summary.Info != 1 {
		t.Errorf("info: want 1, got %d", report.Summary.Info)
	}
	if report.Summary.Total() != 6 {
		t.Errorf("total: want 6, got %d", report.Summary.Total())
	}
}

func TestFilterByType(t *testing.T) {
	rules := []Rule{
		{ID: "p1", Type: RuleTypePassive, Enabled: true},
		{ID: "a1", Type: RuleTypeActive, Enabled: true},
		{ID: "p2", Type: RuleTypePassive, Enabled: true},
	}

	passive := filterByType(rules, RuleTypePassive)
	if len(passive) != 2 {
		t.Fatalf("want 2 passive rules, got %d", len(passive))
	}
	if passive[0].ID != "p1" || passive[1].ID != "p2" {
		t.Error("wrong passive rules returned")
	}

	active := filterByType(rules, RuleTypeActive)
	if len(active) != 1 {
		t.Fatalf("want 1 active rule, got %d", len(active))
	}
}

func TestNewScanner_WithCustomRulesDir(t *testing.T) {
	exec := &mockExecutor{}
	s := NewScanner(exec, mockChecker, WithCustomRulesDir("/custom/path"))
	if s.customRulesDir != "/custom/path" {
		t.Errorf("want /custom/path, got %s", s.customRulesDir)
	}
}
