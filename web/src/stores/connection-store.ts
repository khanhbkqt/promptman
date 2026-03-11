import { create } from 'zustand'
import type { ConnectionStatus, DaemonConfig, DaemonInfo } from '@/types/daemon'
import { configureApiClient, api } from '@/lib/api-client'
import { WsClient } from '@/lib/ws-client'

interface ConnectionState {
  status: ConnectionStatus
  daemonInfo: DaemonInfo | null
  config: DaemonConfig | null
  error: string | null

  // Actions
  connect: (config: DaemonConfig) => Promise<void>
  disconnect: () => void
  checkHealth: () => Promise<boolean>
}

const wsClient = new WsClient()

export const useConnectionStore = create<ConnectionState>((set, get) => {
  // Wire WS status changes to store
  wsClient.onStatusChange = (wsStatus) => {
    if (wsStatus === 'connected') {
      set({ status: 'connected' })
    } else if (wsStatus === 'reconnecting') {
      set({ status: 'reconnecting' })
    } else if (wsStatus === 'disconnected') {
      set({ status: 'disconnected' })
    }
  }

  return {
    status: 'disconnected',
    daemonInfo: null,
    config: null,
    error: null,

    connect: async (config: DaemonConfig) => {
      const baseUrl = `http://127.0.0.1:${config.port}`
      set({ status: 'connecting', config, error: null })

      // Configure the REST client
      configureApiClient(baseUrl, config.token)

      // Health check first
      const res = await api.get<DaemonInfo>('/api/v1/status')
      if (!res.ok) {
        set({
          status: 'disconnected',
          error: res.error?.message || 'Failed to connect to daemon',
        })
        return
      }

      set({ daemonInfo: res.data, status: 'connected' })

      // Start WebSocket connection
      wsClient.connect(baseUrl, config.token)
    },

    disconnect: () => {
      wsClient.disconnect()
      set({ status: 'disconnected', daemonInfo: null, config: null, error: null })
    },

    checkHealth: async () => {
      const { config } = get()
      if (!config) return false
      const res = await api.get<DaemonInfo>('/api/v1/status')
      if (res.ok && res.data) {
        set({ daemonInfo: res.data, status: 'connected' })
        return true
      }
      set({ status: 'disconnected', error: res.error?.message || 'Health check failed' })
      return false
    },
  }
})

// Export wsClient for event subscription in components
export { wsClient }
