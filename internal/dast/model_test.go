package dast

import (
	"encoding/json"
	"testing"
)

func TestSeverity_String(t *testing.T) {
	tests := []struct {
		sev  Severity
		want string
	}{
		{SeverityCritical, "critical"},
		{SeverityHigh, "high"},
		{SeverityMedium, "medium"},
		{SeverityLow, "low"},
		{SeverityInfo, "info"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.sev.String(); got != tt.want {
				t.Errorf("Severity.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSeverity_IsValid(t *testing.T) {
	tests := []struct {
		sev  Severity
		want bool
	}{
		{SeverityCritical, true},
		{SeverityHigh, true},
		{SeverityMedium, true},
		{SeverityLow, true},
		{SeverityInfo, true},
		{Severity("unknown"), false},
		{Severity(""), false},
	}
	for _, tt := range tests {
		t.Run(string(tt.sev), func(t *testing.T) {
			if got := tt.sev.IsValid(); got != tt.want {
				t.Errorf("Severity(%q).IsValid() = %v, want %v", tt.sev, got, tt.want)
			}
		})
	}
}

func TestRuleType_String(t *testing.T) {
	tests := []struct {
		rt   RuleType
		want string
	}{
		{RuleTypePassive, "passive"},
		{RuleTypeActive, "active"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.rt.String(); got != tt.want {
				t.Errorf("RuleType.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRuleType_IsValid(t *testing.T) {
	tests := []struct {
		rt   RuleType
		want bool
	}{
		{RuleTypePassive, true},
		{RuleTypeActive, true},
		{RuleType("unknown"), false},
		{RuleType(""), false},
	}
	for _, tt := range tests {
		t.Run(string(tt.rt), func(t *testing.T) {
			if got := tt.rt.IsValid(); got != tt.want {
				t.Errorf("RuleType(%q).IsValid() = %v, want %v", tt.rt, got, tt.want)
			}
		})
	}
}

func TestSeverityCount_Total(t *testing.T) {
	tests := []struct {
		name string
		sc   SeverityCount
		want int
	}{
		{"zero", SeverityCount{}, 0},
		{"all ones", SeverityCount{1, 1, 1, 1, 1}, 5},
		{"mixed", SeverityCount{Critical: 2, High: 3, Medium: 5, Low: 1, Info: 10}, 21},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.sc.Total(); got != tt.want {
				t.Errorf("SeverityCount.Total() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestDASTReport_JSONRoundTrip(t *testing.T) {
	report := DASTReport{
		Collection: "users",
		Profile:    "standard",
		Mode:       "passive",
		Summary:    SeverityCount{Critical: 1, High: 2},
		Findings: []Finding{
			{
				Rule:        "missing-csp-header",
				Severity:    "high",
				Request:     "get-users",
				Description: "Content-Security-Policy header is missing",
				Evidence:    map[string]string{"header": "Content-Security-Policy"},
				Remediation: "Add Content-Security-Policy header",
			},
		},
	}

	data, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var got DASTReport
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	if got.Collection != report.Collection {
		t.Errorf("Collection = %q, want %q", got.Collection, report.Collection)
	}
	if got.Profile != report.Profile {
		t.Errorf("Profile = %q, want %q", got.Profile, report.Profile)
	}
	if got.Summary.Critical != 1 || got.Summary.High != 2 {
		t.Errorf("Summary = %+v, want Critical=1, High=2", got.Summary)
	}
	if len(got.Findings) != 1 {
		t.Fatalf("len(Findings) = %d, want 1", len(got.Findings))
	}
	if got.Findings[0].Rule != "missing-csp-header" {
		t.Errorf("Finding.Rule = %q, want %q", got.Findings[0].Rule, "missing-csp-header")
	}
}

func TestRule_WithCheck(t *testing.T) {
	rule := Rule{
		ID:          "custom-auth-check",
		Name:        "Custom Auth Header Check",
		Severity:    SeverityHigh,
		Type:        RuleTypePassive,
		Description: "Verify custom auth header is present",
		Remediation: "Add X-Custom-Auth header",
		Enabled:     true,
		Check: &RuleCheck{
			Response: &ResponseCheck{
				Headers: []HeaderCheck{
					{Name: "X-Custom-Auth", Required: true},
				},
				Body: []BodyCheck{
					{NotContains: "internal_error_trace"},
				},
			},
		},
	}

	if rule.Check == nil {
		t.Fatal("Check should not be nil")
	}
	if len(rule.Check.Response.Headers) != 1 {
		t.Fatalf("expected 1 header check, got %d", len(rule.Check.Response.Headers))
	}
	if rule.Check.Response.Headers[0].Name != "X-Custom-Auth" {
		t.Errorf("header name = %q, want %q", rule.Check.Response.Headers[0].Name, "X-Custom-Auth")
	}
	if len(rule.Check.Response.Body) != 1 {
		t.Fatalf("expected 1 body check, got %d", len(rule.Check.Response.Body))
	}
	if rule.Check.Response.Body[0].NotContains != "internal_error_trace" {
		t.Errorf("body not_contains = %q, want %q", rule.Check.Response.Body[0].NotContains, "internal_error_trace")
	}
}

func TestScanOpts_JSONOmitEmpty(t *testing.T) {
	opts := ScanOpts{
		Profile: "standard",
		Mode:    "passive",
		Report:  "json",
	}

	data, err := json.Marshal(opts)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	raw := string(data)
	if contains(raw, "outputFile") {
		t.Errorf("expected outputFile to be omitted, got: %s", raw)
	}

	// With OutputFile set
	opts.OutputFile = "report.json"
	data, err = json.Marshal(opts)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}
	raw = string(data)
	if !contains(raw, "outputFile") {
		t.Errorf("expected outputFile to be present, got: %s", raw)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
