import { useConnectionStore } from '@/stores/connection-store'
import type { ConnectionStatus as Status } from '@/types/daemon'

const STATUS_CONFIG: Record<Status, { label: string; dotClass: string }> = {
  disconnected: { label: 'Disconnected', dotClass: 'bg-destructive' },
  connecting: { label: 'Connecting', dotClass: 'bg-yellow-500 animate-pulse' },
  connected: { label: 'Connected', dotClass: 'bg-green-500' },
  reconnecting: { label: 'Reconnecting', dotClass: 'bg-yellow-500 animate-pulse' },
}

export function ConnectionStatus() {
  const status = useConnectionStore((s) => s.status)
  const config = STATUS_CONFIG[status]

  return (
    <div className="flex items-center gap-1.5 px-2 py-1 rounded-full bg-muted text-[10px] font-medium uppercase tracking-wider text-muted-foreground border border-border">
      <div className={`w-1.5 h-1.5 rounded-full ${config.dotClass}`} />
      {config.label}
    </div>
  )
}
