import { create } from 'zustand'
import { fetchBadgeCounts } from '@/api/badge'
import type { BadgeCounts } from '@/types/api/badge'

interface BadgeStoreState {
  counts: BadgeCounts
  loading: boolean
  loadingForCharacterId: number | null
  loadedForCharacterId: number | null
  load: (characterId: number) => Promise<void>
  clear: () => void
}

export const useBadgeStore = create<BadgeStoreState>((set, get) => ({
  counts: {},
  loading: false,
  loadingForCharacterId: null,
  loadedForCharacterId: null,
  load: async (characterId) => {
    const state = get()
    if (
      state.loadingForCharacterId === characterId ||
      state.loadedForCharacterId === characterId
    ) {
      return
    }

    set({ loading: true, loadingForCharacterId: characterId })
    try {
      const counts = await fetchBadgeCounts()
      if (get().loadingForCharacterId === characterId) {
        set({
          counts,
          loadedForCharacterId: characterId,
        })
      }
    } finally {
      if (get().loadingForCharacterId === characterId) {
        set({ loading: false, loadingForCharacterId: null })
      }
    }
  },
  clear: () =>
    set({
      counts: {},
      loading: false,
      loadingForCharacterId: null,
      loadedForCharacterId: null,
    }),
}))
