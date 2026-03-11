import { Badge } from '@/components/ui/badge'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import type { HistoryEntry } from '@/types/history'

interface HistoryTableProps {
  entries: HistoryEntry[]
}

const methodColors: Record<string, string> = {
  GET: 'bg-emerald-500/15 text-emerald-400 border-emerald-500/30',
  POST: 'bg-blue-500/15 text-blue-400 border-blue-500/30',
  PUT: 'bg-amber-500/15 text-amber-400 border-amber-500/30',
  PATCH: 'bg-orange-500/15 text-orange-400 border-orange-500/30',
  DELETE: 'bg-red-500/15 text-red-400 border-red-500/30',
  HEAD: 'bg-purple-500/15 text-purple-400 border-purple-500/30',
  OPTIONS: 'bg-gray-500/15 text-gray-400 border-gray-500/30',
}

function statusColor(status: number): string {
  if (status >= 200 && status < 300)
    return 'bg-emerald-500/15 text-emerald-400 border-emerald-500/30'
  if (status >= 300 && status < 400)
    return 'bg-blue-500/15 text-blue-400 border-blue-500/30'
  if (status >= 400 && status < 500)
    return 'bg-amber-500/15 text-amber-400 border-amber-500/30'
  if (status >= 500)
    return 'bg-red-500/15 text-red-400 border-red-500/30'
  return ''
}

function formatTime(iso: string): string {
  const d = new Date(iso)
  return d.toLocaleString(undefined, {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  })
}

export function HistoryTable({ entries }: HistoryTableProps) {
  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead className="w-[140px]">Time</TableHead>
          <TableHead className="w-[70px]">Method</TableHead>
          <TableHead>URL</TableHead>
          <TableHead className="w-[70px]">Status</TableHead>
          <TableHead className="w-[70px] text-right">Duration</TableHead>
          <TableHead className="w-[90px]">Collection</TableHead>
          <TableHead className="w-[60px]">Source</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {entries.map((entry, i) => (
          <TableRow key={`${entry.ts}-${entry.reqId}-${i}`} className="text-xs font-mono">
            <TableCell className="text-muted-foreground">
              {formatTime(entry.ts)}
            </TableCell>
            <TableCell>
              <Badge
                variant="outline"
                className={`text-[10px] px-1.5 py-0 font-semibold ${
                  methodColors[entry.method] ?? ''
                }`}
              >
                {entry.method}
              </Badge>
            </TableCell>
            <TableCell className="max-w-[300px] truncate" title={entry.url}>
              {entry.url}
            </TableCell>
            <TableCell>
              <Badge
                variant="outline"
                className={`text-[10px] px-1.5 py-0 tabular-nums ${statusColor(entry.status)}`}
              >
                {entry.status}
              </Badge>
            </TableCell>
            <TableCell className="text-right tabular-nums text-muted-foreground">
              {entry.time}ms
            </TableCell>
            <TableCell className="truncate text-muted-foreground" title={entry.collection}>
              {entry.collection}
            </TableCell>
            <TableCell>
              <Badge variant="outline" className="text-[10px] px-1.5 py-0">
                {entry.source}
              </Badge>
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  )
}
