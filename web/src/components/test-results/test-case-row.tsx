import { CheckCircle, XCircle, Clock, AlertTriangle, SkipForward, ChevronDown, ChevronRight } from 'lucide-react'
import { useState } from 'react'
import { Badge } from '@/components/ui/badge'
import { AssertionDetail } from './assertion-detail'
import type { TestCase } from '@/types/testing'

interface TestCaseRowProps {
  testCase: TestCase
}

const statusConfig = {
  passed: {
    icon: CheckCircle,
    label: 'Passed',
    variant: 'default' as const,
    className: 'bg-emerald-500/15 text-emerald-400 border-emerald-500/30 hover:bg-emerald-500/25',
  },
  failed: {
    icon: XCircle,
    label: 'Failed',
    variant: 'destructive' as const,
    className: '',
  },
  timeout: {
    icon: Clock,
    label: 'Timeout',
    variant: 'secondary' as const,
    className: 'bg-amber-500/15 text-amber-400 border-amber-500/30 hover:bg-amber-500/25',
  },
  error: {
    icon: AlertTriangle,
    label: 'Error',
    variant: 'destructive' as const,
    className: '',
  },
  skipped: {
    icon: SkipForward,
    label: 'Skipped',
    variant: 'outline' as const,
    className: '',
  },
} as const

export function TestCaseRow({ testCase }: TestCaseRowProps) {
  const [expanded, setExpanded] = useState(false)
  const config = statusConfig[testCase.status] ?? statusConfig.error
  const StatusIcon = config.icon
  const hasDetails = !!testCase.error || (testCase.console && testCase.console.length > 0)

  return (
    <div className="border-b border-border/50 last:border-0">
      <button
        className="flex w-full items-center gap-3 px-4 py-2.5 text-left text-sm hover:bg-muted/40 transition-colors"
        onClick={() => hasDetails && setExpanded(!expanded)}
        disabled={!hasDetails}
      >
        {hasDetails ? (
          expanded ? (
            <ChevronDown className="size-3.5 shrink-0 text-muted-foreground" />
          ) : (
            <ChevronRight className="size-3.5 shrink-0 text-muted-foreground" />
          )
        ) : (
          <span className="size-3.5 shrink-0" />
        )}

        <StatusIcon className={`size-4 shrink-0 ${
          testCase.status === 'passed' ? 'text-emerald-400' :
          testCase.status === 'failed' || testCase.status === 'error' ? 'text-destructive' :
          testCase.status === 'timeout' ? 'text-amber-400' :
          'text-muted-foreground'
        }`} />

        <span className="flex-1 min-w-0 truncate font-mono">{testCase.name}</span>

        <span className="shrink-0 text-xs text-muted-foreground font-mono">
          {testCase.request}
        </span>

        <Badge variant={config.variant} className={`shrink-0 text-[10px] px-1.5 py-0 ${config.className}`}>
          {config.label}
        </Badge>

        <span className="shrink-0 w-16 text-right text-xs text-muted-foreground tabular-nums">
          {testCase.duration}ms
        </span>
      </button>

      {expanded && hasDetails && (
        <div className="px-4 pb-3 pl-11">
          {testCase.error && <AssertionDetail error={testCase.error} />}

          {testCase.console && testCase.console.length > 0 && (
            <div className="mt-2">
              <span className="text-xs text-muted-foreground">Console output:</span>
              <pre className="mt-1 rounded bg-muted/50 p-2 text-xs font-mono text-foreground max-h-32 overflow-auto">
                {testCase.console.join('\n')}
              </pre>
            </div>
          )}
        </div>
      )}
    </div>
  )
}
