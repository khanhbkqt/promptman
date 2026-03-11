import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api-client'
import { wsClient } from '@/stores/connection-store'
import { useConnectionStore } from '@/stores/connection-store'
import { useEnvironmentStore } from '@/stores/environment-store'
import { useEffect } from 'react'
import type { EnvListItem, Environment, UpdateEnvInput, SetActiveResponse } from '@/types/environment'
import type { DataChangedPayload } from '@/types/daemon'

// --- Query keys ---
const ENV_KEYS = {
  all: ['environments'] as const,
  list: () => [...ENV_KEYS.all, 'list'] as const,
  detail: (name: string) => [...ENV_KEYS.all, 'detail', name] as const,
}

/**
 * Fetch all environments as summaries with active marker.
 * Automatically invalidated when a `data.changed` WS event fires for `environments`.
 * Also syncs the Zustand activeEnv store from server data.
 */
export function useEnvironments() {
  const status = useConnectionStore((s) => s.status)
  const queryClient = useQueryClient()
  const setActiveEnv = useEnvironmentStore((s) => s.setActiveEnv)

  // Invalidate on WS data.changed for environments
  useEffect(() => {
    const unsub = wsClient.on('data.changed', (event) => {
      const payload = event.payload as DataChangedPayload
      if (payload.source === 'environments') {
        queryClient.invalidateQueries({ queryKey: ENV_KEYS.list() })
      }
    })
    return unsub
  }, [queryClient])

  return useQuery({
    queryKey: ENV_KEYS.list(),
    queryFn: async () => {
      const res = await api.get<EnvListItem[]>('/api/v1/environments')
      if (!res.ok || !res.data) {
        throw new Error(res.error?.message ?? 'Failed to fetch environments')
      }
      // Sync active env to Zustand store
      const active = res.data.find((e) => e.active)
      if (active) {
        setActiveEnv(active.name)
      }
      return res.data
    },
    enabled: status === 'connected',
  })
}

/**
 * Fetch a single environment by name with full details.
 * Secrets are masked by the daemon.
 */
export function useEnvironment(name: string | null) {
  const status = useConnectionStore((s) => s.status)
  const queryClient = useQueryClient()

  // Invalidate on WS data.changed for environments
  useEffect(() => {
    const unsub = wsClient.on('data.changed', (event) => {
      const payload = event.payload as DataChangedPayload
      if (payload.source === 'environments' && name) {
        queryClient.invalidateQueries({ queryKey: ENV_KEYS.detail(name) })
      }
    })
    return unsub
  }, [queryClient, name])

  return useQuery({
    queryKey: ENV_KEYS.detail(name ?? ''),
    queryFn: async () => {
      const res = await api.get<Environment>(`/api/v1/environments/${name}`)
      if (!res.ok || !res.data) {
        throw new Error(res.error?.message ?? 'Failed to fetch environment')
      }
      return res.data
    },
    enabled: status === 'connected' && !!name,
  })
}

/**
 * Mutation: set the active environment via POST /api/v1/environments/active.
 * Optimistically updates the Zustand store and invalidates the environments list.
 */
export function useSetActiveEnvironment() {
  const queryClient = useQueryClient()
  const setActiveEnv = useEnvironmentStore((s) => s.setActiveEnv)

  return useMutation({
    mutationFn: async (name: string) => {
      const res = await api.post<SetActiveResponse>('/api/v1/environments/active', { name })
      if (!res.ok || !res.data) {
        throw new Error(res.error?.message ?? 'Failed to set active environment')
      }
      return res.data
    },
    onMutate: async (name) => {
      // Optimistic update: set active in Zustand immediately
      const previousActiveEnv = useEnvironmentStore.getState().activeEnv
      setActiveEnv(name)

      // Optimistically update the list cache
      await queryClient.cancelQueries({ queryKey: ENV_KEYS.list() })
      const previousList = queryClient.getQueryData<EnvListItem[]>(ENV_KEYS.list())

      if (previousList) {
        queryClient.setQueryData<EnvListItem[]>(
          ENV_KEYS.list(),
          previousList.map((env) => ({
            ...env,
            active: env.name === name,
          })),
        )
      }

      return { previousActiveEnv, previousList }
    },
    onError: (_err, _name, context) => {
      // Rollback on error
      if (context?.previousActiveEnv !== undefined) {
        setActiveEnv(context.previousActiveEnv)
      }
      if (context?.previousList) {
        queryClient.setQueryData(ENV_KEYS.list(), context.previousList)
      }
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: ENV_KEYS.list() })
    },
  })
}

/**
 * Mutation: update environment variables via PUT /api/v1/environments/{name}.
 * Invalidates both list and detail caches on success.
 */
export function useUpdateEnvironment() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async ({ name, data }: { name: string; data: UpdateEnvInput }) => {
      const res = await api.put<Environment>(`/api/v1/environments/${name}`, data)
      if (!res.ok || !res.data) {
        throw new Error(res.error?.message ?? 'Failed to update environment')
      }
      return res.data
    },
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: ENV_KEYS.list() })
      queryClient.invalidateQueries({ queryKey: ENV_KEYS.detail(variables.name) })
    },
  })
}
