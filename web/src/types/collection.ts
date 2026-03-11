// Collection types matching Go daemon models (internal/collection/model.go)

/** Lightweight projection used in listings — from GET /api/v1/collections */
export interface CollectionSummary {
  id: string
  name: string
  requestCount: number
}

/** Full collection — from GET /api/v1/collections/{id} */
export interface Collection {
  name: string
  baseUrl?: string
  defaults?: RequestDefaults
  auth?: AuthConfig
  requests?: RequestItem[]
  folders?: Folder[]
}

/** Single HTTP request within a collection or folder */
export interface RequestItem {
  id: string
  method: string
  path: string
  headers?: Record<string, string>
  body?: RequestBody
  auth?: AuthConfig
  timeout?: number
}

/** Logical grouping of requests — can nest arbitrarily */
export interface Folder {
  id: string
  name: string
  auth?: AuthConfig
  defaults?: RequestDefaults
  requests?: RequestItem[]
  folders?: Folder[]
}

/** Inheritable default values — cascade: collection → folder → request */
export interface RequestDefaults {
  headers?: Record<string, string>
  timeout?: number
}

/** Auth config — exactly one variant should be set matching `type` */
export interface AuthConfig {
  type: 'bearer' | 'basic' | 'api-key'
  bearer?: { token: string }
  basic?: { username: string; password: string }
  apiKey?: { key: string; value: string }
}

/** Request body descriptor */
export interface RequestBody {
  type: 'json' | 'form' | 'raw'
  content?: unknown
}

// --- Request execution types (matching Go internal/request/models.go) ---

/** Input for POST /api/v1/run */
export interface ExecuteInput {
  collection: string
  requestId: string
  env?: string
  variables?: Record<string, unknown>
  skipTlsVerify?: boolean
  source?: 'cli' | 'gui' | 'test'
}

/** Response from POST /api/v1/run */
export interface ExecuteResponse {
  requestId: string
  method: string
  url: string
  status: number
  headers: Record<string, string>
  body: string
  timing: RequestTiming
  error?: string
}

/** Timing breakdown in milliseconds */
export interface RequestTiming {
  dns: number
  connect: number
  tls: number
  ttfb: number
  transfer: number
  total: number
}
