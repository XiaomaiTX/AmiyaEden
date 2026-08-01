import { create } from 'zustand'
import { createJSONStorage, persist } from 'zustand/middleware'
import { SIDEBAR_NAVIGATION_STORE_KEY } from '@/stores/persistence-keys'

interface SidebarNavigationStoreState {
  ownerCharacterId: number | null
  expandedMenuGroupKeys: string[]
  expandMenuGroup: (groupKey: string) => void
  collapseMenuGroup: (groupKey: string) => void
  resetForCharacter: (characterId: number | null) => void
  clear: () => void
}

export const useSidebarNavigationStore = create<SidebarNavigationStoreState>()(
  persist(
    (set) => ({
      ownerCharacterId: null,
      expandedMenuGroupKeys: [],
      expandMenuGroup: (groupKey) =>
        set((state) =>
          state.expandedMenuGroupKeys.includes(groupKey)
            ? state
            : { expandedMenuGroupKeys: [...state.expandedMenuGroupKeys, groupKey] }
        ),
      collapseMenuGroup: (groupKey) =>
        set((state) =>
          state.expandedMenuGroupKeys.includes(groupKey)
            ? {
                expandedMenuGroupKeys: state.expandedMenuGroupKeys.filter((key) => key !== groupKey),
              }
            : state
        ),
      resetForCharacter: (characterId) =>
        set((state) =>
          state.ownerCharacterId === characterId
            ? state
            : { ownerCharacterId: characterId, expandedMenuGroupKeys: [] }
        ),
      clear: () => set({ ownerCharacterId: null, expandedMenuGroupKeys: [] }),
    }),
    {
      name: SIDEBAR_NAVIGATION_STORE_KEY,
      storage: createJSONStorage(() => localStorage),
      partialize: (state) => ({
        ownerCharacterId: state.ownerCharacterId,
        expandedMenuGroupKeys: state.expandedMenuGroupKeys,
      }),
    }
  )
)
