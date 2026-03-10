package variable

import (
	"errors"
	"testing"
)

// --- Resolve tests ---

func TestResolve_Simple(t *testing.T) {
	vars := map[string]any{"name": "world"}
	result, err := Resolve("hello {{name}}", vars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "hello world" {
		t.Errorf("expected 'hello world', got %q", result)
	}
}

func TestResolve_Multiple(t *testing.T) {
	vars := map[string]any{"host": "localhost", "port": "8080"}
	result, err := Resolve("{{host}}:{{port}}", vars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "localhost:8080" {
		t.Errorf("expected 'localhost:8080', got %q", result)
	}
}

func TestResolve_Nested(t *testing.T) {
	vars := map[string]any{
		"baseUrl": "https://api.example.com",
		"version": "v2",
	}
	result, err := Resolve("{{baseUrl}}/{{version}}/users", vars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "https://api.example.com/v2/users"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestResolve_Recursive(t *testing.T) {
	vars := map[string]any{
		"url":  "{{host}}:{{port}}",
		"host": "localhost",
		"port": "3000",
	}
	result, err := Resolve("{{url}}", vars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "localhost:3000" {
		t.Errorf("expected 'localhost:3000', got %q", result)
	}
}

func TestResolve_RecursiveMultiLevel(t *testing.T) {
	vars := map[string]any{
		"endpoint": "{{baseUrl}}/api",
		"baseUrl":  "{{protocol}}://{{host}}",
		"protocol": "https",
		"host":     "api.example.com",
	}
	result, err := Resolve("{{endpoint}}", vars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "https://api.example.com/api"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestResolve_MaxDepthExceeded(t *testing.T) {
	vars := map[string]any{
		"a": "{{b}}",
		"b": "{{a}}", // circular reference
	}
	_, err := Resolve("{{a}}", vars)
	if err == nil {
		t.Fatal("expected error for circular reference, got nil")
	}
	var depthErr *ErrMaxDepthExceeded
	if !errors.As(err, &depthErr) {
		t.Errorf("expected ErrMaxDepthExceeded, got %T: %v", err, err)
	}
}

func TestResolve_CustomMaxDepth(t *testing.T) {
	vars := map[string]any{
		"a": "{{b}}",
		"b": "{{c}}",
		"c": "done",
	}
	// With depth 1, resolving "{{a}}" → "{{b}}" → should stop
	_, err := Resolve("{{a}}", vars, Options{MaxDepth: 1})
	if err == nil {
		t.Fatal("expected depth error with MaxDepth=1")
	}
}

func TestResolve_Escape(t *testing.T) {
	vars := map[string]any{"name": "world"}
	result, err := Resolve(`\{\{literal\}\} and {{name}}`, vars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "{{literal}} and world"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestResolve_StrictMode_Missing(t *testing.T) {
	vars := map[string]any{"a": "1"}
	_, err := Resolve("{{a}} and {{b}}", vars, Options{Strict: true})
	if err == nil {
		t.Fatal("expected error in strict mode for missing variable")
	}
	var notFound *ErrVariableNotFound
	if !errors.As(err, &notFound) {
		t.Errorf("expected ErrVariableNotFound, got %T: %v", err, err)
	}
	if notFound.Name != "b" {
		t.Errorf("expected variable name 'b', got %q", notFound.Name)
	}
}

func TestResolve_LenientMode_Missing(t *testing.T) {
	vars := map[string]any{"a": "1"}
	result, err := Resolve("{{a}} and {{b}}", vars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "1 and {{b}}" {
		t.Errorf("expected '1 and {{b}}', got %q", result)
	}
}

func TestResolve_NoVariables(t *testing.T) {
	result, err := Resolve("no variables here", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "no variables here" {
		t.Errorf("expected 'no variables here', got %q", result)
	}
}

func TestResolve_EmptyTemplate(t *testing.T) {
	result, err := Resolve("", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestResolve_NonStringValue(t *testing.T) {
	vars := map[string]any{"count": 42, "rate": 3.14, "ok": true}
	result, err := Resolve("count={{count}} rate={{rate}} ok={{ok}}", vars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "count=42 rate=3.14 ok=true"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestResolve_WhitespaceInVarName(t *testing.T) {
	vars := map[string]any{"name": "world"}
	result, err := Resolve("{{ name }}", vars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "world" {
		t.Errorf("expected 'world', got %q", result)
	}
}

// --- MergeScopes tests ---

func TestMergeScopes_Empty(t *testing.T) {
	result := MergeScopes()
	if len(result) != 0 {
		t.Errorf("expected empty map, got %v", result)
	}
}

func TestMergeScopes_Single(t *testing.T) {
	result := MergeScopes(map[string]any{"a": "1"})
	if result["a"] != "1" {
		t.Errorf("expected a=1, got %v", result["a"])
	}
}

func TestMergeScopes_Override(t *testing.T) {
	env := map[string]any{"host": "prod.api.com", "port": "443"}
	collection := map[string]any{"port": "8080", "version": "v1"}
	request := map[string]any{"port": "3000"}

	result := MergeScopes(env, collection, request)

	if result["host"] != "prod.api.com" {
		t.Errorf("expected host from env, got %v", result["host"])
	}
	if result["version"] != "v1" {
		t.Errorf("expected version from collection, got %v", result["version"])
	}
	if result["port"] != "3000" {
		t.Errorf("expected port from request (last wins), got %v", result["port"])
	}
}

// --- ResolveStruct tests ---

func TestResolveStruct_Simple(t *testing.T) {
	type Config struct {
		URL    string
		Header string
	}
	cfg := Config{URL: "{{baseUrl}}/api", Header: "Bearer {{token}}"}
	vars := map[string]any{"baseUrl": "https://api.example.com", "token": "abc123"}

	if err := ResolveStruct(&cfg, vars); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.URL != "https://api.example.com/api" {
		t.Errorf("URL not resolved: %q", cfg.URL)
	}
	if cfg.Header != "Bearer abc123" {
		t.Errorf("Header not resolved: %q", cfg.Header)
	}
}

func TestResolveStruct_Nested(t *testing.T) {
	type Inner struct {
		Value string
	}
	type Outer struct {
		Name  string
		Inner Inner
	}

	obj := Outer{Name: "{{name}}", Inner: Inner{Value: "{{val}}"}}
	vars := map[string]any{"name": "test", "val": "resolved"}

	if err := ResolveStruct(&obj, vars); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if obj.Name != "test" {
		t.Errorf("Name not resolved: %q", obj.Name)
	}
	if obj.Inner.Value != "resolved" {
		t.Errorf("Inner.Value not resolved: %q", obj.Inner.Value)
	}
}

func TestResolveStruct_Slice(t *testing.T) {
	type Config struct {
		Items []string
	}
	cfg := Config{Items: []string{"{{a}}", "{{b}}"}}
	vars := map[string]any{"a": "first", "b": "second"}

	if err := ResolveStruct(&cfg, vars); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Items[0] != "first" || cfg.Items[1] != "second" {
		t.Errorf("Items not resolved: %v", cfg.Items)
	}
}

func TestResolveStruct_Map(t *testing.T) {
	type Config struct {
		Headers map[string]string
	}
	cfg := Config{Headers: map[string]string{
		"Authorization": "Bearer {{token}}",
		"X-Custom":      "{{custom}}",
	}}
	vars := map[string]any{"token": "tok123", "custom": "val"}

	if err := ResolveStruct(&cfg, vars); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Headers["Authorization"] != "Bearer tok123" {
		t.Errorf("Authorization not resolved: %q", cfg.Headers["Authorization"])
	}
	if cfg.Headers["X-Custom"] != "val" {
		t.Errorf("X-Custom not resolved: %q", cfg.Headers["X-Custom"])
	}
}

func TestResolveStruct_Nil(t *testing.T) {
	if err := ResolveStruct(nil, nil); err != nil {
		t.Fatalf("unexpected error on nil: %v", err)
	}
}

// --- Parser edge cases ---

func TestParse_UnclosedBrace(t *testing.T) {
	result, err := Resolve("hello {{name", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "hello {{name" {
		t.Errorf("expected literal preservation, got %q", result)
	}
}

func TestParse_EmptyBraces(t *testing.T) {
	result, err := Resolve("hello {{}}", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "hello {{}}" {
		t.Errorf("expected '{{}}' preserved, got %q", result)
	}
}
