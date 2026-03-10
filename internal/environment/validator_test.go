package environment

import (
	"testing"
)

// --- isValidName ---

func TestIsValidName(t *testing.T) {
	tests := []struct {
		name  string
		valid bool
	}{
		{"dev", true},
		{"staging", true},
		{"my-env", true},
		{"a", true},
		{"a1", true},
		{"env-123", true},
		{"dev-staging-v2", true},
		{"", false},
		{"-leading", false},
		{"trailing-", false},
		{"-both-", false},
		{"has space", false},
		{"HAS_UPPER", false},
		{"has_underscore", false},
		{"has.dot", false},
		{"has/slash", false},
		{"1", true},    // single digit OK via nameReSingle
		{"123", false}, // multi-char must start with [a-z]
		{"1a", false},  // starts with digit, multi-char needs to start with alpha
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidName(tt.name)
			if got != tt.valid {
				t.Errorf("isValidName(%q) = %v, want %v", tt.name, got, tt.valid)
			}
		})
	}
}

// --- ValidateCreateInput ---

func TestValidateCreateInput_Valid(t *testing.T) {
	tests := []struct {
		name  string
		input *CreateEnvInput
	}{
		{"simple name", &CreateEnvInput{Name: "dev"}},
		{"with variables", &CreateEnvInput{
			Name:      "staging",
			Variables: map[string]any{"host": "localhost", "port": 3000},
		}},
		{"with secrets", &CreateEnvInput{
			Name:    "prod",
			Secrets: map[string]string{"apiKey": "$ENV{API_KEY}"},
		}},
		{"single char name", &CreateEnvInput{Name: "a"}},
		{"kebab-case", &CreateEnvInput{Name: "my-dev-env"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidateCreateInput(tt.input); err != nil {
				t.Errorf("ValidateCreateInput() error = %v", err)
			}
		})
	}
}

func TestValidateCreateInput_Invalid(t *testing.T) {
	tests := []struct {
		name  string
		input *CreateEnvInput
	}{
		{"nil input", nil},
		{"empty name", &CreateEnvInput{Name: ""}},
		{"whitespace name", &CreateEnvInput{Name: "   "}},
		{"leading hyphen", &CreateEnvInput{Name: "-dev"}},
		{"trailing hyphen", &CreateEnvInput{Name: "dev-"}},
		{"uppercase", &CreateEnvInput{Name: "DEV"}},
		{"spaces in name", &CreateEnvInput{Name: "my env"}},
		{"special chars", &CreateEnvInput{Name: "env@prod"}},
		{"empty variable key", &CreateEnvInput{
			Name:      "dev",
			Variables: map[string]any{"": "value"},
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCreateInput(tt.input)
			if err == nil {
				t.Error("expected validation error")
			}
			if _, ok := err.(*ValidationError); !ok {
				t.Errorf("expected *ValidationError, got %T", err)
			}
		})
	}
}

// --- ValidateUpdateInput ---

func TestValidateUpdateInput_Valid(t *testing.T) {
	newName := "staging"
	newVars := map[string]any{"host": "prod.example.com"}
	newSecrets := map[string]string{"apiKey": "$ENV{PROD_KEY}"}

	tests := []struct {
		name  string
		input *UpdateEnvInput
	}{
		{"name only", &UpdateEnvInput{Name: &newName}},
		{"variables only", &UpdateEnvInput{Variables: &newVars}},
		{"secrets only", &UpdateEnvInput{Secrets: &newSecrets}},
		{"all fields", &UpdateEnvInput{Name: &newName, Variables: &newVars, Secrets: &newSecrets}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidateUpdateInput(tt.input); err != nil {
				t.Errorf("ValidateUpdateInput() error = %v", err)
			}
		})
	}
}

func TestValidateUpdateInput_Invalid(t *testing.T) {
	emptyName := ""
	badName := "-invalid-"
	emptyKeyVars := map[string]any{"": "val"}

	tests := []struct {
		name  string
		input *UpdateEnvInput
	}{
		{"nil input", nil},
		{"no fields set", &UpdateEnvInput{}},
		{"empty name", &UpdateEnvInput{Name: &emptyName}},
		{"invalid name", &UpdateEnvInput{Name: &badName}},
		{"empty variable key", &UpdateEnvInput{Variables: &emptyKeyVars}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateUpdateInput(tt.input)
			if err == nil {
				t.Error("expected validation error")
			}
			if _, ok := err.(*ValidationError); !ok {
				t.Errorf("expected *ValidationError, got %T", err)
			}
		})
	}
}

// --- ValidationError ---

func TestValidationError_Error(t *testing.T) {
	ve := &ValidationError{Errors: []string{"name required", "invalid format"}}
	got := ve.Error()
	if got == "" {
		t.Error("expected non-empty error string")
	}
	// Should contain both messages
	if !contains(got, "name required") || !contains(got, "invalid format") {
		t.Errorf("error string missing messages: %q", got)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
