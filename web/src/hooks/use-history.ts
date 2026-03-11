import { useInfiniteQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api-client'
import { useConnectionStore } from '@/stores/connection-store'
import type { HistoryListResponse, HistoryFilters } from '@/types/history'

const PAGE_SIZE = 50

// --- Query keys ---
const HISTORY_KEYS = {
  all: ['history'] as const,
  list: (filters: HistoryFilters) => [...HISTORY_KEYS.all, 'list', filters] as const,
}

/**
 * Fetch paginated history entries with filters.
 * Uses infinite query for load-more pagination.
 */
export function useHistory(filters: HistoryFilters = {}) {
  const status = useConnectionStore((s) => s.status)

  return useInfiniteQuery({
    queryKey: HISTORY_KEYS.list(filters),
    queryFn: async ({ pageParam = 0 }) => {
      const params = new URLSearchParams()
      params.set('limit', String(PAGE_SIZE))
      params.set('offset', String(pageParam))
      if (filters.collection) params.set('collection', filters.collection)
      if (filters.env) params.set('env', filters.env)
      if (filters.source) params.set('source', filters.source)
      if (filters.status) params.set('status', filters.status)

      const res = await api.get<HistoryListResponse>(`/api/v1/history?${params.toString()}`)
      if (!res.ok || !res.data) {
        throw new Error(res.error?.message ?? 'Failed to fetch history')
      }
      return res.data
    },
    initialPageParam: 0,
    getNextPageParam: (lastPage) => {
      const nextOffset = lastPage.offset + lastPage.limit
      // If we got fewer results than the limit, there are no more pages
      if (lastPage.data.length < lastPage.limit) return undefined
      return nextOffset
    },
    enabled: status === 'connected',
  })
}

/**
 * Mutation: clear history via DELETE /api/v1/history.
 * Invalidates all history queries on success.
 */
export function useClearHistory() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (before?: string) => {
      const params = before ? `?before=${encodeURIComponent(before)}` : ''
      const res = await api.delete<{ deleted: number; message: string }>(
        `/api/v1/history${params}`,
      )
      if (!res.ok || !res.data) {
        throw new Error(res.error?.message ?? 'Failed to clear history')
      }
      return res.data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: HISTORY_KEYS.all })
    },
  })
}
