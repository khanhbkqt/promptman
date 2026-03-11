import { useState, useMemo } from 'react'
import { History, Trash2, Loader2, ChevronDown } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { HistoryFiltersBar } from '@/components/history/history-filters'
import { HistoryTable } from '@/components/history/history-table'
import { useHistory, useClearHistory } from '@/hooks/use-history'
import type { HistoryFilters } from '@/types/history'

export function HistoryPage() {
  const [filters, setFilters] = useState<HistoryFilters>({})
  const {
    data,
    isLoading,
    isError,
    error,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
  } = useHistory(filters)
  const clearHistory = useClearHistory()

  const entries = useMemo(
    () => data?.pages.flatMap((p) => p.data) ?? [],
    [data],
  )

  const totalShown = entries.length

  return (
    <div className="flex flex-col h-full">
      {/* Header bar */}
      <div className="shrink-0 border-b border-border px-6 py-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <History className="size-5 text-muted-foreground" />
            <div>
              <h1 className="text-lg font-semibold">History</h1>
              <p className="text-xs text-muted-foreground">Request execution log</p>
            </div>
          </div>

          <div className="flex items-center gap-2">
            <HistoryFiltersBar filters={filters} onChange={setFilters} />

            <Button
              variant="ghost"
              size="sm"
              className="h-8 gap-1.5 text-destructive hover:text-destructive"
              onClick={() => clearHistory.mutate(undefined)}
              disabled={clearHistory.isPending}
            >
              {clearHistory.isPending ? (
                <Loader2 className="size-3.5 animate-spin" />
              ) : (
                <Trash2 className="size-3.5" />
              )}
              Clear
            </Button>
          </div>
        </div>

        {totalShown > 0 && (
          <p className="mt-2 text-xs text-muted-foreground">
            Showing {totalShown} entr{totalShown === 1 ? 'y' : 'ies'}
          </p>
        )}
      </div>

      {/* Table */}
      <div className="flex-1 overflow-auto">
        {isLoading ? (
          <div className="space-y-2 p-6">
            <Skeleton className="h-10 w-full" />
            <Skeleton className="h-8 w-full" />
            <Skeleton className="h-8 w-full" />
            <Skeleton className="h-8 w-full" />
            <Skeleton className="h-8 w-full" />
          </div>
        ) : isError ? (
          <div className="flex flex-col items-center justify-center h-full text-sm text-destructive gap-2">
            <p>Failed to load history</p>
            <p className="text-xs text-muted-foreground">{error?.message}</p>
          </div>
        ) : entries.length > 0 ? (
          <div className="p-6">
            <HistoryTable entries={entries} />

            {hasNextPage && (
              <div className="flex justify-center mt-4">
                <Button
                  variant="outline"
                  size="sm"
                  className="gap-1.5"
                  onClick={() => fetchNextPage()}
                  disabled={isFetchingNextPage}
                >
                  {isFetchingNextPage ? (
                    <Loader2 className="size-3.5 animate-spin" />
                  ) : (
                    <ChevronDown className="size-3.5" />
                  )}
                  Load More
                </Button>
              </div>
            )}
          </div>
        ) : (
          <div className="flex flex-col items-center justify-center h-full text-muted-foreground gap-3">
            <History className="size-10 opacity-30" />
            <p className="text-sm">No history entries</p>
            <p className="text-xs">Execute some requests to see them here</p>
          </div>
        )}
      </div>
    </div>
  )
}
