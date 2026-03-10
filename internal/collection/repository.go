package collection

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/khanhnguyen/promptman/pkg/fsutil"
)

// validID matches safe collection identifiers: alphanumeric, hyphens, underscores.
// This prevents path-traversal attacks (e.g. "../../etc/passwd").
var validID = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// Repository defines the persistence interface for collections.
type Repository interface {
	// List scans the collections directory and returns a summary of each collection.
	List() ([]CollectionSummary, error)

	// Get loads a single collection by its ID (filename without extension).
	Get(id string) (*Collection, error)

	// Save persists a collection to disk using atomic writes.
	// The id determines the filename: <id>.yaml.
	Save(id string, c *Collection) error

	// Delete removes the YAML file for the given collection ID.
	Delete(id string) error
}

// FileRepository implements Repository using the local filesystem.
// Collections are stored as YAML files under .promptman/collections/.
type FileRepository struct {
	// dir is the resolved collections directory path.
	// When empty, it is resolved lazily via fsutil.CollectionsDir().
	dir string
}

// NewFileRepository creates a FileRepository. Pass an empty string for dir
// to use the default .promptman/collections/ directory (resolved at call time).
func NewFileRepository(dir string) *FileRepository {
	return &FileRepository{dir: dir}
}

// collectionsDir returns the collections directory, resolving it lazily if needed.
func (r *FileRepository) collectionsDir() (string, error) {
	if r.dir != "" {
		return r.dir, nil
	}
	dir, err := fsutil.CollectionsDir()
	if err != nil {
		return "", fmt.Errorf("resolve collections dir: %w", err)
	}
	r.dir = dir
	return dir, nil
}

// validateID checks that id is safe to use as a filename component.
func validateID(id string) error {
	if !validID.MatchString(id) {
		return ErrInvalidRequest.Wrapf("invalid collection id %q: must match [a-zA-Z0-9_-]+", id)
	}
	return nil
}

// filePath returns the full path for a collection ID after validation.
func (r *FileRepository) filePath(id string) (string, error) {
	if err := validateID(id); err != nil {
		return "", err
	}
	dir, err := r.collectionsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, id+".yaml"), nil
}

// List scans .promptman/collections/*.yaml and returns a summary per file.
func (r *FileRepository) List() ([]CollectionSummary, error) {
	dir, err := r.collectionsDir()
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // no collections directory yet → empty list
		}
		return nil, fmt.Errorf("read collections dir: %w", err)
	}

	var summaries []CollectionSummary
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}

		id := strings.TrimSuffix(e.Name(), ".yaml")
		path := filepath.Join(dir, e.Name())

		var c Collection
		if err := fsutil.ReadYAML(path, &c); err != nil {
			// Skip files that can't be parsed — don't break the listing.
			continue
		}

		summaries = append(summaries, CollectionSummary{
			ID:           id,
			Name:         c.Name,
			RequestCount: countRequests(&c),
		})
	}

	return summaries, nil
}

// Get loads a collection by ID.
func (r *FileRepository) Get(id string) (*Collection, error) {
	path, err := r.filePath(id)
	if err != nil {
		return nil, err
	}

	var c Collection
	if err := fsutil.ReadYAML(path, &c); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrCollectionNotFound.Wrapf("collection %q not found", id)
		}
		return nil, ErrInvalidYAML.Wrapf("load collection %q: %v", id, err)
	}

	if err := ValidateCollection(&c); err != nil {
		return nil, ErrInvalidYAML.Wrapf("collection %q failed validation: %v", id, err)
	}

	return &c, nil
}

// Save persists a collection to disk. Validates the collection before writing.
func (r *FileRepository) Save(id string, c *Collection) error {
	if err := ValidateCollection(c); err != nil {
		return err
	}

	path, err := r.filePath(id)
	if err != nil {
		return err
	}

	if err := fsutil.WriteYAML(path, c); err != nil {
		return fmt.Errorf("save collection %q: %w", id, err)
	}

	return nil
}

// Delete removes the YAML file for the given collection ID.
func (r *FileRepository) Delete(id string) error {
	path, err := r.filePath(id)
	if err != nil {
		return err
	}

	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return ErrCollectionNotFound.Wrapf("collection %q not found", id)
		}
		return fmt.Errorf("delete collection %q: %w", id, err)
	}

	return nil
}

// countRequests counts all requests in a collection, including those nested in folders.
func countRequests(c *Collection) int {
	count := len(c.Requests)
	for i := range c.Folders {
		count += countFolderRequests(&c.Folders[i])
	}
	return count
}

// countFolderRequests recursively counts requests within a folder and its sub-folders.
func countFolderRequests(f *Folder) int {
	count := len(f.Requests)
	for i := range f.Folders {
		count += countFolderRequests(&f.Folders[i])
	}
	return count
}
