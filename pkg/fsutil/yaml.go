package fsutil

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ReadYAML reads a YAML file and unmarshals it into the given output.
func ReadYAML(path string, out any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read yaml %s: %w", path, err)
	}
	if err := yaml.Unmarshal(data, out); err != nil {
		return fmt.Errorf("parse yaml %s: %w", path, err)
	}
	return nil
}

// WriteYAML marshals data as YAML and writes it atomically to a file.
// Parent directories are created automatically.
func WriteYAML(path string, data any) error {
	bytes, err := yaml.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal yaml: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create dir for %s: %w", path, err)
	}
	return AtomicWrite(path, bytes)
}
