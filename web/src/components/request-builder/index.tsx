import { useState, useCallback, useMemo } from 'react'
import { MethodUrlBar } from './method-url-bar'
import { HeadersEditor } from './headers-editor'
import { BodyEditor } from './body-editor'
import { AuthEditor } from './auth-editor'
import type {
  RequestItem,
  RequestBody,
  AuthConfig,
  Collection,
} from '@/types/collection'

type Tab = 'headers' | 'body' | 'auth'

interface RequestBuilderProps {
  request: RequestItem
  collection: Collection
  onSend: (request: RequestItem) => void
  isSending: boolean
}

export function RequestBuilder({
  request,
  collection,
  onSend,
  isSending,
}: RequestBuilderProps) {
  const [activeTab, setActiveTab] = useState<Tab>('headers')

  // Local editable state (derived from prop, edited locally)
  const [method, setMethod] = useState(request.method)
  const [url, setUrl] = useState(resolveUrl(collection.baseUrl, request.path))
  const [headers, setHeaders] = useState(
    toHeaderRows(request.headers),
  )
  const [body, setBody] = useState<RequestBody | null>(request.body ?? null)
  const [auth, setAuth] = useState<AuthConfig | null>(request.auth ?? null)

  // Rebuild editable state when request changes
  const requestId = request.id
  const [prevRequestId, setPrevRequestId] = useState(requestId)
  if (requestId !== prevRequestId) {
    setPrevRequestId(requestId)
    setMethod(request.method)
    setUrl(resolveUrl(collection.baseUrl, request.path))
    setHeaders(toHeaderRows(request.headers))
    setBody(request.body ?? null)
    setAuth(request.auth ?? null)
  }

  const handleSend = useCallback(() => {
    onSend({
      ...request,
      method,
      path: url,
      headers: fromHeaderRows(headers),
      body: body ?? undefined,
      auth: auth ?? undefined,
    })
  }, [request, method, url, headers, body, auth, onSend])

  const tabs: { key: Tab; label: string; count?: number }[] = useMemo(
    () => [
      {
        key: 'headers',
        label: 'Headers',
        count: headers.filter((h) => h.key).length,
      },
      { key: 'body', label: 'Body' },
      { key: 'auth', label: 'Auth' },
    ],
    [headers],
  )

  return (
    <div className="flex flex-col h-full">
      {/* Method + URL bar */}
      <div className="px-4 py-3 border-b border-border">
        <MethodUrlBar
          method={method}
          url={url}
          onMethodChange={setMethod}
          onUrlChange={setUrl}
          onSend={handleSend}
          isSending={isSending}
        />
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
            {t.count !== undefined && t.count > 0 && (
              <span className="ml-1.5 text-[10px] bg-muted rounded-full px-1.5 py-0.5 tabular-nums">
                {t.count}
              </span>
            )}
          </button>
        ))}
      </div>

      {/* Tab content */}
      <div className="flex-1 overflow-y-auto p-4">
        {activeTab === 'headers' && (
          <HeadersEditor
            headers={headers}
            onChange={setHeaders}
            inheritedHeaders={collection.defaults?.headers}
          />
        )}
        {activeTab === 'body' && (
          <BodyEditor body={body} onChange={setBody} />
        )}
        {activeTab === 'auth' && (
          <AuthEditor
            auth={auth}
            onChange={setAuth}
            inheritedAuth={collection.auth}
          />
        )}
      </div>
    </div>
  )
}

// --- Helpers ---

function resolveUrl(baseUrl: string | undefined, path: string): string {
  if (!baseUrl) return path
  return `${baseUrl.replace(/\/$/, '')}/${path.replace(/^\//, '')}`
}

function toHeaderRows(headers?: Record<string, string>) {
  if (!headers || Object.keys(headers).length === 0) {
    return [{ key: '', value: '' }]
  }
  return Object.entries(headers).map(([key, value]) => ({ key, value }))
}

function fromHeaderRows(rows: { key: string; value: string }[]) {
  const result: Record<string, string> = {}
  for (const r of rows) {
    if (r.key.trim()) {
      result[r.key.trim()] = r.value
    }
  }
  return Object.keys(result).length > 0 ? result : undefined
}
