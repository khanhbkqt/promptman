// Envelope response format matching Go pkg/envelope
export interface ApiResponse<T = unknown> {
  ok: boolean
  data: T | null
  error: ApiError | null
}

export interface ApiError {
  code: string
  message: string
  details?: unknown
}

// DaemonInfo matches Go DaemonInfo model
export interface DaemonInfo {
  pid: number
  port: number
  token: string
  projectDir: string
  startedAt: string
  uptime?: string
}

// WebSocket event types
export type WsEventType =
  | 'data.changed'
  | 'request.completed'
  | 'test.completed'
  | 'approval.pending'
  | 'stress.tick'
  | 'stress.completed'

export interface WsEvent<T = unknown> {
  type: WsEventType
  payload: T
  ts: string
}

// Typed payloads for each event type
export interface DataChangedPayload {
  source: 'collections' | 'environments' | 'tests' | 'config'
}

export interface RequestCompletedPayload {
  reqId: string
  status: number
  time: number // ms
}

export interface TestCompletedPayload {
  collection: string
  passed: number
  failed: number
  total: number
  duration: number // ms
}

export interface ApprovalPendingPayload {
  actionId: string
  actionType: string
  details: string
}

export interface StressTickPayload {
  elapsed: number
  rps: number
  p95: number
  errorRate: number
  activeUsers: number
}

export interface StressCompletedPayload {
  scenario: string
  summary: unknown
}

// Connection status
export type ConnectionStatus = 'disconnected' | 'connecting' | 'connected' | 'reconnecting'

// Daemon config — either from lockfile discovery or manual input
export interface DaemonConfig {
  port: number
  token: string
}
