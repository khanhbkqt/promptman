interface ResponseHeadersProps {
  headers: Record<string, string>
}

export function ResponseHeaders({ headers }: ResponseHeadersProps) {
  const entries = Object.entries(headers)

  if (entries.length === 0) {
    return (
      <p className="text-xs text-muted-foreground py-4 text-center">
        No response headers
      </p>
    )
  }

  return (
    <div className="space-y-0.5">
      {entries.map(([key, value]) => (
        <div
          key={key}
          className="flex gap-3 py-1.5 px-2 rounded-md hover:bg-accent/30 transition-colors"
        >
          <span className="text-xs font-mono font-medium text-muted-foreground shrink-0 w-[200px] truncate">
            {key}
          </span>
          <span className="text-xs font-mono text-foreground break-all">
            {value}
          </span>
        </div>
      ))}
    </div>
  )
}
