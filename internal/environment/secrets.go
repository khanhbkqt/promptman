package environment

import (
	"os"
	"regexp"
)

// envVarPattern matches the $ENV{VAR_NAME} syntax used in the secrets section.
// Captures the variable name, which must start with a letter or underscore
// followed by uppercase letters, digits, or underscores.
var envVarPattern = regexp.MustCompile(`^\$ENV\{([A-Z_][A-Z0-9_]*)\}$`)

// ResolveSecrets resolves all $ENV{VAR_NAME} references in a secrets map
// by looking up the corresponding OS environment variables via os.Getenv.
//
// Values that do not match the $ENV{} syntax are passed through unchanged.
// If any referenced OS variable is not set (empty string), an ErrSecretResolveFailed
// error is returned identifying the secret key and the missing variable.
func ResolveSecrets(secrets map[string]string) (map[string]string, error) {
	if len(secrets) == 0 {
		return secrets, nil
	}

	resolved := make(map[string]string, len(secrets))
	for key, value := range secrets {
		matches := envVarPattern.FindStringSubmatch(value)
		if matches == nil {
			// Not an $ENV{} reference — pass through unchanged.
			resolved[key] = value
			continue
		}

		envVar := matches[1]
		envValue, ok := os.LookupEnv(envVar)
		if !ok {
			return nil, ErrSecretResolveFailed.Wrapf(
				"secret %q references $ENV{%s} but the environment variable is not set",
				key, envVar,
			)
		}
		resolved[key] = envValue
	}

	return resolved, nil
}

// MaskSecrets returns a copy of the secrets map with every value replaced
// by the mask string "***". This is used to prevent secrets from leaking
// through API responses.
func MaskSecrets(secrets map[string]string) map[string]string {
	if len(secrets) == 0 {
		return secrets
	}

	masked := make(map[string]string, len(secrets))
	for key := range secrets {
		masked[key] = "***"
	}
	return masked
}
