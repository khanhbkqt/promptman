package daemon

import (
	"encoding/json"
	"net/http"

	"github.com/khanhnguyen/promptman/internal/environment"
	"github.com/khanhnguyen/promptman/pkg/envelope"
)

// envListItem is the JSON shape returned by the list endpoint.
// It extends EnvSummary with an Active boolean.
type envListItem struct {
	Name          string `json:"name"`
	VariableCount int    `json:"variableCount"`
	SecretCount   int    `json:"secretCount"`
	Active        bool   `json:"active"`
}

// setActiveRequest is the JSON body for POST /environments/active.
type setActiveRequest struct {
	Name string `json:"name"`
}

// setActiveResponse is the JSON body returned after setting the active environment.
type setActiveResponse struct {
	Message string `json:"message"`
}

// EnvironmentRegistrar registers environment management endpoints on the daemon router.
type EnvironmentRegistrar struct {
	svc *environment.Service
}

// NewEnvironmentRegistrar creates an EnvironmentRegistrar for the given Service.
func NewEnvironmentRegistrar(svc *environment.Service) *EnvironmentRegistrar {
	return &EnvironmentRegistrar{svc: svc}
}

// RegisterRoutes mounts the environment endpoints under the given prefix.
func (er *EnvironmentRegistrar) RegisterRoutes(mux *http.ServeMux, prefix string) {
	mux.HandleFunc("GET "+prefix+"environments", er.handleList())
	mux.HandleFunc("GET "+prefix+"environments/{name}", er.handleGet())
	mux.HandleFunc("POST "+prefix+"environments/active", er.handleSetActive())
	mux.HandleFunc("PUT "+prefix+"environments/{name}", er.handleUpdate())
}

// handleList handles GET /api/v1/environments — list all environments with active marker.
func (er *EnvironmentRegistrar) handleList() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		summaries, err := er.svc.List()
		if err != nil {
			er.writeEnvError(w, err)
			return
		}

		// Get the active environment name (may be empty if none set).
		activeName, _ := er.svc.GetActive()

		items := make([]envListItem, len(summaries))
		for i, s := range summaries {
			items[i] = envListItem{
				Name:          s.Name,
				VariableCount: s.VariableCount,
				SecretCount:   s.SecretCount,
				Active:        s.Name == activeName,
			}
		}

		envelope.WriteSuccess(w, http.StatusOK, items)
	}
}

// handleGet handles GET /api/v1/environments/{name} — get a single environment (secrets masked).
func (er *EnvironmentRegistrar) handleGet() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		if name == "" {
			envelope.WriteError(w, http.StatusBadRequest,
				envelope.CodeInvalidInput, "environment name is required")
			return
		}

		env, err := er.svc.Get(name)
		if err != nil {
			er.writeEnvError(w, err)
			return
		}

		envelope.WriteSuccess(w, http.StatusOK, env)
	}
}

// handleSetActive handles POST /api/v1/environments/active — set the active environment.
func (er *EnvironmentRegistrar) handleSetActive() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req setActiveRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			envelope.WriteError(w, http.StatusBadRequest,
				envelope.CodeInvalidInput, "invalid JSON body: "+err.Error())
			return
		}

		if req.Name == "" {
			envelope.WriteError(w, http.StatusBadRequest,
				envelope.CodeInvalidInput, "field 'name' is required")
			return
		}

		if err := er.svc.SetActive(req.Name); err != nil {
			er.writeEnvError(w, err)
			return
		}

		envelope.WriteSuccess(w, http.StatusOK, setActiveResponse{
			Message: "Active environment set to: " + req.Name,
		})
	}
}

// handleUpdate handles PUT /api/v1/environments/{name} — upsert environment variables.
// If the environment already exists it is updated (HTTP 200).
// If the environment does not yet exist it is created (HTTP 201) — PUT is idempotent upsert.
func (er *EnvironmentRegistrar) handleUpdate() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		if name == "" {
			envelope.WriteError(w, http.StatusBadRequest,
				envelope.CodeInvalidInput, "environment name is required")
			return
		}

		var input environment.UpdateEnvInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			envelope.WriteError(w, http.StatusBadRequest,
				envelope.CodeInvalidInput, "invalid JSON body: "+err.Error())
			return
		}

		env, err := er.svc.Update(name, &input)
		if err != nil {
			// If the environment doesn't exist yet, create it (upsert semantics).
			de, ok := err.(*environment.DomainError)
			if ok && de.Code == envelope.CodeEnvNotFound {
				createInput := &environment.CreateEnvInput{Name: name}
				if input.Variables != nil {
					createInput.Variables = *input.Variables
				}
				if input.Secrets != nil {
					createInput.Secrets = *input.Secrets
				}
				created, createErr := er.svc.Create(createInput)
				if createErr != nil {
					er.writeEnvError(w, createErr)
					return
				}
				envelope.WriteSuccess(w, http.StatusCreated, created)
				return
			}
			er.writeEnvError(w, err)
			return
		}

		envelope.WriteSuccess(w, http.StatusOK, env)
	}
}

// writeEnvError writes an appropriate error envelope based on the environment error type.
func (er *EnvironmentRegistrar) writeEnvError(w http.ResponseWriter, err error) {
	de, ok := err.(*environment.DomainError)
	if ok {
		statusCode := envelope.HTTPStatusForCode(de.Code)
		envelope.WriteError(w, statusCode, de.Code, de.Message)
		return
	}
	envelope.WriteError(w, http.StatusInternalServerError,
		envelope.CodeInternalError, err.Error())
}
