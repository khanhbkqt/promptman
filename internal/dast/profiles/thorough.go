package profiles

import "github.com/khanhnguyen/promptman/internal/dast"

// ThoroughExtraRules contains additional rules added by the thorough profile
// on top of the standard profile. The thorough profile combines
// QuickRules + StandardExtraRules + ThoroughExtraRules for ~40 total rules.
var ThoroughExtraRules = []dast.Rule{
	// --- Fuzzing & Input Validation ---
	{
		ID:          "fuzz-large-payload",
		Name:        "Large Payload Fuzzing",
		Severity:    dast.SeverityMedium,
		Type:        dast.RuleTypeActive,
		Description: "Send oversized payloads to detect buffer overflow or DoS vulnerabilities.",
		Remediation: "Implement request body size limits and validate input length.",
		Enabled:     true,
	},
	{
		ID:          "fuzz-special-chars",
		Name:        "Special Character Fuzzing",
		Severity:    dast.SeverityMedium,
		Type:        dast.RuleTypeActive,
		Description: "Inject special characters (null bytes, unicode, control chars) to test input handling.",
		Remediation: "Sanitize and validate all input. Reject unexpected characters.",
		Enabled:     true,
	},
	{
		ID:          "fuzz-content-type-mismatch",
		Name:        "Content-Type Mismatch Fuzzing",
		Severity:    dast.SeverityMedium,
		Type:        dast.RuleTypeActive,
		Description: "Send requests with mismatched Content-Type headers to test parser confusion attacks.",
		Remediation: "Strictly validate Content-Type and reject mismatched requests.",
		Enabled:     true,
	},
	{
		ID:          "fuzz-negative-numbers",
		Name:        "Negative Number Fuzzing",
		Severity:    dast.SeverityLow,
		Type:        dast.RuleTypeActive,
		Description: "Inject negative values in numeric fields to test business logic validation.",
		Remediation: "Validate numeric ranges server-side. Reject invalid values.",
		Enabled:     true,
	},

	// --- Rate Limiting ---
	{
		ID:          "rate-limit-missing",
		Name:        "Missing Rate Limiting",
		Severity:    dast.SeverityMedium,
		Type:        dast.RuleTypePassive,
		Description: "Response lacks rate-limit headers (X-RateLimit-Limit, RateLimit-Limit), indicating no rate limiting.",
		Remediation: "Implement rate limiting on all endpoints. Return standard rate-limit headers.",
		Enabled:     true,
	},
	{
		ID:          "rate-limit-bypass",
		Name:        "Rate Limit Bypass Attempt",
		Severity:    dast.SeverityHigh,
		Type:        dast.RuleTypeActive,
		Description: "Attempt to bypass rate limiting via header manipulation (X-Forwarded-For, X-Real-IP).",
		Remediation: "Do not trust client-provided IP headers for rate limiting.",
		Enabled:     true,
	},

	// --- SSRF Patterns ---
	{
		ID:          "ssrf-internal-url",
		Name:        "SSRF — Internal URL Probe",
		Severity:    dast.SeverityCritical,
		Type:        dast.RuleTypeActive,
		Description: "Inject internal URLs (127.0.0.1, localhost, metadata endpoints) to detect SSRF.",
		Remediation: "Validate and restrict outgoing URLs. Block internal address ranges.",
		Enabled:     true,
	},
	{
		ID:          "ssrf-cloud-metadata",
		Name:        "SSRF — Cloud Metadata Access",
		Severity:    dast.SeverityCritical,
		Type:        dast.RuleTypeActive,
		Description: "Probe for cloud metadata endpoints (169.254.169.254) via URL parameters.",
		Remediation: "Block access to cloud metadata IPs. Use IMDSv2 with hop limit.",
		Enabled:     true,
	},

	// --- Path Traversal ---
	{
		ID:          "path-traversal-dot-dot",
		Name:        "Path Traversal — Dot-Dot-Slash",
		Severity:    dast.SeverityHigh,
		Type:        dast.RuleTypeActive,
		Description: "Inject ../../../ sequences to test for directory traversal vulnerabilities.",
		Remediation: "Normalize file paths and validate against a base directory.",
		Enabled:     true,
	},
	{
		ID:          "path-traversal-encoded",
		Name:        "Path Traversal — URL Encoded",
		Severity:    dast.SeverityHigh,
		Type:        dast.RuleTypeActive,
		Description: "Inject URL-encoded path traversal sequences (%2e%2e%2f) to bypass input filters.",
		Remediation: "Decode and normalize paths before validation. Never rely on encoding alone.",
		Enabled:     true,
	},

	// --- HTTP Method & Verb Tampering ---
	{
		ID:          "http-method-override",
		Name:        "HTTP Method Override",
		Severity:    dast.SeverityMedium,
		Type:        dast.RuleTypeActive,
		Description: "Test X-HTTP-Method-Override and X-Method-Override headers for method tampering.",
		Remediation: "Ignore method override headers or restrict to trusted proxies only.",
		Enabled:     true,
	},
	{
		ID:          "http-trace-enabled",
		Name:        "HTTP TRACE Method Enabled",
		Severity:    dast.SeverityMedium,
		Type:        dast.RuleTypeActive,
		Description: "TRACE HTTP method is enabled, which can be used for cross-site tracing attacks.",
		Remediation: "Disable TRACE method on the server.",
		Enabled:     true,
	},

	// --- Content Security ---
	{
		ID:          "json-hijacking",
		Name:        "JSON Hijacking Protection",
		Severity:    dast.SeverityMedium,
		Type:        dast.RuleTypePassive,
		Description: "JSON responses with arrays at the top level lack anti-hijacking prefix or X-Content-Type-Options.",
		Remediation: "Wrap JSON arrays in objects. Set X-Content-Type-Options: nosniff.",
		Enabled:     true,
	},
	{
		ID:          "insecure-deserialization",
		Name:        "Insecure Deserialization Pattern",
		Severity:    dast.SeverityHigh,
		Type:        dast.RuleTypePassive,
		Description: "Response indicates deserialization of user-controlled data (Java serialized objects, pickle, etc.).",
		Remediation: "Avoid deserializing untrusted data. Use safe serialization formats.",
		Enabled:     true,
	},

	// --- Advanced Header Checks ---
	{
		ID:          "csp-unsafe-inline",
		Name:        "CSP Contains unsafe-inline",
		Severity:    dast.SeverityMedium,
		Type:        dast.RuleTypePassive,
		Description: "Content-Security-Policy contains 'unsafe-inline', significantly weakening XSS protection.",
		Remediation: "Replace unsafe-inline with nonce-based or hash-based CSP directives.",
		Enabled:     true,
	},
	{
		ID:          "cors-origin-reflection",
		Name:        "CORS Origin Reflection",
		Severity:    dast.SeverityHigh,
		Type:        dast.RuleTypeActive,
		Description: "Server reflects the Origin header value in Access-Control-Allow-Origin without validation.",
		Remediation: "Validate Origin against an explicit allowlist. Never reflect arbitrary origins.",
		Enabled:     true,
	},
}

func init() {
	rules := make([]dast.Rule, 0, len(QuickRules)+len(StandardExtraRules)+len(ThoroughExtraRules))
	rules = append(rules, QuickRules...)
	rules = append(rules, StandardExtraRules...)
	rules = append(rules, ThoroughExtraRules...)

	Register(&Profile{
		Name:        "thorough",
		Description: "Deep scan: all checks plus fuzzing, rate limiting, SSRF, path traversal (~40 rules).",
		Rules:       rules,
	})
}
