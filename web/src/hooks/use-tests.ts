import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api-client'
import { wsClient } from '@/stores/connection-store'
import { useConnectionStore } from '@/stores/connection-store'
import { useEffect } from 'react'
import type { TestResult, TestRunInput } from '@/types/testing'
import type { TestCompletedPayload } from '@/types/daemon'

// --- Query keys ---
const TEST_KEYS = {
  all: ['tests'] as const,
  results: () => [...TEST_KEYS.all, 'results'] as const,
  result: (runId: string) => [...TEST_KEYS.all, 'result', runId] as const,
}

/**
 * Fetch all test results.
 * Automatically invalidated when a `test.completed` WS event fires.
 */
export function useTestResults() {
  const status = useConnectionStore((s) => s.status)
  const queryClient = useQueryClient()

  // Invalidate on WS test.completed
  useEffect(() => {
    const unsub = wsClient.on('test.completed', () => {
      queryClient.invalidateQueries({ queryKey: TEST_KEYS.results() })
    })
    return unsub
  }, [queryClient])

  return useQuery({
    queryKey: TEST_KEYS.results(),
    queryFn: async () => {
      const res = await api.get<TestResult[]>('/api/v1/tests/results')
      if (!res.ok || !res.data) {
        throw new Error(res.error?.message ?? 'Failed to fetch test results')
      }
      return res.data
    },
    enabled: status === 'connected',
  })
}

/**
 * Fetch a single test result by runId.
 */
export function useTestResult(runId: string | null) {
  const status = useConnectionStore((s) => s.status)

  return useQuery({
    queryKey: TEST_KEYS.result(runId ?? ''),
    queryFn: async () => {
      const res = await api.get<TestResult>(`/api/v1/tests/results/${runId}`)
      if (!res.ok || !res.data) {
        throw new Error(res.error?.message ?? 'Failed to fetch test result')
      }
      return res.data
    },
    enabled: status === 'connected' && !!runId,
  })
}

/**
 * Mutation: run tests for a collection via POST /api/v1/tests/run.
 * On success, invalidates the results cache.
 */
export function useRunTests() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (input: TestRunInput) => {
      const res = await api.post<TestResult>('/api/v1/tests/run', input)
      if (!res.ok || !res.data) {
        throw new Error(res.error?.message ?? 'Test execution failed')
      }
      return res.data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: TEST_KEYS.results() })
    },
  })
}
