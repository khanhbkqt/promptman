package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/khanhnguyen/promptman/internal/daemon"
	"github.com/khanhnguyen/promptman/pkg/envelope"
)

const (
	// defaultClientTimeout is the maximum time to wait for a daemon response.
	defaultClientTimeout = 10 * time.Second
)

// Client is an HTTP client that communicates with the running Promptman daemon.
// It reads the .daemon.lock file to discover the daemon's port and auth token,
// validates that the daemon process is alive, and sends Bearer-authenticated
// requests, parsing the envelope.Envelope responses.
type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

// NewClient creates a new Client for the given project directory.
// It reads the .promptman/.daemon.lock file to discover the daemon's address
// and validates that the daemon process is still alive.
// Returns ErrDaemonNotRunning if the lock file is missing or the PID is dead.
func NewClient(projectDir string) (*Client, error) {
	info, err := daemon.ReadLockFile(projectDir)
	if err != nil {
		// Treat missing lock file and corrupt lock file as "daemon not running".
		return nil, ErrDaemonNotRunning.Wrap("lock file unavailable", err)
	}

	// Validate that the PID recorded in the lock file is still alive.
	if !daemon.IsPIDAlive(info.PID) {
		return nil, ErrDaemonNotRunning.Wrap(
			fmt.Sprintf("daemon process %d is not running", info.PID), nil)
	}

	return &Client{
		baseURL: fmt.Sprintf("http://127.0.0.1:%d/api/v1", info.Port),
		token:   info.Token,
		httpClient: &http.Client{
			Timeout: defaultClientTimeout,
		},
	}, nil
}

// NewClientDirect creates a Client with explicit baseURL, token, and http.Client.
// This is intended for tests that provide a pre-configured httptest.Server
// transport, bypassing lock-file discovery and PID validation.
func NewClientDirect(baseURL, token string, httpClient *http.Client) *Client {
	return &Client{
		baseURL:    baseURL,
		token:      token,
		httpClient: httpClient,
	}
}

// Get sends a GET request to the daemon at the given API path (e.g. "/status").
// Returns the parsed envelope.Envelope or a *CLIError on failure.
func (c *Client) Get(path string) (*envelope.Envelope, error) {
	req, err := http.NewRequest(http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return nil, ErrHTTPError.Wrap(fmt.Sprintf("building GET request for %s", path), err)
	}
	c.addAuth(req)

	return c.do(req)
}

// Post sends a POST request with a JSON body to the daemon at the given API path.
// Returns the parsed envelope.Envelope or a *CLIError on failure.
func (c *Client) Post(path string, body any) (*envelope.Envelope, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, ErrHTTPError.Wrap(fmt.Sprintf("marshalling POST body for %s", path), err)
	}

	req, err := http.NewRequest(http.MethodPost, c.baseURL+path, bytes.NewReader(payload))
	if err != nil {
		return nil, ErrHTTPError.Wrap(fmt.Sprintf("building POST request for %s", path), err)
	}
	req.Header.Set("Content-Type", "application/json")
	c.addAuth(req)

	return c.do(req)
}

// Put sends a PUT request with a JSON body to the daemon at the given API path.
// Returns the parsed envelope.Envelope or a *CLIError on failure.
func (c *Client) Put(path string, body any) (*envelope.Envelope, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, ErrHTTPError.Wrap(fmt.Sprintf("marshalling PUT body for %s", path), err)
	}

	req, err := http.NewRequest(http.MethodPut, c.baseURL+path, bytes.NewReader(payload))
	if err != nil {
		return nil, ErrHTTPError.Wrap(fmt.Sprintf("building PUT request for %s", path), err)
	}
	req.Header.Set("Content-Type", "application/json")
	c.addAuth(req)

	return c.do(req)
}

// addAuth sets the Authorization header with the daemon's Bearer token.
func (c *Client) addAuth(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+c.token)
}

// do executes the HTTP request and decodes the response as an envelope.Envelope.
func (c *Client) do(req *http.Request) (*envelope.Envelope, error) {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, ErrDaemonUnreachable.Wrap(
			fmt.Sprintf("daemon did not respond to %s %s", req.Method, req.URL.Path), err)
	}
	defer func() { _ = resp.Body.Close() }()

	var env envelope.Envelope
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		return nil, ErrResponseDecodeError.Wrap(
			fmt.Sprintf("decoding response from %s %s", req.Method, req.URL.Path), err)
	}

	return &env, nil
}
