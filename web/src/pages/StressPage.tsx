import { useCallback } from 'react'
import { Zap, Loader2 } from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import { Skeleton } from '@/components/ui/skeleton'
import { StressForm } from '@/components/stress/stress-form'
import { StressChart } from '@/components/stress/stress-chart'
import { StressSummary } from '@/components/stress/stress-summary'
import { useRunStress, useStressTicks } from '@/hooks/use-stress'
import type { StressRunInput } from '@/types/stress'

export function StressPage() {
  const runStress = useRunStress()
  const { ticks, isRunning, completedReport, trackJob, reset } = useStressTicks()

  const handleSubmit = useCallback(
    (input: StressRunInput) => {
      reset()
      runStress.mutate(input, {
        onSuccess: (data) => {
          trackJob(data.jobId)
        },
      })
    },
    [runStress, trackJob, reset],
  )

  return (
    <div className="flex flex-col h-full">
      {/* Header */}
      <div className="shrink-0 border-b border-border px-6 py-4">
        <div className="flex items-center gap-3 mb-4">
          <Zap className="size-5 text-muted-foreground" />
          <div>
            <h1 className="text-lg font-semibold">Stress Tests</h1>
            <p className="text-xs text-muted-foreground">
              Load test your API endpoints with configurable virtual users
            </p>
          </div>

          {isRunning && (
            <Badge className="ml-auto bg-blue-500/15 text-blue-400 border-blue-500/30 text-[10px] px-2 py-0.5 gap-1.5">
              <Loader2 className="size-3 animate-spin" />
              Running
            </Badge>
          )}

          {completedReport && !isRunning && (
            <Badge className="ml-auto bg-emerald-500/15 text-emerald-400 border-emerald-500/30 text-[10px] px-2 py-0.5">
              Completed
            </Badge>
          )}
        </div>

        <StressForm
          onSubmit={handleSubmit}
          isPending={runStress.isPending}
          disabled={isRunning}
        />

        {runStress.isError && (
          <div className="mt-3 rounded-md border border-destructive/30 bg-destructive/5 p-2 text-xs text-destructive">
            Failed to start: {runStress.error?.message}
          </div>
        )}
      </div>

      {/* Content */}
      <div className="flex-1 overflow-auto p-6 space-y-4">
        {/* Live chart — visible during run or after completion */}
        {(ticks.length > 0 || isRunning) && (
          <StressChart data={ticks} />
        )}

        {/* Loading placeholder while running and no ticks yet */}
        {isRunning && ticks.length === 0 && (
          <div className="space-y-3">
            <Skeleton className="h-[320px] w-full" />
          </div>
        )}

        {/* Completed summary */}
        {completedReport && !isRunning && (
          <StressSummary report={completedReport} />
        )}

        {/* Empty state */}
        {!isRunning && !completedReport && ticks.length === 0 && (
          <div className="flex flex-col items-center justify-center h-full text-muted-foreground gap-3">
            <Zap className="size-10 opacity-30" />
            <p className="text-sm">No stress tests running</p>
            <p className="text-xs">Configure and start a test above to see live metrics</p>
          </div>
        )}
      </div>
    </div>
  )
}
