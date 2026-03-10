package rules

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/khanhnguyen/promptman/internal/dast"
	"github.com/khanhnguyen/promptman/pkg/envelope"
)

func TestLoadCustomRules_ValidRule(t *testing.T) {
	dir := t.TempDir()
	writeRule(t, dir, "custom-header-check.yaml", `
name: Custom Header Check
severity: high
type: passive
check:
  response:
    headers:
      - name: X-Custom-Auth
        required: true
    body:
      - not_contains: internal_error_trace
`)

	rules, err := LoadCustomRules(dir)
	if err != nil {
		t.Fatalf("LoadCustomRules() error: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("len(rules) = %d, want 1", len(rules))
	}

	r := rules[0]
	if r.ID != "custom-header-check" {
		t.Errorf("ID = %q, want %q", r.ID, "custom-header-check")
	}
	if r.Name != "Custom Header Check" {
		t.Errorf("Name = %q, want %q", r.Name, "Custom Header Check")
	}
	if r.Severity != dast.SeverityHigh {
		t.Errorf("Severity = %q, want %q", r.Severity, dast.SeverityHigh)
	}
	if r.Type != dast.RuleTypePassive {
		t.Errorf("Type = %q, want %q", r.Type, dast.RuleTypePassive)
	}
	if !r.Enabled {
		t.Error("Enabled = false, want true")
	}
	if r.Check == nil || r.Check.Response == nil {
		t.Fatal("Check.Response is nil")
	}
	if len(r.Check.Response.Headers) != 1 {
		t.Fatalf("len(Headers) = %d, want 1", len(r.Check.Response.Headers))
	}
	if r.Check.Response.Headers[0].Name != "X-Custom-Auth" {
		t.Errorf("Header.Name = %q, want %q", r.Check.Response.Headers[0].Name, "X-Custom-Auth")
	}
}

func TestLoadCustomRules_MissingDirectory(t *testing.T) {
	rules, err := LoadCustomRules("/nonexistent/path")
	if err != nil {
		t.Fatalf("LoadCustomRules() error: %v", err)
	}
	if len(rules) != 0 {
		t.Errorf("expected empty slice, got %d rules", len(rules))
	}
}

func TestLoadCustomRules_EmptyDirectory(t *testing.T) {
	dir := t.TempDir()
	rules, err := LoadCustomRules(dir)
	if err != nil {
		t.Fatalf("LoadCustomRules() error: %v", err)
	}
	if len(rules) != 0 {
		t.Errorf("expected empty slice, got %d rules", len(rules))
	}
}

func TestLoadCustomRules_MultipleFiles(t *testing.T) {
	dir := t.TempDir()
	writeRule(t, dir, "rule-a.yaml", `
name: Rule A
severity: low
type: passive
`)
	writeRule(t, dir, "rule-b.yaml", `
name: Rule B
severity: high
type: active
`)

	rules, err := LoadCustomRules(dir)
	if err != nil {
		t.Fatalf("LoadCustomRules() error: %v", err)
	}
	if len(rules) != 2 {
		t.Fatalf("len(rules) = %d, want 2", len(rules))
	}
}

func TestLoadCustomRules_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	writeRule(t, dir, "bad.yaml", `{invalid: yaml: [`)

	_, err := LoadCustomRules(dir)
	if err == nil {
		t.Fatal("expected error for invalid YAML, got nil")
	}
	if !dast.IsDomainError(err, envelope.CodeRuleLoadFailed) {
		t.Errorf("expected RULE_LOAD_FAILED error, got: %v", err)
	}
}

func TestLoadCustomRules_MissingName(t *testing.T) {
	dir := t.TempDir()
	writeRule(t, dir, "no-name.yaml", `
severity: high
type: passive
`)

	_, err := LoadCustomRules(dir)
	if err == nil {
		t.Fatal("expected error for missing name, got nil")
	}
	if !dast.IsDomainError(err, envelope.CodeInvalidRule) {
		t.Errorf("expected INVALID_RULE error, got: %v", err)
	}
}

func TestLoadCustomRules_InvalidSeverity(t *testing.T) {
	dir := t.TempDir()
	writeRule(t, dir, "bad-sev.yaml", `
name: Bad Severity
severity: ultra
type: passive
`)

	_, err := LoadCustomRules(dir)
	if err == nil {
		t.Fatal("expected error for invalid severity, got nil")
	}
	if !dast.IsDomainError(err, envelope.CodeInvalidRule) {
		t.Errorf("expected INVALID_RULE error, got: %v", err)
	}
}

func TestLoadCustomRules_InvalidType(t *testing.T) {
	dir := t.TempDir()
	writeRule(t, dir, "bad-type.yaml", `
name: Bad Type
severity: high
type: nuclear
`)

	_, err := LoadCustomRules(dir)
	if err == nil {
		t.Fatal("expected error for invalid type, got nil")
	}
	if !dast.IsDomainError(err, envelope.CodeInvalidRule) {
		t.Errorf("expected INVALID_RULE error, got: %v", err)
	}
}

func TestLoadCustomRules_DefaultsToPassive(t *testing.T) {
	dir := t.TempDir()
	writeRule(t, dir, "no-type.yaml", `
name: No Type Specified
severity: medium
`)

	rules, err := LoadCustomRules(dir)
	if err != nil {
		t.Fatalf("LoadCustomRules() error: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("len(rules) = %d, want 1", len(rules))
	}
	if rules[0].Type != dast.RuleTypePassive {
		t.Errorf("Type = %q, want %q (default)", rules[0].Type, dast.RuleTypePassive)
	}
}

func TestMergeRules(t *testing.T) {
	profile := []dast.Rule{{ID: "p1"}, {ID: "p2"}}
	custom := []dast.Rule{{ID: "c1"}}

	merged := MergeRules(profile, custom)
	if len(merged) != 3 {
		t.Fatalf("len(merged) = %d, want 3", len(merged))
	}
	// Profile rules first
	if merged[0].ID != "p1" || merged[1].ID != "p2" || merged[2].ID != "c1" {
		t.Errorf("order = [%s, %s, %s], want [p1, p2, c1]", merged[0].ID, merged[1].ID, merged[2].ID)
	}
}

func TestMergeRules_EmptyInputs(t *testing.T) {
	if merged := MergeRules(nil, nil); len(merged) != 0 {
		t.Errorf("MergeRules(nil, nil) = %d rules, want 0", len(merged))
	}
	if merged := MergeRules([]dast.Rule{{ID: "a"}}, nil); len(merged) != 1 {
		t.Errorf("MergeRules(1, nil) = %d, want 1", len(merged))
	}
}

func TestFilterDisabled(t *testing.T) {
	tests := []struct {
		name     string
		rules    []dast.Rule
		disabled []string
		wantIDs  []string
	}{
		{
			"no disabled",
			[]dast.Rule{{ID: "a"}, {ID: "b"}, {ID: "c"}},
			nil,
			[]string{"a", "b", "c"},
		},
		{
			"disable one",
			[]dast.Rule{{ID: "a"}, {ID: "b"}, {ID: "c"}},
			[]string{"b"},
			[]string{"a", "c"},
		},
		{
			"disable all",
			[]dast.Rule{{ID: "a"}, {ID: "b"}},
			[]string{"a", "b"},
			nil,
		},
		{
			"disable nonexistent",
			[]dast.Rule{{ID: "a"}, {ID: "b"}},
			[]string{"x", "y"},
			[]string{"a", "b"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FilterDisabled(tt.rules, tt.disabled)
			gotIDs := make([]string, len(got))
			for i, r := range got {
				gotIDs[i] = r.ID
			}
			if len(gotIDs) != len(tt.wantIDs) {
				t.Fatalf("len = %d, want %d: %v", len(gotIDs), len(tt.wantIDs), gotIDs)
			}
			for i := range gotIDs {
				if gotIDs[i] != tt.wantIDs[i] {
					t.Errorf("ID[%d] = %q, want %q", i, gotIDs[i], tt.wantIDs[i])
				}
			}
		})
	}
}

func TestValidateRules_Valid(t *testing.T) {
	rules := []dast.Rule{
		{ID: "rule-1", Severity: dast.SeverityHigh, Type: dast.RuleTypePassive},
		{ID: "rule-2", Severity: dast.SeverityLow, Type: dast.RuleTypeActive},
	}
	if err := ValidateRules(rules); err != nil {
		t.Errorf("ValidateRules() = %v, want nil", err)
	}
}

func TestValidateRules_DuplicateID(t *testing.T) {
	rules := []dast.Rule{
		{ID: "dup", Severity: dast.SeverityHigh, Type: dast.RuleTypePassive},
		{ID: "dup", Severity: dast.SeverityLow, Type: dast.RuleTypeActive},
	}
	if err := ValidateRules(rules); err == nil {
		t.Error("expected error for duplicate ID, got nil")
	}
}

func TestValidateRules_EmptyID(t *testing.T) {
	rules := []dast.Rule{
		{ID: "", Name: "No ID", Severity: dast.SeverityHigh, Type: dast.RuleTypePassive},
	}
	if err := ValidateRules(rules); err == nil {
		t.Error("expected error for empty ID, got nil")
	}
}

// writeRule writes a YAML rule file to the given directory.
func writeRule(t *testing.T, dir, filename, content string) {
	t.Helper()
	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write %s: %v", filename, err)
	}
}
