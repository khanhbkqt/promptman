import { cn } from '@/lib/utils'

interface StatusBadgeProps {
  status: number
  className?: string
}

export function StatusBadge({ status, className }: StatusBadgeProps) {
  const color = getStatusColor(status)
  const text = getStatusText(status)

  return (
    <span
      className={cn(
        'inline-flex items-center gap-1.5 px-2.5 py-1 rounded-md font-mono text-xs font-bold',
        color.bg,
        color.text,
        className,
      )}
    >
      {status} {text}
    </span>
  )
}

function getStatusColor(status: number) {
  if (status >= 200 && status < 300)
    return { bg: 'bg-emerald-500/10', text: 'text-emerald-500' }
  if (status >= 300 && status < 400)
    return { bg: 'bg-blue-500/10', text: 'text-blue-500' }
  if (status >= 400 && status < 500)
    return { bg: 'bg-amber-500/10', text: 'text-amber-500' }
  return { bg: 'bg-red-500/10', text: 'text-red-500' }
}

function getStatusText(status: number): string {
  const texts: Record<number, string> = {
    200: 'OK',
    201: 'Created',
    204: 'No Content',
    301: 'Moved',
    302: 'Found',
    304: 'Not Modified',
    400: 'Bad Request',
    401: 'Unauthorized',
    403: 'Forbidden',
    404: 'Not Found',
    405: 'Not Allowed',
    408: 'Timeout',
    409: 'Conflict',
    422: 'Unprocessable',
    429: 'Rate Limited',
    500: 'Server Error',
    502: 'Bad Gateway',
    503: 'Unavailable',
    504: 'Gateway Timeout',
  }
  return texts[status] ?? ''
}
