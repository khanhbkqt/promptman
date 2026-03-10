package environment

import (
	"testing"
)

// --- Service.Resolve ---

func TestService_Resolve_SimpleVar(t *testing.T) {
	svc, _ := newTestService(t)

	if _, err := svc.Create(&CreateEnvInput{
		Name:      "dev",
		Variables: map[string]any{"host": "localhost", "port": 3000},
	}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := svc.SetActive("dev"); err != nil {
		t.Fatalf("SetActive: %v", err)
	}

	got, err := svc.Resolve("{{host}}:{{port}}")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got != "localhost:3000" {
		t.Errorf("Resolve = %q, want %q", got, "localhost:3000")
	}
}

func TestService_Resolve_NoActiveEnv(t *testing.T) {
	svc, _ := newTestService(t)

	_, err := svc.Resolve("{{host}}")
	if err == nil {
		t.Fatal("expected error")
	}
	if !IsDomainError(err, "ENV_NOT_SET") {
		t.Errorf("expected ENV_NOT_SET, got: %v", err)
	}
}

func TestService_Resolve_NestedVars(t *testing.T) {
	svc, _ := newTestService(t)

	if _, err := svc.Create(&CreateEnvInput{
		Name: "dev",
		Variables: map[string]any{
			"host": "localhost",
			"port": 3000,
			"url":  "{{host}}:{{port}}",
		},
	}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := svc.SetActive("dev"); err != nil {
		t.Fatalf("SetActive: %v", err)
	}

	got, err := svc.Resolve("{{url}}/api")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got != "localhost:3000/api" {
		t.Errorf("Resolve = %q, want %q", got, "localhost:3000/api")
	}
}

func TestService_Resolve_SecretAsVariable(t *testing.T) {
	t.Setenv("TEST_API_KEY", "sk-test-12345")
	svc, _ := newTestService(t)

	if _, err := svc.Create(&CreateEnvInput{
		Name:      "dev",
		Variables: map[string]any{"host": "https://api.example.com"},
		Secrets:   map[string]string{"apiKey": "$ENV{TEST_API_KEY}"},
	}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := svc.SetActive("dev"); err != nil {
		t.Fatalf("SetActive: %v", err)
	}

	// Secret values should be available in the variable map for resolution.
	got, err := svc.Resolve("{{host}}/auth?key={{apiKey}}")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got != "https://api.example.com/auth?key=sk-test-12345" {
		t.Errorf("Resolve = %q, want %q", got, "https://api.example.com/auth?key=sk-test-12345")
	}
}

func TestService_Resolve_MissingVarLenient(t *testing.T) {
	svc, _ := newTestService(t)

	if _, err := svc.Create(&CreateEnvInput{
		Name:      "dev",
		Variables: map[string]any{"host": "localhost"},
	}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := svc.SetActive("dev"); err != nil {
		t.Fatalf("SetActive: %v", err)
	}

	// Missing vars are preserved in lenient mode (default).
	got, err := svc.Resolve("{{host}}:{{missing}}")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got != "localhost:{{missing}}" {
		t.Errorf("Resolve = %q, want %q", got, "localhost:{{missing}}")
	}
}

func TestService_Resolve_ExtraScopeOverrides(t *testing.T) {
	svc, _ := newTestService(t)

	if _, err := svc.Create(&CreateEnvInput{
		Name:      "dev",
		Variables: map[string]any{"host": "env-host", "port": 3000},
	}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := svc.SetActive("dev"); err != nil {
		t.Fatalf("SetActive: %v", err)
	}

	// Extra scope should override env variable.
	extra := map[string]any{"host": "override-host"}
	got, err := svc.Resolve("{{host}}:{{port}}", extra)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got != "override-host:3000" {
		t.Errorf("Resolve = %q, want %q", got, "override-host:3000")
	}
}

// --- Service.ResolveWith ---

func TestService_ResolveWith_SpecificEnv(t *testing.T) {
	svc, _ := newTestService(t)

	// Create two envs.
	if _, err := svc.Create(&CreateEnvInput{
		Name:      "dev",
		Variables: map[string]any{"host": "localhost"},
	}); err != nil {
		t.Fatalf("Create dev: %v", err)
	}
	if _, err := svc.Create(&CreateEnvInput{
		Name:      "prod",
		Variables: map[string]any{"host": "prod.example.com"},
	}); err != nil {
		t.Fatalf("Create prod: %v", err)
	}

	// Set active to dev, but resolve with prod.
	if err := svc.SetActive("dev"); err != nil {
		t.Fatalf("SetActive: %v", err)
	}

	got, err := svc.ResolveWith("{{host}}", "prod")
	if err != nil {
		t.Fatalf("ResolveWith: %v", err)
	}
	if got != "prod.example.com" {
		t.Errorf("ResolveWith = %q, want %q", got, "prod.example.com")
	}
}

func TestService_ResolveWith_NonexistentEnv(t *testing.T) {
	svc, _ := newTestService(t)

	_, err := svc.ResolveWith("{{host}}", "nonexistent")
	if err == nil {
		t.Fatal("expected error")
	}
	if !IsDomainError(err, "ENV_NOT_FOUND") {
		t.Errorf("expected ENV_NOT_FOUND, got: %v", err)
	}
}

func TestService_ResolveWith_MultipleScopePriority(t *testing.T) {
	t.Setenv("TEST_DB_PASS", "secret-pass")
	svc, _ := newTestService(t)

	if _, err := svc.Create(&CreateEnvInput{
		Name:      "dev",
		Variables: map[string]any{"host": "env-host", "port": 3000},
		Secrets:   map[string]string{"dbPass": "$ENV{TEST_DB_PASS}"},
	}); err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Scoping: collectionDefaults (mid) → requestOverrides (highest)
	collDefaults := map[string]any{"host": "coll-host", "timeout": 30}
	requestVars := map[string]any{"host": "req-host"}

	got, err := svc.ResolveWith("{{host}}:{{port}} pass={{dbPass}} timeout={{timeout}}", "dev", collDefaults, requestVars)
	if err != nil {
		t.Fatalf("ResolveWith: %v", err)
	}

	// host: req-host wins (request > collection > env)
	// port: 3000 (only in env)
	// dbPass: secret-pass (from resolved secret)
	// timeout: 30 (from collection defaults)
	expected := "req-host:3000 pass=secret-pass timeout=30"
	if got != expected {
		t.Errorf("ResolveWith = %q, want %q", got, expected)
	}
}

func TestService_Resolve_NoTemplate(t *testing.T) {
	svc, _ := newTestService(t)

	if _, err := svc.Create(&CreateEnvInput{
		Name:      "dev",
		Variables: map[string]any{"host": "localhost"},
	}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := svc.SetActive("dev"); err != nil {
		t.Fatalf("SetActive: %v", err)
	}

	// Plain string with no {{}} should pass through unchanged.
	got, err := svc.Resolve("https://plain.example.com/api")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got != "https://plain.example.com/api" {
		t.Errorf("Resolve = %q, want %q", got, "https://plain.example.com/api")
	}
}
