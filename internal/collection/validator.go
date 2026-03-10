package collection

import (
	"fmt"
	"strings"
)

// validMethods lists the HTTP methods accepted by the validator.
var validMethods = map[string]bool{
	"GET":     true,
	"POST":    true,
	"PUT":     true,
	"PATCH":   true,
	"DELETE":  true,
	"HEAD":    true,
	"OPTIONS": true,
}

// validAuthTypes lists the authentication types supported by AuthConfig.
var validAuthTypes = map[string]bool{
	"bearer":  true,
	"basic":   true,
	"api-key": true,
}

// ValidationError collects one or more validation failures.
type ValidationError struct {
	Errors []string
}

// Error implements the error interface, joining all messages.
func (v *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", ErrInvalidRequest.Code, strings.Join(v.Errors, "; "))
}

// ValidateCollection validates a Collection against the schema rules:
//   - name is required
//   - auth block (if present) must have a valid type with matching credentials
//   - every request must have id and a valid method
//   - folders are validated recursively
//
// It returns a *ValidationError listing all violations, or nil if valid.
func ValidateCollection(c *Collection) error {
	var errs []string

	if strings.TrimSpace(c.Name) == "" {
		errs = append(errs, "collection name is required")
	}

	if c.Auth != nil {
		if msgs := validateAuth(c.Auth, "collection"); len(msgs) > 0 {
			errs = append(errs, msgs...)
		}
	}

	for i, r := range c.Requests {
		if msgs := validateRequest(&r, fmt.Sprintf("requests[%d]", i)); len(msgs) > 0 {
			errs = append(errs, msgs...)
		}
	}

	for i, f := range c.Folders {
		if msgs := validateFolder(&f, fmt.Sprintf("folders[%d]", i)); len(msgs) > 0 {
			errs = append(errs, msgs...)
		}
	}

	if len(errs) > 0 {
		return &ValidationError{Errors: errs}
	}
	return nil
}

// validateRequest checks a single Request for required fields and valid method.
func validateRequest(r *Request, path string) []string {
	var errs []string

	if strings.TrimSpace(r.ID) == "" {
		errs = append(errs, fmt.Sprintf("%s: request id is required", path))
	}

	method := strings.ToUpper(strings.TrimSpace(r.Method))
	if method == "" {
		errs = append(errs, fmt.Sprintf("%s: request method is required", path))
	} else if !validMethods[method] {
		errs = append(errs, fmt.Sprintf("%s: invalid method %q", path, r.Method))
	}

	if r.Auth != nil {
		if msgs := validateAuth(r.Auth, path); len(msgs) > 0 {
			errs = append(errs, msgs...)
		}
	}

	return errs
}

// validateFolder recursively validates a Folder and its nested content.
func validateFolder(f *Folder, path string) []string {
	var errs []string

	if strings.TrimSpace(f.ID) == "" {
		errs = append(errs, fmt.Sprintf("%s: folder id is required", path))
	}
	if strings.TrimSpace(f.Name) == "" {
		errs = append(errs, fmt.Sprintf("%s: folder name is required", path))
	}

	if f.Auth != nil {
		if msgs := validateAuth(f.Auth, path); len(msgs) > 0 {
			errs = append(errs, msgs...)
		}
	}

	for i, r := range f.Requests {
		if msgs := validateRequest(&r, fmt.Sprintf("%s.requests[%d]", path, i)); len(msgs) > 0 {
			errs = append(errs, msgs...)
		}
	}

	for i, sub := range f.Folders {
		if msgs := validateFolder(&sub, fmt.Sprintf("%s.folders[%d]", path, i)); len(msgs) > 0 {
			errs = append(errs, msgs...)
		}
	}

	return errs
}

// validateAuth checks that an AuthConfig has a valid type and matching credentials.
func validateAuth(a *AuthConfig, path string) []string {
	var errs []string

	authType := strings.TrimSpace(a.Type)
	if authType == "" {
		errs = append(errs, fmt.Sprintf("%s: auth type is required", path))
		return errs
	}

	if !validAuthTypes[authType] {
		errs = append(errs, fmt.Sprintf("%s: invalid auth type %q (want bearer, basic, or api-key)", path, a.Type))
		return errs
	}

	switch authType {
	case "bearer":
		if a.Bearer == nil {
			errs = append(errs, fmt.Sprintf("%s: auth type is bearer but bearer config is missing", path))
		}
	case "basic":
		if a.Basic == nil {
			errs = append(errs, fmt.Sprintf("%s: auth type is basic but basic config is missing", path))
		}
	case "api-key":
		if a.APIKey == nil {
			errs = append(errs, fmt.Sprintf("%s: auth type is api-key but apiKey config is missing", path))
		}
	}

	return errs
}
