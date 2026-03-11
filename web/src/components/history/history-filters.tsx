import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Input } from '@/components/ui/input'
import { useCollections } from '@/hooks/use-collections'
import type { HistoryFilters } from '@/types/history'

interface HistoryFiltersBarProps {
  filters: HistoryFilters
  onChange: (filters: HistoryFilters) => void
}

const ALL = '__all__'

export function HistoryFiltersBar({ filters, onChange }: HistoryFiltersBarProps) {
  const { data: collections } = useCollections()

  return (
    <div className="flex items-center gap-2 flex-wrap">
      {/* Collection filter */}
      <Select
        value={filters.collection || ALL}
        onValueChange={(v) => onChange({ ...filters, collection: v === ALL ? undefined : v })}
      >
        <SelectTrigger className="w-[180px] h-8 text-xs">
          <SelectValue placeholder="All collections" />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value={ALL} className="text-xs">All collections</SelectItem>
          {collections?.map((c) => (
            <SelectItem key={c.id} value={c.id} className="text-xs">
              {c.name || c.id}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>

      {/* Source filter */}
      <Select
        value={filters.source || ALL}
        onValueChange={(v) => onChange({ ...filters, source: v === ALL ? undefined : v })}
      >
        <SelectTrigger className="w-[120px] h-8 text-xs">
          <SelectValue placeholder="All sources" />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value={ALL} className="text-xs">All sources</SelectItem>
          <SelectItem value="cli" className="text-xs">CLI</SelectItem>
          <SelectItem value="gui" className="text-xs">GUI</SelectItem>
          <SelectItem value="test" className="text-xs">Test</SelectItem>
        </SelectContent>
      </Select>

      {/* Status code filter */}
      <Input
        type="text"
        inputMode="numeric"
        placeholder="Status code"
        className="w-[110px] h-8 text-xs"
        value={filters.status ?? ''}
        onChange={(e) => {
          const val = e.target.value.replace(/\D/g, '')
          onChange({ ...filters, status: val || undefined })
        }}
      />
    </div>
  )
}
