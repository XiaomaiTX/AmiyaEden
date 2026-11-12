import { create } from 'zustand'
import { fetchBadgeCounts } from '@/api/badge'
import { hasNonGuestRole } from '@/app/route-access'
import { useSessionStore } from '@/stores/session-store'
import type { BadgeCounts } from '@/types/api/badge'

interface BadgeStoreState {
  counts: BadgeCounts
  loading: boolean
  loadingForCharacterId: number | null
  loadedForCharacterId: number | null
  load: (characterId: number, force?: boolean) => Promise<void>
  clear: () => void
}

export const useBadgeStore = create<BadgeStoreState>((set, get) => ({
  counts: {},
  loading: false,
  loadingForCharacterId: null,
  loadedForCharacterId: null,
  load: async (characterId, force = false) => {
    const state = get()
    if (
      state.loadingForCharacterId === characterId ||
      (!force && state.loadedForCharacterId === characterId)
    ) {
      return
    }

    set({ loading: true, loadingForCharacterId: characterId })
    try {
      // badge-counts is a Login route; guests and recruiter-only accounts would only get a 403.
      const counts = hasNonGuestRole(useSessionStore.getState().roles) ? await fetchBadgeCounts() : {}
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
