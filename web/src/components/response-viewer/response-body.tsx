import { useState } from 'react'
import { Copy, Check, WrapText } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'

interface ResponseBodyProps {
  body: string
}

export function ResponseBody({ body }: ResponseBodyProps) {
  const [copied, setCopied] = useState(false)
  const [wrap, setWrap] = useState(true)

  const formatted = tryFormatJson(body)
  const isJson = formatted !== null

  async function handleCopy() {
    await navigator.clipboard.writeText(body)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  return (
    <div className="relative">
      {/* Toolbar */}
      <div className="flex items-center justify-end gap-1 mb-2">
        <Button
          variant="ghost"
          size="icon"
          className="size-7"
          onClick={() => setWrap(!wrap)}
          title={wrap ? 'Disable word wrap' : 'Enable word wrap'}
        >
          <WrapText
            className={cn('size-3.5', wrap && 'text-brand-500')}
          />
        </Button>
        <Button
          variant="ghost"
          size="icon"
          className="size-7"
          onClick={handleCopy}
          title="Copy response body"
        >
          {copied ? (
            <Check className="size-3.5 text-emerald-500" />
          ) : (
            <Copy className="size-3.5" />
          )}
        </Button>
      </div>

      {/* Body content */}
      <pre
        className={cn(
          'p-3 rounded-md border border-border bg-muted/30 text-xs font-mono overflow-x-auto max-h-[600px] overflow-y-auto',
          wrap && 'whitespace-pre-wrap break-words',
        )}
      >
        {isJson ? <JsonHighlight json={formatted} /> : body || '(empty response)'}
      </pre>
    </div>
  )
}

// --- Simple JSON syntax highlighting ---
function JsonHighlight({ json }: { json: string }) {
  // A lightweight regex-based highlighter for JSON
  const highlighted = json
    .replace(
      /("(?:[^"\\]|\\.)*")\s*:/g,
      '<span class="text-blue-400">$1</span>:',
    )
    .replace(
      /:\s*("(?:[^"\\]|\\.)*")/g,
      ': <span class="text-emerald-400">$1</span>',
    )
    .replace(
      /:\s*(\d+\.?\d*)/g,
      ': <span class="text-amber-400">$1</span>',
    )
    .replace(
      /:\s*(true|false)/g,
      ': <span class="text-purple-400">$1</span>',
    )
    .replace(
      /:\s*(null)/g,
      ': <span class="text-red-400">$1</span>',
    )

  return <code dangerouslySetInnerHTML={{ __html: highlighted }} />
}

function tryFormatJson(raw: string): string | null {
  try {
    const parsed = JSON.parse(raw)
    return JSON.stringify(parsed, null, 2)
  } catch {
    return null
  }
}
