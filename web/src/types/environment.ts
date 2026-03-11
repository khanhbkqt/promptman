// Environment types matching Go daemon API responses.
// Go source: internal/environment/model.go

/**
 * EnvListItem matches the JSON shape returned by GET /api/v1/environments.
 * Extends EnvSummary with an Active boolean set by the daemon.
 */
export interface EnvListItem {
  name: string
  variableCount: number
  secretCount: number
  active: boolean
}

/**
 * Environment matches the JSON shape returned by GET /api/v1/environments/{name}.
 * Variables use Record<string, unknown> to preserve native YAML types.
 * Secrets are map[string]string containing raw $ENV{} references (masked by daemon).
 */
export interface Environment {
  name: string
  variables?: Record<string, unknown>
  secrets?: Record<string, string>
}

/**
 * UpdateEnvInput matches the Go UpdateEnvInput struct.
 * Only non-undefined fields are applied; undefined fields are left unchanged.
 */
export interface UpdateEnvInput {
  name?: string
  variables?: Record<string, unknown>
  secrets?: Record<string, string>
}

/**
 * SetActiveRequest is the JSON body for POST /api/v1/environments/active.
 */
export interface SetActiveRequest {
  name: string
}

/**
 * SetActiveResponse is the JSON body returned after setting the active env.
 */
export interface SetActiveResponse {
  message: string
}
