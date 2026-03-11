import { useState } from 'react'
import { ChevronDown, ChevronRight, CheckCircle, XCircle, Clock } from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import { Card, CardHeader, CardTitle, CardContent } from '@/components/ui/card'
import { TestCaseRow } from './test-case-row'
import type { TestResult } from '@/types/testing'

interface TestResultCardProps {
  result: TestResult
}

export function TestResultCard({ result }: TestResultCardProps) {
  const [expanded, setExpanded] = useState(result.summary.failed > 0)
  const allPassed = result.summary.failed === 0 && result.summary.total > 0
  const hasFailures = result.summary.failed > 0

  return (
    <Card className={`overflow-hidden transition-colors ${
      hasFailures ? 'border-destructive/40' : allPassed ? 'border-emerald-500/30' : ''
    }`}>
      <CardHeader
        className="cursor-pointer select-none py-3 px-4 hover:bg-muted/30 transition-colors"
        onClick={() => setExpanded(!expanded)}
      >
        <div className="flex items-center gap-3">
          {expanded ? (
            <ChevronDown className="size-4 text-muted-foreground" />
          ) : (
            <ChevronRight className="size-4 text-muted-foreground" />
          )}

          {hasFailures ? (
            <XCircle className="size-4.5 text-destructive" />
          ) : allPassed ? (
            <CheckCircle className="size-4.5 text-emerald-400" />
          ) : (
            <Clock className="size-4.5 text-muted-foreground" />
          )}

          <CardTitle className="flex-1 text-sm font-medium">{result.collection}</CardTitle>

          {result.env && (
            <Badge variant="outline" className="text-[10px] px-1.5 py-0">
              {result.env}
            </Badge>
          )}

          <div className="flex items-center gap-2 text-xs">
            {result.summary.passed > 0 && (
              <span className="text-emerald-400 tabular-nums">{result.summary.passed} passed</span>
            )}
            {result.summary.failed > 0 && (
              <span className="text-destructive tabular-nums">{result.summary.failed} failed</span>
            )}
            {result.summary.skipped > 0 && (
              <span className="text-muted-foreground tabular-nums">{result.summary.skipped} skipped</span>
            )}
            <span className="text-muted-foreground tabular-nums">
              ({result.summary.duration}ms)
            </span>
          </div>
        </div>
      </CardHeader>

      {expanded && (
        <CardContent className="p-0 border-t border-border/50">
          {result.tests.length > 0 ? (
            result.tests.map((tc, i) => (
              <TestCaseRow key={`${tc.request}-${tc.name}-${i}`} testCase={tc} />
            ))
          ) : (
            <p className="px-4 py-3 text-sm text-muted-foreground">No test cases found.</p>
          )}

          {result.console && result.console.length > 0 && (
            <div className="border-t border-border/50 px-4 py-3">
              <span className="text-xs text-muted-foreground font-medium">Suite console output:</span>
              <pre className="mt-1 rounded bg-muted/50 p-2 text-xs font-mono text-foreground max-h-40 overflow-auto">
                {result.console.join('\n')}
              </pre>
            </div>
          )}
        </CardContent>
      )}
    </Card>
  )
}
