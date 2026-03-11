package core

import (
	"github.com/khanhnguyen/promptman/internal/collection"
)

// CollectionGetter loads a collection by ID. It is a subset of
// collection.Service that the CollectionListerAdapter needs.
type CollectionGetter interface {
	Get(id string) (*collection.Collection, error)
}

// CollectionListerAdapter adapts a CollectionGetter to the
// CollectionLister interface required by Runner.
type CollectionListerAdapter struct {
	getter CollectionGetter
}

// NewCollectionListerAdapter creates a CollectionListerAdapter.
func NewCollectionListerAdapter(getter CollectionGetter) *CollectionListerAdapter {
	return &CollectionListerAdapter{getter: getter}
}

// ListRequestPaths returns all request paths (slash-separated) in the
// collection. It walks the collection's Requests and nested Folders
// recursively.
func (a *CollectionListerAdapter) ListRequestPaths(collectionID string) ([]string, error) {
	coll, err := a.getter.Get(collectionID)
	if err != nil {
		return nil, err
	}

	var paths []string
	collectPaths(&paths, "", coll.Requests, coll.Folders)
	return paths, nil
}

// collectPaths recursively gathers request IDs from requests and folders,
// building slash-separated paths.
func collectPaths(out *[]string, prefix string, requests []collection.Request, folders []collection.Folder) {
	for _, r := range requests {
		path := r.ID
		if prefix != "" {
			path = prefix + "/" + r.ID
		}
		*out = append(*out, path)
	}
	for _, f := range folders {
		folderPrefix := f.ID
		if prefix != "" {
			folderPrefix = prefix + "/" + f.ID
		}
		collectPaths(out, folderPrefix, f.Requests, f.Folders)
	}
}
