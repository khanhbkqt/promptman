package environment

// Environment represents an environment configuration stored as a YAML file.
// Variables use map[string]any to preserve native YAML types (string, int, bool, float64).
// Secrets use map[string]string to store raw $ENV{} references.
type Environment struct {
	Name      string            `yaml:"name"`
	Variables map[string]any    `yaml:"variables,omitempty"`
	Secrets   map[string]string `yaml:"secrets,omitempty"`
}

// EnvSummary is a lightweight projection of an Environment used in listings.
type EnvSummary struct {
	Name          string `yaml:"-"` // derived from filename
	VariableCount int    `yaml:"-"` // computed at load time
	SecretCount   int    `yaml:"-"` // computed at load time
}

// CreateEnvInput holds the data needed to create a new environment.
// Name is required; Variables and Secrets are optional.
type CreateEnvInput struct {
	Name      string
	Variables map[string]any
	Secrets   map[string]string
}

// UpdateEnvInput holds optional fields for partial environment updates.
// Only non-nil fields are applied; nil fields are left unchanged.
type UpdateEnvInput struct {
	Name      *string
	Variables *map[string]any
	Secrets   *map[string]string
}
