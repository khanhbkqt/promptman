import { useState, useCallback } from 'react'
import { Play, FlaskConical, Loader2, RefreshCw } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { TestResultCard } from '@/components/test-results/test-result-card'
import { useTestResults, useRunTests } from '@/hooks/use-tests'
import { useCollections } from '@/hooks/use-collections'
import { useEnvironmentStore } from '@/stores/environment-store'

export function TestsPage() {
  const [selectedCollection, setSelectedCollection] = useState<string>('')
  const { data: results, isLoading, isError, error, refetch } = useTestResults()
  const { data: collections } = useCollections()
  const runTests = useRunTests()
  const activeEnv = useEnvironmentStore((s) => s.activeEnv)

  const handleRunTests = useCallback(() => {
    if (!selectedCollection) return
    runTests.mutate({
      collection: selectedCollection,
      ...(activeEnv ? { env: activeEnv } : {}),
    })
  }, [selectedCollection, runTests, activeEnv])

  // Compute aggregate stats
  const stats = results?.reduce(
    (acc, r) => ({
      totalRuns: acc.totalRuns + 1,
      totalTests: acc.totalTests + r.summary.total,
      totalPassed: acc.totalPassed + r.summary.passed,
      totalFailed: acc.totalFailed + r.summary.failed,
    }),
    { totalRuns: 0, totalTests: 0, totalPassed: 0, totalFailed: 0 },
  )

  return (
    <div className="flex flex-col h-full">
      {/* Header bar */}
      <div className="shrink-0 border-b border-border px-6 py-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <FlaskConical className="size-5 text-muted-foreground" />
            <div>
              <h1 className="text-lg font-semibold">Tests</h1>
              <p className="text-xs text-muted-foreground">Run and review API test suites</p>
            </div>
          </div>

          <div className="flex items-center gap-2">
            <Select value={selectedCollection} onValueChange={setSelectedCollection}>
              <SelectTrigger className="w-[200px] h-8 text-xs">
                <SelectValue placeholder="Select collection" />
              </SelectTrigger>
              <SelectContent>
                {collections?.map((c) => (
                  <SelectItem key={c.id} value={c.id} className="text-xs">
                    {c.name || c.id}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>

            <Button
              size="sm"
              className="h-8 gap-1.5"
              onClick={handleRunTests}
              disabled={!selectedCollection || runTests.isPending}
            >
              {runTests.isPending ? (
                <Loader2 className="size-3.5 animate-spin" />
              ) : (
                <Play className="size-3.5" />
              )}
              Run Tests
            </Button>

            <Button
              variant="ghost"
              size="sm"
              className="h-8 w-8 p-0"
              onClick={() => refetch()}
              title="Refresh results"
            >
              <RefreshCw className="size-3.5" />
            </Button>
          </div>
        </div>

        {/* Stats bar */}
        {stats && stats.totalRuns > 0 && (
          <div className="flex items-center gap-4 mt-3 text-xs">
            <span className="text-muted-foreground">
              {stats.totalRuns} run{stats.totalRuns !== 1 ? 's' : ''}
            </span>
            <span className="text-muted-foreground">·</span>
            <span className="tabular-nums">
              {stats.totalTests} test{stats.totalTests !== 1 ? 's' : ''}
            </span>
            {stats.totalPassed > 0 && (
              <Badge variant="outline" className="bg-emerald-500/15 text-emerald-400 border-emerald-500/30 text-[10px] px-1.5 py-0">
                {stats.totalPassed} passed
              </Badge>
            )}
            {stats.totalFailed > 0 && (
              <Badge variant="destructive" className="text-[10px] px-1.5 py-0">
                {stats.totalFailed} failed
              </Badge>
            )}
          </div>
        )}
      </div>

      {/* Results list */}
      <div className="flex-1 overflow-auto p-6">
        {isLoading ? (
          <div className="space-y-3">
            <Skeleton className="h-16 w-full" />
            <Skeleton className="h-16 w-full" />
            <Skeleton className="h-16 w-full" />
          </div>
        ) : isError ? (
          <div className="flex flex-col items-center justify-center h-full text-sm text-destructive gap-2">
            <p>Failed to load test results</p>
            <p className="text-xs text-muted-foreground">{error?.message}</p>
          </div>
        ) : results && results.length > 0 ? (
          <div className="space-y-3">
            {results.map((result) => (
              <TestResultCard key={result.runId} result={result} />
            ))}
          </div>
        ) : (
          <div className="flex flex-col items-center justify-center h-full text-muted-foreground gap-3">
            <FlaskConical className="size-10 opacity-30" />
            <p className="text-sm">No test results yet</p>
            <p className="text-xs">Select a collection and run tests to see results here</p>
          </div>
        )}

        {/* Run error toast-style inline */}
        {runTests.isError && (
          <div className="mt-4 rounded-md border border-destructive/30 bg-destructive/5 p-3 text-sm text-destructive">
            Test run failed: {runTests.error?.message}
          </div>
        )}
      </div>
    </div>
  )
}
