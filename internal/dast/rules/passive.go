package rules

import (
	"net/http"
	"regexp"
	"strings"

	"github.com/khanhnguyen/promptman/internal/dast"
	"github.com/khanhnguyen/promptman/internal/request"
)

// CheckAllPassive evaluates every passive rule against the given response
// and returns any findings produced. Active rules are silently skipped.
func CheckAllPassive(rules []dast.Rule, reqID string, resp *request.Response) []dast.Finding {
	var findings []dast.Finding
	for i := range rules {
		if rules[i].Type != dast.RuleTypePassive || !rules[i].Enabled {
			continue
		}
		if f := CheckPassive(&rules[i], reqID, resp); f != nil {
			findings = append(findings, *f)
		}
	}
	return findings
}

// CheckPassive evaluates a single passive rule against a response.
// It returns a Finding if the rule triggers, or nil if everything is clean.
// Active rules return nil immediately.
func CheckPassive(rule *dast.Rule, reqID string, resp *request.Response) *dast.Finding {
	if rule.Type != dast.RuleTypePassive || !rule.Enabled {
		return nil
	}

	// If the rule has a custom Check definition (from YAML), use generic matching.
	if rule.Check != nil {
		return checkCustomRule(rule, reqID, resp)
	}

	// Otherwise, use compiled Go logic keyed by rule ID.
	fn, ok := builtinChecks[rule.ID]
	if !ok {
		return nil
	}
	return fn(rule, reqID, resp)
}

// --------------------------------------------------------------------------
// Built-in passive rule check functions
// --------------------------------------------------------------------------

// checkFunc is the signature for a compiled built-in passive rule check.
type checkFunc func(rule *dast.Rule, reqID string, resp *request.Response) *dast.Finding

// builtinChecks maps rule IDs to their compiled Go check functions.
var builtinChecks = map[string]checkFunc{
	// --- Missing security headers ---
	"missing-csp-header":             checkMissingHeader("Content-Security-Policy"),
	"missing-hsts-header":            checkMissingHeader("Strict-Transport-Security"),
	"missing-x-frame-options":        checkMissingHeader("X-Frame-Options"),
	"missing-x-content-type-options": checkMissingHeader("X-Content-Type-Options"),
	"missing-referrer-policy":        checkMissingHeader("Referrer-Policy"),
	"missing-permissions-policy":     checkMissingHeader("Permissions-Policy"),
	"missing-cache-control":          checkMissingCacheControl,

	// --- CORS ---
	"cors-wildcard-origin":      checkCORSWildcard,
	"cors-credentials-wildcard": checkCORSCredentialsWildcard,

	// --- Information disclosure ---
	"server-version-disclosure": checkServerVersionDisclosure,
	"x-powered-by-disclosure":   checkHeaderPresent("X-Powered-By"),

	// --- Body pattern checks ---
	"sqli-error-pattern":       checkSQLiErrorPattern,
	"stack-trace-disclosure":   checkStackTraceDisclosure,
	"error-message-verbose":    checkVerboseErrorMessage,
	"debug-endpoint-exposed":   checkDebugEndpoint,
	"bearer-token-in-body":     checkBearerTokenInBody,
	"internal-ip-disclosure":   checkInternalIPDisclosure,
	"json-hijacking":           checkJSONHijacking,
	"insecure-deserialization": checkInsecureDeserialization,
	"csp-unsafe-inline":        checkCSPUnsafeInline,
	"xss-error-page":           checkXSSErrorPage,

	// --- Cookie security ---
	"insecure-cookie-flags": checkInsecureCookieFlags,

	// --- URL patterns ---
	"sensitive-data-in-url": checkSensitiveDataInURL,

	// --- Rate limiting ---
	"rate-limit-missing": checkRateLimitMissing,
}

// --------------------------------------------------------------------------
// Header check helpers
// --------------------------------------------------------------------------

// checkMissingHeader returns a checkFunc that triggers when the named header
// is absent from the response.
func checkMissingHeader(header string) checkFunc {
	return func(rule *dast.Rule, reqID string, resp *request.Response) *dast.Finding {
		if _, ok := headerLookup(resp.Headers, header); ok {
			return nil
		}
		return makeFinding(rule, reqID, map[string]string{
			"missing_header": header,
		})
	}
}

// checkHeaderPresent returns a checkFunc that triggers when the named header
// IS present (information disclosure).
func checkHeaderPresent(header string) checkFunc {
	return func(rule *dast.Rule, reqID string, resp *request.Response) *dast.Finding {
		val, ok := headerLookup(resp.Headers, header)
		if !ok {
			return nil
		}
		return makeFinding(rule, reqID, map[string]string{
			"header": header,
			"value":  val,
		})
	}
}

// --------------------------------------------------------------------------
// CORS checks
// --------------------------------------------------------------------------

func checkCORSWildcard(rule *dast.Rule, reqID string, resp *request.Response) *dast.Finding {
	val, ok := headerLookup(resp.Headers, "Access-Control-Allow-Origin")
	if !ok || val != "*" {
		return nil
	}
	return makeFinding(rule, reqID, map[string]string{
		"header": "Access-Control-Allow-Origin",
		"value":  val,
	})
}

func checkCORSCredentialsWildcard(rule *dast.Rule, reqID string, resp *request.Response) *dast.Finding {
	origin, okO := headerLookup(resp.Headers, "Access-Control-Allow-Origin")
	creds, okC := headerLookup(resp.Headers, "Access-Control-Allow-Credentials")
	if !okO || !okC {
		return nil
	}
	if origin == "*" && strings.EqualFold(creds, "true") {
		return makeFinding(rule, reqID, map[string]string{
			"Access-Control-Allow-Origin":      origin,
			"Access-Control-Allow-Credentials": creds,
		})
	}
	return nil
}

// --------------------------------------------------------------------------
// Server / cache checks
// --------------------------------------------------------------------------

// serverVersionRe detects version strings like "Apache/2.4.49", "nginx/1.21.6".
var serverVersionRe = regexp.MustCompile(`(?i)(apache|nginx|iis|lighttpd|tomcat|jetty|gunicorn|express|kestrel)[/ ]\d`)

func checkServerVersionDisclosure(rule *dast.Rule, reqID string, resp *request.Response) *dast.Finding {
	val, ok := headerLookup(resp.Headers, "Server")
	if !ok {
		return nil
	}
	if serverVersionRe.MatchString(val) {
		return makeFinding(rule, reqID, map[string]string{
			"header": "Server",
			"value":  val,
		})
	}
	return nil
}

func checkMissingCacheControl(rule *dast.Rule, reqID string, resp *request.Response) *dast.Finding {
	// Only flag on responses that likely contain sensitive data (2xx with a body).
	if resp.Status < 200 || resp.Status >= 300 || resp.Body == "" {
		return nil
	}
	val, ok := headerLookup(resp.Headers, "Cache-Control")
	if ok && (strings.Contains(strings.ToLower(val), "no-store") || strings.Contains(strings.ToLower(val), "private")) {
		return nil
	}
	return makeFinding(rule, reqID, map[string]string{
		"missing_directive": "no-store or private",
		"current_value":     val,
	})
}

// --------------------------------------------------------------------------
// Body pattern checks
// --------------------------------------------------------------------------

var sqliPatterns = []string{
	"SQL syntax",
	"mysql_fetch",
	"ORA-",
	"PG::SyntaxError",
	"SQLite3::",
	"SQLSTATE[",
	"Unclosed quotation mark",
	"quoted string not properly terminated",
	"Microsoft OLE DB Provider",
	"ODBC SQL Server Driver",
}

func checkSQLiErrorPattern(rule *dast.Rule, reqID string, resp *request.Response) *dast.Finding {
	body := resp.Body
	for _, pattern := range sqliPatterns {
		if strings.Contains(body, pattern) {
			return makeFinding(rule, reqID, map[string]string{
				"matched_pattern": pattern,
			})
		}
	}
	return nil
}

var stackTracePatterns = []string{
	"at java.",
	"at sun.",
	"Traceback (most recent call last)",
	"File \"",
	"goroutine ",
	"panic:",
	"System.NullReferenceException",
	"System.Exception",
	"Microsoft.AspNetCore",
	"node_modules/",
	"  at Object.",
	"  at Module.",
}

func checkStackTraceDisclosure(rule *dast.Rule, reqID string, resp *request.Response) *dast.Finding {
	body := resp.Body
	for _, pattern := range stackTracePatterns {
		if strings.Contains(body, pattern) {
			return makeFinding(rule, reqID, map[string]string{
				"matched_pattern": pattern,
			})
		}
	}
	return nil
}

var verboseErrorPatterns = []string{
	"SQLSTATE",
	"pg_query",
	"mysql_",
	"/var/www/",
	"/home/",
	"C:\\Users\\",
	"stack trace:",
	"Exception in thread",
	"at line ",
	"on line ",
}

func checkVerboseErrorMessage(rule *dast.Rule, reqID string, resp *request.Response) *dast.Finding {
	// Only check error responses (4xx/5xx).
	if resp.Status < 400 {
		return nil
	}
	body := resp.Body
	for _, pattern := range verboseErrorPatterns {
		if strings.Contains(body, pattern) {
			return makeFinding(rule, reqID, map[string]string{
				"matched_pattern": pattern,
				"status":          http.StatusText(resp.Status),
			})
		}
	}
	return nil
}

var debugEndpointPatterns = []string{
	"phpinfo()",
	"<title>phpinfo()</title>",
	"Xdebug",
	"Django Debug Toolbar",
	"Debug mode is on",
	"DJANGO_SETTINGS_MODULE",
	"\"debug\":true",
	"\"debug\": true",
	"Werkzeug Debugger",
	"Express Debug",
}

func checkDebugEndpoint(rule *dast.Rule, reqID string, resp *request.Response) *dast.Finding {
	body := resp.Body
	for _, pattern := range debugEndpointPatterns {
		if strings.Contains(body, pattern) {
			return makeFinding(rule, reqID, map[string]string{
				"matched_pattern": pattern,
			})
		}
	}
	return nil
}

var bearerTokenRe = regexp.MustCompile(`(?i)(eyJ[A-Za-z0-9_-]+\.eyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+|Bearer\s+[A-Za-z0-9_-]{20,})`)

func checkBearerTokenInBody(rule *dast.Rule, reqID string, resp *request.Response) *dast.Finding {
	if bearerTokenRe.MatchString(resp.Body) {
		return makeFinding(rule, reqID, map[string]string{
			"pattern": "JWT or Bearer token detected in response body",
		})
	}
	return nil
}

var internalIPRe = regexp.MustCompile(`\b(10\.\d{1,3}\.\d{1,3}\.\d{1,3}|172\.(1[6-9]|2[0-9]|3[01])\.\d{1,3}\.\d{1,3}|192\.168\.\d{1,3}\.\d{1,3})\b`)

func checkInternalIPDisclosure(rule *dast.Rule, reqID string, resp *request.Response) *dast.Finding {
	// Check both headers and body.
	for _, val := range resp.Headers {
		if match := internalIPRe.FindString(val); match != "" {
			return makeFinding(rule, reqID, map[string]string{
				"location": "header",
				"ip":       match,
			})
		}
	}
	if match := internalIPRe.FindString(resp.Body); match != "" {
		return makeFinding(rule, reqID, map[string]string{
			"location": "body",
			"ip":       match,
		})
	}
	return nil
}

func checkJSONHijacking(rule *dast.Rule, reqID string, resp *request.Response) *dast.Finding {
	ct, ok := headerLookup(resp.Headers, "Content-Type")
	if !ok || !strings.Contains(strings.ToLower(ct), "json") {
		return nil
	}
	body := strings.TrimSpace(resp.Body)
	if strings.HasPrefix(body, "[") {
		return makeFinding(rule, reqID, map[string]string{
			"issue": "JSON response starts with array — vulnerable to JSON hijacking",
		})
	}
	return nil
}

var deserializationPatterns = []string{
	"java.io.ObjectInputStream",
	"rO0AB",
	"com.sun.org.apache",
	"pickle.loads",
	"yaml.load(",
	"Marshal.load",
	"unserialize(",
	"__PHP_Incomplete_Class",
}

func checkInsecureDeserialization(rule *dast.Rule, reqID string, resp *request.Response) *dast.Finding {
	body := resp.Body
	for _, pattern := range deserializationPatterns {
		if strings.Contains(body, pattern) {
			return makeFinding(rule, reqID, map[string]string{
				"matched_pattern": pattern,
			})
		}
	}
	return nil
}

func checkCSPUnsafeInline(rule *dast.Rule, reqID string, resp *request.Response) *dast.Finding {
	val, ok := headerLookup(resp.Headers, "Content-Security-Policy")
	if !ok {
		return nil
	}
	if strings.Contains(val, "'unsafe-inline'") {
		return makeFinding(rule, reqID, map[string]string{
			"header": "Content-Security-Policy",
			"issue":  "contains 'unsafe-inline'",
			"value":  val,
		})
	}
	return nil
}

func checkXSSErrorPage(rule *dast.Rule, reqID string, resp *request.Response) *dast.Finding {
	// Only check error responses.
	if resp.Status < 400 {
		return nil
	}
	// Check if the URL path appears reflected in the error body.
	if resp.URL == "" || resp.Body == "" {
		return nil
	}
	// Extract the path portion for reflection check.
	path := resp.URL
	if idx := strings.Index(path, "://"); idx >= 0 {
		path = path[idx+3:]
		if idx2 := strings.Index(path, "/"); idx2 >= 0 {
			path = path[idx2:]
		}
	}
	if path != "" && path != "/" && strings.Contains(resp.Body, path) {
		return makeFinding(rule, reqID, map[string]string{
			"issue":          "URL path reflected in error response",
			"reflected_path": path,
		})
	}
	return nil
}

// --------------------------------------------------------------------------
// Cookie security
// --------------------------------------------------------------------------

func checkInsecureCookieFlags(rule *dast.Rule, reqID string, resp *request.Response) *dast.Finding {
	val, ok := headerLookup(resp.Headers, "Set-Cookie")
	if !ok {
		return nil
	}
	lower := strings.ToLower(val)
	var missing []string
	if !strings.Contains(lower, "secure") {
		missing = append(missing, "Secure")
	}
	if !strings.Contains(lower, "httponly") {
		missing = append(missing, "HttpOnly")
	}
	if !strings.Contains(lower, "samesite") {
		missing = append(missing, "SameSite")
	}
	if len(missing) == 0 {
		return nil
	}
	return makeFinding(rule, reqID, map[string]string{
		"cookie":        val,
		"missing_flags": strings.Join(missing, ", "),
	})
}

// --------------------------------------------------------------------------
// URL pattern checks
// --------------------------------------------------------------------------

var sensitiveURLRe = regexp.MustCompile(`(?i)(password|passwd|secret|token|api_key|apikey|auth|session_id|access_token)=`)

func checkSensitiveDataInURL(rule *dast.Rule, reqID string, resp *request.Response) *dast.Finding {
	url := resp.URL
	if match := sensitiveURLRe.FindString(url); match != "" {
		return makeFinding(rule, reqID, map[string]string{
			"url":             url,
			"matched_pattern": match,
		})
	}
	return nil
}

// --------------------------------------------------------------------------
// Rate limiting
// --------------------------------------------------------------------------

var rateLimitHeaders = []string{
	"X-RateLimit-Limit",
	"RateLimit-Limit",
	"X-Rate-Limit-Limit",
	"Retry-After",
}

func checkRateLimitMissing(rule *dast.Rule, reqID string, resp *request.Response) *dast.Finding {
	for _, h := range rateLimitHeaders {
		if _, ok := headerLookup(resp.Headers, h); ok {
			return nil
		}
	}
	return makeFinding(rule, reqID, map[string]string{
		"issue": "no rate-limit headers found",
	})
}

// --------------------------------------------------------------------------
// Custom YAML rule matching
// --------------------------------------------------------------------------

func checkCustomRule(rule *dast.Rule, reqID string, resp *request.Response) *dast.Finding {
	if rule.Check == nil || rule.Check.Response == nil {
		return nil
	}
	rc := rule.Check.Response

	// Check header criteria.
	for _, hc := range rc.Headers {
		val, ok := headerLookup(resp.Headers, hc.Name)
		if hc.Required && !ok {
			return makeFinding(rule, reqID, map[string]string{
				"missing_header": hc.Name,
			})
		}
		if hc.Contains != "" && ok && !strings.Contains(val, hc.Contains) {
			return makeFinding(rule, reqID, map[string]string{
				"header":          hc.Name,
				"expected_substr": hc.Contains,
				"actual_value":    val,
			})
		}
	}

	// Check body criteria.
	for _, bc := range rc.Body {
		if bc.Contains != "" && !strings.Contains(resp.Body, bc.Contains) {
			return makeFinding(rule, reqID, map[string]string{
				"body_expected": bc.Contains,
			})
		}
		if bc.NotContains != "" && strings.Contains(resp.Body, bc.NotContains) {
			return makeFinding(rule, reqID, map[string]string{
				"body_unwanted": bc.NotContains,
			})
		}
	}

	return nil
}

// --------------------------------------------------------------------------
// Helpers
// --------------------------------------------------------------------------

// headerLookup performs case-insensitive header lookup in a flat string map.
func headerLookup(headers map[string]string, name string) (string, bool) {
	// Try exact match first (fast path).
	if val, ok := headers[name]; ok {
		return val, true
	}
	// Fall back to case-insensitive comparison.
	lower := strings.ToLower(name)
	for k, v := range headers {
		if strings.ToLower(k) == lower {
			return v, true
		}
	}
	return "", false
}

// makeFinding constructs a Finding from a rule and evidence map.
func makeFinding(rule *dast.Rule, reqID string, evidence map[string]string) *dast.Finding {
	return &dast.Finding{
		Rule:        rule.ID,
		Severity:    string(rule.Severity),
		Request:     reqID,
		Description: rule.Description,
		Evidence:    evidence,
		Remediation: rule.Remediation,
	}
}
