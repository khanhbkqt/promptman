// Stress test types matching Go internal/stress/model.go

/** Complete results of a stress test run */
export interface StressReport {
  scenario: string
  duration: number // ms
  summary: StressSummary
  thresholds?: ThresholdResult[]
  timeline: TimelinePoint[]
}

/** Aggregate metrics for a completed stress test */
export interface StressSummary {
  totalRequests: number
  rps: number
  latency: LatencyMetrics
  errorRate: number // 0–100
  throughput: number // bytes/sec
  peakConnections: number
}

/** Latency percentile values in milliseconds */
export interface LatencyMetrics {
  p50: number
  p95: number
  p99: number
}

/** Per-second metrics snapshot for live charting */
export interface TimelinePoint {
  elapsed: number // seconds since start
  rps: number
  p95: number // ms
  errorRate: number // percentage
  activeUsers: number
}

/** Outcome of a single threshold evaluation */
export interface ThresholdResult {
  name: string
  operator: string
  expected: number
  actual: number
  passed: boolean
}

/** Input for POST /api/v1/stress/run */
export interface StressRunInput {
  collection: string
  requestId: string
  users: number
  duration: string
  rampUp?: string
  thresholds?: string[]
  configPath?: string
}

/** Async job response from POST /api/v1/stress/run */
export interface StressJobResponse {
  jobId: string
  message: string
}

/** Stored stress result entry (from GET /api/v1/stress/results) */
export interface StressResultEntry {
  jobId: string
  report: StressReport
}
