package collection

// Collection represents a top-level API collection stored as a single YAML file.
// It groups requests, folders, default settings, and shared authentication.
type Collection struct {
	Name     string           `yaml:"name" json:"name"`
	BaseURL  string           `yaml:"baseUrl,omitempty" json:"baseUrl,omitempty"`
	Defaults *RequestDefaults `yaml:"defaults,omitempty" json:"defaults,omitempty"`
	Auth     *AuthConfig      `yaml:"auth,omitempty" json:"auth,omitempty"`
	Requests []Request        `yaml:"requests,omitempty" json:"requests,omitempty"`
	Folders  []Folder         `yaml:"folders,omitempty" json:"folders,omitempty"`
}

// CollectionSummary is a lightweight projection of a Collection used in listings.
type CollectionSummary struct {
	ID           string `yaml:"-" json:"id"`
	Name         string `yaml:"name" json:"name"`
	RequestCount int    `yaml:"-" json:"requestCount"`
}

// Request represents a single HTTP request within a collection or folder.
type Request struct {
	ID      string            `yaml:"id" json:"id"`
	Method  string            `yaml:"method" json:"method"`
	Path    string            `yaml:"path" json:"path"`
	Headers map[string]string `yaml:"headers,omitempty" json:"headers,omitempty"`
	Body    *RequestBody      `yaml:"body,omitempty" json:"body,omitempty"`
	Auth    *AuthConfig       `yaml:"auth,omitempty" json:"auth,omitempty"`
	Timeout *int              `yaml:"timeout,omitempty" json:"timeout,omitempty"`
}

// Folder is a logical grouping of requests within a collection.
// Folders can nest arbitrarily and may override defaults and auth.
type Folder struct {
	ID       string           `yaml:"id" json:"id"`
	Name     string           `yaml:"name" json:"name"`
	Auth     *AuthConfig      `yaml:"auth,omitempty" json:"auth,omitempty"`
	Defaults *RequestDefaults `yaml:"defaults,omitempty" json:"defaults,omitempty"`
	Requests []Request        `yaml:"requests,omitempty" json:"requests,omitempty"`
	Folders  []Folder         `yaml:"folders,omitempty" json:"folders,omitempty"`
}

// RequestDefaults holds inheritable default values for requests.
// Values cascade from collection → folder → request; child overrides parent.
type RequestDefaults struct {
	Headers map[string]string `yaml:"headers,omitempty" json:"headers,omitempty"`
	Timeout *int              `yaml:"timeout,omitempty" json:"timeout,omitempty"`
}

// AuthConfig describes the authentication block.
// Exactly one of Bearer, Basic, or APIKey should be set, matching Type.
type AuthConfig struct {
	Type   string      `yaml:"type" json:"type"` // "bearer", "basic", or "api-key"
	Bearer *BearerAuth `yaml:"bearer,omitempty" json:"bearer,omitempty"`
	Basic  *BasicAuth  `yaml:"basic,omitempty" json:"basic,omitempty"`
	APIKey *APIKeyAuth `yaml:"apiKey,omitempty" json:"apiKey,omitempty"`
}

// BearerAuth holds a bearer token value (may contain {{variables}}).
type BearerAuth struct {
	Token string `yaml:"token" json:"token"`
}

// BasicAuth holds username/password credentials (may contain {{variables}}).
type BasicAuth struct {
	Username string `yaml:"username" json:"username"`
	Password string `yaml:"password" json:"password"`
}

// APIKeyAuth holds an API key sent as a header (may contain {{variables}}).
type APIKeyAuth struct {
	Key   string `yaml:"key" json:"key"`
	Value string `yaml:"value" json:"value"`
}

// RequestBody describes the body of an HTTP request.
type RequestBody struct {
	Type    string `yaml:"type" json:"type"` // e.g. "json", "form", "raw"
	Content any    `yaml:"content,omitempty" json:"content,omitempty"`
}

// ResolvedRequest is the output of the defaults resolution chain.
// It contains a fully merged request ready for execution, with URL assembled
// from baseUrl + path, headers merged from the inheritance chain, and the
// winning auth and timeout values.
type ResolvedRequest struct {
	URL     string            // baseUrl + request.path
	Method  string            // HTTP method (GET, POST, etc.)
	Headers map[string]string // merged headers: collection → folder(s) → request
	Body    *RequestBody      // request body (not inherited)
	Auth    *AuthConfig       // resolved auth (child overrides parent)
	Timeout *int              // resolved timeout (child overrides parent)
}

// CreateCollectionInput holds the data needed to create a new collection.
// Name is required; all other fields are optional.
type CreateCollectionInput struct {
	Name     string
	BaseURL  string
	Defaults *RequestDefaults
	Auth     *AuthConfig
	Requests []Request
	Folders  []Folder
}

// UpdateCollectionInput holds optional fields for partial collection updates.
// Only non-nil fields are applied; nil fields are left unchanged.
type UpdateCollectionInput struct {
	Name     *string          `json:"name,omitempty"`
	BaseURL  *string          `json:"baseUrl,omitempty"`
	Defaults *RequestDefaults `json:"defaults,omitempty"`
	Auth     *AuthConfig      `json:"auth,omitempty"`
	Requests *[]Request       `json:"requests,omitempty"`
	Folders  *[]Folder        `json:"folders,omitempty"`
}
