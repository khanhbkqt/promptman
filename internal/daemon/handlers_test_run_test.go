package daemon

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	tmod "github.com/khanhnguyen/promptman/internal/testing"
	"github.com/khanhnguyen/promptman/pkg/envelope"
)

// --- Mock SuiteRunner ---

type mockSuiteRunner struct {
	result *tmod.TestResult
	err    error
}

func (m *mockSuiteRunner) RunSuite(_ context.Context, collID, env string, _ time.Duration) (*tmod.TestResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.result, nil
}

// --- Helper ---

func newTestRunRegistrar(result *tmod.TestResult, err error) *TestRunRegistrar {
	runner := &mockSuiteRunner{result: result, err: err}
	store := NewResultStore(5)
	return NewTestRunRegistrar(runner, nil, store)
}

func makeResult() *tmod.TestResult {
	return &tmod.TestResult{
		RunID:      "run-abc",
		Collection: "users",
		Env:        "dev",
		Summary: tmod.TestSummary{
			Total:    2,
			Passed:   2,
			Duration: 100,
		},
		Tests: []tmod.TestCase{
			{Request: "users/list", Name: "is 200", Status: "passed", Duration: 50},
			{Request: "users/get", Name: "returns user", Status: "passed", Duration: 50},
		},
	}
}

// --- POST /api/v1/tests/run ---

func TestHandleRun_Success(t *testing.T) {
	reg := newTestRunRegistrar(makeResult(), nil)

	mux := http.NewServeMux()
	reg.RegisterRoutes(mux, "/api/v1/")

	body := `{"collection":"users","env":"dev"}`
	req := httptest.NewRequest("POST", "/api/v1/tests/run", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d; want %d\n%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var env envelope.Envelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("unmarshal envelope: %v", err)
	}
	if !env.OK {
		t.Error("expected success envelope")
	}
}

func TestHandleRun_MissingCollection(t *testing.T) {
	reg := newTestRunRegistrar(makeResult(), nil)

	mux := http.NewServeMux()
	reg.RegisterRoutes(mux, "/api/v1/")

	body := `{"env":"dev"}`
	req := httptest.NewRequest("POST", "/api/v1/tests/run", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d; want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandleRun_InvalidJSON(t *testing.T) {
	reg := newTestRunRegistrar(makeResult(), nil)

	mux := http.NewServeMux()
	reg.RegisterRoutes(mux, "/api/v1/")

	req := httptest.NewRequest("POST", "/api/v1/tests/run", bytes.NewBufferString("not json"))
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d; want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandleRun_RunnerError(t *testing.T) {
	reg := newTestRunRegistrar(nil, errors.New("boom"))

	mux := http.NewServeMux()
	reg.RegisterRoutes(mux, "/api/v1/")

	body := `{"collection":"users"}`
	req := httptest.NewRequest("POST", "/api/v1/tests/run", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d; want %d", rec.Code, http.StatusInternalServerError)
	}
}

func TestHandleRun_MinimalFormat(t *testing.T) {
	reg := newTestRunRegistrar(makeResult(), nil)

	mux := http.NewServeMux()
	reg.RegisterRoutes(mux, "/api/v1/")

	body := `{"collection":"users","format":"minimal"}`
	req := httptest.NewRequest("POST", "/api/v1/tests/run", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d; want %d", rec.Code, http.StatusOK)
	}

	// Minimal format returns plain text, not JSON envelope.
	ct := rec.Header().Get("Content-Type")
	if ct != "text/plain; charset=utf-8" {
		t.Errorf("Content-Type = %q; want text/plain", ct)
	}
}

func TestHandleRun_UnknownFormat(t *testing.T) {
	reg := newTestRunRegistrar(makeResult(), nil)

	mux := http.NewServeMux()
	reg.RegisterRoutes(mux, "/api/v1/")

	body := `{"collection":"users","format":"html"}`
	req := httptest.NewRequest("POST", "/api/v1/tests/run", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d; want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandleRun_InvalidTimeout(t *testing.T) {
	reg := newTestRunRegistrar(makeResult(), nil)

	mux := http.NewServeMux()
	reg.RegisterRoutes(mux, "/api/v1/")

	body := `{"collection":"users","timeout":"not-a-duration"}`
	req := httptest.NewRequest("POST", "/api/v1/tests/run", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d; want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandleRun_StoresResult(t *testing.T) {
	runner := &mockSuiteRunner{result: makeResult()}
	store := NewResultStore(5)
	reg := NewTestRunRegistrar(runner, nil, store)

	mux := http.NewServeMux()
	reg.RegisterRoutes(mux, "/api/v1/")

	body := `{"collection":"users"}`
	req := httptest.NewRequest("POST", "/api/v1/tests/run", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if store.Len() != 1 {
		t.Errorf("store.Len() = %d; want 1", store.Len())
	}
}

// --- GET /api/v1/tests/results ---

func TestHandleListResults_Empty(t *testing.T) {
	reg := newTestRunRegistrar(makeResult(), nil)

	mux := http.NewServeMux()
	reg.RegisterRoutes(mux, "/api/v1/")

	req := httptest.NewRequest("GET", "/api/v1/tests/results", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d; want %d", rec.Code, http.StatusOK)
	}
}

func TestHandleListResults_WithData(t *testing.T) {
	runner := &mockSuiteRunner{result: makeResult()}
	store := NewResultStore(5)
	reg := NewTestRunRegistrar(runner, nil, store)

	// Pre-populate store.
	store.Store(makeResult())

	mux := http.NewServeMux()
	reg.RegisterRoutes(mux, "/api/v1/")

	req := httptest.NewRequest("GET", "/api/v1/tests/results", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d; want %d", rec.Code, http.StatusOK)
	}

	var env envelope.Envelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !env.OK {
		t.Error("expected success envelope")
	}
}

// --- GET /api/v1/tests/results/{runId} ---

func TestHandleGetResult_Found(t *testing.T) {
	runner := &mockSuiteRunner{result: makeResult()}
	store := NewResultStore(5)
	reg := NewTestRunRegistrar(runner, nil, store)

	store.Store(makeResult())

	mux := http.NewServeMux()
	reg.RegisterRoutes(mux, "/api/v1/")

	req := httptest.NewRequest("GET", "/api/v1/tests/results/run-abc", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d; want %d\n%s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

func TestHandleGetResult_NotFound(t *testing.T) {
	reg := newTestRunRegistrar(makeResult(), nil)

	mux := http.NewServeMux()
	reg.RegisterRoutes(mux, "/api/v1/")

	req := httptest.NewRequest("GET", "/api/v1/tests/results/nonexistent", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d; want %d", rec.Code, http.StatusNotFound)
	}
}
