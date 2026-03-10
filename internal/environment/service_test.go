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

	// Verify persistence.
	got, err := svc.Get("dev")
	if err != nil {
		t.Fatalf("Get after Create: %v", err)
	}
	if got.Name != "dev" {
		t.Errorf("Get Name = %q, want %q", got.Name, "dev")
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

// --- Service.Update ---

func TestService_Update_PartialMerge(t *testing.T) {
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

	// 3. Get
	got, err := svc.Get("staging")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Name != "staging" {
		t.Errorf("Name = %q", got.Name)
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
