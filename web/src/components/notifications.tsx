import { useEffect } from 'react'
import { toast, Toaster as SonnerToaster } from 'sonner'
import { wsClient } from '@/stores/connection-store'
import { useConnectionStore } from '@/stores/connection-store'
import type { DataChangedPayload } from '@/types/daemon'

const SOURCE_LABELS: Record<string, string> = {
  collections: 'Collection',
  environments: 'Environment',
  tests: 'Test',
  config: 'Configuration',
}

/**
 * Listens to WebSocket `data.changed` events and shows toast notifications
 * when data is updated externally (e.g., from CLI or another source).
 */
export function NotificationListener() {
  const status = useConnectionStore((s) => s.status)

  useEffect(() => {
    if (status !== 'connected') return

    const unsub = wsClient.on('data.changed', (event) => {
      const payload = event.payload as DataChangedPayload
      const label = SOURCE_LABELS[payload.source] ?? payload.source
      toast.info(`${label} updated`, {
        description: `${label} data was updated from an external source.`,
        duration: 4000,
      })
    })

    return unsub
  }, [status])

  return null
}

/**
 * Sonner Toaster component configured for the dark theme.
 * Mount this once in the app root (main.tsx).
 */
export function Toaster() {
  return (
    <SonnerToaster
      theme="dark"
      position="bottom-right"
      toastOptions={{
        className: 'border-border bg-card text-card-foreground',
      }}
      richColors
      closeButton
    />
  )
}
