package core

import (
	"fmt"
	"os"
	"path/filepath"
)

// FileLoader implements TestFileLoader by reading test scripts from the
// .promptman/tests/ directory on disk.
type FileLoader struct {
	baseDir string // project root (empty = cwd)
}

// NewFileLoader creates a FileLoader rooted at the given project directory.
// If baseDir is empty, the current working directory is used.
func NewFileLoader(baseDir string) *FileLoader {
	return &FileLoader{baseDir: baseDir}
}

// Load reads the JavaScript test file for the given collection ID.
// The file is expected at .promptman/tests/<collectionID>.test.js.
func (fl *FileLoader) Load(collectionID string) (string, error) {
	path := filepath.Join(fl.baseDir, ".promptman", "tests", collectionID+".test.js")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading test file %s: %w", path, err)
	}
	return string(data), nil
}
