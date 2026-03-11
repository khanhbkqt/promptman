import type { AuthConfig } from '@/types/collection'

const AUTH_TYPES = [
  { value: 'none', label: 'None' },
  { value: 'inherit', label: 'Inherit' },
  { value: 'bearer', label: 'Bearer Token' },
  { value: 'basic', label: 'Basic Auth' },
  { value: 'api-key', label: 'API Key' },
] as const

interface AuthEditorProps {
  auth: AuthConfig | null
  onChange: (auth: AuthConfig | null) => void
  inheritedAuth?: AuthConfig | null
}

export function AuthEditor({ auth, onChange, inheritedAuth }: AuthEditorProps) {
  // Determine current mode
  const mode = auth === null ? 'none' : auth.type === 'bearer' ? 'bearer' : auth.type

  function setMode(newMode: string) {
    switch (newMode) {
      case 'none':
        onChange(null)
        break
      case 'inherit':
        onChange(null) // inherit = no override
        break
      case 'bearer':
        onChange({ type: 'bearer', bearer: { token: '' } })
        break
      case 'basic':
        onChange({ type: 'basic', basic: { username: '', password: '' } })
        break
      case 'api-key':
        onChange({ type: 'api-key', apiKey: { key: '', value: '' } })
        break
    }
  }

  return (
    <div className="space-y-3">
      {/* Type selector */}
      <div className="flex gap-1 border-b border-border pb-2">
        {AUTH_TYPES.map((t) => (
          <button
            key={t.value}
            onClick={() => setMode(t.value)}
            className={`px-3 py-1 text-xs rounded-md transition-colors ${
              (t.value === 'none' && auth === null && !inheritedAuth) ||
              (t.value === 'inherit' && auth === null && inheritedAuth) ||
              (t.value === mode && auth !== null)
                ? 'bg-accent text-accent-foreground font-medium'
                : 'text-muted-foreground hover:text-foreground hover:bg-accent/50'
            }`}
          >
            {t.label}
          </button>
        ))}
      </div>

      {/* Auth fields */}
      {auth === null && inheritedAuth && (
        <div className="text-xs text-muted-foreground bg-muted/50 rounded-md p-3">
          <p className="font-medium mb-1">Inherited: {inheritedAuth.type}</p>
          {inheritedAuth.bearer && (
            <p className="font-mono truncate">Token: {inheritedAuth.bearer.token}</p>
          )}
          {inheritedAuth.basic && (
            <p className="font-mono">User: {inheritedAuth.basic.username}</p>
          )}
          {inheritedAuth.apiKey && (
            <p className="font-mono">Key: {inheritedAuth.apiKey.key}</p>
          )}
        </div>
      )}

      {auth?.type === 'bearer' && (
        <div>
          <label className="text-xs text-muted-foreground mb-1 block">Token</label>
          <input
            type="text"
            value={auth.bearer?.token ?? ''}
            onChange={(e) =>
              onChange({ type: 'bearer', bearer: { token: e.target.value } })
            }
            placeholder="Bearer token (supports {{variables}})"
            className="w-full h-8 px-2 rounded-md border border-border bg-background text-xs font-mono placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-ring"
          />
        </div>
      )}

      {auth?.type === 'basic' && (
        <div className="space-y-2">
          <div>
            <label className="text-xs text-muted-foreground mb-1 block">Username</label>
            <input
              type="text"
              value={auth.basic?.username ?? ''}
              onChange={(e) =>
                onChange({
                  type: 'basic',
                  basic: { username: e.target.value, password: auth.basic?.password ?? '' },
                })
              }
              placeholder="Username"
              className="w-full h-8 px-2 rounded-md border border-border bg-background text-xs font-mono placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-ring"
            />
          </div>
          <div>
            <label className="text-xs text-muted-foreground mb-1 block">Password</label>
            <input
              type="password"
              value={auth.basic?.password ?? ''}
              onChange={(e) =>
                onChange({
                  type: 'basic',
                  basic: { username: auth.basic?.username ?? '', password: e.target.value },
                })
              }
              placeholder="Password"
              className="w-full h-8 px-2 rounded-md border border-border bg-background text-xs font-mono placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-ring"
            />
          </div>
        </div>
      )}

      {auth?.type === 'api-key' && (
        <div className="space-y-2">
          <div>
            <label className="text-xs text-muted-foreground mb-1 block">Header Name</label>
            <input
              type="text"
              value={auth.apiKey?.key ?? ''}
              onChange={(e) =>
                onChange({
                  type: 'api-key',
                  apiKey: { key: e.target.value, value: auth.apiKey?.value ?? '' },
                })
              }
              placeholder="X-API-Key"
              className="w-full h-8 px-2 rounded-md border border-border bg-background text-xs font-mono placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-ring"
            />
          </div>
          <div>
            <label className="text-xs text-muted-foreground mb-1 block">Value</label>
            <input
              type="text"
              value={auth.apiKey?.value ?? ''}
              onChange={(e) =>
                onChange({
                  type: 'api-key',
                  apiKey: { key: auth.apiKey?.key ?? '', value: e.target.value },
                })
              }
              placeholder="API key value"
              className="w-full h-8 px-2 rounded-md border border-border bg-background text-xs font-mono placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-ring"
            />
          </div>
        </div>
      )}

      {auth === null && !inheritedAuth && (
        <p className="text-xs text-muted-foreground py-4 text-center">
          No authentication configured for this request.
        </p>
      )}
    </div>
  )
}
