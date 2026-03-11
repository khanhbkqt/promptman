import type { RequestTiming } from '@/types/collection'
import { cn } from '@/lib/utils'

interface ResponseTimingProps {
  timing: RequestTiming
}

const SEGMENTS = [
  { key: 'dns' as const, label: 'DNS Lookup', color: 'bg-cyan-500' },
  { key: 'connect' as const, label: 'TCP Connect', color: 'bg-blue-500' },
  { key: 'tls' as const, label: 'TLS Handshake', color: 'bg-purple-500' },
  { key: 'ttfb' as const, label: 'TTFB', color: 'bg-amber-500' },
  { key: 'transfer' as const, label: 'Transfer', color: 'bg-emerald-500' },
]

export function ResponseTiming({ timing }: ResponseTimingProps) {
  const total = timing.total || 1 // Avoid division by zero

  return (
    <div className="space-y-4">
      {/* Timeline bar */}
      <div className="flex h-6 rounded-md overflow-hidden bg-muted/30">
        {SEGMENTS.map((seg) => {
          const value = timing[seg.key]
          const pct = (value / total) * 100
          if (pct < 0.5) return null
          return (
            <div
              key={seg.key}
              className={cn(seg.color, 'relative transition-all')}
              style={{ width: `${pct}%` }}
              title={`${seg.label}: ${value.toFixed(1)}ms`}
            />
          )
        })}
      </div>

      {/* Legend */}
      <div className="space-y-2">
        {SEGMENTS.map((seg) => {
          const value = timing[seg.key]
          const pct = (value / total) * 100
          return (
            <div key={seg.key} className="flex items-center gap-3">
              <div className={cn('size-2.5 rounded-sm shrink-0', seg.color)} />
              <span className="text-xs text-muted-foreground w-[120px]">
                {seg.label}
              </span>
              <div className="flex-1 relative h-4">
                <div
                  className={cn(seg.color, 'h-full rounded-sm opacity-20')}
                  style={{ width: `${Math.max(pct, 2)}%` }}
                />
              </div>
              <span className="text-xs font-mono tabular-nums w-[60px] text-right">
                {value.toFixed(1)}ms
              </span>
            </div>
          )
        })}
        {/* Total */}
        <div className="flex items-center gap-3 pt-2 border-t border-border">
          <div className="size-2.5 shrink-0" />
          <span className="text-xs font-medium w-[120px]">Total</span>
          <div className="flex-1" />
          <span className="text-xs font-mono font-bold tabular-nums w-[60px] text-right">
            {timing.total.toFixed(1)}ms
          </span>
        </div>
      </div>
    </div>
  )
}
