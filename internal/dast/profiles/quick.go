package profiles

import "github.com/khanhnguyen/promptman/internal/dast"

// QuickRules contains ~10 fast passive checks focused on
// security headers, CORS, and basic information disclosure.
var QuickRules = []dast.Rule{
	{
		ID:          "missing-csp-header",
		Name:        "Missing Content-Security-Policy",
		Severity:    dast.SeverityMedium,
		Type:        dast.RuleTypePassive,
		Description: "Content-Security-Policy header is missing, allowing potential XSS and data injection attacks.",
		Remediation: "Add a Content-Security-Policy header with a strict policy.",
		Enabled:     true,
	},
	{
		ID:          "missing-hsts-header",
		Name:        "Missing Strict-Transport-Security",
		Severity:    dast.SeverityHigh,
		Type:        dast.RuleTypePassive,
		Description: "Strict-Transport-Security header is missing, exposing users to protocol downgrade and MITM attacks.",
		Remediation: "Add Strict-Transport-Security header with max-age ≥ 31536000.",
		Enabled:     true,
	},
	{
		ID:          "missing-x-frame-options",
		Name:        "Missing X-Frame-Options",
		Severity:    dast.SeverityMedium,
		Type:        dast.RuleTypePassive,
		Description: "X-Frame-Options header is missing, allowing the page to be framed and enabling clickjacking.",
		Remediation: "Add X-Frame-Options: DENY or SAMEORIGIN header.",
		Enabled:     true,
	},
	{
		ID:          "missing-x-content-type-options",
		Name:        "Missing X-Content-Type-Options",
		Severity:    dast.SeverityLow,
		Type:        dast.RuleTypePassive,
		Description: "X-Content-Type-Options header is missing, allowing MIME-type sniffing.",
		Remediation: "Add X-Content-Type-Options: nosniff header.",
		Enabled:     true,
	},
	{
		ID:          "cors-wildcard-origin",
		Name:        "CORS Wildcard Origin",
		Severity:    dast.SeverityHigh,
		Type:        dast.RuleTypePassive,
		Description: "Access-Control-Allow-Origin is set to *, allowing any domain to make cross-origin requests.",
		Remediation: "Restrict Access-Control-Allow-Origin to specific trusted domains.",
		Enabled:     true,
	},
	{
		ID:          "cors-credentials-wildcard",
		Name:        "CORS Credentials with Wildcard",
		Severity:    dast.SeverityCritical,
		Type:        dast.RuleTypePassive,
		Description: "CORS allows credentials with wildcard origin, which is a severe misconfiguration.",
		Remediation: "Never combine Access-Control-Allow-Credentials: true with wildcard origin.",
		Enabled:     true,
	},
	{
		ID:          "server-version-disclosure",
		Name:        "Server Version Disclosure",
		Severity:    dast.SeverityLow,
		Type:        dast.RuleTypePassive,
		Description: "Server header discloses version information that aids attackers in targeting known vulnerabilities.",
		Remediation: "Remove or genericize the Server header to hide version details.",
		Enabled:     true,
	},
	{
		ID:          "x-powered-by-disclosure",
		Name:        "X-Powered-By Disclosure",
		Severity:    dast.SeverityLow,
		Type:        dast.RuleTypePassive,
		Description: "X-Powered-By header reveals technology stack information.",
		Remediation: "Remove the X-Powered-By header from responses.",
		Enabled:     true,
	},
	{
		ID:          "sensitive-data-in-url",
		Name:        "Sensitive Data in URL",
		Severity:    dast.SeverityMedium,
		Type:        dast.RuleTypePassive,
		Description: "URL contains patterns that may include sensitive data (tokens, passwords, API keys).",
		Remediation: "Move sensitive parameters to request body or headers.",
		Enabled:     true,
	},
	{
		ID:          "missing-cache-control",
		Name:        "Missing Cache-Control for Sensitive Response",
		Severity:    dast.SeverityLow,
		Type:        dast.RuleTypePassive,
		Description: "Responses containing sensitive data lack Cache-Control: no-store, risking data exposure via caches.",
		Remediation: "Add Cache-Control: no-store for responses containing sensitive user data.",
		Enabled:     true,
	},
}

func init() {
	Register(&Profile{
		Name:        "quick",
		Description: "Fast passive scan: security headers, CORS, and information disclosure (~10 rules).",
		Rules:       QuickRules,
	})
}
