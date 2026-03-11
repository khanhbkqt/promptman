import { useCallback, useMemo } from 'react'
import { Send } from 'lucide-react'
import { CollectionTree } from '@/components/collection-tree'
import { RequestBuilder } from '@/components/request-builder'
import { ResponseViewer } from '@/components/response-viewer'
import { useCollectionStore } from '@/stores/collection-store'
import { useCollection, useSendRequest } from '@/hooks/use-collections'
import type { RequestItem, Folder } from '@/types/collection'

export function CollectionsPage() {
  const { selectedCollectionId, selectedRequestPath } = useCollectionStore()
  const { data: collection } = useCollection(selectedCollectionId)
  const sendMutation = useSendRequest()

  // Find the selected request in the collection tree
  const selectedRequest = useMemo(() => {
    if (!collection || !selectedRequestPath) return null
    return findRequest(
      selectedRequestPath,
      collection.requests,
      collection.folders,
    )
  }, [collection, selectedRequestPath])

  const handleSend = useCallback(
    (request: RequestItem) => {
      if (!selectedCollectionId) return
      sendMutation.mutate({
        collection: selectedCollectionId,
        requestId: request.id,
        source: 'gui',
      })
    },
    [selectedCollectionId, sendMutation],
  )

  return (
    <div className="flex h-full">
      {/* Left: Collection tree sidebar */}
      <div className="w-[280px] shrink-0 border-r border-border bg-sidebar">
        <CollectionTree />
      </div>

      {/* Right: Request builder + Response viewer */}
      <div className="flex-1 min-w-0">
        {selectedRequest && collection ? (
          <div className="flex flex-col h-full">
            {/* Top half: Request builder */}
            <div className="h-1/2 border-b border-border overflow-hidden">
              <RequestBuilder
                request={selectedRequest}
                collection={collection}
                onSend={handleSend}
                isSending={sendMutation.isPending}
              />
            </div>

            {/* Bottom half: Response viewer */}
            <div className="h-1/2 overflow-hidden">
              <ResponseViewer
                response={sendMutation.data ?? null}
                isLoading={sendMutation.isPending}
                error={
                  sendMutation.error
                    ? sendMutation.error.message
                    : null
                }
              />
            </div>
          </div>
        ) : (
          <div className="flex flex-col items-center justify-center h-full text-muted-foreground gap-3">
            <Send className="size-10 opacity-30" />
            <p className="text-sm">Select a request from the tree to get started</p>
          </div>
        )}
      </div>
    </div>
  )
}

/**
 * Recursively find a request by ID in requests/folders.
 */
function findRequest(
  id: string,
  requests?: RequestItem[],
  folders?: Folder[],
): RequestItem | null {
  if (requests) {
    for (const r of requests) {
      if (r.id === id) return r
    }
  }
  if (folders) {
    for (const f of folders) {
      const found = findRequest(id, f.requests, f.folders)
      if (found) return found
    }
  }
  return null
}
