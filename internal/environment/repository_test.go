package environment

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/khanhnguyen/promptman/pkg/envelope"
)

// --- NewFileRepository ---

func TestNewFileRepository(t *testing.T) {
	repo := NewFileRepository("/some/dir")
	if repo.dir != "/some/dir" {
		t.Errorf("dir = %q, want %q", repo.dir, "/some/dir")
	}
}

// --- validateName ---

func TestValidateName(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{"dev", false},
		{"staging", false},
		{"my-env", false},
		{"env-123", false},
		{"a", false},
		{"123", false},
		{"../../etc/passwd", true},
		{"", true},
		{"has space", true},
		{"has.dot", true},
		{"has/slash", true},
		{"has\\backslash", true},
		{"HAS_UPPER", true},
		{"has_underscore", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateName(tt.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateName(%q) error = %v, wantErr = %v", tt.name, err, tt.wantErr)
			}
		})
	}
}

// --- List ---

func TestFileRepository_List_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	repo := NewFileRepository(dir)

	summaries, err := repo.List()
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(summaries) != 0 {
		t.Errorf("expected empty list, got %d items", len(summaries))
	}
}

func TestFileRepository_List_NonExistentDir(t *testing.T) {
	repo := NewFileRepository(filepath.Join(t.TempDir(), "nonexistent"))

	summaries, err := repo.List()
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if summaries != nil {
		t.Errorf("expected nil, got %v", summaries)
	}
}

func TestFileRepository_List_MultipleEnvironments(t *testing.T) {
	dir := t.TempDir()
	repo := NewFileRepository(dir)

	// Save two environments
	env1 := &Environment{
		Name:      "Development",
		Variables: map[string]any{"host": "localhost", "port": 3000},
		Secrets:   map[string]string{"apiKey": "$ENV{API_KEY}"},
	}
	env2 := &Environment{
		Name:      "Production",
		Variables: map[string]any{"host": "prod.example.com"},
	}

	if err := repo.Save("dev", env1); err != nil {
		t.Fatalf("Save dev: %v", err)
	}
	if err := repo.Save("prod", env2); err != nil {
		t.Fatalf("Save prod: %v", err)
	}

	summaries, err := repo.List()
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(summaries) != 2 {
		t.Fatalf("expected 2 summaries, got %d", len(summaries))
	}

	// Build map for order-independent assertion
	byName := make(map[string]EnvSummary)
	for _, s := range summaries {
		byName[s.Name] = s
	}

	dev := byName["dev"]
	if dev.VariableCount != 2 || dev.SecretCount != 1 {
		t.Errorf("dev: VarCount=%d, SecretCount=%d, want 2, 1", dev.VariableCount, dev.SecretCount)
	}

	prod := byName["prod"]
	if prod.VariableCount != 1 || prod.SecretCount != 0 {
		t.Errorf("prod: VarCount=%d, SecretCount=%d, want 1, 0", prod.VariableCount, prod.SecretCount)
	}
}

func TestFileRepository_List_SkipsSecretsFiles(t *testing.T) {
	dir := t.TempDir()
	repo := NewFileRepository(dir)

	env := &Environment{Name: "Development", Variables: map[string]any{"host": "localhost"}}
	if err := repo.Save("dev", env); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Write a secrets file directly
	secretsPath := filepath.Join(dir, "dev.secrets.yaml")
	if err := os.WriteFile(secretsPath, []byte("variables:\n  apiKey: real-key\n"), 0644); err != nil {
		t.Fatalf("write secrets: %v", err)
	}

	summaries, err := repo.List()
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(summaries) != 1 {
		t.Errorf("expected 1 summary (secrets file skipped), got %d", len(summaries))
	}
}

func TestFileRepository_List_SkipsInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	repo := NewFileRepository(dir)

	env := &Environment{Name: "Valid"}
	if err := repo.Save("valid", env); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Write an invalid YAML file
	badPath := filepath.Join(dir, "bad.yaml")
	if err := os.WriteFile(badPath, []byte("not: valid: yaml: [[["), 0644); err != nil {
		t.Fatalf("write bad yaml: %v", err)
	}

	summaries, err := repo.List()
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(summaries) != 1 {
		t.Errorf("expected 1 summary, got %d", len(summaries))
	}
}

func TestFileRepository_List_SkipsNonYAMLFiles(t *testing.T) {
	dir := t.TempDir()
	repo := NewFileRepository(dir)

	env := &Environment{Name: "Valid"}
	if err := repo.Save("valid", env); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Create a non-YAML file and a directory
	os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("hello"), 0644)
	os.Mkdir(filepath.Join(dir, "subdir"), 0755)

	summaries, err := repo.List()
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(summaries) != 1 {
		t.Errorf("expected 1 summary, got %d", len(summaries))
	}
}

// --- Get ---

func TestFileRepository_Get_Valid(t *testing.T) {
	dir := t.TempDir()
	repo := NewFileRepository(dir)

	want := &Environment{
		Name:      "Development",
		Variables: map[string]any{"host": "localhost", "port": 3000},
		Secrets:   map[string]string{"apiKey": "$ENV{API_KEY}"},
	}
	if err := repo.Save("dev", want); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := repo.Get("dev")
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}

	if got.Name != want.Name {
		t.Errorf("Name = %q, want %q", got.Name, want.Name)
	}
	if len(got.Variables) != len(want.Variables) {
		t.Errorf("Variables count = %d, want %d", len(got.Variables), len(want.Variables))
	}
	if len(got.Secrets) != len(want.Secrets) {
		t.Errorf("Secrets count = %d, want %d", len(got.Secrets), len(want.Secrets))
	}
}

func TestFileRepository_Get_NotFound(t *testing.T) {
	dir := t.TempDir()
	repo := NewFileRepository(dir)

	_, err := repo.Get("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent environment")
	}
	if !IsDomainError(err, envelope.CodeEnvNotFound) {
		t.Errorf("expected ENV_NOT_FOUND, got: %v", err)
	}
}

func TestFileRepository_Get_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	repo := NewFileRepository(dir)

	badPath := filepath.Join(dir, "bad.yaml")
	if err := os.WriteFile(badPath, []byte("{{{{not yaml"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	_, err := repo.Get("bad")
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
	if !IsDomainError(err, envelope.CodeInvalidYAML) {
		t.Errorf("expected INVALID_YAML, got: %v", err)
	}
}

func TestFileRepository_Get_InvalidName(t *testing.T) {
	dir := t.TempDir()
	repo := NewFileRepository(dir)

	_, err := repo.Get("../../etc/passwd")
	if err == nil {
		t.Fatal("expected error for path-traversal name")
	}
}

func TestFileRepository_Get_FillsEmptyName(t *testing.T) {
	dir := t.TempDir()
	repo := NewFileRepository(dir)

	// Write a YAML file without a name field
	path := filepath.Join(dir, "unnamed.yaml")
	if err := os.WriteFile(path, []byte("variables:\n  host: localhost\n"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	got, err := repo.Get("unnamed")
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	if got.Name != "unnamed" {
		t.Errorf("Name = %q, want %q (filled from filename)", got.Name, "unnamed")
	}
}

// --- Save ---

func TestFileRepository_Save_Valid(t *testing.T) {
	dir := t.TempDir()
	repo := NewFileRepository(dir)

	env := &Environment{Name: "Development"}
	if err := repo.Save("dev", env); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	// Verify file exists
	path := filepath.Join(dir, "dev.yaml")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Size() == 0 {
		t.Error("expected non-empty file")
	}
}

func TestFileRepository_Save_InvalidName(t *testing.T) {
	dir := t.TempDir()
	repo := NewFileRepository(dir)

	env := &Environment{Name: "Development"}
	err := repo.Save("../sneaky", env)
	if err == nil {
		t.Fatal("expected error for invalid name")
	}
}

func TestFileRepository_Save_Overwrite(t *testing.T) {
	dir := t.TempDir()
	repo := NewFileRepository(dir)

	env1 := &Environment{Name: "version-1"}
	if err := repo.Save("dev", env1); err != nil {
		t.Fatalf("Save v1: %v", err)
	}

	env2 := &Environment{Name: "version-2"}
	if err := repo.Save("dev", env2); err != nil {
		t.Fatalf("Save v2: %v", err)
	}

	got, err := repo.Get("dev")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Name != "version-2" {
		t.Errorf("Name = %q, want %q", got.Name, "version-2")
	}
}

// --- Delete ---

func TestFileRepository_Delete_Valid(t *testing.T) {
	dir := t.TempDir()
	repo := NewFileRepository(dir)

	env := &Environment{Name: "Development"}
	if err := repo.Save("dev", env); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if err := repo.Delete("dev"); err != nil {
		t.Fatalf("Delete() error: %v", err)
	}

	// File should be gone
	path := filepath.Join(dir, "dev.yaml")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("file should not exist after delete")
	}
}

func TestFileRepository_Delete_NotFound(t *testing.T) {
	dir := t.TempDir()
	repo := NewFileRepository(dir)

	err := repo.Delete("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent environment")
	}
	if !IsDomainError(err, envelope.CodeEnvNotFound) {
		t.Errorf("expected ENV_NOT_FOUND, got: %v", err)
	}
}

func TestFileRepository_Delete_InvalidName(t *testing.T) {
	dir := t.TempDir()
	repo := NewFileRepository(dir)

	err := repo.Delete("../../etc/passwd")
	if err == nil {
		t.Fatal("expected error for path-traversal name")
	}
}

func TestFileRepository_Delete_WithSecretsFile(t *testing.T) {
	dir := t.TempDir()
	repo := NewFileRepository(dir)

	env := &Environment{Name: "Development"}
	if err := repo.Save("dev", env); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Create a secrets file alongside the env file
	secretsPath := filepath.Join(dir, "dev.secrets.yaml")
	if err := os.WriteFile(secretsPath, []byte("variables:\n  apiKey: real-key\n"), 0644); err != nil {
		t.Fatalf("write secrets: %v", err)
	}

	if err := repo.Delete("dev"); err != nil {
		t.Fatalf("Delete() error: %v", err)
	}

	// Both files should be gone
	if _, err := os.Stat(filepath.Join(dir, "dev.yaml")); !os.IsNotExist(err) {
		t.Error("env file should not exist after delete")
	}
	if _, err := os.Stat(secretsPath); !os.IsNotExist(err) {
		t.Error("secrets file should not exist after delete")
	}
}

// --- YAML Round-Trip ---

func TestFileRepository_YAMLRoundTrip(t *testing.T) {
	dir := t.TempDir()
	repo := NewFileRepository(dir)

	original := &Environment{
		Name:      "Development",
		Variables: map[string]any{"host": "localhost", "debug": true},
		Secrets:   map[string]string{"apiKey": "$ENV{API_KEY}", "dbPass": "$ENV{DB_PASS}"},
	}
	if err := repo.Save("dev", original); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := repo.Get("dev")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if loaded.Name != original.Name {
		t.Errorf("Name: %q != %q", loaded.Name, original.Name)
	}
	if len(loaded.Variables) != len(original.Variables) {
		t.Errorf("Variables count: %d != %d", len(loaded.Variables), len(original.Variables))
	}
	if len(loaded.Secrets) != len(original.Secrets) {
		t.Errorf("Secrets count: %d != %d", len(loaded.Secrets), len(original.Secrets))
	}
	// Check specific values
	if loaded.Variables["host"] != "localhost" {
		t.Errorf("host = %v, want localhost", loaded.Variables["host"])
	}
	if loaded.Variables["debug"] != true {
		t.Errorf("debug = %v, want true", loaded.Variables["debug"])
	}
	if loaded.Secrets["apiKey"] != "$ENV{API_KEY}" {
		t.Errorf("apiKey = %q, want $ENV{API_KEY}", loaded.Secrets["apiKey"])
	}
}
