import { useState } from 'react'
import { Play, Loader2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { useCollections, useCollection } from '@/hooks/use-collections'
import type { StressRunInput } from '@/types/stress'

interface StressFormProps {
  onSubmit: (input: StressRunInput) => void
  isPending: boolean
  disabled?: boolean
}

export function StressForm({ onSubmit, isPending, disabled }: StressFormProps) {
  const [collectionId, setCollectionId] = useState('')
  const [requestId, setRequestId] = useState('')
  const [users, setUsers] = useState('10')
  const [duration, setDuration] = useState('30s')
  const [rampUp, setRampUp] = useState('5s')

  const { data: collections } = useCollections()
  const { data: collection } = useCollection(collectionId || null)

  // Flatten requests from the collection for the request picker
  const requests = flattenRequests(collection)

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (!collectionId || !requestId) return
    onSubmit({
      collection: collectionId,
      requestId,
      users: parseInt(users, 10) || 10,
      duration,
      rampUp: rampUp || undefined,
    })
  }

  return (
    <form onSubmit={handleSubmit} className="flex items-end gap-3 flex-wrap">
      {/* Collection */}
      <div className="space-y-1">
        <label className="text-[10px] text-muted-foreground uppercase tracking-wider">Collection</label>
        <Select value={collectionId} onValueChange={(v) => { setCollectionId(v); setRequestId('') }}>
          <SelectTrigger className="w-[180px] h-8 text-xs">
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
      </div>

      {/* Request */}
      <div className="space-y-1">
        <label className="text-[10px] text-muted-foreground uppercase tracking-wider">Request</label>
        <Select value={requestId} onValueChange={setRequestId} disabled={!collectionId}>
          <SelectTrigger className="w-[180px] h-8 text-xs">
            <SelectValue placeholder="Select request" />
          </SelectTrigger>
          <SelectContent>
            {requests.map((r) => (
              <SelectItem key={r.id} value={r.id} className="text-xs">
                <span className="font-semibold mr-1">{r.method}</span> {r.path}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      {/* VUs */}
      <div className="space-y-1">
        <label className="text-[10px] text-muted-foreground uppercase tracking-wider">Users</label>
        <Input
          type="number"
          min="1"
          max="1000"
          className="w-[80px] h-8 text-xs"
          value={users}
          onChange={(e) => setUsers(e.target.value)}
        />
      </div>

      {/* Duration */}
      <div className="space-y-1">
        <label className="text-[10px] text-muted-foreground uppercase tracking-wider">Duration</label>
        <Input
          type="text"
          placeholder="30s"
          className="w-[80px] h-8 text-xs"
          value={duration}
          onChange={(e) => setDuration(e.target.value)}
        />
      </div>

      {/* Ramp-up */}
      <div className="space-y-1">
        <label className="text-[10px] text-muted-foreground uppercase tracking-wider">Ramp-up</label>
        <Input
          type="text"
          placeholder="5s"
          className="w-[80px] h-8 text-xs"
          value={rampUp}
          onChange={(e) => setRampUp(e.target.value)}
        />
      </div>

      {/* Submit */}
      <Button
        type="submit"
        size="sm"
        className="h-8 gap-1.5"
        disabled={!collectionId || !requestId || isPending || disabled}
      >
        {isPending ? (
          <Loader2 className="size-3.5 animate-spin" />
        ) : (
          <Play className="size-3.5" />
        )}
        Start Stress Test
      </Button>
    </form>
  )
}

// Helper: flatten all requests from a collection tree
function flattenRequests(
  collection?: { requests?: { id: string; method: string; path: string }[]; folders?: { requests?: { id: string; method: string; path: string }[]; folders?: any[] }[] } | null,
): { id: string; method: string; path: string }[] {
  if (!collection) return []
  const result: { id: string; method: string; path: string }[] = []
  if (collection.requests) result.push(...collection.requests)
  if (collection.folders) {
    for (const folder of collection.folders) {
      result.push(...flattenRequests(folder as any))
    }
  }
  return result
}
