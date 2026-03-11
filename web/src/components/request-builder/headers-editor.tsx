import { Plus, X } from 'lucide-react'
import { Button } from '@/components/ui/button'

interface HeaderRow {
  key: string
  value: string
}

interface HeadersEditorProps {
  headers: HeaderRow[]
  onChange: (headers: HeaderRow[]) => void
  inheritedHeaders?: Record<string, string>
}

export function HeadersEditor({
  headers,
  onChange,
  inheritedHeaders,
}: HeadersEditorProps) {
  function addRow() {
    onChange([...headers, { key: '', value: '' }])
  }

  function removeRow(index: number) {
    onChange(headers.filter((_, i) => i !== index))
  }

  function updateRow(index: number, field: 'key' | 'value', val: string) {
    const updated = headers.map((h, i) =>
      i === index ? { ...h, [field]: val } : h,
    )
    onChange(updated)
  }

  const inheritedEntries = inheritedHeaders
    ? Object.entries(inheritedHeaders)
    : []

  return (
    <div className="space-y-1">
      {/* Inherited headers (read-only) */}
      {inheritedEntries.map(([k, v]) => (
        <div key={`inherited-${k}`} className="flex items-center gap-2 opacity-50">
          <input
            value={k}
            readOnly
            className="h-8 flex-1 px-2 rounded-md border border-border/50 bg-muted text-xs font-mono"
            tabIndex={-1}
          />
          <input
            value={v}
            readOnly
            className="h-8 flex-1 px-2 rounded-md border border-border/50 bg-muted text-xs font-mono"
            tabIndex={-1}
          />
          <div className="w-7" /> {/* spacer for delete button alignment */}
        </div>
      ))}

      {/* Editable headers */}
      {headers.map((h, i) => (
        <div key={i} className="flex items-center gap-2">
          <input
            value={h.key}
            onChange={(e) => updateRow(i, 'key', e.target.value)}
            placeholder="Header name"
            className="h-8 flex-1 px-2 rounded-md border border-border bg-background text-xs font-mono placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-ring"
          />
          <input
            value={h.value}
            onChange={(e) => updateRow(i, 'value', e.target.value)}
            placeholder="Value"
            className="h-8 flex-1 px-2 rounded-md border border-border bg-background text-xs font-mono placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-ring"
          />
          <button
            onClick={() => removeRow(i)}
            className="size-7 flex items-center justify-center rounded-md text-muted-foreground hover:text-destructive hover:bg-destructive/10 transition-colors"
          >
            <X className="size-3.5" />
          </button>
        </div>
      ))}

      <Button
        variant="ghost"
        size="sm"
        onClick={addRow}
        className="text-xs gap-1 mt-1"
      >
        <Plus className="size-3" />
        Add Header
      </Button>
    </div>
  )
}
