import { useState } from 'react'
import {
  ChevronRight,
  ChevronDown,
  FolderOpen,
  Folder as FolderIcon,
  RefreshCw,
  Inbox,
} from 'lucide-react'
import { cn } from '@/lib/utils'
import { useCollections, useCollection } from '@/hooks/use-collections'
import { useCollectionStore } from '@/stores/collection-store'
import { Skeleton } from '@/components/ui/skeleton'
import { Button } from '@/components/ui/button'
import type { CollectionSummary, Folder, RequestItem } from '@/types/collection'

// --- Method badge colors ---
const METHOD_COLORS: Record<string, string> = {
  GET: 'text-emerald-500',
  POST: 'text-blue-500',
  PUT: 'text-amber-500',
  PATCH: 'text-purple-500',
  DELETE: 'text-red-500',
  HEAD: 'text-slate-400',
  OPTIONS: 'text-slate-400',
}

function MethodBadge({ method }: { method: string }) {
  const upper = method.toUpperCase()
  return (
    <span
      className={cn(
        'font-mono text-[10px] font-bold leading-none w-12 shrink-0 text-right',
        METHOD_COLORS[upper] ?? 'text-muted-foreground',
      )}
    >
      {upper}
    </span>
  )
}

// --- Request row ---
function RequestRow({
  request,
  collectionId,
  depth,
  pathPrefix,
}: {
  request: RequestItem
  collectionId: string
  depth: number
  pathPrefix: string
}) {
  const { selectedRequestPath, selectRequest } = useCollectionStore()
  // Full path = folder prefix + request ID (e.g. "auth-testing/bearer-auth")
  const fullPath = pathPrefix ? `${pathPrefix}/${request.id}` : request.id
  const isActive = selectedRequestPath === fullPath

  return (
    <button
      onClick={() => selectRequest(collectionId, fullPath)}
      className={cn(
        'flex items-center gap-2 w-full px-2 py-1 text-left text-sm rounded-md transition-colors',
        'hover:bg-accent/50',
        isActive && 'bg-accent text-accent-foreground',
      )}
      style={{ paddingLeft: `${depth * 16 + 8}px` }}
    >
      <MethodBadge method={request.method} />
      <span className="truncate">{request.path}</span>
    </button>
  )
}

// --- Folder tree node ---
function FolderNode({
  folder,
  collectionId,
  depth,
  pathPrefix,
}: {
  folder: Folder
  collectionId: string
  depth: number
  pathPrefix: string
}) {
  const [open, setOpen] = useState(false)
  // Build this folder's path for its children
  const folderPath = pathPrefix ? `${pathPrefix}/${folder.id}` : folder.id

  return (
    <div>
      <button
        onClick={() => setOpen(!open)}
        className="flex items-center gap-1.5 w-full px-2 py-1 text-left text-sm rounded-md hover:bg-accent/50 transition-colors"
        style={{ paddingLeft: `${depth * 16 + 8}px` }}
      >
        {open ? (
          <ChevronDown className="size-3.5 text-muted-foreground shrink-0" />
        ) : (
          <ChevronRight className="size-3.5 text-muted-foreground shrink-0" />
        )}
        {open ? (
          <FolderOpen className="size-3.5 text-muted-foreground shrink-0" />
        ) : (
          <FolderIcon className="size-3.5 text-muted-foreground shrink-0" />
        )}
        <span className="truncate font-medium">{folder.name}</span>
      </button>
      {open && (
        <div>
          {folder.folders?.map((sf) => (
            <FolderNode
              key={sf.id}
              folder={sf}
              collectionId={collectionId}
              depth={depth + 1}
              pathPrefix={folderPath}
            />
          ))}
          {folder.requests?.map((r) => (
            <RequestRow
              key={r.id}
              request={r}
              collectionId={collectionId}
              depth={depth + 1}
              pathPrefix={folderPath}
            />
          ))}
        </div>
      )}
    </div>
  )
}

// --- Collection root node ---
function CollectionNode({ summary }: { summary: CollectionSummary }) {
  const { selectedCollectionId, selectCollection } = useCollectionStore()
  const isSelected = selectedCollectionId === summary.id
  const [open, setOpen] = useState(false)

  // Fetch full collection when expanded
  const { data: collection } = useCollection(open ? summary.id : null)

  function handleToggle() {
    const willOpen = !open
    setOpen(willOpen)
    if (willOpen) {
      selectCollection(summary.id)
    }
  }

  return (
    <div>
      <button
        onClick={handleToggle}
        className={cn(
          'flex items-center gap-1.5 w-full px-2 py-1.5 text-left text-sm rounded-md transition-colors',
          'hover:bg-accent/50',
          isSelected && 'bg-accent/30',
        )}
      >
        {open ? (
          <ChevronDown className="size-4 text-muted-foreground shrink-0" />
        ) : (
          <ChevronRight className="size-4 text-muted-foreground shrink-0" />
        )}
        <span className="truncate font-semibold">{summary.name}</span>
        <span className="ml-auto text-xs text-muted-foreground tabular-nums">
          {summary.requestCount}
        </span>
      </button>
      {open && collection && (
        <div className="ml-1">
          {collection.folders?.map((f) => (
            <FolderNode
              key={f.id}
              folder={f}
              collectionId={summary.id}
              depth={1}
              pathPrefix=""
            />
          ))}
          {collection.requests?.map((r) => (
            <RequestRow
              key={r.id}
              request={r}
              collectionId={summary.id}
              depth={1}
              pathPrefix=""
            />
          ))}
        </div>
      )}
    </div>
  )
}

// --- Loading skeleton ---
function TreeSkeleton() {
  return (
    <div className="space-y-2 p-2">
      {[1, 2, 3].map((i) => (
        <div key={i} className="space-y-1">
          <Skeleton className="h-7 w-full" />
          <div className="ml-4 space-y-1">
            <Skeleton className="h-5 w-3/4" />
            <Skeleton className="h-5 w-2/3" />
          </div>
        </div>
      ))}
    </div>
  )
}

// --- Empty state ---
function EmptyState() {
  return (
    <div className="flex flex-col items-center justify-center py-12 text-muted-foreground gap-2">
      <Inbox className="size-8" />
      <p className="text-sm">No collections yet</p>
      <p className="text-xs">
        Add <code className="font-mono">.yaml</code> files to your collections
        directory
      </p>
    </div>
  )
}

// --- Main export ---
export function CollectionTree() {
  const { data: collections, isLoading, refetch, isFetching } = useCollections()

  return (
    <div className="flex flex-col h-full">
      {/* Header */}
      <div className="flex items-center justify-between px-3 py-2 border-b border-border">
        <h2 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
          Collections
        </h2>
        <Button
          variant="ghost"
          size="icon"
          className="size-6"
          onClick={() => refetch()}
          disabled={isFetching}
        >
          <RefreshCw
            className={cn('size-3.5', isFetching && 'animate-spin')}
          />
        </Button>
      </div>

      {/* Tree body */}
      <div className="flex-1 overflow-y-auto p-1 space-y-0.5">
        {isLoading ? (
          <TreeSkeleton />
        ) : !collections?.length ? (
          <EmptyState />
        ) : (
          collections.map((c) => <CollectionNode key={c.id} summary={c} />)
        )}
      </div>
    </div>
  )
}
