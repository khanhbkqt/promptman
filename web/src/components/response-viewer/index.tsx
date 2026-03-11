import { useState } from 'react'
import { Loader2, Send, AlertCircle } from 'lucide-react'
import { StatusBadge } from './status-badge'
import { ResponseBody } from './response-body'
import { ResponseHeaders } from './response-headers'
import { ResponseTiming } from './response-timing'
import type { ExecuteResponse } from '@/types/collection'

type Tab = 'body' | 'headers' | 'timing'

interface ResponseViewerProps {
  response: ExecuteResponse | null
  isLoading: boolean
  error: string | null
}

export function ResponseViewer({
  response,
  isLoading,
  error,
}: ResponseViewerProps) {
  const [activeTab, setActiveTab] = useState<Tab>('body')

  // --- Loading state ---
  if (isLoading) {
    return (
      <div className="flex flex-col items-center justify-center h-full text-muted-foreground gap-3">
        <Loader2 className="size-6 animate-spin" />
        <p className="text-sm">Sending request…</p>
      </div>
    )
  }

  // --- Error state ---
  if (error) {
    return (
      <div className="flex flex-col items-center justify-center h-full text-muted-foreground gap-3 px-8">
        <AlertCircle className="size-8 text-destructive" />
        <p className="text-sm font-medium text-destructive">Request Failed</p>
        <p className="text-xs text-center max-w-sm">{error}</p>
      </div>
    )
  }

  // --- Empty state ---
  if (!response) {
    return (
      <div className="flex flex-col items-center justify-center h-full text-muted-foreground gap-3">
        <Send className="size-8 opacity-20" />
        <p className="text-sm">Send a request to see the response</p>
        <p className="text-xs opacity-60">Use Ctrl+Enter or click Send</p>
      </div>
    )
  }

  // --- Response display ---
  const tabs: { key: Tab; label: string }[] = [
    { key: 'body', label: 'Body' },
    { key: 'headers', label: `Headers (${Object.keys(response.headers).length})` },
    { key: 'timing', label: 'Timing' },
  ]

  const responseSize = new Blob([response.body]).size
  const sizeLabel =
    responseSize > 1024
      ? `${(responseSize / 1024).toFixed(1)} KB`
      : `${responseSize} B`

  return (
    <div className="flex flex-col h-full">
      {/* Status bar */}
      <div className="flex items-center gap-3 px-4 py-2 border-b border-border">
        <StatusBadge status={response.status} />
        <span className="text-xs text-muted-foreground font-mono tabular-nums">
          {response.timing.total.toFixed(0)}ms
        </span>
        <span className="text-xs text-muted-foreground font-mono tabular-nums">
          {sizeLabel}
        </span>
        {response.error && (
          <span className="text-xs text-destructive ml-auto">
            {response.error}
          </span>
        )}
      </div>

      {/* Tabs */}
      <div className="flex gap-0 border-b border-border px-4">
        {tabs.map((t) => (
          <button
            key={t.key}
            onClick={() => setActiveTab(t.key)}
            className={`px-4 py-2 text-xs font-medium border-b-2 transition-colors ${
              activeTab === t.key
                ? 'border-brand-500 text-foreground'
                : 'border-transparent text-muted-foreground hover:text-foreground'
            }`}
          >
            {t.label}
          </button>
        ))}
      </div>

      {/* Content */}
      <div className="flex-1 overflow-y-auto p-4">
        {activeTab === 'body' && <ResponseBody body={response.body} />}
        {activeTab === 'headers' && <ResponseHeaders headers={response.headers} />}
        {activeTab === 'timing' && <ResponseTiming timing={response.timing} />}
      </div>
    </div>
  )
}
