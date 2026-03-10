package daemon

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	testing "github.com/khanhnguyen/promptman/internal/testing"
	"github.com/khanhnguyen/promptman/internal/testing/reporter"
	"github.com/khanhnguyen/promptman/internal/ws"
	"github.com/khanhnguyen/promptman/pkg/envelope"
)

// SuiteRunner is the interface required by TestRunRegistrar to execute
// test suites. It decouples the handler from the concrete Runner
// implementation in internal/testing/core.
//
// The suiteTimeout parameter overrides the default suite timeout.
// Pass 0 to use the runner's default.
type SuiteRunner interface {
	RunSuite(ctx context.Context, collID, env string, suiteTimeout time.Duration) (*testing.TestResult, error)
}

// testRunRequest is the JSON body for POST /api/v1/tests/run.
type testRunRequest struct {
	Collection string `json:"collection"`
	Env        string `json:"env,omitempty"`
	Format     string `json:"format,omitempty"`
	Timeout    string `json:"timeout,omitempty"`
}

// TestRunRegistrar registers test execution and result query endpoints
// on the daemon router. It follows the same Registrar pattern as
// RequestRegistrar and EnvironmentRegistrar.
type TestRunRegistrar struct {
	runner SuiteRunner
	hub    *ws.Hub
	store  *ResultStore
}

// NewTestRunRegistrar creates a TestRunRegistrar.
// If hub is nil, WebSocket broadcasting is skipped.
func NewTestRunRegistrar(runner SuiteRunner, hub *ws.Hub, store *ResultStore) *TestRunRegistrar {
	return &TestRunRegistrar{
		runner: runner,
		hub:    hub,
		store:  store,
	}
}

// RegisterRoutes mounts test run endpoints under the given prefix.
func (tr *TestRunRegistrar) RegisterRoutes(mux *http.ServeMux, prefix string) {
	mux.HandleFunc("POST "+prefix+"tests/run", tr.handleRun())
	mux.HandleFunc("GET "+prefix+"tests/results", tr.handleListResults())
	mux.HandleFunc("GET "+prefix+"tests/results/{runId}", tr.handleGetResult())
}

// handleRun handles POST /api/v1/tests/run — runs a test suite and
// returns the result. Optionally formats output via the reporter package.
func (tr *TestRunRegistrar) handleRun() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req testRunRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			envelope.WriteError(w, http.StatusBadRequest,
				envelope.CodeInvalidInput, "invalid JSON body: "+err.Error())
			return
		}

		if req.Collection == "" {
			envelope.WriteError(w, http.StatusBadRequest,
				envelope.CodeInvalidInput, "field 'collection' is required")
			return
		}

		// Parse optional timeout override.
		var suiteTimeout time.Duration
		if req.Timeout != "" {
			d, err := time.ParseDuration(req.Timeout)
			if err != nil {
				envelope.WriteError(w, http.StatusBadRequest,
					envelope.CodeInvalidInput, "invalid timeout duration: "+err.Error())
				return
			}
			suiteTimeout = d
		}

		// Execute the test suite.
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
		defer cancel()

		result, err := tr.runner.RunSuite(ctx, req.Collection, req.Env, suiteTimeout)
		if err != nil {
			envelope.WriteError(w, http.StatusInternalServerError,
				"TEST_EXECUTION_ERROR", "test execution failed: "+err.Error())
			return
		}

		// Store the result.
		tr.store.Store(result)

		// Broadcast test.completed event.
		if tr.hub != nil {
			tr.hub.Broadcast(ws.NewEvent(ws.EventTestCompleted,
				ws.TestCompletedPayload{
					Collection: result.Collection,
					Passed:     result.Summary.Passed,
					Failed:     result.Summary.Failed,
					Total:      result.Summary.Total,
					Duration:   int64(result.Summary.Duration),
				}))
		}

		// Format output if a specific format is requested.
		if req.Format != "" && req.Format != reporter.FormatJSON {
			rep, err := reporter.ForFormat(req.Format)
			if err != nil {
				envelope.WriteError(w, http.StatusBadRequest,
					envelope.CodeInvalidInput, err.Error())
				return
			}
			out, err := rep.Format(result)
			if err != nil {
				envelope.WriteError(w, http.StatusInternalServerError,
					"TEST_FORMAT_ERROR", "formatting test results: "+err.Error())
				return
			}
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(out)
			return
		}

		// Default: JSON envelope.
		envelope.WriteSuccess(w, http.StatusOK, result)
	}
}

// handleListResults handles GET /api/v1/tests/results — returns latest results.
func (tr *TestRunRegistrar) handleListResults() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		results := tr.store.Latest(defaultResultCapacity)
		if results == nil {
			results = []*testing.TestResult{}
		}
		envelope.WriteSuccess(w, http.StatusOK, results)
	}
}

// handleGetResult handles GET /api/v1/tests/results/{runId} — returns a specific result.
func (tr *TestRunRegistrar) handleGetResult() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		runID := r.PathValue("runId")
		if runID == "" {
			envelope.WriteError(w, http.StatusBadRequest,
				envelope.CodeInvalidInput, "runId is required")
			return
		}

		result, ok := tr.store.Get(runID)
		if !ok {
			envelope.WriteError(w, http.StatusNotFound,
				"TEST_RESULT_NOT_FOUND", "no test result with runId: "+runID)
			return
		}
		envelope.WriteSuccess(w, http.StatusOK, result)
	}
}
