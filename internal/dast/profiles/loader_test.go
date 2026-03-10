package profiles

import (
	"testing"

	"github.com/khanhnguyen/promptman/internal/dast"
)

func TestGet_BuiltInProfiles(t *testing.T) {
	tests := []struct {
		name     string
		minRules int
	}{
		{"quick", 10},
		{"standard", 25},
		{"thorough", 40},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := Get(tt.name)
			if p == nil {
				t.Fatalf("Get(%q) returned nil", tt.name)
			}
			if p.Name != tt.name {
				t.Errorf("Name = %q, want %q", p.Name, tt.name)
			}
			if len(p.Rules) < tt.minRules {
				t.Errorf("len(Rules) = %d, want >= %d", len(p.Rules), tt.minRules)
			}
		})
	}
}

func TestGet_NotFound(t *testing.T) {
	if p := Get("nonexistent"); p != nil {
		t.Errorf("Get(nonexistent) = %v, want nil", p)
	}
}

func TestListProfiles_ReturnsAll(t *testing.T) {
	infos := ListProfiles()

	if len(infos) < 3 {
		t.Fatalf("ListProfiles() returned %d profiles, want >= 3", len(infos))
	}

	// Verify sorted alphabetically
	for i := 1; i < len(infos); i++ {
		if infos[i-1].Name >= infos[i].Name {
			t.Errorf("profiles not sorted: %q >= %q", infos[i-1].Name, infos[i].Name)
		}
	}

	// Verify each has description and rule count
	for _, info := range infos {
		if info.Description == "" {
			t.Errorf("profile %q has empty description", info.Name)
		}
		if info.RuleCount == 0 {
			t.Errorf("profile %q has 0 rules", info.Name)
		}
	}
}

func TestNames_ReturnsSortedNames(t *testing.T) {
	names := Names()

	if len(names) < 3 {
		t.Fatalf("Names() returned %d names, want >= 3", len(names))
	}

	for i := 1; i < len(names); i++ {
		if names[i-1] >= names[i] {
			t.Errorf("names not sorted: %q >= %q", names[i-1], names[i])
		}
	}

	// Verify expected profiles exist
	expected := map[string]bool{"quick": false, "standard": false, "thorough": false}
	for _, name := range names {
		expected[name] = true
	}
	for name, found := range expected {
		if !found {
			t.Errorf("expected profile %q not in Names()", name)
		}
	}
}

func TestProfiles_RulesHaveRequiredFields(t *testing.T) {
	for _, name := range Names() {
		p := Get(name)
		for _, rule := range p.Rules {
			t.Run(name+"/"+rule.ID, func(t *testing.T) {
				if rule.ID == "" {
					t.Error("rule has empty ID")
				}
				if rule.Name == "" {
					t.Errorf("rule %q has empty Name", rule.ID)
				}
				if !rule.Severity.IsValid() {
					t.Errorf("rule %q has invalid severity: %q", rule.ID, rule.Severity)
				}
				if !rule.Type.IsValid() {
					t.Errorf("rule %q has invalid type: %q", rule.ID, rule.Type)
				}
				if rule.Description == "" {
					t.Errorf("rule %q has empty Description", rule.ID)
				}
				if rule.Remediation == "" {
					t.Errorf("rule %q has empty Remediation", rule.ID)
				}
				if !rule.Enabled {
					t.Errorf("rule %q is not Enabled", rule.ID)
				}
			})
		}
	}
}

func TestProfiles_NoDuplicateRuleIDs(t *testing.T) {
	for _, name := range Names() {
		p := Get(name)
		seen := make(map[string]bool, len(p.Rules))
		for _, rule := range p.Rules {
			if seen[rule.ID] {
				t.Errorf("profile %q has duplicate rule ID: %q", name, rule.ID)
			}
			seen[rule.ID] = true
		}
	}
}

func TestProfiles_StandardIncludesQuick(t *testing.T) {
	quick := Get("quick")
	standard := Get("standard")

	quickIDs := make(map[string]bool, len(quick.Rules))
	for _, r := range quick.Rules {
		quickIDs[r.ID] = true
	}

	standardIDs := make(map[string]bool, len(standard.Rules))
	for _, r := range standard.Rules {
		standardIDs[r.ID] = true
	}

	for id := range quickIDs {
		if !standardIDs[id] {
			t.Errorf("standard profile missing quick rule: %q", id)
		}
	}
}

func TestProfiles_ThoroughIncludesStandard(t *testing.T) {
	standard := Get("standard")
	thorough := Get("thorough")

	standardIDs := make(map[string]bool, len(standard.Rules))
	for _, r := range standard.Rules {
		standardIDs[r.ID] = true
	}

	thoroughIDs := make(map[string]bool, len(thorough.Rules))
	for _, r := range thorough.Rules {
		thoroughIDs[r.ID] = true
	}

	for id := range standardIDs {
		if !thoroughIDs[id] {
			t.Errorf("thorough profile missing standard rule: %q", id)
		}
	}
}

func TestProfiles_RuleCountsInRange(t *testing.T) {
	tests := []struct {
		name string
		min  int
		max  int
	}{
		{"quick", 8, 15},
		{"standard", 20, 30},
		{"thorough", 35, 50},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := Get(tt.name)
			n := len(p.Rules)
			if n < tt.min || n > tt.max {
				t.Errorf("profile %q has %d rules, want %d–%d", tt.name, n, tt.min, tt.max)
			}
		})
	}
}

// Compile-time check that Rule conforms to expected fields.
var _ = dast.Rule{
	ID:          "",
	Name:        "",
	Severity:    dast.SeverityCritical,
	Type:        dast.RuleTypePassive,
	Description: "",
	Remediation: "",
	Enabled:     true,
}
