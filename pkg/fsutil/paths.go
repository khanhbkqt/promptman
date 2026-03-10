package fsutil

import (
	"fmt"
	"os"
	"path/filepath"
)

const promptmanDir = ".promptman"

// ProjectDir walks up the directory tree from the current working directory
// to find a directory containing a .promptman/ folder.
// Returns the path to the project root (parent of .promptman/).
func ProjectDir() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get working directory: %w", err)
	}
	return projectDirFrom(dir)
}

// projectDirFrom walks up from the given directory to find .promptman/.
func projectDirFrom(startDir string) (string, error) {
	dir := startDir
	for {
		candidate := filepath.Join(dir, promptmanDir)
		info, err := os.Stat(candidate)
		if err == nil && info.IsDir() {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root
			return "", fmt.Errorf("no %s directory found (searched from %s to root)", promptmanDir, startDir)
		}
		dir = parent
	}
}

// CollectionsDir returns the path to the collections directory.
func CollectionsDir() (string, error) {
	root, err := ProjectDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, promptmanDir, "collections"), nil
}

// EnvironmentsDir returns the path to the environments directory.
func EnvironmentsDir() (string, error) {
	root, err := ProjectDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, promptmanDir, "environments"), nil
}

// TestsDir returns the path to the tests directory.
func TestsDir() (string, error) {
	root, err := ProjectDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, promptmanDir, "tests"), nil
}

// HistoryDir returns the path to the history directory.
func HistoryDir() (string, error) {
	root, err := ProjectDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, promptmanDir, "history"), nil
}
