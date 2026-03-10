package collection

import (
	"strings"
)

// Resolve walks the folder tree using requestPath and merges defaults from
// Collection → Folder(s) → Request, returning a fully resolved request.
//
// requestPath is a slash-separated path where intermediate segments are folder
// IDs and the last segment is a request ID. For example:
//
//	"health"              → root-level request "health"
//	"admin/list-admins"   → request "list-admins" inside folder "admin"
//	"admin/settings/list" → request "list" inside folder "settings" inside "admin"
//
// Returns ErrRequestNotFound if any segment in the path does not match.
func Resolve(c *Collection, requestPath string) (*ResolvedRequest, error) {
	requestPath = strings.TrimSpace(requestPath)
	if requestPath == "" {
		return nil, ErrRequestNotFound.Wrap("request path is empty")
	}

	segments := strings.Split(requestPath, "/")

	// Walk the folder tree, collecting defaults and auth at each level.
	type level struct {
		defaults *RequestDefaults
		auth     *AuthConfig
	}

	chain := []level{{defaults: c.Defaults, auth: c.Auth}}

	// Current scope: start at collection level.
	folders := c.Folders
	requests := c.Requests

	// Traverse intermediate segments (all except the last) as folder IDs.
	for i := 0; i < len(segments)-1; i++ {
		folder, found := findFolder(folders, segments[i])
		if !found {
			return nil, ErrRequestNotFound.Wrapf("folder %q not found in path %q", segments[i], requestPath)
		}
		chain = append(chain, level{defaults: folder.Defaults, auth: folder.Auth})
		folders = folder.Folders
		requests = folder.Requests
	}

	// Last segment is the request ID.
	reqID := segments[len(segments)-1]
	req, found := findRequest(requests, reqID)
	if !found {
		return nil, ErrRequestNotFound.Wrapf("request %q not found in path %q", reqID, requestPath)
	}

	// Merge the defaults chain.
	resolved := &ResolvedRequest{
		URL:     joinURL(c.BaseURL, req.Path),
		Method:  req.Method,
		Headers: make(map[string]string),
		Body:    req.Body,
	}

	// Layer headers from collection defaults → folder defaults → request headers.
	// Each layer overrides keys from the previous.
	for _, lv := range chain {
		if lv.defaults != nil {
			for k, v := range lv.defaults.Headers {
				resolved.Headers[k] = v
			}
		}
	}
	for k, v := range req.Headers {
		resolved.Headers[k] = v
	}

	// Resolve timeout: last non-nil wins.
	for _, lv := range chain {
		if lv.defaults != nil && lv.defaults.Timeout != nil {
			resolved.Timeout = lv.defaults.Timeout
		}
	}
	if req.Timeout != nil {
		resolved.Timeout = req.Timeout
	}

	// Resolve auth: last non-nil wins (child overrides parent entirely).
	for _, lv := range chain {
		if lv.auth != nil {
			resolved.Auth = lv.auth
		}
	}
	if req.Auth != nil {
		resolved.Auth = req.Auth
	}

	return resolved, nil
}

// findFolder searches a slice of folders for one matching the given ID.
func findFolder(folders []Folder, id string) (*Folder, bool) {
	for i := range folders {
		if folders[i].ID == id {
			return &folders[i], true
		}
	}
	return nil, false
}

// findRequest searches a slice of requests for one matching the given ID.
func findRequest(requests []Request, id string) (*Request, bool) {
	for i := range requests {
		if requests[i].ID == id {
			return &requests[i], true
		}
	}
	return nil, false
}

// joinURL concatenates a base URL and a path, ensuring exactly one slash between them.
func joinURL(base, path string) string {
	if base == "" {
		return path
	}
	if path == "" {
		return base
	}
	return strings.TrimRight(base, "/") + "/" + strings.TrimLeft(path, "/")
}
