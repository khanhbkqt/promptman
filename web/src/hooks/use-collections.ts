import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api-client'
import { wsClient } from '@/stores/connection-store'
import { useConnectionStore } from '@/stores/connection-store'
import { useEffect } from 'react'
import type {
  CollectionSummary,
  Collection,
  ExecuteInput,
  ExecuteResponse,
} from '@/types/collection'
import type { DataChangedPayload } from '@/types/daemon'

// --- Query keys ---
const COLLECTION_KEYS = {
  all: ['collections'] as const,
  list: () => [...COLLECTION_KEYS.all, 'list'] as const,
  detail: (id: string) => [...COLLECTION_KEYS.all, 'detail', id] as const,
}

/**
 * Fetch all collections as summaries.
 * Automatically invalidated when a `data.changed` WS event fires for `collections`.
 */
export function useCollections() {
  const status = useConnectionStore((s) => s.status)
  const queryClient = useQueryClient()

  // Invalidate on WS data.changed for collections
  useEffect(() => {
    const unsub = wsClient.on('data.changed', (event) => {
      const payload = event.payload as DataChangedPayload
      if (payload.source === 'collections') {
        queryClient.invalidateQueries({ queryKey: COLLECTION_KEYS.list() })
      }
    })
    return unsub
  }, [queryClient])

  return useQuery({
    queryKey: COLLECTION_KEYS.list(),
    queryFn: async () => {
      const res = await api.get<CollectionSummary[]>('/api/v1/collections')
      if (!res.ok || !res.data) {
        throw new Error(res.error?.message ?? 'Failed to fetch collections')
      }
      return res.data
    },
    enabled: status === 'connected',
  })
}

/**
 * Fetch a single collection by ID with full details (folders, requests).
 */
export function useCollection(id: string | null) {
  const status = useConnectionStore((s) => s.status)
  const queryClient = useQueryClient()

  // Invalidate on WS data.changed for collections
  useEffect(() => {
    const unsub = wsClient.on('data.changed', (event) => {
      const payload = event.payload as DataChangedPayload
      if (payload.source === 'collections' && id) {
        queryClient.invalidateQueries({ queryKey: COLLECTION_KEYS.detail(id) })
      }
    })
    return unsub
  }, [queryClient, id])

  return useQuery({
    queryKey: COLLECTION_KEYS.detail(id ?? ''),
    queryFn: async () => {
      const res = await api.get<Collection>(`/api/v1/collections/${id}`)
      if (!res.ok || !res.data) {
        throw new Error(res.error?.message ?? 'Failed to fetch collection')
      }
      return res.data
    },
    enabled: status === 'connected' && !!id,
  })
}

/**
 * Mutation: execute a single request via POST /api/v1/run.
 */
export function useSendRequest() {
  return useMutation({
    mutationFn: async (input: ExecuteInput) => {
      const res = await api.post<ExecuteResponse>('/api/v1/run', input)
      if (!res.ok || !res.data) {
        throw new Error(res.error?.message ?? 'Request execution failed')
      }
      return res.data
    },
  })
}

/**
 * Mutation: update a collection via PUT /api/v1/collections/{id}.
 * Invalidates both the list and detail caches on success.
 */
export function useUpdateCollection() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async ({ id, data }: { id: string; data: Partial<Collection> }) => {
      const res = await api.put<Collection>(`/api/v1/collections/${id}`, data)
      if (!res.ok || !res.data) {
        throw new Error(res.error?.message ?? 'Failed to update collection')
      }
      return res.data
    },
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: COLLECTION_KEYS.list() })
      queryClient.invalidateQueries({ queryKey: COLLECTION_KEYS.detail(variables.id) })
    },
  })
}
