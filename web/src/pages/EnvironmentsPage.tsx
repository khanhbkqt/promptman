import * as React from 'react'
import { useEnvironments, useEnvironment, useUpdateEnvironment } from '@/hooks/use-environments'
import { useConnectionStore } from '@/stores/connection-store'
import { VariableTable } from '@/components/variable-table'
import { VariablePreview } from '@/components/variable-preview'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Skeleton } from '@/components/ui/skeleton'
import { Globe, AlertCircle } from 'lucide-react'

export function EnvironmentsPage() {
  const status = useConnectionStore((s) => s.status)
  const { data: environments, isLoading: isListLoading, error: listError } = useEnvironments()
  const [selectedEnv, setSelectedEnv] = React.useState<string | null>(null)
  const { data: envDetail, isLoading: isDetailLoading } = useEnvironment(selectedEnv)
  const updateMutation = useUpdateEnvironment()

  // Auto-select first env or active env on load
  React.useEffect(() => {
    if (!selectedEnv && environments && environments.length > 0) {
      const active = environments.find((e) => e.active)
      setSelectedEnv(active?.name ?? environments[0].name)
    }
  }, [environments, selectedEnv])

  const handleSave = (variables: Record<string, unknown>, secrets: Record<string, string>) => {
    if (!selectedEnv) return
    updateMutation.mutate({
      name: selectedEnv,
      data: { variables, secrets },
    })
  }

  if (status !== 'connected') {
    return (
      <div className="flex items-center justify-center h-full p-6">
        <div className="text-center text-muted-foreground">
          <AlertCircle className="h-8 w-8 mx-auto mb-2" />
          <p>Connect to a daemon to manage environments</p>
        </div>
      </div>
    )
  }

  if (isListLoading) {
    return (
      <div className="p-6 space-y-4">
        <Skeleton className="h-8 w-48" />
        <div className="grid grid-cols-3 gap-3">
          {[1, 2, 3].map((i) => (
            <Skeleton key={i} className="h-20" />
          ))}
        </div>
      </div>
    )
  }

  if (listError) {
    return (
      <div className="flex items-center justify-center h-full p-6">
        <div className="text-center text-destructive">
          <AlertCircle className="h-8 w-8 mx-auto mb-2" />
          <p>Failed to load environments</p>
          <p className="text-xs text-muted-foreground mt-1">{listError.message}</p>
        </div>
      </div>
    )
  }

  const isEmpty = !environments || environments.length === 0

  return (
    <div className="flex h-full">
      {/* Sidebar: environment list */}
      <div className="w-64 shrink-0 border-r bg-muted/30 p-4 space-y-2 overflow-y-auto">
        <h3 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider mb-3">
          Environments
        </h3>
        {isEmpty ? (
          <div className="text-sm text-muted-foreground py-4 text-center">
            No environments found.
            <br />
            <span className="text-xs">Create one via CLI:</span>
            <code className="block mt-1 text-xs bg-muted px-2 py-1 rounded">
              pm env create dev
            </code>
          </div>
        ) : (
          environments.map((env) => (
            <Button
              key={env.name}
              variant={selectedEnv === env.name ? 'secondary' : 'ghost'}
              className="w-full justify-start gap-2 h-auto py-2"
              onClick={() => setSelectedEnv(env.name)}
            >
              <Globe className="h-4 w-4 shrink-0" />
              <div className="flex flex-col items-start text-left min-w-0">
                <div className="flex items-center gap-1.5">
                  <span className="text-sm font-medium truncate">{env.name}</span>
                  {env.active && (
                    <Badge variant="default" className="text-[9px] px-1 py-0 h-3.5">
                      active
                    </Badge>
                  )}
                </div>
                <span className="text-[10px] text-muted-foreground">
                  {env.variableCount} vars · {env.secretCount} secrets
                </span>
              </div>
            </Button>
          ))
        )}
      </div>

      {/* Main content: environment detail */}
      <div className="flex-1 overflow-y-auto p-6 space-y-6">
        {selectedEnv && envDetail ? (
          <>
            <div className="flex items-center gap-3">
              <h2 className="text-xl font-semibold">{envDetail.name}</h2>
              {environments?.find((e) => e.name === envDetail.name)?.active && (
                <Badge variant="default">active</Badge>
              )}
            </div>

            <Card>
              <CardHeader className="pb-3">
                <CardTitle className="text-base">Variables</CardTitle>
              </CardHeader>
              <CardContent>
                <VariableTable
                  variables={envDetail.variables}
                  secrets={envDetail.secrets}
                  onSave={handleSave}
                  isSaving={updateMutation.isPending}
                />
              </CardContent>
            </Card>

            <Card>
              <CardHeader className="pb-3">
                <CardTitle className="text-base">Preview</CardTitle>
              </CardHeader>
              <CardContent>
                <VariablePreview variables={envDetail.variables} />
              </CardContent>
            </Card>
          </>
        ) : isDetailLoading ? (
          <div className="space-y-4">
            <Skeleton className="h-8 w-48" />
            <Skeleton className="h-48" />
          </div>
        ) : (
          <div className="flex items-center justify-center h-full text-muted-foreground">
            <p>Select an environment to view details</p>
          </div>
        )}
      </div>
    </div>
  )
}
