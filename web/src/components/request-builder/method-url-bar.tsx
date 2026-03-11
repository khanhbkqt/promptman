import { cn } from '@/lib/utils'

const HTTP_METHODS = ['GET', 'POST', 'PUT', 'PATCH', 'DELETE', 'HEAD', 'OPTIONS'] as const

const METHOD_COLORS: Record<string, { bg: string; text: string }> = {
  GET: { bg: 'bg-emerald-500/10', text: 'text-emerald-500' },
  POST: { bg: 'bg-blue-500/10', text: 'text-blue-500' },
  PUT: { bg: 'bg-amber-500/10', text: 'text-amber-500' },
  PATCH: { bg: 'bg-purple-500/10', text: 'text-purple-500' },
  DELETE: { bg: 'bg-red-500/10', text: 'text-red-500' },
  HEAD: { bg: 'bg-slate-500/10', text: 'text-slate-400' },
  OPTIONS: { bg: 'bg-slate-500/10', text: 'text-slate-400' },
}

interface MethodUrlBarProps {
  method: string
  url: string
  onMethodChange: (method: string) => void
  onUrlChange: (url: string) => void
  onSend: () => void
  isSending: boolean
}

export function MethodUrlBar({
  method,
  url,
  onMethodChange,
  onUrlChange,
  onSend,
  isSending,
}: MethodUrlBarProps) {
  const colors = METHOD_COLORS[method.toUpperCase()] ?? METHOD_COLORS.GET

  return (
    <div className="flex items-center gap-2">
      {/* Method selector */}
      <select
        value={method}
        onChange={(e) => onMethodChange(e.target.value)}
        className={cn(
          'h-9 px-3 rounded-md font-mono text-xs font-bold border border-border bg-background',
          'appearance-none cursor-pointer',
          colors.bg,
          colors.text,
        )}
        style={{ minWidth: '90px' }}
      >
        {HTTP_METHODS.map((m) => (
          <option key={m} value={m}>
            {m}
          </option>
        ))}
      </select>

      {/* URL input */}
      <input
        type="text"
        value={url}
        onChange={(e) => onUrlChange(e.target.value)}
        placeholder="Enter request URL"
        className="flex-1 h-9 px-3 rounded-md border border-border bg-background text-sm font-mono placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring"
        onKeyDown={(e) => {
          if ((e.metaKey || e.ctrlKey) && e.key === 'Enter') {
            e.preventDefault()
            onSend()
          }
        }}
      />

      {/* Send button */}
      <button
        onClick={onSend}
        disabled={isSending}
        className={cn(
          'h-9 px-5 rounded-md font-semibold text-sm transition-colors',
          'bg-brand-600 text-white hover:bg-brand-700',
          'disabled:opacity-50 disabled:cursor-not-allowed',
        )}
      >
        {isSending ? 'Sending…' : 'Send'}
      </button>
    </div>
  )
}
