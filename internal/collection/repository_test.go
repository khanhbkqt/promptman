package collection

import (
	"os"
	"path/filepath"
	"reflect"
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

// --- validateID ---

func TestValidateID(t *testing.T) {
	tests := []struct {
		id      string
		wantErr bool
	}{
		{"my-collection", false},
		{"my_collection", false},
		{"Collection123", false},
		{"a", false},
		{"../../etc/passwd", true},
		{"", true},
		{"has space", true},
		{"has.dot", true},
		{"has/slash", true},
		{"has\\backslash", true},
	}
	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			err := validateID(tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateID(%q) error = %v, wantErr = %v", tt.id, err, tt.wantErr)
			}
			if err != nil && !IsDomainError(err, envelope.CodeInvalidRequest) {
				t.Errorf("expected INVALID_REQUEST error code, got: %v", err)
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

func TestFileRepository_List_MultipleCollections(t *testing.T) {
	dir := t.TempDir()
	repo := NewFileRepository(dir)

	// Save two collections
	c1 := &Collection{Name: "API One", Requests: []Request{
		{ID: "r1", Method: "GET", Path: "/one"},
	}}
	c2 := &Collection{Name: "API Two", Requests: []Request{
		{ID: "r1", Method: "GET", Path: "/two"},
		{ID: "r2", Method: "POST", Path: "/two"},
	}}

	if err := repo.Save("api-one", c1); err != nil {
		t.Fatalf("Save c1: %v", err)
	}
	if err := repo.Save("api-two", c2); err != nil {
		t.Fatalf("Save c2: %v", err)
	}

	summaries, err := repo.List()
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(summaries) != 2 {
		t.Fatalf("expected 2 summaries, got %d", len(summaries))
	}

	// Build a map for order-independent assertion
	byID := make(map[string]CollectionSummary)
	for _, s := range summaries {
		byID[s.ID] = s
	}

	s1, ok := byID["api-one"]
	if !ok {
		t.Fatal("missing api-one in summaries")
	}
	if s1.Name != "API One" || s1.RequestCount != 1 {
		t.Errorf("api-one: Name=%q, RequestCount=%d", s1.Name, s1.RequestCount)
	}

	s2, ok := byID["api-two"]
	if !ok {
		t.Fatal("missing api-two in summaries")
	}
	if s2.Name != "API Two" || s2.RequestCount != 2 {
		t.Errorf("api-two: Name=%q, RequestCount=%d", s2.Name, s2.RequestCount)
	}
}

func TestFileRepository_List_SkipsInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	repo := NewFileRepository(dir)

	// Save one valid collection
	c := &Collection{Name: "Valid"}
	if err := repo.Save("valid", c); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Write an invalid YAML file directly
	badPath := filepath.Join(dir, "bad.yaml")
	if err := os.WriteFile(badPath, []byte("not: valid: yaml: [[["), 0644); err != nil {
		t.Fatalf("write bad yaml: %v", err)
	}

	summaries, err := repo.List()
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	// Should only have "valid", not "bad"
	if len(summaries) != 1 {
		t.Errorf("expected 1 summary, got %d", len(summaries))
	}
	if summaries[0].ID != "valid" {
		t.Errorf("expected id=valid, got %q", summaries[0].ID)
	}
}

func TestFileRepository_List_SkipsNonYAMLFiles(t *testing.T) {
	dir := t.TempDir()
	repo := NewFileRepository(dir)

	c := &Collection{Name: "Valid"}
	if err := repo.Save("valid", c); err != nil {
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

func TestFileRepository_List_CountsNestedRequests(t *testing.T) {
	dir := t.TempDir()
	repo := NewFileRepository(dir)

	c := &Collection{
		Name: "Nested",
		Requests: []Request{
			{ID: "r1", Method: "GET", Path: "/top"},
		},
		Folders: []Folder{
			{
				ID:   "f1",
				Name: "Folder 1",
				Requests: []Request{
					{ID: "r2", Method: "POST", Path: "/f1"},
				},
				Folders: []Folder{
					{
						ID:   "f2",
						Name: "Nested",
						Requests: []Request{
							{ID: "r3", Method: "DELETE", Path: "/f1/f2"},
							{ID: "r4", Method: "PUT", Path: "/f1/f2/b"},
						},
					},
				},
			},
		},
	}
	if err := repo.Save("nested", c); err != nil {
		t.Fatalf("Save: %v", err)
	}

	summaries, err := repo.List()
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(summaries) != 1 {
		t.Fatalf("expected 1 summary, got %d", len(summaries))
	}
	if summaries[0].RequestCount != 4 {
		t.Errorf("expected 4 requests, got %d", summaries[0].RequestCount)
	}
}

// --- Get ---

func TestFileRepository_Get_Valid(t *testing.T) {
	dir := t.TempDir()
	repo := NewFileRepository(dir)

	want := validFullCollection()
	if err := repo.Save("full", want); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := repo.Get("full")
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}

	if got.Name != want.Name {
		t.Errorf("Name = %q, want %q", got.Name, want.Name)
	}
	if got.BaseURL != want.BaseURL {
		t.Errorf("BaseURL = %q, want %q", got.BaseURL, want.BaseURL)
	}
	if len(got.Requests) != len(want.Requests) {
		t.Errorf("Requests count = %d, want %d", len(got.Requests), len(want.Requests))
	}
	if len(got.Folders) != len(want.Folders) {
		t.Errorf("Folders count = %d, want %d", len(got.Folders), len(want.Folders))
	}
}

func TestFileRepository_Get_NotFound(t *testing.T) {
	dir := t.TempDir()
	repo := NewFileRepository(dir)

	_, err := repo.Get("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent collection")
	}
	if !IsDomainError(err, envelope.CodeCollectionNotFound) {
		t.Errorf("expected COLLECTION_NOT_FOUND, got: %v", err)
	}
}

func TestFileRepository_Get_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	repo := NewFileRepository(dir)

	// Write malformed YAML
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

func TestFileRepository_Get_ValidationFailure(t *testing.T) {
	dir := t.TempDir()
	repo := NewFileRepository(dir)

	// Write YAML that parses but fails validation (missing name)
	yamlContent := `requests:
  - id: r1
    method: GET
    path: /test
`
	path := filepath.Join(dir, "invalid.yaml")
	if err := os.WriteFile(path, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	_, err := repo.Get("invalid")
	if err == nil {
		t.Fatal("expected error for validation failure")
	}
	if !IsDomainError(err, envelope.CodeInvalidYAML) {
		t.Errorf("expected INVALID_YAML error, got: %v", err)
	}
}

func TestFileRepository_Get_InvalidID(t *testing.T) {
	dir := t.TempDir()
	repo := NewFileRepository(dir)

	_, err := repo.Get("../../etc/passwd")
	if err == nil {
		t.Fatal("expected error for path-traversal ID")
	}
	if !IsDomainError(err, envelope.CodeInvalidRequest) {
		t.Errorf("expected INVALID_REQUEST, got: %v", err)
	}
}

// --- Save ---

func TestFileRepository_Save_Valid(t *testing.T) {
	dir := t.TempDir()
	repo := NewFileRepository(dir)

	c := validMinimalCollection()
	if err := repo.Save("test", c); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	// Verify file exists
	path := filepath.Join(dir, "test.yaml")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Size() == 0 {
		t.Error("expected non-empty file")
	}
}

func TestFileRepository_Save_ValidationFailure(t *testing.T) {
	dir := t.TempDir()
	repo := NewFileRepository(dir)

	c := &Collection{Name: ""} // invalid
	err := repo.Save("bad", c)
	if err == nil {
		t.Fatal("expected error for invalid collection")
	}

	// File should not have been created
	path := filepath.Join(dir, "bad.yaml")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("file should not exist after validation failure")
	}
}

func TestFileRepository_Save_InvalidID(t *testing.T) {
	dir := t.TempDir()
	repo := NewFileRepository(dir)

	c := validMinimalCollection()
	err := repo.Save("../sneaky", c)
	if err == nil {
		t.Fatal("expected error for invalid ID")
	}
	if !IsDomainError(err, envelope.CodeInvalidRequest) {
		t.Errorf("expected INVALID_REQUEST, got: %v", err)
	}
}

func TestFileRepository_Save_Overwrite(t *testing.T) {
	dir := t.TempDir()
	repo := NewFileRepository(dir)

	c1 := &Collection{Name: "version-1"}
	if err := repo.Save("test", c1); err != nil {
		t.Fatalf("Save v1: %v", err)
	}

	c2 := &Collection{Name: "version-2"}
	if err := repo.Save("test", c2); err != nil {
		t.Fatalf("Save v2: %v", err)
	}

	got, err := repo.Get("test")
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

	c := validMinimalCollection()
	if err := repo.Save("to-delete", c); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if err := repo.Delete("to-delete"); err != nil {
		t.Fatalf("Delete() error: %v", err)
	}

	// File should be gone
	path := filepath.Join(dir, "to-delete.yaml")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("file should not exist after delete")
	}
}

func TestFileRepository_Delete_NotFound(t *testing.T) {
	dir := t.TempDir()
	repo := NewFileRepository(dir)

	err := repo.Delete("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent collection")
	}
	if !IsDomainError(err, envelope.CodeCollectionNotFound) {
		t.Errorf("expected COLLECTION_NOT_FOUND, got: %v", err)
	}
}

func TestFileRepository_Delete_InvalidID(t *testing.T) {
	dir := t.TempDir()
	repo := NewFileRepository(dir)

	err := repo.Delete("../../etc/passwd")
	if err == nil {
		t.Fatal("expected error for path-traversal ID")
	}
	if !IsDomainError(err, envelope.CodeInvalidRequest) {
		t.Errorf("expected INVALID_REQUEST, got: %v", err)
	}
}

// --- YAML Round-Trip ---

func TestFileRepository_YAMLRoundTrip_MinimalCollection(t *testing.T) {
	dir := t.TempDir()
	repo := NewFileRepository(dir)

	original := validMinimalCollection()
	if err := repo.Save("roundtrip", original); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := repo.Get("roundtrip")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if !reflect.DeepEqual(original, loaded) {
		t.Errorf("round-trip mismatch:\n  original: %+v\n  loaded:   %+v", original, loaded)
	}
}

func TestFileRepository_YAMLRoundTrip_FullCollection(t *testing.T) {
	dir := t.TempDir()
	repo := NewFileRepository(dir)

	original := validFullCollection()
	if err := repo.Save("full-roundtrip", original); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := repo.Get("full-roundtrip")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if original.Name != loaded.Name {
		t.Errorf("Name: %q != %q", original.Name, loaded.Name)
	}
	if original.BaseURL != loaded.BaseURL {
		t.Errorf("BaseURL: %q != %q", original.BaseURL, loaded.BaseURL)
	}
	if len(original.Requests) != len(loaded.Requests) {
		t.Fatalf("Requests count: %d != %d", len(original.Requests), len(loaded.Requests))
	}
	for i := range original.Requests {
		if original.Requests[i].ID != loaded.Requests[i].ID {
			t.Errorf("Request[%d].ID: %q != %q", i, original.Requests[i].ID, loaded.Requests[i].ID)
		}
		if original.Requests[i].Method != loaded.Requests[i].Method {
			t.Errorf("Request[%d].Method: %q != %q", i, original.Requests[i].Method, loaded.Requests[i].Method)
		}
	}
	if original.Auth != nil && loaded.Auth != nil {
		if original.Auth.Type != loaded.Auth.Type {
			t.Errorf("Auth.Type: %q != %q", original.Auth.Type, loaded.Auth.Type)
		}
	}
	if len(original.Folders) != len(loaded.Folders) {
		t.Fatalf("Folders count: %d != %d", len(original.Folders), len(loaded.Folders))
	}
	if original.Folders[0].ID != loaded.Folders[0].ID {
		t.Errorf("Folder[0].ID: %q != %q", original.Folders[0].ID, loaded.Folders[0].ID)
	}
	if original.Defaults != nil && loaded.Defaults != nil {
		if *original.Defaults.Timeout != *loaded.Defaults.Timeout {
			t.Errorf("Defaults.Timeout: %d != %d", *original.Defaults.Timeout, *loaded.Defaults.Timeout)
		}
	}
}

// --- countRequests ---

func TestCountRequests(t *testing.T) {
	tests := []struct {
		name  string
		col   *Collection
		count int
	}{
		{"nil requests", &Collection{Name: "t"}, 0},
		{"top-level only", &Collection{Name: "t", Requests: []Request{
			{ID: "r1", Method: "GET", Path: "/"},
			{ID: "r2", Method: "POST", Path: "/"},
		}}, 2},
		{"with nested folders", &Collection{
			Name:     "t",
			Requests: []Request{{ID: "r1", Method: "GET", Path: "/"}},
			Folders: []Folder{{
				ID: "f1", Name: "F1",
				Requests: []Request{{ID: "r2", Method: "GET", Path: "/"}},
				Folders: []Folder{{
					ID: "f2", Name: "F2",
					Requests: []Request{
						{ID: "r3", Method: "GET", Path: "/"},
						{ID: "r4", Method: "GET", Path: "/"},
					},
				}},
			}},
		}, 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := countRequests(tt.col); got != tt.count {
				t.Errorf("countRequests() = %d, want %d", got, tt.count)
			}
		})
	}
}
