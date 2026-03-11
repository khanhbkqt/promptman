import { CheckCircle, XCircle } from 'lucide-react'
import { Card, CardHeader, CardTitle, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import type { StressReport } from '@/types/stress'

interface StressSummaryProps {
  report: StressReport
}

export function StressSummary({ report }: StressSummaryProps) {
  const { summary, thresholds } = report

  return (
    <div className="space-y-4">
      {/* Metric cards */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
        <MetricCard label="Total Requests" value={summary.totalRequests.toLocaleString()} />
        <MetricCard label="Requests/sec" value={summary.rps.toFixed(1)} />
        <MetricCard
          label="Error Rate"
          value={`${summary.errorRate.toFixed(1)}%`}
          variant={summary.errorRate > 5 ? 'danger' : summary.errorRate > 0 ? 'warning' : 'success'}
        />
        <MetricCard label="Peak Connections" value={String(summary.peakConnections)} />
      </div>

      {/* Latency breakdown */}
      <Card>
        <CardHeader className="py-3 px-4">
          <CardTitle className="text-sm font-medium">Latency (ms)</CardTitle>
        </CardHeader>
        <CardContent className="pb-4 px-4">
          <div className="grid grid-cols-3 gap-4 text-center">
            <div>
              <p className="text-2xl font-semibold tabular-nums">{summary.latency.p50}</p>
              <p className="text-[10px] text-muted-foreground uppercase tracking-wider">p50 (median)</p>
            </div>
            <div>
              <p className="text-2xl font-semibold tabular-nums text-amber-400">{summary.latency.p95}</p>
              <p className="text-[10px] text-muted-foreground uppercase tracking-wider">p95</p>
            </div>
            <div>
              <p className="text-2xl font-semibold tabular-nums text-red-400">{summary.latency.p99}</p>
              <p className="text-[10px] text-muted-foreground uppercase tracking-wider">p99</p>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Threshold results */}
      {thresholds && thresholds.length > 0 && (
        <Card>
          <CardHeader className="py-3 px-4">
            <CardTitle className="text-sm font-medium">Thresholds</CardTitle>
          </CardHeader>
          <CardContent className="pb-4 px-4">
            <div className="space-y-2">
              {thresholds.map((t, i) => (
                <div
                  key={`${t.name}-${i}`}
                  className="flex items-center gap-3 text-sm"
                >
                  {t.passed ? (
                    <CheckCircle className="size-4 text-emerald-400" />
                  ) : (
                    <XCircle className="size-4 text-destructive" />
                  )}
                  <span className="font-mono text-xs flex-1">
                    {t.name} {t.operator} {t.expected}
                  </span>
                  <span className="font-mono text-xs tabular-nums text-muted-foreground">
                    actual: {t.actual.toFixed(2)}
                  </span>
                  <Badge
                    variant={t.passed ? 'outline' : 'destructive'}
                    className={`text-[10px] px-1.5 py-0 ${
                      t.passed
                        ? 'bg-emerald-500/15 text-emerald-400 border-emerald-500/30'
                        : ''
                    }`}
                  >
                    {t.passed ? 'PASS' : 'FAIL'}
                  </Badge>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  )
}

// --- Internal ---

function MetricCard({
  label,
  value,
  variant = 'default',
}: {
  label: string
  value: string
  variant?: 'default' | 'success' | 'warning' | 'danger'
}) {
  const colorClass =
    variant === 'success'
      ? 'text-emerald-400'
      : variant === 'warning'
        ? 'text-amber-400'
        : variant === 'danger'
          ? 'text-destructive'
          : 'text-foreground'

  return (
    <Card>
      <CardContent className="p-4">
        <p className={`text-2xl font-semibold tabular-nums ${colorClass}`}>{value}</p>
        <p className="text-[10px] text-muted-foreground uppercase tracking-wider mt-1">{label}</p>
      </CardContent>
    </Card>
  )
}
