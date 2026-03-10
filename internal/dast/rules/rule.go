package rules

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/khanhnguyen/promptman/internal/dast"
	"github.com/khanhnguyen/promptman/pkg/fsutil"
)

// LoadCustomRules loads custom DAST rule definitions from YAML files
// in the given directory. Files must match the pattern *.yaml.
// If the directory does not exist, an empty slice is returned (not an error).
// Invalid YAML files produce clear error messages.
func LoadCustomRules(dir string) ([]dast.Rule, error) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, nil
	}

	pattern := filepath.Join(dir, "*.yaml")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, dast.ErrRuleLoadFailed.Wrapf("glob %s: %v", pattern, err)
	}

	var rules []dast.Rule
	for _, file := range files {
		rule, err := loadRuleFile(file)
		if err != nil {
			return nil, err
		}
		rules = append(rules, *rule)
	}
	return rules, nil
}

// yamlRule is the on-disk YAML representation of a custom rule.
// It matches the M5 spec §6 custom rule format.
type yamlRule struct {
	Name     string          `yaml:"name"`
	Severity string          `yaml:"severity"`
	Type     string          `yaml:"type"`
	Check    *dast.RuleCheck `yaml:"check,omitempty"`
}

// loadRuleFile reads a single YAML rule file and converts it to a Rule.
func loadRuleFile(path string) (*dast.Rule, error) {
	var yr yamlRule
	if err := fsutil.ReadYAML(path, &yr); err != nil {
		return nil, dast.ErrRuleLoadFailed.Wrapf("loading %s: %v", filepath.Base(path), err)
	}

	rule, err := convertYAMLRule(filepath.Base(path), yr)
	if err != nil {
		return nil, err
	}
	return rule, nil
}

// convertYAMLRule validates and converts a yamlRule to a dast.Rule.
func convertYAMLRule(filename string, yr yamlRule) (*dast.Rule, error) {
	if yr.Name == "" {
		return nil, dast.ErrInvalidRule.Wrapf("rule in %s: missing required field 'name'", filename)
	}

	severity := dast.Severity(yr.Severity)
	if !severity.IsValid() {
		return nil, dast.ErrInvalidRule.Wrapf("rule %q in %s: invalid severity %q (must be critical|high|medium|low|info)",
			yr.Name, filename, yr.Severity)
	}

	// Default to passive if type is omitted
	ruleType := dast.RuleType(yr.Type)
	if yr.Type == "" {
		ruleType = dast.RuleTypePassive
	} else if !ruleType.IsValid() {
		return nil, dast.ErrInvalidRule.Wrapf("rule %q in %s: invalid type %q (must be passive|active)",
			yr.Name, filename, yr.Type)
	}

	// Generate ID from filename: "custom-auth-check.yaml" → "custom-auth-check"
	id := strings.TrimSuffix(filename, filepath.Ext(filename))

	return &dast.Rule{
		ID:       id,
		Name:     yr.Name,
		Severity: severity,
		Type:     ruleType,
		Enabled:  true,
		Check:    yr.Check,
	}, nil
}

// MergeRules combines profile rules with custom rules.
// Custom rules are appended after profile rules.
func MergeRules(profileRules, customRules []dast.Rule) []dast.Rule {
	merged := make([]dast.Rule, 0, len(profileRules)+len(customRules))
	merged = append(merged, profileRules...)
	merged = append(merged, customRules...)
	return merged
}

// FilterDisabled removes rules whose IDs appear in the disabled set.
func FilterDisabled(rules []dast.Rule, disabled []string) []dast.Rule {
	if len(disabled) == 0 {
		return rules
	}

	disabledSet := make(map[string]bool, len(disabled))
	for _, id := range disabled {
		disabledSet[id] = true
	}

	filtered := make([]dast.Rule, 0, len(rules))
	for _, r := range rules {
		if !disabledSet[r.ID] {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

// ValidateRules checks that all rules in the slice have valid required fields.
func ValidateRules(rules []dast.Rule) error {
	seen := make(map[string]bool, len(rules))
	for _, r := range rules {
		if r.ID == "" {
			return fmt.Errorf("rule with name %q has empty ID", r.Name)
		}
		if seen[r.ID] {
			return fmt.Errorf("duplicate rule ID: %q", r.ID)
		}
		seen[r.ID] = true

		if !r.Severity.IsValid() {
			return dast.ErrInvalidRule.Wrapf("rule %q: invalid severity %q", r.ID, r.Severity)
		}
		if !r.Type.IsValid() {
			return dast.ErrInvalidRule.Wrapf("rule %q: invalid type %q", r.ID, r.Type)
		}
	}
	return nil
}
