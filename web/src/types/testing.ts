// Testing types matching Go internal/testing/model.go

/** Complete result of a test suite execution — from GET /api/v1/tests/results */
export interface TestResult {
  runId: string
  collection: string
  env: string
  summary: TestSummary
  tests: TestCase[]
  console?: string[]
}

/** Aggregate counts for a test run */
export interface TestSummary {
  total: number
  passed: number
  failed: number
  skipped: number
  duration: number // milliseconds
}

/** Individual test case result */
export interface TestCase {
  request: string
  name: string
  status: 'passed' | 'failed' | 'timeout' | 'error' | 'skipped'
  duration: number // milliseconds
  error?: TestError
  response?: ResponseSnapshot
  console?: string[]
}

/** Assertion/execution error details */
export interface TestError {
  expected: unknown
  actual: unknown
  message: string
}

/** Lightweight response snapshot attached to failed tests */
export interface ResponseSnapshot {
  status: number
  headers?: Record<string, string>
  body?: string
  time: number // ms
}

/** Input for POST /api/v1/tests/run */
export interface TestRunInput {
  collection: string
  env?: string
  format?: string
  timeout?: string
}
