package environment

import (
	"fmt"
	"regexp"
	"strings"
)

// nameReSingle matches a single lowercase alphanumeric character.
var nameReSingle = regexp.MustCompile(`^[a-z0-9]$`)

// nameReMulti matches kebab-case names: starts with alpha, ends with alphanumeric,
// allows hyphens in the middle.
var nameReMulti = regexp.MustCompile(`^[a-z][a-z0-9-]*[a-z0-9]$`)

// ValidationError collects one or more validation failures.
type ValidationError struct {
	Errors []string
}

// Error implements the error interface, joining all messages.
func (v *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", ErrEnvironmentExists.Code, strings.Join(v.Errors, "; "))
}

// isValidName checks whether a name matches the kebab-case naming convention.
// Accepts: single alnum char OR multi-char kebab-case.
func isValidName(name string) bool {
	return nameReSingle.MatchString(name) || nameReMulti.MatchString(name)
}

// ValidateCreateInput validates the input for creating a new environment.
func ValidateCreateInput(input *CreateEnvInput) error {
	var errs []string

	if input == nil {
		return &ValidationError{Errors: []string{"create input is nil"}}
	}

	name := strings.TrimSpace(input.Name)
	if name == "" {
		errs = append(errs, "environment name is required")
	} else if !isValidName(name) {
		errs = append(errs, fmt.Sprintf("environment name %q is invalid: must be kebab-case (lowercase alphanumeric and hyphens, no leading/trailing hyphens)", input.Name))
	}

	// Validate variable keys are non-empty
	for key := range input.Variables {
		if strings.TrimSpace(key) == "" {
			errs = append(errs, "variable key must not be empty")
			break
		}
	}

	if len(errs) > 0 {
		return &ValidationError{Errors: errs}
	}
	return nil
}

// ValidateUpdateInput validates the input for updating an existing environment.
func ValidateUpdateInput(input *UpdateEnvInput) error {
	var errs []string

	if input == nil {
		return &ValidationError{Errors: []string{"update input is nil"}}
	}

	// At least one field must be set.
	if input.Name == nil && input.Variables == nil && input.Secrets == nil {
		errs = append(errs, "at least one field must be set")
	}

	// If Name is provided, validate it.
	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" {
			errs = append(errs, "environment name must not be empty")
		} else if !isValidName(name) {
			errs = append(errs, fmt.Sprintf("environment name %q is invalid: must be kebab-case", *input.Name))
		}
	}

	// Validate variable keys are non-empty
	if input.Variables != nil {
		for key := range *input.Variables {
			if strings.TrimSpace(key) == "" {
				errs = append(errs, "variable key must not be empty")
				break
			}
		}
	}

	if len(errs) > 0 {
		return &ValidationError{Errors: errs}
	}
	return nil
}
