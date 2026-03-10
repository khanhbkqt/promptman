package request

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/khanhnguyen/promptman/internal/collection"
	"github.com/khanhnguyen/promptman/pkg/variable"
)

// bodyTypeContentType maps body types to their Content-Type header values.
var bodyTypeContentType = map[string]string{
	"json": "application/json",
	"form": "application/x-www-form-urlencoded",
	"raw":  "text/plain",
}

// Builder constructs net/http requests from resolved request definitions.
type Builder struct{}

// NewBuilder creates a new Builder.
func NewBuilder() *Builder {
	return &Builder{}
}

// Build creates a ready-to-send *http.Request from a ResolvedRequest and a
// variable map. All {{var}} templates in URL, headers, body, and auth fields
// are resolved using strict mode — missing variables produce an error.
func (b *Builder) Build(resolved *collection.ResolvedRequest, vars map[string]any) (*http.Request, error) {
	opts := variable.Options{Strict: true}

	// 1. Resolve URL
	resolvedURL, err := variable.Resolve(resolved.URL, vars, opts)
	if err != nil {
		return nil, fmt.Errorf("resolving URL: %w", err)
	}

	// 2. Resolve headers
	headers := make(map[string]string, len(resolved.Headers))
	for k, v := range resolved.Headers {
		resolvedVal, err := variable.Resolve(v, vars, opts)
		if err != nil {
			return nil, fmt.Errorf("resolving header %q: %w", k, err)
		}
		headers[k] = resolvedVal
	}

	// 3. Build body
	var bodyStr string
	if resolved.Body != nil && resolved.Body.Content != nil {
		raw, err := json.Marshal(resolved.Body.Content)
		if err != nil {
			return nil, fmt.Errorf("serializing body: %w", err)
		}
		bodyStr, err = variable.Resolve(string(raw), vars, opts)
		if err != nil {
			return nil, fmt.Errorf("resolving body: %w", err)
		}
	}

	// 4. Create http.Request
	var bodyReader *strings.Reader
	if bodyStr != "" {
		bodyReader = strings.NewReader(bodyStr)
	}

	method := strings.ToUpper(resolved.Method)
	var req *http.Request
	if bodyReader != nil {
		req, err = http.NewRequest(method, resolvedURL, bodyReader)
	} else {
		req, err = http.NewRequest(method, resolvedURL, nil)
	}
	if err != nil {
		return nil, fmt.Errorf("creating HTTP request: %w", err)
	}

	// 5. Set headers
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	// 6. Set Content-Type based on body type
	if resolved.Body != nil && resolved.Body.Type != "" {
		if ct, ok := bodyTypeContentType[resolved.Body.Type]; ok {
			req.Header.Set("Content-Type", ct)
		}
	}

	// 7. Apply auth
	if err := applyAuth(req, resolved.Auth, vars, opts); err != nil {
		return nil, err
	}

	return req, nil
}
