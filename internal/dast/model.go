package dast

// Severity represents the severity level of a DAST finding.
type Severity string

// Severity level constants ordered from most to least severe.
const (
	SeverityCritical Severity = "critical"
	SeverityHigh     Severity = "high"
	SeverityMedium   Severity = "medium"
	SeverityLow      Severity = "low"
	SeverityInfo     Severity = "info"
)

// String returns the string representation of the severity.
func (s Severity) String() string { return string(s) }

// IsValid reports whether s is a recognized severity level.
func (s Severity) IsValid() bool {
	switch s {
	case SeverityCritical, SeverityHigh, SeverityMedium, SeverityLow, SeverityInfo:
		return true
	}
	return false
}

// RuleType represents whether a rule is passive or active.
type RuleType string

// Rule type constants.
const (
	RuleTypePassive RuleType = "passive"
	RuleTypeActive  RuleType = "active"
)

// String returns the string representation of the rule type.
func (r RuleType) String() string { return string(r) }

// IsValid reports whether r is a recognized rule type.
func (r RuleType) IsValid() bool {
	return r == RuleTypePassive || r == RuleTypeActive
}

// Rule defines a single DAST check to be applied against a response
// (passive) or a mutated request (active).
type Rule struct {
	ID          string   `json:"id"          yaml:"id"`          // kebab-case identifier
	Name        string   `json:"name"        yaml:"name"`        // human-readable name
	Severity    Severity `json:"severity"    yaml:"severity"`    // critical|high|medium|low|info
	Type        RuleType `json:"type"        yaml:"type"`        // passive|active
	Description string   `json:"description" yaml:"description"` // what this rule checks
	Remediation string   `json:"remediation" yaml:"remediation"` // how to fix a finding
	Enabled     bool     `json:"enabled"     yaml:"enabled"`     // whether this rule is active

	// Check contains the matching criteria for custom YAML rules.
	// Built-in rules leave this nil and use compiled Go logic instead.
	Check *RuleCheck `json:"check,omitempty" yaml:"check,omitempty"`
}

// RuleCheck describes the response inspection criteria for a custom YAML rule.
type RuleCheck struct {
	Response *ResponseCheck `json:"response,omitempty" yaml:"response,omitempty"`
}

// ResponseCheck specifies what to inspect on the HTTP response.
type ResponseCheck struct {
	Headers []HeaderCheck `json:"headers,omitempty" yaml:"headers,omitempty"`
	Body    []BodyCheck   `json:"body,omitempty"    yaml:"body,omitempty"`
}

// HeaderCheck describes a single header inspection criterion.
type HeaderCheck struct {
	Name     string `json:"name"              yaml:"name"`                // header name to inspect
	Required bool   `json:"required"          yaml:"required"`            // header must be present
	Contains string `json:"contains,omitempty" yaml:"contains,omitempty"` // header value must contain
}

// BodyCheck describes a single response body inspection criterion.
type BodyCheck struct {
	Contains    string `json:"contains,omitempty"     yaml:"contains,omitempty"`     // body must contain
	NotContains string `json:"not_contains,omitempty" yaml:"not_contains,omitempty"` // body must not contain
}

// Finding represents a single security issue discovered during a DAST scan.
type Finding struct {
	Rule        string `json:"rule"`        // rule ID that produced this finding
	Severity    string `json:"severity"`    // severity level
	Request     string `json:"request"`     // request ID or name that was tested
	Description string `json:"description"` // human-readable description
	Evidence    any    `json:"evidence"`    // supporting evidence (varies by rule)
	Remediation string `json:"remediation"` // recommended remediation steps
}

// DASTReport holds the complete results of a DAST scan.
type DASTReport struct {
	Collection string        `json:"collection"` // collection that was scanned
	Profile    string        `json:"profile"`    // profile used: quick|standard|thorough
	Mode       string        `json:"mode"`       // scan mode: passive|active
	Summary    SeverityCount `json:"summary"`    // aggregate finding counts by severity
	Findings   []Finding     `json:"findings"`   // individual findings
}

// SeverityCount holds the number of findings at each severity level.
type SeverityCount struct {
	Critical int `json:"critical"`
	High     int `json:"high"`
	Medium   int `json:"medium"`
	Low      int `json:"low"`
	Info     int `json:"info"`
}

// Total returns the total number of findings across all severity levels.
func (s SeverityCount) Total() int {
	return s.Critical + s.High + s.Medium + s.Low + s.Info
}

// ScanOpts configures a DAST scan execution.
type ScanOpts struct {
	Profile    string `json:"profile"`              // quick | standard | thorough
	Mode       string `json:"mode"`                 // passive | active
	Report     string `json:"report"`               // json | html
	OutputFile string `json:"outputFile,omitempty"` // optional output file path
}

// ProfileInfo provides metadata about a built-in scan profile.
type ProfileInfo struct {
	Name        string `json:"name"`        // profile name: quick|standard|thorough
	Description string `json:"description"` // human-readable description
	RuleCount   int    `json:"ruleCount"`   // number of rules in this profile
}
