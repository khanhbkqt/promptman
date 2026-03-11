package daemon

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/khanhnguyen/promptman/internal/collection"
	"github.com/khanhnguyen/promptman/internal/stress"
	"github.com/khanhnguyen/promptman/pkg/envelope"
)

// StressExecutor is the interface required by StressRegistrar to run
// stress tests. It decouples the handler from the concrete StressRunner.
type StressExecutor interface {
	Run(opts *stress.StressOpts) (*stress.StressReport, error)
	RunFromConfig(path string) (*stress.StressReport, error)
}

// stressRunRequest is the JSON body for POST /api/v1/stress/run.
type stressRunRequest struct {
	Collection string   `json:"collection"`
	RequestID  string   `json:"requestId"`
	Users      int      `json:"users"`
	Duration   string   `json:"duration"`
	RampUp     string   `json:"rampUp,omitempty"`
	Thresholds []string `json:"thresholds,omitempty"`
	ConfigPath string   `json:"configPath,omitempty"` // mutually exclusive with above
}

// stressJobResponse is returned by the async run endpoint.
type stressJobResponse struct {
	JobID   string `json:"jobId"`
	Message string `json:"message"`
}

// StressRegistrar registers stress test endpoints on the daemon router.
// It follows the same Registrar pattern as TestRunRegistrar and
// RequestRegistrar.
type StressRegistrar struct {
	runner StressExecutor
	store  *StressResultStore
}

// NewStressRegistrar creates a StressRegistrar.
func NewStressRegistrar(runner StressExecutor, store *StressResultStore) *StressRegistrar {
	return &StressRegistrar{
		runner: runner,
		store:  store,
	}
}

// RegisterRoutes mounts stress test endpoints under the given prefix.
func (sr *StressRegistrar) RegisterRoutes(mux *http.ServeMux, prefix string) {
	mux.HandleFunc("POST "+prefix+"stress/run", sr.handleRun())
	mux.HandleFunc("GET "+prefix+"stress/results", sr.handleListResults())
	mux.HandleFunc("GET "+prefix+"stress/results/{jobId}", sr.handleGetResult())
}

// handleRun handles POST /api/v1/stress/run — kicks off a stress test
// asynchronously and returns a job ID immediately (202 Accepted).
func (sr *StressRegistrar) handleRun() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req stressRunRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			envelope.WriteError(w, http.StatusBadRequest,
				envelope.CodeInvalidInput, "invalid JSON body: "+err.Error())
			return
		}

		// Validate: either configPath OR (collection + requestId + users + duration).
		if req.ConfigPath != "" {
			// Config mode — no other fields required.
		} else {
			if req.Collection == "" {
				envelope.WriteError(w, http.StatusBadRequest,
					envelope.CodeInvalidInput, "field 'collection' is required")
				return
			}
			if req.RequestID == "" {
				envelope.WriteError(w, http.StatusBadRequest,
					envelope.CodeInvalidInput, "field 'requestId' is required")
				return
			}
			if req.Users <= 0 {
				envelope.WriteError(w, http.StatusBadRequest,
					envelope.CodeInvalidInput, "field 'users' must be a positive integer")
				return
			}
			if req.Duration == "" {
				envelope.WriteError(w, http.StatusBadRequest,
					envelope.CodeInvalidInput, "field 'duration' is required")
				return
			}
		}

		var idBytes [16]byte
		_, _ = rand.Read(idBytes[:])
		jobID := hex.EncodeToString(idBytes[:])

		// Run asynchronously — results are stored for later retrieval.
		go func() {
			var report *stress.StressReport
			var err error

			if req.ConfigPath != "" {
				report, err = sr.runner.RunFromConfig(req.ConfigPath)
			} else {
				opts := &stress.StressOpts{
					Collection: req.Collection,
					RequestID:  req.RequestID,
					Users:      req.Users,
					Duration:   req.Duration,
					RampUp:     req.RampUp,
					Thresholds: req.Thresholds,
				}
				report, err = sr.runner.Run(opts)
			}

			if err != nil {
				// Store a minimal report with the error scenario name.
				report = &stress.StressReport{
					Scenario: "error: " + err.Error(),
				}
			}

			sr.store.Store(jobID, report)
		}()

		envelope.WriteSuccess(w, http.StatusAccepted, stressJobResponse{
			JobID:   jobID,
			Message: "stress test started",
		})
	}
}

// handleListResults handles GET /api/v1/stress/results — returns latest results.
func (sr *StressRegistrar) handleListResults() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		results := sr.store.Latest(defaultStressResultCapacity)
		if results == nil {
			results = []*stressResultEntry{}
		}
		envelope.WriteSuccess(w, http.StatusOK, results)
	}
}

// handleGetResult handles GET /api/v1/stress/results/{jobId} — returns a specific result.
func (sr *StressRegistrar) handleGetResult() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		jobID := r.PathValue("jobId")
		if jobID == "" {
			envelope.WriteError(w, http.StatusBadRequest,
				envelope.CodeInvalidInput, "jobId is required")
			return
		}

		report, ok := sr.store.Get(jobID)
		if !ok {
			envelope.WriteError(w, http.StatusNotFound,
				"STRESS_RESULT_NOT_FOUND", "no stress result with jobId: "+jobID)
			return
		}
		envelope.WriteSuccess(w, http.StatusOK, report)
	}
}

// writeStressError writes an appropriate error envelope based on the
// stress domain error type.
func writeStressError(w http.ResponseWriter, err error) {
	var sde *stress.DomainError
	if errors.As(err, &sde) {
		statusCode := envelope.HTTPStatusForCode(sde.Code)
		envelope.WriteError(w, statusCode, sde.Code, sde.Message)
		return
	}
	var cde *collection.DomainError
	if errors.As(err, &cde) {
		statusCode := envelope.HTTPStatusForCode(cde.Code)
		envelope.WriteError(w, statusCode, cde.Code, cde.Message)
		return
	}
	envelope.WriteError(w, http.StatusInternalServerError,
		envelope.CodeInternalError, "internal server error")
}
