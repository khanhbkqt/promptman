package request

import (
	"fmt"
	"net/http"

	"github.com/khanhnguyen/promptman/internal/collection"
	"github.com/khanhnguyen/promptman/pkg/variable"
)

// applyAuth injects authentication headers into the HTTP request based on the
// AuthConfig. All auth field values are resolved with the provided variable map.
// If auth is nil or has an empty type, this is a no-op.
func applyAuth(req *http.Request, auth *collection.AuthConfig, vars map[string]any, opts variable.Options) error {
	if auth == nil || auth.Type == "" {
		return nil
	}

	switch auth.Type {
	case "bearer":
		if auth.Bearer == nil {
			return nil
		}
		token, err := variable.Resolve(auth.Bearer.Token, vars, opts)
		if err != nil {
			return fmt.Errorf("resolving bearer token: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+token)

	case "basic":
		if auth.Basic == nil {
			return nil
		}
		user, err := variable.Resolve(auth.Basic.Username, vars, opts)
		if err != nil {
			return fmt.Errorf("resolving basic auth username: %w", err)
		}
		pass, err := variable.Resolve(auth.Basic.Password, vars, opts)
		if err != nil {
			return fmt.Errorf("resolving basic auth password: %w", err)
		}
		req.SetBasicAuth(user, pass)

	case "api-key":
		if auth.APIKey == nil {
			return nil
		}
		key, err := variable.Resolve(auth.APIKey.Key, vars, opts)
		if err != nil {
			return fmt.Errorf("resolving api-key key: %w", err)
		}
		val, err := variable.Resolve(auth.APIKey.Value, vars, opts)
		if err != nil {
			return fmt.Errorf("resolving api-key value: %w", err)
		}
		req.Header.Set(key, val)
	}

	return nil
}
