package request

// ExecuteInput holds the parameters needed to execute a single HTTP request.
type ExecuteInput struct {
	CollectionID  string         `json:"collection"`              // collection identifier
	RequestID     string         `json:"requestId"`               // request path within collection
	Environment   string         `json:"env,omitempty"`           // override active environment
	Variables     map[string]any `json:"variables,omitempty"`     // runtime variable overrides
	SkipTLSVerify bool           `json:"skipTlsVerify,omitempty"` // skip TLS certificate verification
	Source        string         `json:"source,omitempty"`        // history source: cli | gui | test
}

// Response holds the complete result of an HTTP request execution.
type Response struct {
	RequestID string            `json:"requestId"`       // original request path
	Method    string            `json:"method"`          // HTTP method used
	URL       string            `json:"url"`             // fully resolved URL
	Status    int               `json:"status"`          // HTTP status code
	Headers   map[string]string `json:"headers"`         // response headers
	Body      string            `json:"body"`            // response body as string
	Timing    *RequestTiming    `json:"timing"`          // timing breakdown
	Error     string            `json:"error,omitempty"` // network error description
}

// RequestTiming holds the timing breakdown of an HTTP request in milliseconds.
type RequestTiming struct {
	DNS      int `json:"dns"`      // DNS lookup duration
	Connect  int `json:"connect"`  // TCP connection duration
	TLS      int `json:"tls"`      // TLS handshake duration
	TTFB     int `json:"ttfb"`     // time to first byte
	Transfer int `json:"transfer"` // body transfer duration
	Total    int `json:"total"`    // total request duration
}

// CollectionRunOpts configures how a collection is executed.
type CollectionRunOpts struct {
	CollectionID  string         `json:"collection"`              // collection to execute
	Environment   string         `json:"env,omitempty"`           // environment to use
	Variables     map[string]any `json:"variables,omitempty"`     // runtime variable overrides
	StopOnError   bool           `json:"stopOnError,omitempty"`   // stop on first request error
	SkipTLSVerify bool           `json:"skipTlsVerify,omitempty"` // skip TLS certificate verification
}
