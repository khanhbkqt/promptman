import { useState, useEffect, useCallback, useRef } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api-client'
import { wsClient } from '@/stores/connection-store'
import { useConnectionStore } from '@/stores/connection-store'
import type {
  StressReport,
  StressRunInput,
  StressJobResponse,
  StressResultEntry,
  TimelinePoint,
} from '@/types/stress'
import type { StressTickPayload, StressCompletedPayload } from '@/types/daemon'

// --- Query keys ---
const STRESS_KEYS = {
  all: ['stress'] as const,
  results: () => [...STRESS_KEYS.all, 'results'] as const,
  result: (jobId: string) => [...STRESS_KEYS.all, 'result', jobId] as const,
}

/**
 * Fetch all stress test results.
 */
export function useStressResults() {
  const status = useConnectionStore((s) => s.status)

  return useQuery({
    queryKey: STRESS_KEYS.results(),
    queryFn: async () => {
      const res = await api.get<StressResultEntry[]>('/api/v1/stress/results')
      if (!res.ok || !res.data) {
        throw new Error(res.error?.message ?? 'Failed to fetch stress results')
      }
      return res.data
    },
    enabled: status === 'connected',
  })
}

/**
 * Fetch a single stress result by jobId.
 */
export function useStressResult(jobId: string | null) {
  const status = useConnectionStore((s) => s.status)

  return useQuery({
    queryKey: STRESS_KEYS.result(jobId ?? ''),
    queryFn: async () => {
      const res = await api.get<StressResultEntry>(`/api/v1/stress/results/${jobId}`)
      if (!res.ok || !res.data) {
        throw new Error(res.error?.message ?? 'Failed to fetch stress result')
      }
      return res.data
    },
    enabled: status === 'connected' && !!jobId,
  })
}

/**
 * Mutation: start a stress test via POST /api/v1/stress/run.
 * Returns the jobId immediately (202).
 */
export function useRunStress() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (input: StressRunInput) => {
      const res = await api.post<StressJobResponse>('/api/v1/stress/run', input)
      if (!res.ok || !res.data) {
        throw new Error(res.error?.message ?? 'Stress test failed to start')
      }
      return res.data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: STRESS_KEYS.results() })
    },
  })
}

/**
 * Live tick accumulator — subscribes to WS `stress.tick` events and
 * accumulates TimelinePoint data for charting.
 * Returns:
 * - ticks: accumulated timeline data
 * - isRunning: whether we're actively receiving ticks
 * - completedReport: the full report once stress.completed fires
 * - reset: clears accumulated data
 */
export function useStressTicks() {
  const [ticks, setTicks] = useState<TimelinePoint[]>([])
  const [isRunning, setIsRunning] = useState(false)
  const [completedReport, setCompletedReport] = useState<StressReport | null>(null)
  const queryClient = useQueryClient()
  const jobIdRef = useRef<string | null>(null)

  // Set the job ID to track
  const trackJob = useCallback((jobId: string) => {
    jobIdRef.current = jobId
    setTicks([])
    setIsRunning(true)
    setCompletedReport(null)
  }, [])

  const reset = useCallback(() => {
    jobIdRef.current = null
    setTicks([])
    setIsRunning(false)
    setCompletedReport(null)
  }, [])

  useEffect(() => {
    const unsubTick = wsClient.on('stress.tick', (payload: StressTickPayload) => {
      if (!isRunning) return
      setTicks((prev) => [
        ...prev,
        {
          elapsed: payload.elapsed,
          rps: payload.rps,
          p95: payload.p95,
          errorRate: payload.errorRate,
          activeUsers: payload.activeUsers,
        },
      ])
    })

    const unsubCompleted = wsClient.on('stress.completed', (payload: StressCompletedPayload) => {
      setIsRunning(false)
      // Fetch the full report
      if (jobIdRef.current) {
        api
          .get<StressResultEntry>(`/api/v1/stress/results/${jobIdRef.current}`)
          .then((res) => {
            if (res.ok && res.data) {
              setCompletedReport(res.data.report)
            }
          })
      }
      queryClient.invalidateQueries({ queryKey: STRESS_KEYS.results() })
    })

    return () => {
      unsubTick()
      unsubCompleted()
    }
  }, [isRunning, queryClient])

  return { ticks, isRunning, completedReport, trackJob, reset }
}
