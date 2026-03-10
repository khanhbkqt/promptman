package environment

import (
	"fmt"
)

// Service provides business-level operations on environments.
// It wires together a Repository (persistence) and validation
// into a single public API.
type Service struct {
	repo Repository
}

// NewService creates a Service backed by the given repository.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// List returns a summary of every environment on disk.
func (s *Service) List() ([]EnvSummary, error) {
	return s.repo.List()
}

// Get loads a single environment by name, merges .secrets.yaml overrides,
// resolves $ENV{} references, and masks all secret values as "***".
// Use this for API responses where secrets must not leak.
func (s *Service) Get(name string) (*Environment, error) {
	env, err := s.repo.GetWithSecrets(name)
	if err != nil {
		return nil, err
	}

	// Resolve $ENV{} references in secrets.
	resolved, err := ResolveSecrets(env.Secrets)
	if err != nil {
		return nil, err
	}

	// Mask all resolved secret values for safe output.
	env.Secrets = MaskSecrets(resolved)
	return env, nil
}

// GetRaw loads a single environment by name, merges .secrets.yaml overrides,
// and resolves $ENV{} references — returning the actual secret values.
// This is for internal use only (e.g., by the M3 request execution engine).
// GetRaw MUST NOT be exposed via the REST API.
func (s *Service) GetRaw(name string) (*Environment, error) {
	env, err := s.repo.GetWithSecrets(name)
	if err != nil {
		return nil, err
	}

	resolved, err := ResolveSecrets(env.Secrets)
	if err != nil {
		return nil, err
	}

	env.Secrets = resolved
	return env, nil
}

// Create validates the input, checks for duplicates, persists the environment,
// and returns the created environment.
func (s *Service) Create(input *CreateEnvInput) (*Environment, error) {
	if err := ValidateCreateInput(input); err != nil {
		return nil, err
	}

	// Check for duplicate name.
	if _, err := s.repo.Get(input.Name); err == nil {
		return nil, ErrEnvironmentExists.Wrapf("environment %q already exists", input.Name)
	}

	env := &Environment{
		Name:      input.Name,
		Variables: input.Variables,
		Secrets:   input.Secrets,
	}

	if err := s.repo.Save(input.Name, env); err != nil {
		return nil, fmt.Errorf("create environment: %w", err)
	}

	return env, nil
}

// Update loads an existing environment, merges the non-nil fields from input,
// validates the result, and saves it back.
func (s *Service) Update(name string, input *UpdateEnvInput) (*Environment, error) {
	if err := ValidateUpdateInput(input); err != nil {
		return nil, err
	}

	env, err := s.repo.Get(name)
	if err != nil {
		return nil, err
	}

	// Merge non-nil fields.
	if input.Name != nil {
		env.Name = *input.Name
	}
	if input.Variables != nil {
		env.Variables = *input.Variables
	}
	if input.Secrets != nil {
		env.Secrets = *input.Secrets
	}

	if err := s.repo.Save(name, env); err != nil {
		return nil, fmt.Errorf("update environment %q: %w", name, err)
	}

	return env, nil
}

// Delete removes an environment by name.
func (s *Service) Delete(name string) error {
	return s.repo.Delete(name)
}
