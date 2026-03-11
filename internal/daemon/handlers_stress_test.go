package daemon

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/khanhnguyen/promptman/internal/stress"
	"github.com/khanhnguyen/promptman/pkg/envelope"
)

// mockStressExecutor is a test double for StressExecutor.
type mockStressExecutor struct {
	report *stress.StressReport
	err    error
}

func (m *mockStressExecutor) Run(_ *stress.StressOpts) (*stress.StressReport, error) {
	return m.report, m.err
}

func (m *mockStressExecutor) RunFromConfig(_ string) (*stress.StressReport, error) {
	return m.report, m.err
}

func TestStressRegistrar_HandleRun_MissingCollection(t *testing.T) {
	reg := NewStressRegistrar(&mockStressExecutor{}, NewStressResultStore(0))
	mux := http.NewServeMux()
	reg.RegisterRoutes(mux, "/api/v1/")

	body := `{"requestId":"r1","users":10,"duration":"5s"}`
	req := httptest.NewRequest("POST", "/api/v1/stress/run", strings.NewReader(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d; want %d", w.Code, http.StatusBadRequest)
	}
}

func TestStressRegistrar_HandleRun_MissingRequestID(t *testing.T) {
	reg := NewStressRegistrar(&mockStressExecutor{}, NewStressResultStore(0))
	mux := http.NewServeMux()
	reg.RegisterRoutes(mux, "/api/v1/")

	body := `{"collection":"users","users":10,"duration":"5s"}`
	req := httptest.NewRequest("POST", "/api/v1/stress/run", strings.NewReader(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d; want %d", w.Code, http.StatusBadRequest)
	}
}

func TestStressRegistrar_HandleRun_InvalidUsers(t *testing.T) {
	reg := NewStressRegistrar(&mockStressExecutor{}, NewStressResultStore(0))
	mux := http.NewServeMux()
	reg.RegisterRoutes(mux, "/api/v1/")

	body := `{"collection":"users","requestId":"r1","users":0,"duration":"5s"}`
	req := httptest.NewRequest("POST", "/api/v1/stress/run", strings.NewReader(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d; want %d", w.Code, http.StatusBadRequest)
	}
}

func TestStressRegistrar_HandleRun_MissingDuration(t *testing.T) {
	reg := NewStressRegistrar(&mockStressExecutor{}, NewStressResultStore(0))
	mux := http.NewServeMux()
	reg.RegisterRoutes(mux, "/api/v1/")

	body := `{"collection":"users","requestId":"r1","users":10}`
	req := httptest.NewRequest("POST", "/api/v1/stress/run", strings.NewReader(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d; want %d", w.Code, http.StatusBadRequest)
	}
}

func TestStressRegistrar_HandleRun_InvalidJSON(t *testing.T) {
	reg := NewStressRegistrar(&mockStressExecutor{}, NewStressResultStore(0))
	mux := http.NewServeMux()
	reg.RegisterRoutes(mux, "/api/v1/")

	req := httptest.NewRequest("POST", "/api/v1/stress/run", strings.NewReader("{bad"))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d; want %d", w.Code, http.StatusBadRequest)
	}
}

func TestStressRegistrar_HandleRun_Accepted(t *testing.T) {
	report := &stress.StressReport{
		Scenario: "test",
		Duration: 5000,
		Summary:  stress.StressSummary{TotalRequests: 100},
	}
	executor := &mockStressExecutor{report: report}
	store := NewStressResultStore(10)
	reg := NewStressRegistrar(executor, store)
	mux := http.NewServeMux()
	reg.RegisterRoutes(mux, "/api/v1/")

	body := `{"collection":"users","requestId":"r1","users":10,"duration":"5s"}`
	req := httptest.NewRequest("POST", "/api/v1/stress/run", strings.NewReader(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("status = %d; want %d", w.Code, http.StatusAccepted)
	}

	var resp envelope.Envelope
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if !resp.OK {
		t.Error("expected ok = true")
	}
}

func TestStressRegistrar_HandleRun_ConfigPath(t *testing.T) {
	report := &stress.StressReport{
		Scenario: "from-config",
		Duration: 10000,
		Summary:  stress.StressSummary{TotalRequests: 500},
	}
	executor := &mockStressExecutor{report: report}
	store := NewStressResultStore(10)
	reg := NewStressRegistrar(executor, store)
	mux := http.NewServeMux()
	reg.RegisterRoutes(mux, "/api/v1/")

	body := `{"configPath":"./stress.yaml"}`
	req := httptest.NewRequest("POST", "/api/v1/stress/run", strings.NewReader(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("status = %d; want %d", w.Code, http.StatusAccepted)
	}
}

func TestStressRegistrar_HandleGetResult_NotFound(t *testing.T) {
	store := NewStressResultStore(10)
	reg := NewStressRegistrar(&mockStressExecutor{}, store)
	mux := http.NewServeMux()
	reg.RegisterRoutes(mux, "/api/v1/")

	req := httptest.NewRequest("GET", "/api/v1/stress/results/nonexistent", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d; want %d", w.Code, http.StatusNotFound)
	}
}

func TestStressRegistrar_HandleGetResult_Found(t *testing.T) {
	store := NewStressResultStore(10)
	report := &stress.StressReport{
		Scenario: "stored",
		Duration: 5000,
		Summary:  stress.StressSummary{TotalRequests: 100, RPS: 20},
	}
	store.Store("job-123", report)

	reg := NewStressRegistrar(&mockStressExecutor{}, store)
	mux := http.NewServeMux()
	reg.RegisterRoutes(mux, "/api/v1/")

	req := httptest.NewRequest("GET", "/api/v1/stress/results/job-123", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d; want %d", w.Code, http.StatusOK)
	}
}

func TestStressRegistrar_HandleListResults_Empty(t *testing.T) {
	store := NewStressResultStore(10)
	reg := NewStressRegistrar(&mockStressExecutor{}, store)
	mux := http.NewServeMux()
	reg.RegisterRoutes(mux, "/api/v1/")

	req := httptest.NewRequest("GET", "/api/v1/stress/results", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d; want %d", w.Code, http.StatusOK)
	}
}

func TestStressRegistrar_HandleListResults_WithData(t *testing.T) {
	store := NewStressResultStore(10)
	for i := 0; i < 3; i++ {
		store.Store("job-"+string(rune('a'+i)), &stress.StressReport{
			Scenario: "test-" + string(rune('a'+i)),
		})
	}

	reg := NewStressRegistrar(&mockStressExecutor{}, store)
	mux := http.NewServeMux()
	reg.RegisterRoutes(mux, "/api/v1/")

	req := httptest.NewRequest("GET", "/api/v1/stress/results", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d; want %d", w.Code, http.StatusOK)
	}
}

func TestStressRegistrar_AsyncExecution_StoresResult(t *testing.T) {
	report := &stress.StressReport{
		Scenario: "async-test",
		Duration: 1000,
		Summary:  stress.StressSummary{TotalRequests: 50},
	}
	executor := &mockStressExecutor{report: report}
	store := NewStressResultStore(10)
	reg := NewStressRegistrar(executor, store)
	mux := http.NewServeMux()
	reg.RegisterRoutes(mux, "/api/v1/")

	// Kick off the async run.
	body := `{"collection":"users","requestId":"r1","users":5,"duration":"1s"}`
	req := httptest.NewRequest("POST", "/api/v1/stress/run", strings.NewReader(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("status = %d; want %d", w.Code, http.StatusAccepted)
	}

	// Extract jobId from response.
	var resp struct {
		Data struct {
			JobID string `json:"jobId"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	jobID := resp.Data.JobID
	if jobID == "" {
		t.Fatal("expected non-empty jobId")
	}

	// Wait briefly for the goroutine to store the result.
	time.Sleep(100 * time.Millisecond)

	// Verify the result is retrievable.
	result, ok := store.Get(jobID)
	if !ok {
		t.Fatal("expected result to be stored after async execution")
	}
	if result.Scenario != "async-test" {
		t.Errorf("Scenario = %q; want %q", result.Scenario, "async-test")
	}
}
