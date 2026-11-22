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

let latestBadgeRequestId = 0

export const useBadgeStore = create<BadgeStoreState>((set, get) => ({
  counts: {},
  loading: false,
  loadingForCharacterId: null,
  loadedForCharacterId: null,
  load: async (characterId, force = false) => {
    const state = get()
    if (
      !force &&
      (state.loadingForCharacterId === characterId || state.loadedForCharacterId === characterId)
    ) {
      return
    }

    const requestId = ++latestBadgeRequestId
    set({ loading: true, loadingForCharacterId: characterId })
    try {
      // badge-counts is a Login route; guests and recruiter-only accounts would only get a 403.
      const counts = hasNonGuestRole(useSessionStore.getState().roles) ? await fetchBadgeCounts() : {}
      if (requestId === latestBadgeRequestId && get().loadingForCharacterId === characterId) {
        set({
          counts,
          loadedForCharacterId: characterId,
        })
      }
    } finally {
      if (requestId === latestBadgeRequestId && get().loadingForCharacterId === characterId) {
        set({ loading: false, loadingForCharacterId: null })
      }
    }
  },
  clear: () => {
    latestBadgeRequestId++
    set({
      counts: {},
      loading: false,
      loadingForCharacterId: null,
      loadedForCharacterId: null,
    })
  },
}))
