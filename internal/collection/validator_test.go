package collection

import (
	"strings"
	"testing"
)

// --- helpers ---

// validMinimalCollection returns the smallest valid Collection.
func validMinimalCollection() *Collection {
	return &Collection{
		Name: "test-api",
	}
}

// validFullCollection returns a Collection exercising most fields.
func validFullCollection() *Collection {
	timeout := 30
	return &Collection{
		Name:    "full-api",
		BaseURL: "https://api.example.com",
		Defaults: &RequestDefaults{
			Headers: map[string]string{"Accept": "application/json"},
			Timeout: &timeout,
		},
		Auth: &AuthConfig{
			Type:   "bearer",
			Bearer: &BearerAuth{Token: "{{token}}"},
		},
		Requests: []Request{
			{ID: "get-users", Method: "GET", Path: "/users"},
			{ID: "create-user", Method: "POST", Path: "/users",
				Body: &RequestBody{Type: "json", Content: map[string]string{"name": "test"}},
			},
		},
		Folders: []Folder{
			{
				ID:   "admin",
				Name: "Admin",
				Auth: &AuthConfig{
					Type:  "basic",
					Basic: &BasicAuth{Username: "admin", Password: "{{pw}}"},
				},
				Requests: []Request{
					{ID: "list-logs", Method: "GET", Path: "/admin/logs"},
				},
				Folders: []Folder{
					{
						ID:   "nested",
						Name: "Nested",
						Auth: &AuthConfig{
							Type:   "api-key",
							APIKey: &APIKeyAuth{Key: "X-Key", Value: "{{key}}"},
						},
						Requests: []Request{
							{ID: "deep-req", Method: "DELETE", Path: "/admin/deep"},
						},
					},
				},
			},
		},
	}
}

// --- ValidateCollection tests ---

func TestValidateCollection_ValidMinimal(t *testing.T) {
	if err := ValidateCollection(validMinimalCollection()); err != nil {
		t.Errorf("expected valid, got: %v", err)
	}
}

func TestValidateCollection_ValidFull(t *testing.T) {
	if err := ValidateCollection(validFullCollection()); err != nil {
		t.Errorf("expected valid, got: %v", err)
	}
}

func TestValidateCollection_MissingName(t *testing.T) {
	c := &Collection{Name: ""}
	err := ValidateCollection(c)
	if err == nil {
		t.Fatal("expected error for missing name")
	}
	assertContains(t, err.Error(), "collection name is required")
}

func TestValidateCollection_WhitespaceName(t *testing.T) {
	c := &Collection{Name: "   "}
	err := ValidateCollection(c)
	if err == nil {
		t.Fatal("expected error for whitespace-only name")
	}
	assertContains(t, err.Error(), "collection name is required")
}

// --- Request validation ---

func TestValidateCollection_RequestMissingID(t *testing.T) {
	c := validMinimalCollection()
	c.Requests = []Request{{Method: "GET", Path: "/test"}}
	err := ValidateCollection(c)
	if err == nil {
		t.Fatal("expected error for missing request id")
	}
	assertContains(t, err.Error(), "request id is required")
}

func TestValidateCollection_RequestMissingMethod(t *testing.T) {
	c := validMinimalCollection()
	c.Requests = []Request{{ID: "req1", Path: "/test"}}
	err := ValidateCollection(c)
	if err == nil {
		t.Fatal("expected error for missing method")
	}
	assertContains(t, err.Error(), "request method is required")
}

func TestValidateCollection_RequestInvalidMethod(t *testing.T) {
	c := validMinimalCollection()
	c.Requests = []Request{{ID: "req1", Method: "INVALID", Path: "/test"}}
	err := ValidateCollection(c)
	if err == nil {
		t.Fatal("expected error for invalid method")
	}
	assertContains(t, err.Error(), "invalid method")
}

func TestValidateCollection_AllValidMethods(t *testing.T) {
	methods := []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}
	for _, m := range methods {
		t.Run(m, func(t *testing.T) {
			c := validMinimalCollection()
			c.Requests = []Request{{ID: "r1", Method: m, Path: "/test"}}
			if err := ValidateCollection(c); err != nil {
				t.Errorf("method %q should be valid, got: %v", m, err)
			}
		})
	}
}

// --- Auth validation ---

func TestValidateCollection_AuthMissingType(t *testing.T) {
	c := validMinimalCollection()
	c.Auth = &AuthConfig{}
	err := ValidateCollection(c)
	if err == nil {
		t.Fatal("expected error for missing auth type")
	}
	assertContains(t, err.Error(), "auth type is required")
}

func TestValidateCollection_AuthInvalidType(t *testing.T) {
	c := validMinimalCollection()
	c.Auth = &AuthConfig{Type: "oauth2"}
	err := ValidateCollection(c)
	if err == nil {
		t.Fatal("expected error for invalid auth type")
	}
	assertContains(t, err.Error(), "invalid auth type")
}

func TestValidateCollection_BearerMissingConfig(t *testing.T) {
	c := validMinimalCollection()
	c.Auth = &AuthConfig{Type: "bearer"} // no Bearer field
	err := ValidateCollection(c)
	if err == nil {
		t.Fatal("expected error for bearer without config")
	}
	assertContains(t, err.Error(), "bearer config is missing")
}

func TestValidateCollection_BasicMissingConfig(t *testing.T) {
	c := validMinimalCollection()
	c.Auth = &AuthConfig{Type: "basic"} // no Basic field
	err := ValidateCollection(c)
	if err == nil {
		t.Fatal("expected error for basic without config")
	}
	assertContains(t, err.Error(), "basic config is missing")
}

func TestValidateCollection_APIKeyMissingConfig(t *testing.T) {
	c := validMinimalCollection()
	c.Auth = &AuthConfig{Type: "api-key"} // no APIKey field
	err := ValidateCollection(c)
	if err == nil {
		t.Fatal("expected error for api-key without config")
	}
	assertContains(t, err.Error(), "apiKey config is missing")
}

func TestValidateCollection_ValidBearerAuth(t *testing.T) {
	c := validMinimalCollection()
	c.Auth = &AuthConfig{Type: "bearer", Bearer: &BearerAuth{Token: "tok"}}
	if err := ValidateCollection(c); err != nil {
		t.Errorf("expected valid, got: %v", err)
	}
}

func TestValidateCollection_ValidBasicAuth(t *testing.T) {
	c := validMinimalCollection()
	c.Auth = &AuthConfig{Type: "basic", Basic: &BasicAuth{Username: "u", Password: "p"}}
	if err := ValidateCollection(c); err != nil {
		t.Errorf("expected valid, got: %v", err)
	}
}

func TestValidateCollection_ValidAPIKeyAuth(t *testing.T) {
	c := validMinimalCollection()
	c.Auth = &AuthConfig{Type: "api-key", APIKey: &APIKeyAuth{Key: "X-Key", Value: "val"}}
	if err := ValidateCollection(c); err != nil {
		t.Errorf("expected valid, got: %v", err)
	}
}

// --- Auth on requests ---

func TestValidateCollection_RequestAuthInvalid(t *testing.T) {
	c := validMinimalCollection()
	c.Requests = []Request{{
		ID:     "r1",
		Method: "GET",
		Path:   "/test",
		Auth:   &AuthConfig{Type: "bearer"}, // missing Bearer
	}}
	err := ValidateCollection(c)
	if err == nil {
		t.Fatal("expected error for request-level auth without config")
	}
	assertContains(t, err.Error(), "bearer config is missing")
}

// --- Folder validation ---

func TestValidateCollection_FolderMissingID(t *testing.T) {
	c := validMinimalCollection()
	c.Folders = []Folder{{Name: "Admin"}}
	err := ValidateCollection(c)
	if err == nil {
		t.Fatal("expected error for folder missing id")
	}
	assertContains(t, err.Error(), "folder id is required")
}

func TestValidateCollection_FolderMissingName(t *testing.T) {
	c := validMinimalCollection()
	c.Folders = []Folder{{ID: "f1"}}
	err := ValidateCollection(c)
	if err == nil {
		t.Fatal("expected error for folder missing name")
	}
	assertContains(t, err.Error(), "folder name is required")
}

func TestValidateCollection_FolderAuthInvalid(t *testing.T) {
	c := validMinimalCollection()
	c.Folders = []Folder{{
		ID:   "f1",
		Name: "Admin",
		Auth: &AuthConfig{Type: "invalid"},
	}}
	err := ValidateCollection(c)
	if err == nil {
		t.Fatal("expected error for folder invalid auth")
	}
	assertContains(t, err.Error(), "invalid auth type")
}

func TestValidateCollection_NestedFolderRequestInvalid(t *testing.T) {
	c := validMinimalCollection()
	c.Folders = []Folder{{
		ID:   "f1",
		Name: "Parent",
		Folders: []Folder{{
			ID:   "f2",
			Name: "Child",
			Requests: []Request{{
				ID:   "", // invalid
				Path: "/test",
			}},
		}},
	}}
	err := ValidateCollection(c)
	if err == nil {
		t.Fatal("expected error for nested folder request missing id")
	}
	assertContains(t, err.Error(), "request id is required")
}

// --- Multiple errors ---

func TestValidateCollection_MultipleErrors(t *testing.T) {
	c := &Collection{
		Name: "", // error 1
		Requests: []Request{
			{ID: "", Method: "INVALID"}, // errors 2 & 3
		},
	}
	err := ValidateCollection(c)
	if err == nil {
		t.Fatal("expected multiple errors")
	}
	ve, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected *ValidationError, got %T", err)
	}
	if len(ve.Errors) < 3 {
		t.Errorf("expected at least 3 errors, got %d: %v", len(ve.Errors), ve.Errors)
	}
}

func TestValidationError_ErrorFormat(t *testing.T) {
	ve := &ValidationError{Errors: []string{"err1", "err2"}}
	s := ve.Error()
	// Should contain the ErrInvalidRequest code and both messages
	assertContains(t, s, "err1")
	assertContains(t, s, "err2")
}

// --- helpers ---

func assertContains(t *testing.T, s, substr string) {
	t.Helper()
	if !strings.Contains(s, substr) {
		t.Errorf("expected %q to contain %q", s, substr)
	}
}
