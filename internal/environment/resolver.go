package environment

import (
	"github.com/khanhnguyen/promptman/pkg/variable"
)

// Resolve resolves all {{variable}} references in a template string using the
// currently active environment's merged variable map (variables + resolved secrets).
//
// Optional extraScopes are merged after the environment variables, with later
// scopes overriding earlier ones. This allows M3 (request execution) to layer
// collection defaults and request overrides on top.
//
// Returns ErrEnvNotSet if no active environment has been configured.
func (s *Service) Resolve(template string, extraScopes ...map[string]any) (string, error) {
	name, err := s.GetActive()
	if err != nil {
		return "", err
	}
	return s.ResolveWith(template, name, extraScopes...)
}

// ResolveWith resolves all {{variable}} references in a template string using
// a specific (non-active) environment by name.
//
// It loads the environment via GetRaw (which resolves $ENV{} secrets),
// merges environment variables and resolved secrets into a single scope,
// then applies any extraScopes on top before delegating to pkg/variable.Resolve.
func (s *Service) ResolveWith(template string, envName string, extraScopes ...map[string]any) (string, error) {
	env, err := s.GetRaw(envName)
	if err != nil {
		return "", err
	}

	// Convert resolved secrets (map[string]string) to map[string]any for merging.
	secretScope := make(map[string]any, len(env.Secrets))
	for k, v := range env.Secrets {
		secretScope[k] = v
	}

	// Build the scope chain: env variables (lowest) → secrets → extra scopes (highest).
	scopes := make([]map[string]any, 0, 2+len(extraScopes))
	scopes = append(scopes, env.Variables, secretScope)
	scopes = append(scopes, extraScopes...)

	merged := variable.MergeScopes(scopes...)

	return variable.Resolve(template, merged)
}
