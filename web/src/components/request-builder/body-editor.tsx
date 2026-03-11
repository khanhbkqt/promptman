import { Plus, X } from 'lucide-react'
import { Button } from '@/components/ui/button'
import type { RequestBody } from '@/types/collection'

const BODY_TYPES = [
  { value: 'none', label: 'None' },
  { value: 'json', label: 'JSON' },
  { value: 'form', label: 'Form Data' },
  { value: 'raw', label: 'Raw' },
] as const

interface FormRow {
  key: string
  value: string
}

interface BodyEditorProps {
  body: RequestBody | null
  onChange: (body: RequestBody | null) => void
}

export function BodyEditor({ body, onChange }: BodyEditorProps) {
  const bodyType = body?.type ?? 'none'

  function setType(type: string) {
    if (type === 'none') {
      onChange(null)
    } else {
      onChange({ type: type as RequestBody['type'], content: body?.content ?? '' })
    }
  }

  return (
    <div className="space-y-3">
      {/* Type selector tabs */}
      <div className="flex gap-1 border-b border-border pb-2">
        {BODY_TYPES.map((t) => (
          <button
            key={t.value}
            onClick={() => setType(t.value)}
            className={`px-3 py-1 text-xs rounded-md transition-colors ${
              bodyType === t.value
                ? 'bg-accent text-accent-foreground font-medium'
                : 'text-muted-foreground hover:text-foreground hover:bg-accent/50'
            }`}
          >
            {t.label}
          </button>
        ))}
      </div>

      {/* Body content */}
      {bodyType === 'none' && (
        <p className="text-xs text-muted-foreground py-4 text-center">
          This request does not have a body.
        </p>
      )}

      {bodyType === 'json' && (
        <JsonBodyEditor
          content={typeof body?.content === 'string' ? body.content : JSON.stringify(body?.content ?? '', null, 2)}
          onChange={(c) => onChange({ type: 'json', content: c })}
        />
      )}

      {bodyType === 'form' && (
        <FormBodyEditor
          rows={parseFormData(body?.content)}
          onChange={(rows) => onChange({ type: 'form', content: rows })}
        />
      )}

      {bodyType === 'raw' && (
        <textarea
          value={typeof body?.content === 'string' ? body.content : ''}
          onChange={(e) => onChange({ type: 'raw', content: e.target.value })}
          placeholder="Raw body content"
          className="w-full min-h-[200px] p-3 rounded-md border border-border bg-background text-xs font-mono placeholder:text-muted-foreground resize-y focus:outline-none focus:ring-1 focus:ring-ring"
        />
      )}
    </div>
  )
}

// --- JSON editor ---
function JsonBodyEditor({
  content,
  onChange,
}: {
  content: string
  onChange: (content: string) => void
}) {
  return (
    <textarea
      value={content}
      onChange={(e) => onChange(e.target.value)}
      placeholder='{\n  "key": "value"\n}'
      className="w-full min-h-[200px] p-3 rounded-md border border-border bg-background text-xs font-mono placeholder:text-muted-foreground resize-y focus:outline-none focus:ring-1 focus:ring-ring"
      spellCheck={false}
    />
  )
}

// --- Form data editor ---
function FormBodyEditor({
  rows,
  onChange,
}: {
  rows: FormRow[]
  onChange: (rows: FormRow[]) => void
}) {
  function addRow() {
    onChange([...rows, { key: '', value: '' }])
  }

  function removeRow(index: number) {
    onChange(rows.filter((_, i) => i !== index))
  }

  function updateRow(index: number, field: 'key' | 'value', val: string) {
    const updated = rows.map((r, i) =>
      i === index ? { ...r, [field]: val } : r,
    )
    onChange(updated)
  }

  return (
    <div className="space-y-1">
      {rows.map((r, i) => (
        <div key={i} className="flex items-center gap-2">
          <input
            value={r.key}
            onChange={(e) => updateRow(i, 'key', e.target.value)}
            placeholder="Key"
            className="h-8 flex-1 px-2 rounded-md border border-border bg-background text-xs font-mono placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-ring"
          />
          <input
            value={r.value}
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
      <Button variant="ghost" size="sm" onClick={addRow} className="text-xs gap-1 mt-1">
        <Plus className="size-3" />
        Add Field
      </Button>
    </div>
  )
}

// --- Helper: parse form data content to rows ---
function parseFormData(content: unknown): FormRow[] {
  if (Array.isArray(content)) {
    return content.map((item) => ({
      key: String(item?.key ?? ''),
      value: String(item?.value ?? ''),
    }))
  }
  if (content && typeof content === 'object') {
    return Object.entries(content as Record<string, unknown>).map(([k, v]) => ({
      key: k,
      value: String(v ?? ''),
    }))
  }
  return [{ key: '', value: '' }]
}
