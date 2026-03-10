package collection

// Collection represents a top-level API collection stored as a single YAML file.
// It groups requests, folders, default settings, and shared authentication.
type Collection struct {
	Name     string           `yaml:"name"`
	BaseURL  string           `yaml:"baseUrl,omitempty"`
	Defaults *RequestDefaults `yaml:"defaults,omitempty"`
	Auth     *AuthConfig      `yaml:"auth,omitempty"`
	Requests []Request        `yaml:"requests,omitempty"`
	Folders  []Folder         `yaml:"folders,omitempty"`
}

// CollectionSummary is a lightweight projection of a Collection used in listings.
type CollectionSummary struct {
	ID           string `yaml:"-"` // derived from filename, not serialized
	Name         string `yaml:"name"`
	RequestCount int    `yaml:"-"` // computed at load time
}

// Request represents a single HTTP request within a collection or folder.
type Request struct {
	ID      string            `yaml:"id"`
	Method  string            `yaml:"method"`
	Path    string            `yaml:"path"`
	Headers map[string]string `yaml:"headers,omitempty"`
	Body    *RequestBody      `yaml:"body,omitempty"`
	Auth    *AuthConfig       `yaml:"auth,omitempty"`
	Timeout *int              `yaml:"timeout,omitempty"`
}

// Folder is a logical grouping of requests within a collection.
// Folders can nest arbitrarily and may override defaults and auth.
type Folder struct {
	ID       string           `yaml:"id"`
	Name     string           `yaml:"name"`
	Auth     *AuthConfig      `yaml:"auth,omitempty"`
	Defaults *RequestDefaults `yaml:"defaults,omitempty"`
	Requests []Request        `yaml:"requests,omitempty"`
	Folders  []Folder         `yaml:"folders,omitempty"`
}

// RequestDefaults holds inheritable default values for requests.
// Values cascade from collection → folder → request; child overrides parent.
type RequestDefaults struct {
	Headers map[string]string `yaml:"headers,omitempty"`
	Timeout *int              `yaml:"timeout,omitempty"`
}

// AuthConfig describes the authentication block.
// Exactly one of Bearer, Basic, or APIKey should be set, matching Type.
type AuthConfig struct {
	Type   string      `yaml:"type"` // "bearer", "basic", or "api-key"
	Bearer *BearerAuth `yaml:"bearer,omitempty"`
	Basic  *BasicAuth  `yaml:"basic,omitempty"`
	APIKey *APIKeyAuth `yaml:"apiKey,omitempty"`
}

// BearerAuth holds a bearer token value (may contain {{variables}}).
type BearerAuth struct {
	Token string `yaml:"token"`
}

// BasicAuth holds username/password credentials (may contain {{variables}}).
type BasicAuth struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

// APIKeyAuth holds an API key sent as a header (may contain {{variables}}).
type APIKeyAuth struct {
	Key   string `yaml:"key"`
	Value string `yaml:"value"`
}

// RequestBody describes the body of an HTTP request.
type RequestBody struct {
	Type    string `yaml:"type"` // e.g. "json", "form", "raw"
	Content any    `yaml:"content,omitempty"`
}
