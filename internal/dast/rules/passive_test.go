package rules

import (
	"strings"
	"testing"

	"github.com/khanhnguyen/promptman/internal/dast"
	"github.com/khanhnguyen/promptman/internal/request"
)

// --------------------------------------------------------------------------
// helpers
// --------------------------------------------------------------------------

func passiveRule(id string) dast.Rule {
	return dast.Rule{
		ID:          id,
		Name:        "Test: " + id,
		Severity:    dast.SeverityMedium,
		Type:        dast.RuleTypePassive,
		Description: "test rule " + id,
		Remediation: "fix it",
		Enabled:     true,
	}
}

func resp(status int, headers map[string]string, body string) *request.Response {
	return &request.Response{
		RequestID: "test-req",
		Method:    "GET",
		URL:       "https://example.com/api/test",
		Status:    status,
		Headers:   headers,
		Body:      body,
	}
}

// --------------------------------------------------------------------------
// Missing header checks
// --------------------------------------------------------------------------

func TestCheckPassive_MissingCSP(t *testing.T) {
	rule := passiveRule("missing-csp-header")
	// No CSP header → finding expected.
	f := CheckPassive(&rule, "r1", resp(200, map[string]string{}, ""))
	if f == nil {
		t.Fatal("expected finding for missing CSP header")
	}
	if f.Rule != "missing-csp-header" {
		t.Errorf("want rule missing-csp-header, got %s", f.Rule)
	}

	// With CSP header → no finding.
	f = CheckPassive(&rule, "r1", resp(200, map[string]string{"Content-Security-Policy": "default-src 'self'"}, ""))
	if f != nil {
		t.Fatal("expected no finding when CSP header present")
	}
}

func TestCheckPassive_MissingHSTS(t *testing.T) {
	rule := passiveRule("missing-hsts-header")
	f := CheckPassive(&rule, "r1", resp(200, map[string]string{}, ""))
	if f == nil {
		t.Fatal("expected finding for missing HSTS header")
	}
	f = CheckPassive(&rule, "r1", resp(200, map[string]string{"Strict-Transport-Security": "max-age=31536000"}, ""))
	if f != nil {
		t.Fatal("expected no finding when HSTS header present")
	}
}

// --------------------------------------------------------------------------
// CORS checks
// --------------------------------------------------------------------------

func TestCheckPassive_CORSWildcard(t *testing.T) {
	rule := passiveRule("cors-wildcard-origin")

	// Wildcard → finding.
	f := CheckPassive(&rule, "r1", resp(200, map[string]string{
		"Access-Control-Allow-Origin": "*",
	}, ""))
	if f == nil {
		t.Fatal("expected finding for wildcard CORS")
	}

	// Specific origin → no finding.
	f = CheckPassive(&rule, "r1", resp(200, map[string]string{
		"Access-Control-Allow-Origin": "https://example.com",
	}, ""))
	if f != nil {
		t.Fatal("expected no finding for specific origin")
	}
}

func TestCheckPassive_CORSCredentialsWildcard(t *testing.T) {
	rule := passiveRule("cors-credentials-wildcard")

	// Both wildcard + credentials → finding.
	f := CheckPassive(&rule, "r1", resp(200, map[string]string{
		"Access-Control-Allow-Origin":      "*",
		"Access-Control-Allow-Credentials": "true",
	}, ""))
	if f == nil {
		t.Fatal("expected finding for credentials + wildcard")
	}

	// No credentials → no finding.
	f = CheckPassive(&rule, "r1", resp(200, map[string]string{
		"Access-Control-Allow-Origin": "*",
	}, ""))
	if f != nil {
		t.Fatal("expected no finding without credentials header")
	}
}

// --------------------------------------------------------------------------
// Server disclosure
// --------------------------------------------------------------------------

func TestCheckPassive_ServerVersionDisclosure(t *testing.T) {
	rule := passiveRule("server-version-disclosure")

	tests := []struct {
		name    string
		server  string
		finding bool
	}{
		{"nginx with version", "nginx/1.21.6", true},
		{"Apache with version", "Apache/2.4.49", true},
		{"generic server", "cloudflare", false},
		{"no server header", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			headers := map[string]string{}
			if tt.server != "" {
				headers["Server"] = tt.server
			}
			f := CheckPassive(&rule, "r1", resp(200, headers, ""))
			if (f != nil) != tt.finding {
				t.Errorf("server=%q: got finding=%v, want finding=%v", tt.server, f != nil, tt.finding)
			}
		})
	}
}

// --------------------------------------------------------------------------
// Body pattern checks
// --------------------------------------------------------------------------

func TestCheckPassive_SQLiErrorPattern(t *testing.T) {
	rule := passiveRule("sqli-error-pattern")

	f := CheckPassive(&rule, "r1", resp(500, map[string]string{}, "Error: You have an error in your SQL syntax near..."))
	if f == nil {
		t.Fatal("expected finding for SQL error pattern")
	}

	f = CheckPassive(&rule, "r1", resp(200, map[string]string{}, `{"status":"ok"}`))
	if f != nil {
		t.Fatal("expected no finding for clean response")
	}
}

func TestCheckPassive_StackTraceDisclosure(t *testing.T) {
	rule := passiveRule("stack-trace-disclosure")

	f := CheckPassive(&rule, "r1", resp(500, map[string]string{}, "Traceback (most recent call last):\n  File \"app.py\""))
	if f == nil {
		t.Fatal("expected finding for stack trace")
	}

	f = CheckPassive(&rule, "r1", resp(500, map[string]string{}, "goroutine 1 [running]:"))
	if f == nil {
		t.Fatal("expected finding for Go panic trace")
	}
}

func TestCheckPassive_VerboseErrorMessage(t *testing.T) {
	rule := passiveRule("error-message-verbose")

	// Only triggers on 4xx/5xx.
	f := CheckPassive(&rule, "r1", resp(200, map[string]string{}, "/var/www/html/app.py"))
	if f != nil {
		t.Fatal("should not trigger on 200")
	}

	f = CheckPassive(&rule, "r1", resp(500, map[string]string{}, "Error at /var/www/html/app.py"))
	if f == nil {
		t.Fatal("expected finding for verbose error on 500")
	}
}

func TestCheckPassive_BearerTokenInBody(t *testing.T) {
	rule := passiveRule("bearer-token-in-body")

	jwt := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U"
	f := CheckPassive(&rule, "r1", resp(200, map[string]string{}, `{"token":"`+jwt+`"}`))
	if f == nil {
		t.Fatal("expected finding for JWT in body")
	}

	f = CheckPassive(&rule, "r1", resp(200, map[string]string{}, `{"status":"ok"}`))
	if f != nil {
		t.Fatal("expected no finding for clean body")
	}
}

func TestCheckPassive_InternalIPDisclosure(t *testing.T) {
	rule := passiveRule("internal-ip-disclosure")

	f := CheckPassive(&rule, "r1", resp(200, map[string]string{
		"X-Backend": "192.168.1.100:8080",
	}, ""))
	if f == nil {
		t.Fatal("expected finding for internal IP in header")
	}

	f = CheckPassive(&rule, "r1", resp(200, map[string]string{}, `{"server":"10.0.0.1"}`))
	if f == nil {
		t.Fatal("expected finding for internal IP in body")
	}
}

// --------------------------------------------------------------------------
// Cookie checks
// --------------------------------------------------------------------------

func TestCheckPassive_InsecureCookieFlags(t *testing.T) {
	rule := passiveRule("insecure-cookie-flags")

	// Missing all flags.
	f := CheckPassive(&rule, "r1", resp(200, map[string]string{
		"Set-Cookie": "session=abc123; Path=/",
	}, ""))
	if f == nil {
		t.Fatal("expected finding for insecure cookie")
	}
	ev := f.Evidence.(map[string]string)
	if !strings.Contains(ev["missing_flags"], "Secure") {
		t.Error("should report missing Secure flag")
	}

	// All flags present → no finding.
	f = CheckPassive(&rule, "r1", resp(200, map[string]string{
		"Set-Cookie": "session=abc123; Path=/; Secure; HttpOnly; SameSite=Lax",
	}, ""))
	if f != nil {
		t.Fatal("expected no finding when all flags present")
	}
}

// --------------------------------------------------------------------------
// URL pattern checks
// --------------------------------------------------------------------------

func TestCheckPassive_SensitiveDataInURL(t *testing.T) {
	rule := passiveRule("sensitive-data-in-url")

	r := &request.Response{
		RequestID: "r1",
		URL:       "https://api.example.com/login?password=secret123",
		Status:    200,
		Headers:   map[string]string{},
	}
	f := CheckPassive(&rule, "r1", r)
	if f == nil {
		t.Fatal("expected finding for sensitive data in URL")
	}

	r.URL = "https://api.example.com/users"
	f = CheckPassive(&rule, "r1", r)
	if f != nil {
		t.Fatal("expected no finding for clean URL")
	}
}

// --------------------------------------------------------------------------
// Rate limiting
// --------------------------------------------------------------------------

func TestCheckPassive_RateLimitMissing(t *testing.T) {
	rule := passiveRule("rate-limit-missing")

	// No rate limit headers → finding.
	f := CheckPassive(&rule, "r1", resp(200, map[string]string{}, ""))
	if f == nil {
		t.Fatal("expected finding when no rate limit headers")
	}

	// With rate limit header → no finding.
	f = CheckPassive(&rule, "r1", resp(200, map[string]string{
		"X-RateLimit-Limit": "100",
	}, ""))
	if f != nil {
		t.Fatal("expected no finding when rate limit header present")
	}
}

// --------------------------------------------------------------------------
// CSP checks
// --------------------------------------------------------------------------

func TestCheckPassive_CSPUnsafeInline(t *testing.T) {
	rule := passiveRule("csp-unsafe-inline")

	f := CheckPassive(&rule, "r1", resp(200, map[string]string{
		"Content-Security-Policy": "default-src 'self' 'unsafe-inline'",
	}, ""))
	if f == nil {
		t.Fatal("expected finding for unsafe-inline in CSP")
	}

	f = CheckPassive(&rule, "r1", resp(200, map[string]string{
		"Content-Security-Policy": "default-src 'self'",
	}, ""))
	if f != nil {
		t.Fatal("expected no finding for clean CSP")
	}
}

// --------------------------------------------------------------------------
// JSON hijacking
// --------------------------------------------------------------------------

func TestCheckPassive_JSONHijacking(t *testing.T) {
	rule := passiveRule("json-hijacking")

	f := CheckPassive(&rule, "r1", resp(200, map[string]string{
		"Content-Type": "application/json",
	}, `[{"id":1},{"id":2}]`))
	if f == nil {
		t.Fatal("expected finding for JSON array response")
	}

	f = CheckPassive(&rule, "r1", resp(200, map[string]string{
		"Content-Type": "application/json",
	}, `{"data":[1,2,3]}`))
	if f != nil {
		t.Fatal("expected no finding for JSON object response")
	}
}

// --------------------------------------------------------------------------
// CheckAllPassive
// --------------------------------------------------------------------------

func TestCheckAllPassive(t *testing.T) {
	rules := []dast.Rule{
		passiveRule("missing-csp-header"),
		passiveRule("missing-hsts-header"),
		{
			ID: "active-rule", Name: "Active", Severity: dast.SeverityHigh,
			Type: dast.RuleTypeActive, Enabled: true,
		},
		{
			ID: "disabled-passive", Name: "Disabled", Severity: dast.SeverityLow,
			Type: dast.RuleTypePassive, Enabled: false,
		},
	}

	r := resp(200, map[string]string{}, "")
	findings := CheckAllPassive(rules, "r1", r)

	// Should only process the 2 enabled passive rules.
	if len(findings) != 2 {
		t.Fatalf("want 2 findings, got %d", len(findings))
	}
	if findings[0].Rule != "missing-csp-header" {
		t.Errorf("first finding: want missing-csp-header, got %s", findings[0].Rule)
	}
	if findings[1].Rule != "missing-hsts-header" {
		t.Errorf("second finding: want missing-hsts-header, got %s", findings[1].Rule)
	}
}

// --------------------------------------------------------------------------
// Custom YAML rule matching
// --------------------------------------------------------------------------

func TestCheckPassive_CustomRuleHeaderRequired(t *testing.T) {
	rule := dast.Rule{
		ID:       "custom-check",
		Name:     "Custom Header",
		Severity: dast.SeverityMedium,
		Type:     dast.RuleTypePassive,
		Enabled:  true,
		Check: &dast.RuleCheck{
			Response: &dast.ResponseCheck{
				Headers: []dast.HeaderCheck{
					{Name: "X-Custom-Header", Required: true},
				},
			},
		},
	}

	// Missing header → finding.
	f := CheckPassive(&rule, "r1", resp(200, map[string]string{}, ""))
	if f == nil {
		t.Fatal("expected finding for missing custom header")
	}

	// Header present → no finding.
	f = CheckPassive(&rule, "r1", resp(200, map[string]string{"X-Custom-Header": "present"}, ""))
	if f != nil {
		t.Fatal("expected no finding when custom header present")
	}
}

func TestCheckPassive_CustomRuleBodyNotContains(t *testing.T) {
	rule := dast.Rule{
		ID:       "custom-body",
		Name:     "No Debug Info",
		Severity: dast.SeverityHigh,
		Type:     dast.RuleTypePassive,
		Enabled:  true,
		Check: &dast.RuleCheck{
			Response: &dast.ResponseCheck{
				Body: []dast.BodyCheck{
					{NotContains: "DEBUG_MODE"},
				},
			},
		},
	}

	// Body contains unwanted string → finding.
	f := CheckPassive(&rule, "r1", resp(200, map[string]string{}, "app started in DEBUG_MODE"))
	if f == nil {
		t.Fatal("expected finding for body containing unwanted pattern")
	}

	// Clean body → no finding.
	f = CheckPassive(&rule, "r1", resp(200, map[string]string{}, "app started"))
	if f != nil {
		t.Fatal("expected no finding for clean body")
	}
}

// --------------------------------------------------------------------------
// Edge cases
// --------------------------------------------------------------------------

func TestCheckPassive_ActiveRuleReturnsNil(t *testing.T) {
	rule := dast.Rule{
		ID: "active-rule", Type: dast.RuleTypeActive, Enabled: true,
	}
	f := CheckPassive(&rule, "r1", resp(200, map[string]string{}, ""))
	if f != nil {
		t.Fatal("active rules should always return nil from CheckPassive")
	}
}

func TestCheckPassive_DisabledRuleReturnsNil(t *testing.T) {
	rule := passiveRule("missing-csp-header")
	rule.Enabled = false
	f := CheckPassive(&rule, "r1", resp(200, map[string]string{}, ""))
	if f != nil {
		t.Fatal("disabled rules should always return nil")
	}
}

func TestCheckPassive_UnknownRuleIDReturnsNil(t *testing.T) {
	rule := passiveRule("unknown-rule-id-that-does-not-exist")
	f := CheckPassive(&rule, "r1", resp(200, map[string]string{}, ""))
	if f != nil {
		t.Fatal("unknown rule ID should return nil")
	}
}

// --------------------------------------------------------------------------
// headerLookup
// --------------------------------------------------------------------------

func TestHeaderLookup_CaseInsensitive(t *testing.T) {
	headers := map[string]string{
		"content-type": "application/json",
	}
	val, ok := headerLookup(headers, "Content-Type")
	if !ok {
		t.Fatal("expected case-insensitive match")
	}
	if val != "application/json" {
		t.Errorf("got %q, want application/json", val)
	}
}

// --------------------------------------------------------------------------
// Benchmark
// --------------------------------------------------------------------------

func BenchmarkCheckAllPassive(b *testing.B) {
	rules := []dast.Rule{
		passiveRule("missing-csp-header"),
		passiveRule("missing-hsts-header"),
		passiveRule("missing-x-frame-options"),
		passiveRule("cors-wildcard-origin"),
		passiveRule("server-version-disclosure"),
		passiveRule("sqli-error-pattern"),
		passiveRule("stack-trace-disclosure"),
		passiveRule("insecure-cookie-flags"),
		passiveRule("sensitive-data-in-url"),
		passiveRule("rate-limit-missing"),
	}
	r := resp(200, map[string]string{
		"Server":       "nginx/1.21",
		"Content-Type": "application/json",
	}, `{"status":"ok","data":[1,2,3]}`)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CheckAllPassive(rules, "bench-req", r)
	}
}
