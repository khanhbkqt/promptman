import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { TooltipProvider } from '@/components/ui/tooltip'
import { useConnectionStore } from '@/stores/connection-store'
import App from './App'
import './index.css'

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 1,
      staleTime: 30_000,
      refetchOnWindowFocus: false,
    },
  },
})

// Auto-discover daemon from lock file (dev mode via Vite middleware)
async function autoConnect() {
  try {
    const res = await fetch('/api/__daemon_lock')
    if (!res.ok) return
    const lock = await res.json()
    if (lock.port && lock.token) {
      useConnectionStore.getState().connect({ port: lock.port, token: lock.token })
    }
  } catch {
    // Daemon not running — remain disconnected
  }
}

autoConnect()

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <TooltipProvider>
        <App />
      </TooltipProvider>
    </QueryClientProvider>
  </StrictMode>,
)
