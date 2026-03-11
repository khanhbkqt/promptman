import { useQuery } from '@tanstack/react-query'
import { useConnectionStore } from '@/stores/connection-store'

export function useDaemonStatus() {
  const { status, checkHealth } = useConnectionStore()

  useQuery({
    queryKey: ['daemon-health'],
    queryFn: checkHealth,
    enabled: status === 'connected' || status === 'reconnecting',
    refetchInterval: 15_000, // every 15s
    refetchIntervalInBackground: false,
  })

  return useConnectionStore()
}
