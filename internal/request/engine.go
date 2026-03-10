package request

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptrace"
	"time"

	"github.com/khanhnguyen/promptman/internal/collection"
	"github.com/khanhnguyen/promptman/internal/environment"
)

// Default engine configuration values.
const (
	defaultTimeout         = 30 * time.Second
	defaultMaxIdleConns    = 100
	defaultIdleConnTimeout = 90 * time.Second
	defaultMaxConnsPerHost = 10
)

// CollectionFinder finds and resolves a request within a collection.
type CollectionFinder interface {
	FindRequest(collectionID, requestPath string) (*collection.ResolvedRequest, error)
}

// EnvironmentResolver resolves environment variables and secrets.
type EnvironmentResolver interface {
	GetRaw(name string) (*environment.Environment, error)
	GetActive() (string, error)
}

// Engine executes HTTP requests by orchestrating collection lookup,
// environment resolution, request building, and response capture.
type Engine struct {
	collSvc   CollectionFinder
	envSvc    EnvironmentResolver
	builder   *Builder
	transport *http.Transport
	timeout   time.Duration
}

// EngineOption configures the Engine via functional options.
type EngineOption func(*Engine)

// WithTransport sets a custom HTTP transport for the engine.
// Use this for testing or custom connection pooling configuration.
func WithTransport(t *http.Transport) EngineOption {
	return func(e *Engine) {
		e.transport = t
	}
}

// WithDefaultTimeout sets the default request timeout when no
// timeout is specified in the collection defaults chain.
func WithDefaultTimeout(d time.Duration) EngineOption {
	return func(e *Engine) {
		e.timeout = d
	}
}

// NewEngine creates an Engine with the given service dependencies
// and optional configuration. The engine reuses a shared HTTP transport
// for connection pooling across requests.
func NewEngine(collSvc CollectionFinder, envSvc EnvironmentResolver, opts ...EngineOption) *Engine {
	e := &Engine{
		collSvc: collSvc,
		envSvc:  envSvc,
		builder: NewBuilder(),
		transport: &http.Transport{
			MaxIdleConns:        defaultMaxIdleConns,
			MaxConnsPerHost:     defaultMaxConnsPerHost,
			IdleConnTimeout:     defaultIdleConnTimeout,
			TLSHandshakeTimeout: 10 * time.Second,
		},
		timeout: defaultTimeout,
	}

	for _, opt := range opts {
		opt(e)
	}

	return e
}

// Execute sends a single HTTP request and returns the response with timing data.
//
// The execution pipeline:
//  1. Find and resolve the request definition from the collection
//  2. Resolve environment variables and secrets
//  3. Build the HTTP request with variable substitution
//  4. Send the request with timing instrumentation
//  5. Capture and return the response
func (e *Engine) Execute(ctx context.Context, input ExecuteInput) (*Response, error) {
	// 1. Find request in collection.
	resolved, err := e.collSvc.FindRequest(input.CollectionID, input.RequestID)
	if err != nil {
		return nil, fmt.Errorf("finding request: %w", err)
	}

	// 2. Resolve environment.
	vars, err := e.resolveVariables(input)
	if err != nil {
		return nil, err
	}

	// 3. Build the HTTP request.
	httpReq, err := e.builder.Build(resolved, vars)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}

	// 4. Configure client and execute.
	timeout := e.resolveTimeout(resolved)
	client := e.buildClient(input.SkipTLSVerify, timeout)

	trace, timing := newTimingTrace()
	httpReq = httpReq.WithContext(httptrace.WithClientTrace(ctx, trace))

	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, e.classifyError(err, input.RequestID)
	}

	// 5. Capture response.
	timing.done()
	result, err := captureResponse(resp, timing, input.RequestID, httpReq.Method, httpReq.URL.String())
	if err != nil {
		return nil, fmt.Errorf("capturing response: %w", err)
	}

	return result, nil
}

// resolveVariables merges environment variables, secrets, and runtime overrides.
// Priority (highest wins): runtime overrides > secrets > variables.
func (e *Engine) resolveVariables(input ExecuteInput) (map[string]any, error) {
	envName := input.Environment

	// Fall back to active environment.
	if envName == "" {
		active, err := e.envSvc.GetActive()
		if err != nil {
			// No active environment — use only runtime overrides.
			vars := make(map[string]any, len(input.Variables))
			for k, v := range input.Variables {
				vars[k] = v
			}
			return vars, nil
		}
		envName = active
	}

	env, err := e.envSvc.GetRaw(envName)
	if err != nil {
		return nil, fmt.Errorf("resolving environment %q: %w", envName, err)
	}

	// Merge: variables (base) → secrets (override) → runtime (highest).
	vars := make(map[string]any)
	for k, v := range env.Variables {
		vars[k] = v
	}
	for k, v := range env.Secrets {
		vars[k] = v
	}
	for k, v := range input.Variables {
		vars[k] = v
	}

	return vars, nil
}

// resolveTimeout returns the timeout for the request. The priority is:
// 1. Explicitly set timeout on the ResolvedRequest (from defaults chain)
// 2. Engine's default timeout
func (e *Engine) resolveTimeout(resolved *collection.ResolvedRequest) time.Duration {
	if resolved.Timeout != nil && *resolved.Timeout > 0 {
		return time.Duration(*resolved.Timeout) * time.Millisecond
	}
	return e.timeout
}

// buildClient creates an http.Client with the appropriate timeout and TLS
// configuration. When skipTLS is true, certificate verification is disabled.
func (e *Engine) buildClient(skipTLS bool, timeout time.Duration) *http.Client {
	transport := e.transport

	if skipTLS {
		// Clone the transport to avoid mutating the shared one.
		transport = e.transport.Clone()
		transport.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true, //nolint:gosec // user-requested skip
		}
	}

	return &http.Client{
		Transport: transport,
		Timeout:   timeout,
	}
}

// classifyError maps network errors to domain-specific error types.
func (e *Engine) classifyError(err error, reqID string) error {
	// Context cancellation.
	if errors.Is(err, context.Canceled) {
		return ErrRequestFailed.Wrapf("request %q canceled", reqID)
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return ErrRequestTimeout.Wrapf("request %q timed out", reqID)
	}

	// Check for net.Error timeout (from http.Client.Timeout).
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return ErrRequestTimeout.Wrapf("request %q timed out: %v", reqID, err)
	}

	// All other network errors.
	return ErrRequestFailed.Wrapf("request %q failed: %v", reqID, err)
}
