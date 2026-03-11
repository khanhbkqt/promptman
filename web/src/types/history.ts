// History types matching Go internal/history/model.go

/** Single request execution log entry */
export interface HistoryEntry {
  ts: string // ISO 8601 timestamp
  reqId: string
  collection: string
  method: string
  url: string
  status: number
  time: number // milliseconds
  env: string
  source: 'cli' | 'gui' | 'test'
}

/** Paginated response from GET /api/v1/history */
export interface HistoryListResponse {
  data: HistoryEntry[]
  total: number
  limit: number
  offset: number
}

/** Query filters for history */
export interface HistoryFilters {
  collection?: string
  env?: string
  source?: string
  status?: string
}
