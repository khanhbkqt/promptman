import { useEnvironments, useSetActiveEnvironment } from '@/hooks/use-environments'
import { useEnvironmentStore } from '@/stores/environment-store'
import { useConnectionStore } from '@/stores/connection-store'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Badge } from '@/components/ui/badge'
import { Globe } from 'lucide-react'

/**
 * Environment switcher dropdown for the top bar.
 * Shows all environments with the active one highlighted.
 * Supports optimistic switching via the useSetActiveEnvironment mutation.
 */
export function EnvSwitcher() {
  const connectionStatus = useConnectionStore((s) => s.status)
  const activeEnv = useEnvironmentStore((s) => s.activeEnv)
  const { data: environments, isLoading } = useEnvironments()
  const setActiveMutation = useSetActiveEnvironment()

  if (connectionStatus !== 'connected') {
    return null
  }

  const handleSwitch = (envName: string | null) => {
    if (!envName || envName === activeEnv) return
    setActiveMutation.mutate(envName)
  }

  return (
    <div className="flex items-center gap-2">
      <Globe className="h-4 w-4 text-muted-foreground" />
      <Select
        value={activeEnv ?? undefined}
        onValueChange={handleSwitch}
        disabled={isLoading || setActiveMutation.isPending}
      >
        <SelectTrigger
          className="h-8 w-[160px] text-xs border-dashed"
          id="env-switcher-trigger"
        >
          <SelectValue placeholder={isLoading ? 'Loading…' : 'No environment'} />
        </SelectTrigger>
        <SelectContent>
          {environments?.map((env) => (
            <SelectItem key={env.name} value={env.name}>
              <div className="flex items-center gap-2">
                <span>{env.name}</span>
                {env.active && (
                  <Badge variant="secondary" className="text-[10px] px-1 py-0 h-4">
                    active
                  </Badge>
                )}
                <span className="text-muted-foreground text-[10px] ml-auto">
                  {env.variableCount} vars
                </span>
              </div>
            </SelectItem>
          ))}
          {environments?.length === 0 && (
            <div className="px-2 py-1.5 text-xs text-muted-foreground">
              No environments found
            </div>
          )}
        </SelectContent>
      </Select>
    </div>
  )
}
