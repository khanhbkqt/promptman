package environment

import (
	"os"
	"path/filepath"
	"testing"
)

// newTestService creates a Service backed by a temp directory.
func newTestService(t *testing.T) (*Service, string) {
	t.Helper()
	dir := t.TempDir()
	repo := NewFileRepository(dir)
	return NewService(repo), dir
}

// --- Service.Create ---

func TestService_Create_Success(t *testing.T) {
	t.Setenv("API_KEY", "test-key")
	svc, _ := newTestService(t)

	env, err := svc.Create(&CreateEnvInput{
		Name:      "dev",
		Variables: map[string]any{"host": "localhost", "port": 3000},
		Secrets:   map[string]string{"apiKey": "$ENV{API_KEY}"},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if env.Name != "dev" {
		t.Errorf("Name = %q, want %q", env.Name, "dev")
	}
	if len(env.Variables) != 2 {
		t.Errorf("Variables count = %d, want 2", len(env.Variables))
	}
	if len(env.Secrets) != 1 {
		t.Errorf("Secrets count = %d, want 1", len(env.Secrets))
	}

	// Verify persistence — Get now returns masked secrets.
	got, err := svc.Get("dev")
	if err != nil {
		t.Fatalf("Get after Create: %v", err)
	}
	if got.Name != "dev" {
		t.Errorf("Get Name = %q, want %q", got.Name, "dev")
	}
	if got.Secrets["apiKey"] != "***" {
		t.Errorf("Get should mask apiKey, got %q", got.Secrets["apiKey"])
	}
}

func TestService_Create_NilInput(t *testing.T) {
	svc, _ := newTestService(t)

	_, err := svc.Create(nil)
	if err == nil {
		t.Fatal("expected error for nil input")
	}
}

func TestService_Create_EmptyName(t *testing.T) {
	svc, _ := newTestService(t)

	_, err := svc.Create(&CreateEnvInput{Name: ""})
	if err == nil {
		t.Fatal("expected validation error for empty name")
	}
}

func TestService_Create_InvalidName(t *testing.T) {
	svc, _ := newTestService(t)

	_, err := svc.Create(&CreateEnvInput{Name: "Invalid Name"})
	if err == nil {
		t.Fatal("expected validation error for invalid name")
	}
}

func TestService_Create_Duplicate(t *testing.T) {
	svc, _ := newTestService(t)

	_, err := svc.Create(&CreateEnvInput{Name: "dev"})
	if err != nil {
		t.Fatalf("first Create: %v", err)
	}

	_, err = svc.Create(&CreateEnvInput{Name: "dev"})
	if err == nil {
		t.Fatal("expected error for duplicate name")
	}
	if !IsDomainError(err, "INVALID_REQUEST") {
		t.Errorf("expected INVALID_REQUEST, got: %v", err)
	}
}

// --- Service.List ---

func TestService_List_Empty(t *testing.T) {
	svc, _ := newTestService(t)

	summaries, err := svc.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(summaries) != 0 {
		t.Errorf("len = %d, want 0", len(summaries))
	}
}

func TestService_List_Multiple(t *testing.T) {
	svc, _ := newTestService(t)

	if _, err := svc.Create(&CreateEnvInput{
		Name:      "dev",
		Variables: map[string]any{"host": "localhost"},
	}); err != nil {
		t.Fatalf("Create dev: %v", err)
	}

	if _, err := svc.Create(&CreateEnvInput{
		Name:      "prod",
		Variables: map[string]any{"host": "prod.example.com", "port": 443},
		Secrets:   map[string]string{"apiKey": "$ENV{PROD_KEY}"},
	}); err != nil {
		t.Fatalf("Create prod: %v", err)
	}

	summaries, err := svc.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(summaries) != 2 {
		t.Fatalf("len = %d, want 2", len(summaries))
	}
}

// --- Service.Get ---

func TestService_Get_NotFound(t *testing.T) {
	svc, _ := newTestService(t)

	_, err := svc.Get("nonexistent")
	if err == nil {
		t.Fatal("expected error")
	}
	if !IsDomainError(err, "ENV_NOT_FOUND") {
		t.Errorf("expected ENV_NOT_FOUND, got: %v", err)
	}
}

func TestService_Get_MasksSecrets(t *testing.T) {
	t.Setenv("TEST_SECRET", "real-secret-value")
	svc, _ := newTestService(t)

	_, err := svc.Create(&CreateEnvInput{
		Name:    "dev",
		Secrets: map[string]string{"mySecret": "$ENV{TEST_SECRET}", "plainVal": "plain-text"},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := svc.Get("dev")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	// All secrets should be masked.
	for key, val := range got.Secrets {
		if val != "***" {
			t.Errorf("secret %q = %q, want '***'", key, val)
		}
	}
}

func TestService_Get_UnsetEnvVar(t *testing.T) {
	svc, _ := newTestService(t)

	_, err := svc.Create(&CreateEnvInput{
		Name:    "dev",
		Secrets: map[string]string{"mySecret": "$ENV{TOTALLY_UNSET_VAR_XYZ}"},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	_, err = svc.Get("dev")
	if err == nil {
		t.Fatal("expected error for unset env var")
	}
	if !IsDomainError(err, "SECRET_RESOLVE_FAILED") {
		t.Errorf("expected SECRET_RESOLVE_FAILED, got: %v", err)
	}
}

// --- Service.GetRaw ---

func TestService_GetRaw_ReturnsRealValues(t *testing.T) {
	t.Setenv("TEST_RAW_SECRET", "the-real-value")
	svc, _ := newTestService(t)

	_, err := svc.Create(&CreateEnvInput{
		Name:    "dev",
		Secrets: map[string]string{"mySecret": "$ENV{TEST_RAW_SECRET}", "plain": "not-env-ref"},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := svc.GetRaw("dev")
	if err != nil {
		t.Fatalf("GetRaw: %v", err)
	}

	if got.Secrets["mySecret"] != "the-real-value" {
		t.Errorf("mySecret = %q, want %q", got.Secrets["mySecret"], "the-real-value")
	}
	if got.Secrets["plain"] != "not-env-ref" {
		t.Errorf("plain = %q, want %q", got.Secrets["plain"], "not-env-ref")
	}
}

func TestService_GetRaw_NotFound(t *testing.T) {
	svc, _ := newTestService(t)

	_, err := svc.GetRaw("nonexistent")
	if err == nil {
		t.Fatal("expected error")
	}
	if !IsDomainError(err, "ENV_NOT_FOUND") {
		t.Errorf("expected ENV_NOT_FOUND, got: %v", err)
	}
}

func TestService_GetRaw_WithSecretsFile(t *testing.T) {
	t.Setenv("TEST_OVERRIDE_KEY", "from-env")
	dir := t.TempDir()
	repo := NewFileRepository(dir)
	svc := NewService(repo)

	_, err := svc.Create(&CreateEnvInput{
		Name:    "dev",
		Secrets: map[string]string{"apiKey": "$ENV{TEST_OVERRIDE_KEY}"},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Write a .secrets.yaml that overrides the apiKey with a direct value
	secretsYAML := "secrets:\n  apiKey: direct-override-value\n"
	if err := os.WriteFile(filepath.Join(dir, "dev.secrets.yaml"), []byte(secretsYAML), 0644); err != nil {
		t.Fatalf("write secrets: %v", err)
	}

	got, err := svc.GetRaw("dev")
	if err != nil {
		t.Fatalf("GetRaw: %v", err)
	}

	// .secrets.yaml override takes precedence, and since "direct-override-value"
	// doesn't match $ENV{} pattern, it passes through unchanged.
	if got.Secrets["apiKey"] != "direct-override-value" {
		t.Errorf("apiKey = %q, want %q", got.Secrets["apiKey"], "direct-override-value")
	}
}

// --- Service.Update ---

func TestService_Update_PartialMerge(t *testing.T) {
	t.Setenv("API_KEY", "test-key")
	svc, _ := newTestService(t)

	_, err := svc.Create(&CreateEnvInput{
		Name:      "dev",
		Variables: map[string]any{"host": "localhost", "port": 3000},
		Secrets:   map[string]string{"apiKey": "$ENV{API_KEY}"},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	newVars := map[string]any{"host": "new-host", "debug": true}
	updated, err := svc.Update("dev", &UpdateEnvInput{
		Variables: &newVars,
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}

	// Variables should be replaced.
	if len(updated.Variables) != 2 {
		t.Errorf("Variables count = %d, want 2", len(updated.Variables))
	}
	if updated.Variables["host"] != "new-host" {
		t.Errorf("host = %v, want new-host", updated.Variables["host"])
	}
	// Secrets should be preserved.
	if len(updated.Secrets) != 1 {
		t.Errorf("Secrets count = %d, want 1 (preserved)", len(updated.Secrets))
	}
}

func TestService_Update_NilInput(t *testing.T) {
	svc, _ := newTestService(t)

	_, err := svc.Update("any-name", nil)
	if err == nil {
		t.Fatal("expected error for nil input")
	}
}

func TestService_Update_NotFound(t *testing.T) {
	svc, _ := newTestService(t)

	newName := "whatever"
	_, err := svc.Update("nonexistent", &UpdateEnvInput{Name: &newName})
	if err == nil {
		t.Fatal("expected error")
	}
	if !IsDomainError(err, "ENV_NOT_FOUND") {
		t.Errorf("expected ENV_NOT_FOUND, got: %v", err)
	}
}

func TestService_Update_NoFieldsSet(t *testing.T) {
	svc, _ := newTestService(t)

	_, err := svc.Update("dev", &UpdateEnvInput{})
	if err == nil {
		t.Fatal("expected validation error for empty update")
	}
}

// --- Service.Delete ---

func TestService_Delete_Success(t *testing.T) {
	svc, _ := newTestService(t)

	_, err := svc.Create(&CreateEnvInput{Name: "ephemeral"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := svc.Delete("ephemeral"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// Verify it's gone.
	_, err = svc.Get("ephemeral")
	if err == nil {
		t.Fatal("expected not found after delete")
	}
	if !IsDomainError(err, "ENV_NOT_FOUND") {
		t.Errorf("expected ENV_NOT_FOUND, got: %v", err)
	}
}

func TestService_Delete_NotFound(t *testing.T) {
	svc, _ := newTestService(t)

	err := svc.Delete("nonexistent")
	if err == nil {
		t.Fatal("expected error")
	}
	if !IsDomainError(err, "ENV_NOT_FOUND") {
		t.Errorf("expected ENV_NOT_FOUND, got: %v", err)
	}
}

// --- Full CRUD Flow ---

func TestService_FullFlow(t *testing.T) {
	t.Setenv("STAGING_DB_PASS", "staging-password")
	dir := t.TempDir()
	repo := NewFileRepository(dir)
	svc := NewService(repo)

	// 1. Create
	env, err := svc.Create(&CreateEnvInput{
		Name:      "staging",
		Variables: map[string]any{"host": "staging.example.com", "port": 443},
		Secrets:   map[string]string{"dbPass": "$ENV{STAGING_DB_PASS}"},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if env.Name != "staging" {
		t.Errorf("Name = %q, want staging", env.Name)
	}

	// 2. List — should have 1
	summaries, err := svc.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(summaries) != 1 {
		t.Fatalf("List len = %d, want 1", len(summaries))
	}
	if summaries[0].VariableCount != 2 {
		t.Errorf("VariableCount = %d, want 2", summaries[0].VariableCount)
	}
	if summaries[0].SecretCount != 1 {
		t.Errorf("SecretCount = %d, want 1", summaries[0].SecretCount)
	}

	// 3. Get — secrets should be masked
	got, err := svc.Get("staging")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Name != "staging" {
		t.Errorf("Name = %q", got.Name)
	}
	if got.Secrets["dbPass"] != "***" {
		t.Errorf("Get should mask dbPass, got %q", got.Secrets["dbPass"])
	}

	// 3b. GetRaw — secrets should be real values
	raw, err := svc.GetRaw("staging")
	if err != nil {
		t.Fatalf("GetRaw: %v", err)
	}
	if raw.Secrets["dbPass"] != "staging-password" {
		t.Errorf("GetRaw dbPass = %q, want %q", raw.Secrets["dbPass"], "staging-password")
	}

	// 4. Update
	newPort := map[string]any{"host": "staging-v2.example.com", "port": 8443, "debug": false}
	updated, err := svc.Update("staging", &UpdateEnvInput{Variables: &newPort})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Variables["host"] != "staging-v2.example.com" {
		t.Errorf("Updated host = %v", updated.Variables["host"])
	}

	// 5. Delete
	if err := svc.Delete("staging"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// 6. Verify file gone
	yamlPath := filepath.Join(dir, "staging.yaml")
	if _, err := os.Stat(yamlPath); !os.IsNotExist(err) {
		t.Errorf("expected file to be deleted, got err: %v", err)
	}

	// 7. Get returns not found
	_, err = svc.Get("staging")
	if !IsDomainError(err, "ENV_NOT_FOUND") {
		t.Errorf("expected ENV_NOT_FOUND after delete, got: %v", err)
	}
}

// --- Service.SetActive / GetActive ---

func TestService_SetActive_Success(t *testing.T) {
	svc, _ := newTestService(t)

	if _, err := svc.Create(&CreateEnvInput{Name: "dev"}); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := svc.SetActive("dev"); err != nil {
		t.Fatalf("SetActive: %v", err)
	}

	got, err := svc.GetActive()
	if err != nil {
		t.Fatalf("GetActive: %v", err)
	}
	if got != "dev" {
		t.Errorf("GetActive = %q, want %q", got, "dev")
	}
}

func TestService_GetActive_NotSet(t *testing.T) {
	svc, _ := newTestService(t)

	_, err := svc.GetActive()
	if err == nil {
		t.Fatal("expected error")
	}
	if !IsDomainError(err, "ENV_NOT_SET") {
		t.Errorf("expected ENV_NOT_SET, got: %v", err)
	}
}

func TestService_SetActive_NonexistentEnv(t *testing.T) {
	svc, _ := newTestService(t)

	err := svc.SetActive("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent env")
	}
	if !IsDomainError(err, "ENV_NOT_FOUND") {
		t.Errorf("expected ENV_NOT_FOUND, got: %v", err)
	}
}

func TestService_SetActive_Persistence(t *testing.T) {
	dir := t.TempDir()

	// Service #1: create env and set active.
	svc1 := NewService(NewFileRepository(dir))
	if _, err := svc1.Create(&CreateEnvInput{Name: "prod"}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := svc1.SetActive("prod"); err != nil {
		t.Fatalf("SetActive: %v", err)
	}

	// Service #2: new instance, same dir — should read persisted active env.
	svc2 := NewService(NewFileRepository(dir))
	got, err := svc2.GetActive()
	if err != nil {
		t.Fatalf("GetActive (new service): %v", err)
	}
	if got != "prod" {
		t.Errorf("GetActive = %q, want %q", got, "prod")
	}
}

func TestService_SetActive_SwitchEnv(t *testing.T) {
	svc, _ := newTestService(t)

	if _, err := svc.Create(&CreateEnvInput{Name: "dev"}); err != nil {
		t.Fatalf("Create dev: %v", err)
	}
	if _, err := svc.Create(&CreateEnvInput{Name: "prod"}); err != nil {
		t.Fatalf("Create prod: %v", err)
	}

	// Set to dev, then switch to prod.
	if err := svc.SetActive("dev"); err != nil {
		t.Fatalf("SetActive dev: %v", err)
	}
	if err := svc.SetActive("prod"); err != nil {
		t.Fatalf("SetActive prod: %v", err)
	}

	got, err := svc.GetActive()
	if err != nil {
		t.Fatalf("GetActive: %v", err)
	}
	if got != "prod" {
		t.Errorf("GetActive = %q, want %q", got, "prod")
	}
}
