package collection

import (
	"fmt"
	"regexp"
	"strings"
)

// slugRe matches characters that are NOT alphanumeric or hyphens.
var slugRe = regexp.MustCompile(`[^a-z0-9-]+`)

// Service provides business-level operations on collections.
// It wires together a Repository (persistence) and the Resolve function
// (defaults/auth inheritance) into a single public API.
type Service struct {
	repo Repository
}

// NewService creates a Service backed by the given repository.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// List returns a summary of every collection on disk.
func (s *Service) List() ([]CollectionSummary, error) {
	return s.repo.List()
}

// Get loads a single collection by ID, including all requests and folders.
func (s *Service) Get(id string) (*Collection, error) {
	return s.repo.Get(id)
}

// Create validates the input, generates a slug-based ID from the name,
// persists the collection, and returns the generated ID.
func (s *Service) Create(input *CreateCollectionInput) (string, error) {
	if input == nil {
		return "", ErrInvalidRequest.Wrap("create input is nil")
	}

	c := &Collection{
		Name:     input.Name,
		BaseURL:  input.BaseURL,
		Defaults: input.Defaults,
		Auth:     input.Auth,
		Requests: input.Requests,
		Folders:  input.Folders,
	}

	if err := ValidateCollection(c); err != nil {
		return "", err
	}

	id := generateID(input.Name)

	if err := s.repo.Save(id, c); err != nil {
		return "", fmt.Errorf("create collection: %w", err)
	}

	return id, nil
}

// Update loads an existing collection, merges the non-nil fields from input,
// validates the result, and saves it back.
func (s *Service) Update(id string, input *UpdateCollectionInput) (*Collection, error) {
	if input == nil {
		return nil, ErrInvalidRequest.Wrap("update input is nil")
	}

	c, err := s.repo.Get(id)
	if err != nil {
		return nil, err
	}

	// Merge non-nil fields.
	if input.Name != nil {
		c.Name = *input.Name
	}
	if input.BaseURL != nil {
		c.BaseURL = *input.BaseURL
	}
	if input.Defaults != nil {
		c.Defaults = input.Defaults
	}
	if input.Auth != nil {
		c.Auth = input.Auth
	}
	if input.Requests != nil {
		c.Requests = *input.Requests
	}
	if input.Folders != nil {
		c.Folders = *input.Folders
	}

	if err := ValidateCollection(c); err != nil {
		return nil, err
	}

	if err := s.repo.Save(id, c); err != nil {
		return nil, fmt.Errorf("update collection %q: %w", id, err)
	}

	return c, nil
}

// Delete removes a collection by ID.
func (s *Service) Delete(id string) error {
	return s.repo.Delete(id)
}

// FindRequest loads a collection and resolves a request within it using the
// defaults inheritance chain. requestPath is a slash-separated path like
// "admin/list-admins".
func (s *Service) FindRequest(collectionID, requestPath string) (*ResolvedRequest, error) {
	c, err := s.repo.Get(collectionID)
	if err != nil {
		return nil, err
	}
	return Resolve(c, requestPath)
}

// generateID creates a URL-safe slug from a collection name.
// Example: "Users API" → "users-api", "My  Cool--API!" → "my-cool-api".
func generateID(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	s = slugRe.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")

	// Collapse consecutive hyphens.
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}

	if s == "" {
		return "collection"
	}
	return s
}
