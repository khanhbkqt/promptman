import { create } from 'zustand'
import { persist } from 'zustand/middleware'

interface EnvironmentState {
  /** Currently active environment name (persisted in localStorage). */
  activeEnv: string | null

  /** Optimistically set the active environment name. */
  setActiveEnv: (name: string | null) => void
}

/**
 * Zustand store for active environment state.
 * Persisted in localStorage so the UI remembers the last active env across reloads.
 * The actual source of truth is the daemon — this store is for optimistic updates.
 */
export const useEnvironmentStore = create<EnvironmentState>()(
  persist(
    (set) => ({
      activeEnv: null,

      setActiveEnv: (name) => {
        set({ activeEnv: name })
      },
    }),
    {
      name: 'promptman-active-env',
    },
  ),
)
