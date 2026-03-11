import type { TestError } from '@/types/testing'

interface AssertionDetailProps {
  error: TestError
}

export function AssertionDetail({ error }: AssertionDetailProps) {
  return (
    <div className="mt-2 rounded-md border border-destructive/30 bg-destructive/5 p-3 text-sm">
      <p className="font-medium text-destructive">{error.message}</p>
      {(error.expected !== undefined || error.actual !== undefined) && (
        <div className="mt-2 grid grid-cols-2 gap-3 text-xs">
          <div>
            <span className="text-muted-foreground">Expected:</span>
            <pre className="mt-1 rounded bg-muted/50 p-1.5 font-mono text-foreground">
              {formatValue(error.expected)}
            </pre>
          </div>
          <div>
            <span className="text-muted-foreground">Actual:</span>
            <pre className="mt-1 rounded bg-muted/50 p-1.5 font-mono text-foreground">
              {formatValue(error.actual)}
            </pre>
          </div>
        </div>
      )}
    </div>
  )
}

function formatValue(value: unknown): string {
  if (value === undefined) return 'undefined'
  if (value === null) return 'null'
  if (typeof value === 'string') return `"${value}"`
  return JSON.stringify(value, null, 2)
}
