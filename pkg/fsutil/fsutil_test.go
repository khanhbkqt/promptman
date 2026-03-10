package fsutil

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// --- YAML tests ---

func TestReadWriteYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.yaml")

	type Config struct {
		Name    string `yaml:"name"`
		Port    int    `yaml:"port"`
		Enabled bool   `yaml:"enabled"`
	}

	original := Config{Name: "test-app", Port: 8080, Enabled: true}

	if err := WriteYAML(path, original); err != nil {
		t.Fatalf("WriteYAML failed: %v", err)
	}

	var loaded Config
	if err := ReadYAML(path, &loaded); err != nil {
		t.Fatalf("ReadYAML failed: %v", err)
	}

	if loaded.Name != original.Name || loaded.Port != original.Port || loaded.Enabled != original.Enabled {
		t.Errorf("round-trip failed: got %+v, want %+v", loaded, original)
	}
}

func TestReadYAML_NotFound(t *testing.T) {
	var out map[string]any
	err := ReadYAML("/nonexistent/path.yaml", &out)
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestWriteYAML_AutoMkdir(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "dir", "config.yaml")

	if err := WriteYAML(path, map[string]string{"key": "value"}); err != nil {
		t.Fatalf("WriteYAML with auto-mkdir failed: %v", err)
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("expected file to exist after auto-mkdir write")
	}
}

// --- JSON tests ---

func TestReadWriteJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.json")

	type Data struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}

	original := Data{ID: 42, Name: "test"}

	if err := WriteJSON(path, original); err != nil {
		t.Fatalf("WriteJSON failed: %v", err)
	}

	var loaded Data
	if err := ReadJSON(path, &loaded); err != nil {
		t.Fatalf("ReadJSON failed: %v", err)
	}

	if loaded.ID != original.ID || loaded.Name != original.Name {
		t.Errorf("round-trip failed: got %+v, want %+v", loaded, original)
	}
}

func TestReadJSON_NotFound(t *testing.T) {
	var out map[string]any
	err := ReadJSON("/nonexistent/path.json", &out)
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestAppendJSONL(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "history.jsonl")

	type Entry struct {
		ID  int    `json:"id"`
		URL string `json:"url"`
	}

	entries := []Entry{
		{ID: 1, URL: "https://api.example.com/users"},
		{ID: 2, URL: "https://api.example.com/posts"},
		{ID: 3, URL: "https://api.example.com/comments"},
	}

	for _, entry := range entries {
		if err := AppendJSONL(path, entry); err != nil {
			t.Fatalf("AppendJSONL failed: %v", err)
		}
	}

	// Verify: read back and check each line
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("failed to open JSONL file: %v", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	var count int
	for scanner.Scan() {
		var entry Entry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			t.Fatalf("failed to parse JSONL line %d: %v", count+1, err)
		}
		if entry.ID != entries[count].ID || entry.URL != entries[count].URL {
			t.Errorf("line %d: got %+v, want %+v", count+1, entry, entries[count])
		}
		count++
	}

	if count != len(entries) {
		t.Errorf("expected %d lines, got %d", len(entries), count)
	}
}

func TestAppendJSONL_AutoMkdir(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "history.jsonl")

	if err := AppendJSONL(path, map[string]string{"key": "value"}); err != nil {
		t.Fatalf("AppendJSONL with auto-mkdir failed: %v", err)
	}
}

// --- Atomic Write tests ---

func TestAtomicWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")

	data := []byte("hello, world!")
	if err := AtomicWrite(path, data); err != nil {
		t.Fatalf("AtomicWrite failed: %v", err)
	}

	read, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if string(read) != string(data) {
		t.Errorf("data mismatch: got %q, want %q", read, data)
	}
}

func TestAtomicWrite_Overwrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")

	if err := AtomicWrite(path, []byte("first")); err != nil {
		t.Fatalf("first write failed: %v", err)
	}
	if err := AtomicWrite(path, []byte("second")); err != nil {
		t.Fatalf("second write failed: %v", err)
	}

	read, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if string(read) != "second" {
		t.Errorf("expected 'second', got %q", read)
	}
}

func TestAtomicWrite_NoTempFileRemains(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")

	if err := AtomicWrite(path, []byte("data")); err != nil {
		t.Fatalf("AtomicWrite failed: %v", err)
	}

	// Check no .promptman-tmp-* files remain
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("failed to read dir: %v", err)
	}
	for _, entry := range entries {
		if entry.Name() != "test.txt" {
			t.Errorf("unexpected file remaining: %s", entry.Name())
		}
	}
}

func TestAtomicWrite_AutoMkdir(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "deeply", "nested", "dir", "test.txt")

	if err := AtomicWrite(path, []byte("data")); err != nil {
		t.Fatalf("AtomicWrite with auto-mkdir failed: %v", err)
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("expected file to exist")
	}
}

// --- Path helper tests ---

func TestProjectDirFrom(t *testing.T) {
	// Create a temp project with .promptman dir
	dir := t.TempDir()
	promptmanPath := filepath.Join(dir, ".promptman")
	if err := os.MkdirAll(promptmanPath, 0755); err != nil {
		t.Fatalf("failed to create .promptman dir: %v", err)
	}

	// Test finding from project root
	result, err := projectDirFrom(dir)
	if err != nil {
		t.Fatalf("projectDirFrom failed: %v", err)
	}
	if result != dir {
		t.Errorf("expected %s, got %s", dir, result)
	}

	// Test finding from nested subdirectory
	nested := filepath.Join(dir, "internal", "collection")
	if err := os.MkdirAll(nested, 0755); err != nil {
		t.Fatalf("failed to create nested dir: %v", err)
	}

	result, err = projectDirFrom(nested)
	if err != nil {
		t.Fatalf("projectDirFrom from nested dir failed: %v", err)
	}
	if result != dir {
		t.Errorf("expected %s from nested dir, got %s", dir, result)
	}
}

func TestProjectDirFrom_NotFound(t *testing.T) {
	dir := t.TempDir() // No .promptman
	_, err := projectDirFrom(dir)
	if err == nil {
		t.Error("expected error when .promptman not found")
	}
}
