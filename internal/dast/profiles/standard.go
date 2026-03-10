package profiles

import "github.com/khanhnguyen/promptman/internal/dast"

// StandardExtraRules contains additional rules added by the standard profile
// on top of the quick profile rules. The standard profile combines
// QuickRules + StandardExtraRules for ~25 total rules.
var StandardExtraRules = []dast.Rule{
	// --- SQL Injection Probes ---
	{
		ID:          "sqli-error-pattern",
		Name:        "SQL Injection Error Pattern",
		Severity:    dast.SeverityCritical,
		Type:        dast.RuleTypePassive,
		Description: "Response body contains SQL error messages indicating potential SQL injection vulnerability.",
		Remediation: "Use parameterized queries and proper input validation.",
		Enabled:     true,
	},
	{
		ID:          "sqli-probe-single-quote",
		Name:        "SQL Injection Probe — Single Quote",
		Severity:    dast.SeverityCritical,
		Type:        dast.RuleTypeActive,
		Description: "Inject single quote character to detect SQL injection via error-based response differences.",
		Remediation: "Use parameterized queries. Never concatenate user input into SQL.",
		Enabled:     true,
	},
	{
		ID:          "sqli-probe-boolean",
		Name:        "SQL Injection Probe — Boolean",
		Severity:    dast.SeverityCritical,
		Type:        dast.RuleTypeActive,
		Description: "Inject boolean-based SQL payloads (OR 1=1) to detect blind SQL injection.",
		Remediation: "Use parameterized queries and allowlist-based input validation.",
		Enabled:     true,
	},

	// --- XSS Probes ---
	{
		ID:          "xss-reflection-probe",
		Name:        "XSS Reflection Probe",
		Severity:    dast.SeverityHigh,
		Type:        dast.RuleTypeActive,
		Description: "Inject XSS payloads and check if they reflect in the response without encoding.",
		Remediation: "Encode all user-supplied output. Use Content-Security-Policy.",
		Enabled:     true,
	},
	{
		ID:          "xss-error-page",
		Name:        "XSS in Error Pages",
		Severity:    dast.SeverityHigh,
		Type:        dast.RuleTypePassive,
		Description: "Error pages reflect request parameters without encoding, enabling XSS.",
		Remediation: "Sanitize all reflected content in error pages.",
		Enabled:     true,
	},

	// --- Open Redirect ---
	{
		ID:          "open-redirect-probe",
		Name:        "Open Redirect",
		Severity:    dast.SeverityMedium,
		Type:        dast.RuleTypeActive,
		Description: "Inject external URLs in redirect parameters to test for open redirect vulnerabilities.",
		Remediation: "Validate redirect URLs against an allowlist of trusted domains.",
		Enabled:     true,
	},

	// --- Authentication & Session ---
	{
		ID:          "auth-bypass-method-swap",
		Name:        "Auth Bypass — Method Swap",
		Severity:    dast.SeverityHigh,
		Type:        dast.RuleTypeActive,
		Description: "Change HTTP method (GET↔POST) on protected endpoints to test for method-based auth bypass.",
		Remediation: "Enforce authentication checks regardless of HTTP method.",
		Enabled:     true,
	},
	{
		ID:          "insecure-cookie-flags",
		Name:        "Insecure Cookie Flags",
		Severity:    dast.SeverityMedium,
		Type:        dast.RuleTypePassive,
		Description: "Session cookies are missing Secure, HttpOnly, or SameSite flags.",
		Remediation: "Set Secure, HttpOnly, and SameSite=Lax/Strict on session cookies.",
		Enabled:     true,
	},
	{
		ID:          "bearer-token-in-body",
		Name:        "Bearer Token in Response Body",
		Severity:    dast.SeverityHigh,
		Type:        dast.RuleTypePassive,
		Description: "Bearer/JWT tokens appear in response body, risking exposure via logs and caches.",
		Remediation: "Return tokens only in Set-Cookie headers or secure token storage mechanisms.",
		Enabled:     true,
	},

	// --- Information Disclosure ---
	{
		ID:          "stack-trace-disclosure",
		Name:        "Stack Trace Disclosure",
		Severity:    dast.SeverityMedium,
		Type:        dast.RuleTypePassive,
		Description: "Response body contains stack trace or debug information from application errors.",
		Remediation: "Disable detailed error messages in production. Use generic error responses.",
		Enabled:     true,
	},
	{
		ID:          "internal-ip-disclosure",
		Name:        "Internal IP Address Disclosure",
		Severity:    dast.SeverityLow,
		Type:        dast.RuleTypePassive,
		Description: "Response headers or body contain internal IP addresses (10.x, 192.168.x, 172.16-31.x).",
		Remediation: "Remove internal IP addresses from responses and headers.",
		Enabled:     true,
	},
	{
		ID:          "debug-endpoint-exposed",
		Name:        "Debug Endpoint Exposed",
		Severity:    dast.SeverityHigh,
		Type:        dast.RuleTypePassive,
		Description: "Response contains indicators of exposed debug endpoints (/debug, /trace, /phpinfo).",
		Remediation: "Disable debug endpoints in production or restrict access.",
		Enabled:     true,
	},
	{
		ID:          "error-message-verbose",
		Name:        "Verbose Error Messages",
		Severity:    dast.SeverityMedium,
		Type:        dast.RuleTypePassive,
		Description: "Error responses include verbose technical details (database names, file paths, query strings).",
		Remediation: "Return generic error messages. Log details server-side only.",
		Enabled:     true,
	},

	// --- Header Security ---
	{
		ID:          "missing-referrer-policy",
		Name:        "Missing Referrer-Policy",
		Severity:    dast.SeverityLow,
		Type:        dast.RuleTypePassive,
		Description: "Referrer-Policy header is missing, potentially leaking URL information to external sites.",
		Remediation: "Add Referrer-Policy: strict-origin-when-cross-origin header.",
		Enabled:     true,
	},
	{
		ID:          "missing-permissions-policy",
		Name:        "Missing Permissions-Policy",
		Severity:    dast.SeverityLow,
		Type:        dast.RuleTypePassive,
		Description: "Permissions-Policy (formerly Feature-Policy) header is missing.",
		Remediation: "Add Permissions-Policy header to restrict browser feature access.",
		Enabled:     true,
	},
}

func init() {
	rules := make([]dast.Rule, 0, len(QuickRules)+len(StandardExtraRules))
	rules = append(rules, QuickRules...)
	rules = append(rules, StandardExtraRules...)

	Register(&Profile{
		Name:        "standard",
		Description: "Comprehensive scan: security headers, injection probes, auth checks, info disclosure (~25 rules).",
		Rules:       rules,
	})
}
