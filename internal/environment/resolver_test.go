package environment

import (
	"os"
	"path/filepath"
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

// --- Integration Tests ---

func TestIntegration_ResolveWithSecretsFile(t *testing.T) {
	t.Setenv("INTEG_API_KEY", "env-api-key")
	dir := t.TempDir()
	svc := NewService(NewFileRepository(dir))

	// Create an environment with an $ENV{} secret.
	if _, err := svc.Create(&CreateEnvInput{
		Name:      "staging",
		Variables: map[string]any{"host": "staging.example.com", "port": 8080},
		Secrets:   map[string]string{"apiKey": "$ENV{INTEG_API_KEY}"},
	}); err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Write a .secrets.yaml that overrides the apiKey with a local value.
	secretsYAML := "secrets:\n  apiKey: local-override-key\n  dbPass: local-db-password\n"
	if err := os.WriteFile(filepath.Join(dir, "staging.secrets.yaml"), []byte(secretsYAML), 0o644); err != nil {
		t.Fatalf("write .secrets.yaml: %v", err)
	}

	if err := svc.SetActive("staging"); err != nil {
		t.Fatalf("SetActive: %v", err)
	}

	// Resolve should use .secrets.yaml override values (not $ENV{}).
	got, err := svc.Resolve("{{host}}:{{port}}/auth?key={{apiKey}}&db={{dbPass}}")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	expected := "staging.example.com:8080/auth?key=local-override-key&db=local-db-password"
	if got != expected {
		t.Errorf("Resolve = %q, want %q", got, expected)
	}
}

func TestIntegration_ActiveEnvPersistenceAndResolve(t *testing.T) {
	dir := t.TempDir()

	// Service #1: create env, set active.
	svc1 := NewService(NewFileRepository(dir))
	if _, err := svc1.Create(&CreateEnvInput{
		Name:      "production",
		Variables: map[string]any{"host": "prod.example.com", "version": "v2"},
	}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := svc1.SetActive("production"); err != nil {
		t.Fatalf("SetActive: %v", err)
	}

	// Service #2: new instance — verify active env persists and resolution works.
	svc2 := NewService(NewFileRepository(dir))
	got, err := svc2.Resolve("https://{{host}}/api/{{version}}")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	expected := "https://prod.example.com/api/v2"
	if got != expected {
		t.Errorf("Resolve = %q, want %q", got, expected)
	}
}

func TestIntegration_FullLifecycle(t *testing.T) {
	t.Setenv("LIFECYCLE_SECRET", "lifecycle-value")
	dir := t.TempDir()
	svc := NewService(NewFileRepository(dir))

	// 1. Create two environments.
	if _, err := svc.Create(&CreateEnvInput{
		Name:      "dev",
		Variables: map[string]any{"host": "localhost", "port": 3000, "url": "{{host}}:{{port}}"},
		Secrets:   map[string]string{"token": "$ENV{LIFECYCLE_SECRET}"},
	}); err != nil {
		t.Fatalf("Create dev: %v", err)
	}
	if _, err := svc.Create(&CreateEnvInput{
		Name:      "prod",
		Variables: map[string]any{"host": "prod.example.com", "port": 443, "url": "{{host}}:{{port}}"},
	}); err != nil {
		t.Fatalf("Create prod: %v", err)
	}

	// 2. No active env → Resolve should fail.
	if _, err := svc.Resolve("{{host}}"); !IsDomainError(err, "ENV_NOT_SET") {
		t.Errorf("expected ENV_NOT_SET, got: %v", err)
	}

	// 3. Set active to dev, resolve with nested vars + secrets.
	if err := svc.SetActive("dev"); err != nil {
		t.Fatalf("SetActive dev: %v", err)
	}

	got, err := svc.Resolve("{{url}}/api?token={{token}}")
	if err != nil {
		t.Fatalf("Resolve dev: %v", err)
	}
	if got != "localhost:3000/api?token=lifecycle-value" {
		t.Errorf("Resolve dev = %q, want %q", got, "localhost:3000/api?token=lifecycle-value")
	}

	// 4. ResolveWith prod (without changing active).
	got, err = svc.ResolveWith("{{url}}", "prod")
	if err != nil {
		t.Fatalf("ResolveWith prod: %v", err)
	}
	if got != "prod.example.com:443" {
		t.Errorf("ResolveWith prod = %q, want %q", got, "prod.example.com:443")
	}

	// 5. Active should still be dev.
	active, err := svc.GetActive()
	if err != nil {
		t.Fatalf("GetActive: %v", err)
	}
	if active != "dev" {
		t.Errorf("active = %q, want %q", active, "dev")
	}

	// 6. Switch active to prod, resolve again.
	if err := svc.SetActive("prod"); err != nil {
		t.Fatalf("SetActive prod: %v", err)
	}
	got, err = svc.Resolve("{{url}}")
	if err != nil {
		t.Fatalf("Resolve prod: %v", err)
	}
	if got != "prod.example.com:443" {
		t.Errorf("Resolve prod = %q, want %q", got, "prod.example.com:443")
	}

	// 7. Extra scopes override env vars.
	extra := map[string]any{"host": "custom-host", "port": 9999}
	got, err = svc.Resolve("{{host}}:{{port}}", extra)
	if err != nil {
		t.Fatalf("Resolve with extra: %v", err)
	}
	if got != "custom-host:9999" {
		t.Errorf("Resolve with extra = %q, want %q", got, "custom-host:9999")
	}
}
