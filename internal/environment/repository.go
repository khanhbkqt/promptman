package environment

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/khanhnguyen/promptman/pkg/fsutil"
)

// validName matches safe environment names: lowercase alphanumeric and hyphens.
// This prevents path-traversal attacks and enforces kebab-case naming.
var validName = regexp.MustCompile(`^[a-z0-9-]+$`)

// Repository defines the persistence interface for environments.
type Repository interface {
	// List scans the environments directory and returns a summary of each environment.
	List() ([]EnvSummary, error)

	// Get loads a single environment by name (filename without extension).
	Get(name string) (*Environment, error)

	// GetWithSecrets loads an environment and merges the corresponding .secrets.yaml
	// file if it exists. Variables from .secrets.yaml override same-name variables
	// in the base environment. Missing .secrets.yaml is not an error.
	GetWithSecrets(name string) (*Environment, error)

	// Save persists an environment to disk.
	// The name determines the filename: <name>.yaml.
	Save(name string, env *Environment) error

	// Delete removes the environment YAML file (and secrets file if present).
	Delete(name string) error
}

// FileRepository implements Repository using the local filesystem.
// Environments are stored as YAML files under .promptman/environments/.
type FileRepository struct {
	// dir is the resolved environments directory path.
	// When empty, it is resolved lazily via fsutil.EnvironmentsDir().
	dir string
}

// NewFileRepository creates a FileRepository. Pass an empty string for dir
// to use the default .promptman/environments/ directory (resolved at call time).
func NewFileRepository(dir string) *FileRepository {
	return &FileRepository{dir: dir}
}

// environmentsDir returns the environments directory, resolving it lazily if needed.
func (r *FileRepository) environmentsDir() (string, error) {
	if r.dir != "" {
		return r.dir, nil
	}
	dir, err := fsutil.EnvironmentsDir()
	if err != nil {
		return "", fmt.Errorf("resolve environments dir: %w", err)
	}
	r.dir = dir
	return dir, nil
}

// validateName checks that name is safe to use as a filename component.
func validateName(name string) error {
	if !validName.MatchString(name) {
		return ErrEnvironmentExists.Wrapf("invalid environment name %q: must match [a-z0-9-]+", name)
	}
	return nil
}

// filePath returns the full path for an environment name after validation.
func (r *FileRepository) filePath(name string) (string, error) {
	if err := validateName(name); err != nil {
		return "", err
	}
	dir, err := r.environmentsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, name+".yaml"), nil
}

// List scans .promptman/environments/*.yaml (excluding *.secrets.yaml)
// and returns a summary per file.
func (r *FileRepository) List() ([]EnvSummary, error) {
	dir, err := r.environmentsDir()
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // no environments directory yet → empty list
		}
		return nil, fmt.Errorf("read environments dir: %w", err)
	}

	var summaries []EnvSummary
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		// Skip secrets files — they are merged via GetWithSecrets.
		if strings.HasSuffix(e.Name(), ".secrets.yaml") {
			continue
		}

		name := strings.TrimSuffix(e.Name(), ".yaml")
		path := filepath.Join(dir, e.Name())

		var env Environment
		if err := fsutil.ReadYAML(path, &env); err != nil {
			// Skip files that can't be parsed — don't break the listing.
			continue
		}

		summaries = append(summaries, EnvSummary{
			Name:          name,
			VariableCount: len(env.Variables),
			SecretCount:   len(env.Secrets),
		})
	}

	return summaries, nil
}

// Get loads an environment by name.
func (r *FileRepository) Get(name string) (*Environment, error) {
	path, err := r.filePath(name)
	if err != nil {
		return nil, err
	}

	var env Environment
	if err := fsutil.ReadYAML(path, &env); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrEnvironmentNotFound.Wrapf("environment %q not found", name)
		}
		return nil, ErrInvalidYAML.Wrapf("load environment %q: %v", name, err)
	}

	// Ensure the Name field is set (it may be empty in the YAML).
	if env.Name == "" {
		env.Name = name
	}

	return &env, nil
}

// GetWithSecrets loads an environment and merges the corresponding .secrets.yaml
// file if it exists. Variables from the secrets file override same-name secrets
// in the base environment. If no .secrets.yaml exists, the base environment
// is returned unchanged.
func (r *FileRepository) GetWithSecrets(name string) (*Environment, error) {
	env, err := r.Get(name)
	if err != nil {
		return nil, err
	}

	dir, err := r.environmentsDir()
	if err != nil {
		return nil, err
	}

	secretsPath := filepath.Join(dir, name+".secrets.yaml")

	var secretsFile Environment
	if err := fsutil.ReadYAML(secretsPath, &secretsFile); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// No secrets file — return the base environment as-is.
			return env, nil
		}
		return nil, ErrInvalidYAML.Wrapf("load secrets file for %q: %v", name, err)
	}

	// Merge secrets: values from .secrets.yaml override base secrets.
	if len(secretsFile.Secrets) > 0 {
		if env.Secrets == nil {
			env.Secrets = make(map[string]string)
		}
		for k, v := range secretsFile.Secrets {
			env.Secrets[k] = v
		}
	}

	// Merge variables: values from .secrets.yaml override base variables.
	if len(secretsFile.Variables) > 0 {
		if env.Variables == nil {
			env.Variables = make(map[string]any)
		}
		for k, v := range secretsFile.Variables {
			env.Variables[k] = v
		}
	}

	return env, nil
}

// Save persists an environment to disk.
func (r *FileRepository) Save(name string, env *Environment) error {
	path, err := r.filePath(name)
	if err != nil {
		return err
	}

	if err := fsutil.WriteYAML(path, env); err != nil {
		return fmt.Errorf("save environment %q: %w", name, err)
	}

	return nil
}

// Delete removes the YAML file for the given environment name.
// It also removes the corresponding .secrets.yaml file if it exists.
func (r *FileRepository) Delete(name string) error {
	path, err := r.filePath(name)
	if err != nil {
		return err
	}

	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return ErrEnvironmentNotFound.Wrapf("environment %q not found", name)
		}
		return fmt.Errorf("delete environment %q: %w", name, err)
	}

	// Also remove the secrets file if it exists.
	dir, _ := r.environmentsDir()
	secretsPath := filepath.Join(dir, name+".secrets.yaml")
	// Ignore errors — the secrets file may not exist.
	os.Remove(secretsPath)

	return nil
}
