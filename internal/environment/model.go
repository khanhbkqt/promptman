package environment

// Environment represents an environment configuration stored as a YAML file.
// Variables use map[string]any to preserve native YAML types (string, int, bool, float64).
// Secrets use map[string]string to store raw $ENV{} references.
type Environment struct {
	Name      string            `yaml:"name" json:"name"`
	Variables map[string]any    `yaml:"variables,omitempty" json:"variables,omitempty"`
	Secrets   map[string]string `yaml:"secrets,omitempty" json:"secrets,omitempty"`
}

// EnvSummary is a lightweight projection of an Environment used in listings.
type EnvSummary struct {
	Name          string `yaml:"-" json:"name"`
	VariableCount int    `yaml:"-" json:"variableCount"`
	SecretCount   int    `yaml:"-" json:"secretCount"`
}

// CreateEnvInput holds the data needed to create a new environment.
// Name is required; Variables and Secrets are optional.
type CreateEnvInput struct {
	Name      string            `json:"name"`
	Variables map[string]any    `json:"variables,omitempty"`
	Secrets   map[string]string `json:"secrets,omitempty"`
}

// UpdateEnvInput holds optional fields for partial environment updates.
// Only non-nil fields are applied; nil fields are left unchanged.
type UpdateEnvInput struct {
	Name      *string            `json:"name,omitempty"`
	Variables *map[string]any    `json:"variables,omitempty"`
	Secrets   *map[string]string `json:"secrets,omitempty"`
}
