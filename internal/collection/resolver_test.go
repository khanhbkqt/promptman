package collection

import (
	"testing"

	"github.com/khanhnguyen/promptman/pkg/envelope"
)

// intPtr is a test helper that returns a pointer to an int.
func intPtr(v int) *int { return &v }

// baseCollection returns a minimal collection for use in tests.
// Callers can modify fields before passing to Resolve.
func baseCollection() *Collection {
	return &Collection{
		Name:    "Test API",
		BaseURL: "https://api.example.com",
		Defaults: &RequestDefaults{
			Headers: map[string]string{
				"Accept":       "application/json",
				"Content-Type": "application/json",
			},
			Timeout: intPtr(30000),
		},
		Auth: &AuthConfig{
			Type:   "bearer",
			Bearer: &BearerAuth{Token: "{{authToken}}"},
		},
		Requests: []Request{
			{ID: "health", Method: "GET", Path: "/health"},
		},
		Folders: []Folder{
			{
				ID:   "admin",
				Name: "Admin",
				Auth: &AuthConfig{
					Type:   "api-key",
					APIKey: &APIKeyAuth{Key: "X-Admin-Key", Value: "{{adminKey}}"},
				},
				Defaults: &RequestDefaults{
					Headers: map[string]string{
						"X-Admin": "true",
					},
					Timeout: intPtr(60000),
				},
				Requests: []Request{
					{ID: "list-admins", Method: "GET", Path: "/admin/users"},
					{
						ID:     "create-admin",
						Method: "POST",
						Path:   "/admin/users",
						Headers: map[string]string{
							"X-Idempotency": "abc-123",
						},
						Body: &RequestBody{
							Type:    "json",
							Content: map[string]any{"role": "admin"},
						},
					},
				},
				Folders: []Folder{
					{
						ID:   "settings",
						Name: "Settings",
						Requests: []Request{
							{ID: "list-configs", Method: "GET", Path: "/admin/settings"},
						},
					},
				},
			},
		},
	}
}

func TestResolve_SimpleInherit(t *testing.T) {
	c := baseCollection()
	got, err := Resolve(c, "health")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.URL != "https://api.example.com/health" {
		t.Errorf("URL = %q, want %q", got.URL, "https://api.example.com/health")
	}
	if got.Method != "GET" {
		t.Errorf("Method = %q, want GET", got.Method)
	}
	// Should inherit collection defaults headers.
	if got.Headers["Accept"] != "application/json" {
		t.Errorf("Accept header = %q, want application/json", got.Headers["Accept"])
	}
	if got.Headers["Content-Type"] != "application/json" {
		t.Errorf("Content-Type header = %q, want application/json", got.Headers["Content-Type"])
	}
	// Should inherit collection auth.
	if got.Auth == nil || got.Auth.Type != "bearer" {
		t.Errorf("Auth = %+v, want bearer auth", got.Auth)
	}
	// Should inherit collection timeout.
	if got.Timeout == nil || *got.Timeout != 30000 {
		t.Errorf("Timeout = %v, want 30000", got.Timeout)
	}
}

func TestResolve_SingleFolder(t *testing.T) {
	c := baseCollection()
	got, err := Resolve(c, "admin/list-admins")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.URL != "https://api.example.com/admin/users" {
		t.Errorf("URL = %q, want %q", got.URL, "https://api.example.com/admin/users")
	}
	// Auth: folder overrides collection (api-key instead of bearer).
	if got.Auth == nil || got.Auth.Type != "api-key" {
		t.Fatalf("Auth.Type = %v, want api-key", got.Auth)
	}
	if got.Auth.APIKey.Key != "X-Admin-Key" {
		t.Errorf("Auth.APIKey.Key = %q, want X-Admin-Key", got.Auth.APIKey.Key)
	}
	// Timeout: folder overrides collection (60000 instead of 30000).
	if got.Timeout == nil || *got.Timeout != 60000 {
		t.Errorf("Timeout = %v, want 60000", got.Timeout)
	}
	// Headers: collection + folder merged.
	if got.Headers["Accept"] != "application/json" {
		t.Errorf("Accept header should be inherited from collection")
	}
	if got.Headers["X-Admin"] != "true" {
		t.Errorf("X-Admin header should come from folder defaults")
	}
}

func TestResolve_NestedFolders(t *testing.T) {
	c := baseCollection()
	got, err := Resolve(c, "admin/settings/list-configs")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.URL != "https://api.example.com/admin/settings" {
		t.Errorf("URL = %q, want %q", got.URL, "https://api.example.com/admin/settings")
	}
	// Auth should be inherited from admin folder (settings has no auth override).
	if got.Auth == nil || got.Auth.Type != "api-key" {
		t.Errorf("Auth.Type = %v, want api-key (inherited from admin folder)", got.Auth)
	}
	// Timeout should be inherited from admin folder (settings has no timeout).
	if got.Timeout == nil || *got.Timeout != 60000 {
		t.Errorf("Timeout = %v, want 60000 (inherited from admin folder)", got.Timeout)
	}
	// Headers from collection + admin folder should be present.
	if got.Headers["Accept"] != "application/json" {
		t.Errorf("Accept header missing (should inherit from collection)")
	}
	if got.Headers["X-Admin"] != "true" {
		t.Errorf("X-Admin header missing (should inherit from admin folder)")
	}
}

func TestResolve_HeaderMerge(t *testing.T) {
	c := baseCollection()
	got, err := Resolve(c, "admin/create-admin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Collection headers (Accept, Content-Type) + folder header (X-Admin) + request header (X-Idempotency).
	expected := map[string]string{
		"Accept":        "application/json",
		"Content-Type":  "application/json",
		"X-Admin":       "true",
		"X-Idempotency": "abc-123",
	}
	for k, v := range expected {
		if got.Headers[k] != v {
			t.Errorf("Header[%q] = %q, want %q", k, got.Headers[k], v)
		}
	}
	if len(got.Headers) != len(expected) {
		t.Errorf("got %d headers, want %d", len(got.Headers), len(expected))
	}
}

func TestResolve_HeaderOverride(t *testing.T) {
	c := &Collection{
		Name: "Override Test",
		Defaults: &RequestDefaults{
			Headers: map[string]string{
				"Accept":       "text/html",
				"X-Request-ID": "parent-id",
			},
		},
		Requests: []Request{
			{
				ID:     "override",
				Method: "GET",
				Path:   "/test",
				Headers: map[string]string{
					"Accept": "application/xml", // override parent
				},
			},
		},
	}

	got, err := Resolve(c, "override")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Headers["Accept"] != "application/xml" {
		t.Errorf("Accept = %q, want application/xml (child override)", got.Headers["Accept"])
	}
	// Parent header should be preserved if not overridden.
	if got.Headers["X-Request-ID"] != "parent-id" {
		t.Errorf("X-Request-ID = %q, want parent-id (preserved)", got.Headers["X-Request-ID"])
	}
}

func TestResolve_TimeoutOverride(t *testing.T) {
	c := &Collection{
		Name: "Timeout Test",
		Defaults: &RequestDefaults{
			Timeout: intPtr(10000),
		},
		Folders: []Folder{
			{
				ID:   "slow",
				Name: "Slow",
				Defaults: &RequestDefaults{
					Timeout: intPtr(60000), // folder overrides collection
				},
				Requests: []Request{
					{
						ID:      "fast-req",
						Method:  "GET",
						Path:    "/fast",
						Timeout: intPtr(5000), // request overrides folder
					},
					{
						ID:     "inherit-req",
						Method: "GET",
						Path:   "/inherit",
						// no timeout → inherits from folder
					},
				},
			},
		},
	}

	t.Run("request overrides folder", func(t *testing.T) {
		got, err := Resolve(c, "slow/fast-req")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Timeout == nil || *got.Timeout != 5000 {
			t.Errorf("Timeout = %v, want 5000", got.Timeout)
		}
	})

	t.Run("inherits from folder", func(t *testing.T) {
		got, err := Resolve(c, "slow/inherit-req")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Timeout == nil || *got.Timeout != 60000 {
			t.Errorf("Timeout = %v, want 60000", got.Timeout)
		}
	})
}

func TestResolve_TimeoutClear(t *testing.T) {
	zero := 0
	c := &Collection{
		Name: "Timeout Clear Test",
		Defaults: &RequestDefaults{
			Timeout: intPtr(30000),
		},
		Requests: []Request{
			{
				ID:      "no-timeout",
				Method:  "GET",
				Path:    "/test",
				Timeout: &zero, // explicit 0 clears inherited
			},
		},
	}

	got, err := Resolve(c, "no-timeout")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Timeout == nil {
		t.Fatal("Timeout should not be nil — explicit 0 is still set")
	}
	if *got.Timeout != 0 {
		t.Errorf("Timeout = %d, want 0 (explicitly cleared)", *got.Timeout)
	}
}

func TestResolve_AuthOverride(t *testing.T) {
	c := &Collection{
		Name: "Auth Override",
		Auth: &AuthConfig{
			Type:   "bearer",
			Bearer: &BearerAuth{Token: "global-token"},
		},
		Folders: []Folder{
			{
				ID:   "admin",
				Name: "Admin",
				Auth: &AuthConfig{
					Type:   "api-key",
					APIKey: &APIKeyAuth{Key: "X-Key", Value: "admin-key"},
				},
				Requests: []Request{
					{
						ID:     "with-basic",
						Method: "GET",
						Path:   "/test",
						Auth: &AuthConfig{
							Type:  "basic",
							Basic: &BasicAuth{Username: "user", Password: "pass"},
						},
					},
					{
						ID:     "inherit-folder",
						Method: "GET",
						Path:   "/test2",
						// no auth → inherits api-key from folder
					},
				},
			},
		},
		Requests: []Request{
			{
				ID:     "inherit-collection",
				Method: "GET",
				Path:   "/test3",
				// no auth → inherits bearer from collection
			},
		},
	}

	t.Run("request overrides folder auth", func(t *testing.T) {
		got, err := Resolve(c, "admin/with-basic")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Auth == nil || got.Auth.Type != "basic" {
			t.Errorf("Auth.Type = %v, want basic", got.Auth)
		}
	})

	t.Run("inherits from folder", func(t *testing.T) {
		got, err := Resolve(c, "admin/inherit-folder")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Auth == nil || got.Auth.Type != "api-key" {
			t.Errorf("Auth.Type = %v, want api-key (from folder)", got.Auth)
		}
	})

	t.Run("inherits from collection", func(t *testing.T) {
		got, err := Resolve(c, "inherit-collection")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Auth == nil || got.Auth.Type != "bearer" {
			t.Errorf("Auth.Type = %v, want bearer (from collection)", got.Auth)
		}
	})
}

func TestResolve_AllAuthTypes(t *testing.T) {
	tests := []struct {
		name     string
		auth     *AuthConfig
		wantType string
	}{
		{
			name:     "bearer",
			auth:     &AuthConfig{Type: "bearer", Bearer: &BearerAuth{Token: "t"}},
			wantType: "bearer",
		},
		{
			name:     "basic",
			auth:     &AuthConfig{Type: "basic", Basic: &BasicAuth{Username: "u", Password: "p"}},
			wantType: "basic",
		},
		{
			name:     "api-key",
			auth:     &AuthConfig{Type: "api-key", APIKey: &APIKeyAuth{Key: "k", Value: "v"}},
			wantType: "api-key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Collection{
				Name: "Auth Types",
				Auth: tt.auth,
				Requests: []Request{
					{ID: "req", Method: "GET", Path: "/test"},
				},
			}
			got, err := Resolve(c, "req")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Auth == nil || got.Auth.Type != tt.wantType {
				t.Errorf("Auth.Type = %v, want %q", got.Auth, tt.wantType)
			}
		})
	}
}

func TestResolve_NoAuth(t *testing.T) {
	c := &Collection{
		Name: "No Auth",
		Requests: []Request{
			{ID: "req", Method: "GET", Path: "/test"},
		},
	}
	got, err := Resolve(c, "req")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Auth != nil {
		t.Errorf("Auth = %+v, want nil", got.Auth)
	}
}

func TestResolve_URLResolution(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
		path    string
		want    string
	}{
		{"base with trailing slash", "https://api.example.com/", "/v1/health", "https://api.example.com/v1/health"},
		{"base without trailing slash", "https://api.example.com", "/v1/health", "https://api.example.com/v1/health"},
		{"path without leading slash", "https://api.example.com", "v1/health", "https://api.example.com/v1/health"},
		{"both slashes", "https://api.example.com/", "/health", "https://api.example.com/health"},
		{"empty base", "", "/health", "/health"},
		{"empty path", "https://api.example.com", "", "https://api.example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Collection{
				Name:    "URL Test",
				BaseURL: tt.baseURL,
				Requests: []Request{
					{ID: "req", Method: "GET", Path: tt.path},
				},
			}
			got, err := Resolve(c, "req")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.URL != tt.want {
				t.Errorf("URL = %q, want %q", got.URL, tt.want)
			}
		})
	}
}

func TestResolve_InvalidPath(t *testing.T) {
	c := baseCollection()

	tests := []struct {
		name string
		path string
	}{
		{"non-existent request", "does-not-exist"},
		{"non-existent folder", "bogus/health"},
		{"wrong nested path", "admin/bogus/list-configs"},
		{"empty path", ""},
		{"whitespace path", "   "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Resolve(c, tt.path)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !IsDomainError(err, envelope.CodeRequestNotFound) {
				t.Errorf("error code = %v, want REQUEST_NOT_FOUND", err)
			}
		})
	}
}

func TestResolve_NoDefaults(t *testing.T) {
	c := &Collection{
		Name: "No Defaults",
		Requests: []Request{
			{ID: "bare", Method: "DELETE", Path: "/items/1"},
		},
	}
	got, err := Resolve(c, "bare")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Method != "DELETE" {
		t.Errorf("Method = %q, want DELETE", got.Method)
	}
	if len(got.Headers) != 0 {
		t.Errorf("Headers = %v, want empty", got.Headers)
	}
	if got.Timeout != nil {
		t.Errorf("Timeout = %v, want nil", got.Timeout)
	}
	if got.Auth != nil {
		t.Errorf("Auth = %v, want nil", got.Auth)
	}
}

func TestResolve_BodyNotInherited(t *testing.T) {
	c := baseCollection()
	got, err := Resolve(c, "admin/create-admin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Body == nil {
		t.Fatal("Body should not be nil")
	}
	if got.Body.Type != "json" {
		t.Errorf("Body.Type = %q, want json", got.Body.Type)
	}

	// A request without a body should have nil Body.
	got2, err := Resolve(c, "health")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got2.Body != nil {
		t.Errorf("Body = %v, want nil for health request", got2.Body)
	}
}
