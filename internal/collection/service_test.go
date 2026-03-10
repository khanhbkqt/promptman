package collection

import (
	"os"
	"path/filepath"
	"testing"
)

// newTestService creates a Service backed by a temp directory.
func newTestService(t *testing.T) *Service {
	t.Helper()
	dir := t.TempDir()
	repo := NewFileRepository(dir)
	return NewService(repo)
}

// saveTestCollection writes a collection YAML directly for setup purposes.
func saveTestCollection(t *testing.T, dir, id string, c *Collection) {
	t.Helper()
	repo := NewFileRepository(dir)
	if err := repo.Save(id, c); err != nil {
		t.Fatalf("saveTestCollection(%s): %v", id, err)
	}
}

// --- generateID tests ---

func TestGenerateID(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"simple name", "Users API", "users-api"},
		{"extra spaces", "  My  Cool  API  ", "my-cool-api"},
		{"special chars", "Hello World! @#$ v2", "hello-world-v2"},
		{"already slug", "users-api", "users-api"},
		{"empty string", "", "collection"},
		{"only spaces", "   ", "collection"},
		{"hyphens only", "---", "collection"},
		{"unicode", "Café API", "caf-api"},
		{"numbers", "API v2.1", "api-v2-1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateID(tt.in)
			if got != tt.want {
				t.Errorf("generateID(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// --- Service.Create tests ---

func TestService_Create_Success(t *testing.T) {
	svc := newTestService(t)

	id, err := svc.Create(&CreateCollectionInput{
		Name:    "Users API",
		BaseURL: "https://api.example.com",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if id != "users-api" {
		t.Errorf("id = %q, want %q", id, "users-api")
	}

	// Verify the collection was persisted.
	c, err := svc.Get(id)
	if err != nil {
		t.Fatalf("Get after Create: %v", err)
	}
	if c.Name != "Users API" {
		t.Errorf("Name = %q, want %q", c.Name, "Users API")
	}
	if c.BaseURL != "https://api.example.com" {
		t.Errorf("BaseURL = %q, want %q", c.BaseURL, "https://api.example.com")
	}
}

func TestService_Create_NilInput(t *testing.T) {
	svc := newTestService(t)

	_, err := svc.Create(nil)
	if err == nil {
		t.Fatal("expected error for nil input")
	}
}

func TestService_Create_EmptyName(t *testing.T) {
	svc := newTestService(t)

	_, err := svc.Create(&CreateCollectionInput{Name: ""})
	if err == nil {
		t.Fatal("expected validation error for empty name")
	}
}

func TestService_Create_WithRequestsAndFolders(t *testing.T) {
	svc := newTestService(t)

	id, err := svc.Create(&CreateCollectionInput{
		Name:    "Full API",
		BaseURL: "https://api.example.com",
		Requests: []Request{
			{ID: "health", Method: "GET", Path: "/health"},
		},
		Folders: []Folder{
			{
				ID:   "admin",
				Name: "Admin",
				Requests: []Request{
					{ID: "list-admins", Method: "GET", Path: "/admin/users"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	c, err := svc.Get(id)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(c.Requests) != 1 {
		t.Errorf("len(Requests) = %d, want 1", len(c.Requests))
	}
	if len(c.Folders) != 1 {
		t.Errorf("len(Folders) = %d, want 1", len(c.Folders))
	}
}

// --- Service.List tests ---

func TestService_List_Empty(t *testing.T) {
	svc := newTestService(t)

	summaries, err := svc.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(summaries) != 0 {
		t.Errorf("len = %d, want 0", len(summaries))
	}
}

func TestService_List_Multiple(t *testing.T) {
	svc := newTestService(t)

	if _, err := svc.Create(&CreateCollectionInput{
		Name: "Alpha",
		Requests: []Request{
			{ID: "r1", Method: "GET", Path: "/a"},
			{ID: "r2", Method: "POST", Path: "/b"},
		},
	}); err != nil {
		t.Fatalf("Create Alpha: %v", err)
	}

	if _, err := svc.Create(&CreateCollectionInput{
		Name: "Beta",
		Requests: []Request{
			{ID: "r3", Method: "DELETE", Path: "/c"},
		},
	}); err != nil {
		t.Fatalf("Create Beta: %v", err)
	}

	summaries, err := svc.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(summaries) != 2 {
		t.Fatalf("len = %d, want 2", len(summaries))
	}

	// Build a map for order-independent checks.
	byID := make(map[string]CollectionSummary)
	for _, s := range summaries {
		byID[s.ID] = s
	}

	alpha := byID["alpha"]
	if alpha.Name != "Alpha" || alpha.RequestCount != 2 {
		t.Errorf("Alpha summary: %+v", alpha)
	}

	beta := byID["beta"]
	if beta.Name != "Beta" || beta.RequestCount != 1 {
		t.Errorf("Beta summary: %+v", beta)
	}
}

// --- Service.Get tests ---

func TestService_Get_NotFound(t *testing.T) {
	svc := newTestService(t)

	_, err := svc.Get("nonexistent")
	if err == nil {
		t.Fatal("expected error")
	}
	if !IsDomainError(err, "COLLECTION_NOT_FOUND") {
		t.Errorf("expected COLLECTION_NOT_FOUND, got: %v", err)
	}
}

// --- Service.Update tests ---

func TestService_Update_PartialMerge(t *testing.T) {
	svc := newTestService(t)

	id, err := svc.Create(&CreateCollectionInput{
		Name:    "Original",
		BaseURL: "https://old.example.com",
		Requests: []Request{
			{ID: "r1", Method: "GET", Path: "/original"},
		},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	newName := "Updated"
	updated, err := svc.Update(id, &UpdateCollectionInput{
		Name: &newName,
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}

	if updated.Name != "Updated" {
		t.Errorf("Name = %q, want %q", updated.Name, "Updated")
	}
	// BaseURL should be preserved.
	if updated.BaseURL != "https://old.example.com" {
		t.Errorf("BaseURL = %q, want preserved", updated.BaseURL)
	}
	// Requests should be preserved.
	if len(updated.Requests) != 1 {
		t.Errorf("Requests len = %d, want 1", len(updated.Requests))
	}
}

func TestService_Update_NilInput(t *testing.T) {
	svc := newTestService(t)

	_, err := svc.Update("any-id", nil)
	if err == nil {
		t.Fatal("expected error for nil input")
	}
}

func TestService_Update_NotFound(t *testing.T) {
	svc := newTestService(t)

	newName := "Whatever"
	_, err := svc.Update("nonexistent", &UpdateCollectionInput{Name: &newName})
	if err == nil {
		t.Fatal("expected error")
	}
	if !IsDomainError(err, "COLLECTION_NOT_FOUND") {
		t.Errorf("expected COLLECTION_NOT_FOUND, got: %v", err)
	}
}

func TestService_Update_InvalidResult(t *testing.T) {
	svc := newTestService(t)

	id, err := svc.Create(&CreateCollectionInput{Name: "Valid"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	emptyName := ""
	_, err = svc.Update(id, &UpdateCollectionInput{Name: &emptyName})
	if err == nil {
		t.Fatal("expected validation error for empty name after merge")
	}
}

// --- Service.Delete tests ---

func TestService_Delete_Success(t *testing.T) {
	svc := newTestService(t)

	id, err := svc.Create(&CreateCollectionInput{Name: "Ephemeral"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := svc.Delete(id); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// Verify it's gone.
	_, err = svc.Get(id)
	if err == nil {
		t.Fatal("expected not found after delete")
	}
	if !IsDomainError(err, "COLLECTION_NOT_FOUND") {
		t.Errorf("expected COLLECTION_NOT_FOUND, got: %v", err)
	}
}

func TestService_Delete_NotFound(t *testing.T) {
	svc := newTestService(t)

	err := svc.Delete("nonexistent")
	if err == nil {
		t.Fatal("expected error")
	}
	if !IsDomainError(err, "COLLECTION_NOT_FOUND") {
		t.Errorf("expected COLLECTION_NOT_FOUND, got: %v", err)
	}
}

// --- Service.FindRequest tests ---

func TestService_FindRequest_RootLevel(t *testing.T) {
	svc := newTestService(t)

	id, err := svc.Create(&CreateCollectionInput{
		Name:    "Users API",
		BaseURL: "https://api.example.com",
		Defaults: &RequestDefaults{
			Headers: map[string]string{"Accept": "application/json"},
		},
		Requests: []Request{
			{ID: "health", Method: "GET", Path: "/health"},
		},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	resolved, err := svc.FindRequest(id, "health")
	if err != nil {
		t.Fatalf("FindRequest: %v", err)
	}

	if resolved.Method != "GET" {
		t.Errorf("Method = %q, want GET", resolved.Method)
	}
	if resolved.URL != "https://api.example.com/health" {
		t.Errorf("URL = %q, want https://api.example.com/health", resolved.URL)
	}
	if resolved.Headers["Accept"] != "application/json" {
		t.Errorf("Accept header = %q, want application/json", resolved.Headers["Accept"])
	}
}

func TestService_FindRequest_NestedFolder(t *testing.T) {
	svc := newTestService(t)

	id, err := svc.Create(&CreateCollectionInput{
		Name:    "Users",
		BaseURL: "https://api.example.com",
		Defaults: &RequestDefaults{
			Headers: map[string]string{
				"Accept":       "application/json",
				"Content-Type": "application/json",
			},
		},
		Auth: &AuthConfig{
			Type:   "bearer",
			Bearer: &BearerAuth{Token: "{{authToken}}"},
		},
		Folders: []Folder{
			{
				ID:   "admin",
				Name: "Admin",
				Auth: &AuthConfig{
					Type:   "api-key",
					APIKey: &APIKeyAuth{Key: "X-Admin-Key", Value: "{{adminKey}}"},
				},
				Requests: []Request{
					{ID: "list-admins", Method: "GET", Path: "/admin/users"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	resolved, err := svc.FindRequest(id, "admin/list-admins")
	if err != nil {
		t.Fatalf("FindRequest: %v", err)
	}

	if resolved.Method != "GET" {
		t.Errorf("Method = %q, want GET", resolved.Method)
	}
	// Auth should be overridden by folder.
	if resolved.Auth == nil || resolved.Auth.Type != "api-key" {
		t.Errorf("Auth = %+v, want api-key", resolved.Auth)
	}
	// Headers should be inherited from collection defaults.
	if resolved.Headers["Accept"] != "application/json" {
		t.Errorf("Accept = %q, want application/json", resolved.Headers["Accept"])
	}
}

func TestService_FindRequest_CollectionNotFound(t *testing.T) {
	svc := newTestService(t)

	_, err := svc.FindRequest("nonexistent", "any-path")
	if err == nil {
		t.Fatal("expected error")
	}
	if !IsDomainError(err, "COLLECTION_NOT_FOUND") {
		t.Errorf("expected COLLECTION_NOT_FOUND, got: %v", err)
	}
}

func TestService_FindRequest_RequestNotFound(t *testing.T) {
	svc := newTestService(t)

	id, err := svc.Create(&CreateCollectionInput{
		Name: "Empty API",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	_, err = svc.FindRequest(id, "nonexistent")
	if err == nil {
		t.Fatal("expected error")
	}
	if !IsDomainError(err, "REQUEST_NOT_FOUND") {
		t.Errorf("expected REQUEST_NOT_FOUND, got: %v", err)
	}
}

// --- Full CRUD flow ---

func TestService_FullFlow(t *testing.T) {
	dir := t.TempDir()
	repo := NewFileRepository(dir)
	svc := NewService(repo)

	// 1. Create
	id, err := svc.Create(&CreateCollectionInput{
		Name:    "Workflow API",
		BaseURL: "https://workflow.example.com",
		Requests: []Request{
			{ID: "start", Method: "POST", Path: "/workflow/start"},
		},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if id != "workflow-api" {
		t.Errorf("id = %q, want workflow-api", id)
	}

	// 2. List — should have 1
	summaries, err := svc.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(summaries) != 1 {
		t.Fatalf("List len = %d, want 1", len(summaries))
	}
	if summaries[0].RequestCount != 1 {
		t.Errorf("RequestCount = %d, want 1", summaries[0].RequestCount)
	}

	// 3. Get
	c, err := svc.Get(id)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if c.Name != "Workflow API" {
		t.Errorf("Name = %q", c.Name)
	}

	// 4. FindRequest
	resolved, err := svc.FindRequest(id, "start")
	if err != nil {
		t.Fatalf("FindRequest: %v", err)
	}
	if resolved.URL != "https://workflow.example.com/workflow/start" {
		t.Errorf("URL = %q", resolved.URL)
	}

	// 5. Update
	newName := "Updated Workflow"
	updated, err := svc.Update(id, &UpdateCollectionInput{Name: &newName})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Name != "Updated Workflow" {
		t.Errorf("Updated Name = %q", updated.Name)
	}

	// 6. Delete
	if err := svc.Delete(id); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// 7. Verify file gone
	yamlPath := filepath.Join(dir, id+".yaml")
	if _, err := os.Stat(yamlPath); !os.IsNotExist(err) {
		t.Errorf("expected file to be deleted, got err: %v", err)
	}

	// 8. Get returns not found
	_, err = svc.Get(id)
	if !IsDomainError(err, "COLLECTION_NOT_FOUND") {
		t.Errorf("expected COLLECTION_NOT_FOUND after delete, got: %v", err)
	}
}
