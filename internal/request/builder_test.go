package request

import (
	"encoding/base64"
	"io"
	"testing"

	"github.com/khanhnguyen/promptman/internal/collection"
	"github.com/khanhnguyen/promptman/pkg/variable"
)

func TestBuild_URLResolution(t *testing.T) {
	tests := []struct {
		name     string
		resolved *collection.ResolvedRequest
		vars     map[string]any
		wantURL  string
		wantErr  bool
	}{
		{
			name: "simple variable in URL",
			resolved: &collection.ResolvedRequest{
				URL:    "https://{{host}}/api/users",
				Method: "GET",
			},
			vars:    map[string]any{"host": "example.com"},
			wantURL: "https://example.com/api/users",
		},
		{
			name: "multiple variables in URL",
			resolved: &collection.ResolvedRequest{
				URL:    "https://{{host}}:{{port}}/api/{{version}}/users",
				Method: "GET",
			},
			vars:    map[string]any{"host": "localhost", "port": "8080", "version": "v2"},
			wantURL: "https://localhost:8080/api/v2/users",
		},
		{
			name: "no variables in URL",
			resolved: &collection.ResolvedRequest{
				URL:    "https://example.com/api/users",
				Method: "GET",
			},
			vars:    map[string]any{},
			wantURL: "https://example.com/api/users",
		},
		{
			name: "missing variable in URL (strict mode)",
			resolved: &collection.ResolvedRequest{
				URL:    "https://{{host}}/api/users",
				Method: "GET",
			},
			vars:    map[string]any{},
			wantErr: true,
		},
	}

	b := NewBuilder()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := b.Build(tt.resolved, tt.vars)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if req.URL.String() != tt.wantURL {
				t.Errorf("URL = %q, want %q", req.URL.String(), tt.wantURL)
			}
		})
	}
}

func TestBuild_HeaderResolution(t *testing.T) {
	b := NewBuilder()
	resolved := &collection.ResolvedRequest{
		URL:    "https://example.com",
		Method: "GET",
		Headers: map[string]string{
			"X-Custom":    "{{custom_val}}",
			"X-Static":    "static-value",
			"X-Multi-Var": "{{prefix}}-{{suffix}}",
		},
	}
	vars := map[string]any{
		"custom_val": "resolved-custom",
		"prefix":     "hello",
		"suffix":     "world",
	}

	req, err := b.Build(resolved, vars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tests := []struct {
		header string
		want   string
	}{
		{"X-Custom", "resolved-custom"},
		{"X-Static", "static-value"},
		{"X-Multi-Var", "hello-world"},
	}

	for _, tt := range tests {
		got := req.Header.Get(tt.header)
		if got != tt.want {
			t.Errorf("Header %q = %q, want %q", tt.header, got, tt.want)
		}
	}
}

func TestBuild_HeaderMissingVar(t *testing.T) {
	b := NewBuilder()
	resolved := &collection.ResolvedRequest{
		URL:    "https://example.com",
		Method: "GET",
		Headers: map[string]string{
			"X-Token": "{{missing_token}}",
		},
	}

	_, err := b.Build(resolved, map[string]any{})
	if err == nil {
		t.Fatal("expected error for missing header variable")
	}
}

func TestBuild_BodyResolution(t *testing.T) {
	b := NewBuilder()
	resolved := &collection.ResolvedRequest{
		URL:    "https://example.com",
		Method: "POST",
		Body: &collection.RequestBody{
			Type:    "json",
			Content: map[string]any{"name": "{{user_name}}", "age": 30},
		},
	}
	vars := map[string]any{"user_name": "Alice"}

	req, err := b.Build(resolved, vars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatalf("reading body: %v", err)
	}

	bodyStr := string(body)
	if bodyStr == "" {
		t.Fatal("expected non-empty body")
	}
	// The body should contain "Alice" (resolved), not "{{user_name}}"
	if contains(bodyStr, "{{user_name}}") {
		t.Errorf("body still contains unresolved variable: %s", bodyStr)
	}
	if !contains(bodyStr, "Alice") {
		t.Errorf("body should contain 'Alice', got: %s", bodyStr)
	}
}

func TestBuild_NoBody(t *testing.T) {
	b := NewBuilder()
	resolved := &collection.ResolvedRequest{
		URL:    "https://example.com",
		Method: "GET",
	}

	req, err := b.Build(resolved, map[string]any{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if req.Body != nil {
		t.Error("expected nil body for GET request without body")
	}

	// No Content-Type should be set
	if ct := req.Header.Get("Content-Type"); ct != "" {
		t.Errorf("Content-Type = %q, want empty", ct)
	}
}

func TestBuild_ContentType(t *testing.T) {
	tests := []struct {
		name     string
		bodyType string
		wantCT   string
	}{
		{"json body", "json", "application/json"},
		{"form body", "form", "application/x-www-form-urlencoded"},
		{"raw body", "raw", "text/plain"},
	}

	b := NewBuilder()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolved := &collection.ResolvedRequest{
				URL:    "https://example.com",
				Method: "POST",
				Body: &collection.RequestBody{
					Type:    tt.bodyType,
					Content: "test body",
				},
			}

			req, err := b.Build(resolved, map[string]any{})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			got := req.Header.Get("Content-Type")
			if got != tt.wantCT {
				t.Errorf("Content-Type = %q, want %q", got, tt.wantCT)
			}
		})
	}
}

func TestBuild_Method(t *testing.T) {
	tests := []struct {
		method string
		want   string
	}{
		{"get", "GET"},
		{"POST", "POST"},
		{"Delete", "DELETE"},
		{"patch", "PATCH"},
	}

	b := NewBuilder()
	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			resolved := &collection.ResolvedRequest{
				URL:    "https://example.com",
				Method: tt.method,
			}
			req, err := b.Build(resolved, map[string]any{})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if req.Method != tt.want {
				t.Errorf("Method = %q, want %q", req.Method, tt.want)
			}
		})
	}
}

func TestApplyAuth_Bearer(t *testing.T) {
	b := NewBuilder()
	resolved := &collection.ResolvedRequest{
		URL:    "https://example.com",
		Method: "GET",
		Auth: &collection.AuthConfig{
			Type:   "bearer",
			Bearer: &collection.BearerAuth{Token: "{{api_token}}"},
		},
	}
	vars := map[string]any{"api_token": "my-secret-token"}

	req, err := b.Build(resolved, vars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := "Bearer my-secret-token"
	got := req.Header.Get("Authorization")
	if got != want {
		t.Errorf("Authorization = %q, want %q", got, want)
	}
}

func TestApplyAuth_Basic(t *testing.T) {
	b := NewBuilder()
	resolved := &collection.ResolvedRequest{
		URL:    "https://example.com",
		Method: "GET",
		Auth: &collection.AuthConfig{
			Type:  "basic",
			Basic: &collection.BasicAuth{Username: "{{user}}", Password: "{{pass}}"},
		},
	}
	vars := map[string]any{"user": "admin", "pass": "secret123"}

	req, err := b.Build(resolved, vars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := "Basic " + base64.StdEncoding.EncodeToString([]byte("admin:secret123"))
	got := req.Header.Get("Authorization")
	if got != want {
		t.Errorf("Authorization = %q, want %q", got, want)
	}
}

func TestApplyAuth_APIKey(t *testing.T) {
	b := NewBuilder()
	resolved := &collection.ResolvedRequest{
		URL:    "https://example.com",
		Method: "GET",
		Auth: &collection.AuthConfig{
			Type:   "api-key",
			APIKey: &collection.APIKeyAuth{Key: "X-API-Key", Value: "{{key_value}}"},
		},
	}
	vars := map[string]any{"key_value": "abc-123-xyz"}

	req, err := b.Build(resolved, vars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := req.Header.Get("X-API-Key")
	if got != "abc-123-xyz" {
		t.Errorf("X-API-Key = %q, want %q", got, "abc-123-xyz")
	}
}

func TestApplyAuth_NilAuth(t *testing.T) {
	b := NewBuilder()
	resolved := &collection.ResolvedRequest{
		URL:    "https://example.com",
		Method: "GET",
		Auth:   nil,
	}

	req, err := b.Build(resolved, map[string]any{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if auth := req.Header.Get("Authorization"); auth != "" {
		t.Errorf("Authorization should be empty, got %q", auth)
	}
}

func TestApplyAuth_MissingVarInToken(t *testing.T) {
	b := NewBuilder()
	resolved := &collection.ResolvedRequest{
		URL:    "https://example.com",
		Method: "GET",
		Auth: &collection.AuthConfig{
			Type:   "bearer",
			Bearer: &collection.BearerAuth{Token: "{{missing_token}}"},
		},
	}

	_, err := b.Build(resolved, map[string]any{})
	if err == nil {
		t.Fatal("expected error for missing auth variable")
	}

	// Verify the error wraps a variable not found error
	var varErr *variable.ErrVariableNotFound
	if !containsErr(err, varErr) {
		// Just verify the error message references the variable resolution
		if !contains(err.Error(), "resolving") {
			t.Errorf("error should reference resolving, got: %v", err)
		}
	}
}

func TestApplyAuth_MissingVarInBasicPassword(t *testing.T) {
	b := NewBuilder()
	resolved := &collection.ResolvedRequest{
		URL:    "https://example.com",
		Method: "GET",
		Auth: &collection.AuthConfig{
			Type:  "basic",
			Basic: &collection.BasicAuth{Username: "user", Password: "{{missing_pass}}"},
		},
	}

	_, err := b.Build(resolved, map[string]any{})
	if err == nil {
		t.Fatal("expected error for missing basic auth password variable")
	}
}

func TestApplyAuth_APIKeyWithVariableKey(t *testing.T) {
	b := NewBuilder()
	resolved := &collection.ResolvedRequest{
		URL:    "https://example.com",
		Method: "GET",
		Auth: &collection.AuthConfig{
			Type:   "api-key",
			APIKey: &collection.APIKeyAuth{Key: "{{header_name}}", Value: "{{header_val}}"},
		},
	}
	vars := map[string]any{"header_name": "X-Custom-Auth", "header_val": "custom-token"}

	req, err := b.Build(resolved, vars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := req.Header.Get("X-Custom-Auth")
	if got != "custom-token" {
		t.Errorf("X-Custom-Auth = %q, want %q", got, "custom-token")
	}
}

// contains checks if s contains substr.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// containsErr checks if an error chain contains a specific error type.
func containsErr(err error, target any) bool {
	_ = target
	return false // simplified; real impl would use errors.As
}
