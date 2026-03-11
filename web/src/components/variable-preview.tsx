import * as React from 'react'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'

interface VariablePreviewProps {
  /** Current environment variables (key→value). */
  variables?: Record<string, unknown>
}

const VARIABLE_REGEX = /\{\{(\w+)\}\}/g

/**
 * Shows a URL template with highlighted `{{variable}}` placeholders
 * resolved against the current environment variables.
 */
export function VariablePreview({ variables = {} }: VariablePreviewProps) {
  const [url, setUrl] = React.useState('https://{{host}}/api/v1/{{resource}}')

  // Parse and render the URL with highlighted variables
  const rendered = React.useMemo(() => {
    const parts: React.ReactNode[] = []
    let lastIndex = 0
    let match: RegExpExecArray | null

    const regex = new RegExp(VARIABLE_REGEX.source, 'g')
    while ((match = regex.exec(url)) !== null) {
      // Text before the match
      if (match.index > lastIndex) {
        parts.push(<span key={`t-${lastIndex}`}>{url.slice(lastIndex, match.index)}</span>)
      }

      const varName = match[1]
      const resolved = variables[varName]

      if (resolved !== undefined) {
        parts.push(
          <Badge key={`v-${match.index}`} variant="secondary" className="text-[11px] px-1 py-0 font-mono mx-0.5">
            {String(resolved)}
          </Badge>,
        )
      } else {
        parts.push(
          <Badge key={`v-${match.index}`} variant="destructive" className="text-[11px] px-1 py-0 font-mono mx-0.5">
            {`{{${varName}}}`}
          </Badge>,
        )
      }

      lastIndex = match.index + match[0].length
    }

    // Remaining text
    if (lastIndex < url.length) {
      parts.push(<span key={`t-${lastIndex}`}>{url.slice(lastIndex)}</span>)
    }

    return parts
  }, [url, variables])

  // Resolved plain text
  const resolvedUrl = React.useMemo(() => {
    return url.replace(VARIABLE_REGEX, (_, varName) => {
      const val = variables[varName]
      return val !== undefined ? String(val) : `{{${varName}}}`
    })
  }, [url, variables])

  return (
    <div className="space-y-3">
      <h3 className="text-sm font-medium text-muted-foreground">Variable Preview</h3>
      <Input
        value={url}
        onChange={(e) => setUrl(e.target.value)}
        placeholder="Enter a URL template with {{variables}}"
        className="font-mono text-xs"
      />
      <div className="rounded-md border bg-muted/50 p-3">
        <div className="text-xs text-muted-foreground mb-1">Resolved:</div>
        <div className="font-mono text-sm break-all leading-relaxed">{rendered}</div>
      </div>
      <div className="text-xs text-muted-foreground font-mono break-all">
        → {resolvedUrl}
      </div>
    </div>
  )
}
