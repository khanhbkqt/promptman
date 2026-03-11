import { create } from 'zustand'

interface CollectionSelectionState {
  /** Currently selected collection ID (from the tree) */
  selectedCollectionId: string | null
  /** Currently selected request path within the collection (e.g. "folder-id/request-id") */
  selectedRequestPath: string | null

  // Actions
  selectCollection: (id: string | null) => void
  selectRequest: (collectionId: string, requestPath: string) => void
  clearSelection: () => void
}

export const useCollectionStore = create<CollectionSelectionState>((set) => ({
  selectedCollectionId: null,
  selectedRequestPath: null,

  selectCollection: (id) =>
    set({ selectedCollectionId: id, selectedRequestPath: null }),

  selectRequest: (collectionId, requestPath) =>
    set({ selectedCollectionId: collectionId, selectedRequestPath: requestPath }),

  clearSelection: () =>
    set({ selectedCollectionId: null, selectedRequestPath: null }),
}))
