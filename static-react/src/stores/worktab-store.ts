import { create } from 'zustand'
import { createJSONStorage, persist } from 'zustand/middleware'

export interface WorktabItem {
  routeId: string
  path: string
  titleKey: string
  fixed: boolean
}

interface WorktabStoreState {
  ownerCharacterId: number | null
  opened: WorktabItem[]
  open: (tab: Omit<WorktabItem, 'fixed'> & { fixed?: boolean }) => void
  close: (path: string) => string | null
  closeOthers: (path: string) => void
  closeAll: () => string | null
  toggleFixed: (path: string) => void
  resetForCharacter: (characterId: number | null) => void
  clear: () => void
}

export const useWorktabStore = create<WorktabStoreState>()(
  persist(
    (set, get) => ({
      ownerCharacterId: null,
      opened: [],
      open: (tab) =>
        set((state) => {
          const existingIndex = state.opened.findIndex((item) => item.routeId === tab.routeId)
          if (existingIndex < 0) {
            return { opened: [...state.opened, { ...tab, fixed: tab.fixed ?? false }] }
          }
          const opened = [...state.opened]
          opened[existingIndex] = {
            ...opened[existingIndex],
            ...tab,
            fixed: tab.fixed ?? opened[existingIndex].fixed,
          }
          return { opened }
        }),
      close: (path) => {
        const state = get()
        const index = state.opened.findIndex((item) => item.path === path)
        if (index < 0 || state.opened[index].fixed) {
          return null
        }
        const opened = state.opened.filter((item) => item.path !== path)
        set({ opened })
        return opened[Math.min(index, opened.length - 1)]?.path ?? null
      },
      closeOthers: (path) =>
        set((state) => ({
          opened: state.opened.filter((item) => item.path === path || item.fixed),
        })),
      closeAll: () => {
        const opened = get().opened.filter((item) => item.fixed)
        set({ opened })
        return opened[0]?.path ?? null
      },
      toggleFixed: (path) =>
        set((state) => {
          const opened = state.opened.map((item) =>
            item.path === path ? { ...item, fixed: !item.fixed } : item
          )
          return {
            opened: [...opened.filter((item) => item.fixed), ...opened.filter((item) => !item.fixed)],
          }
        }),
      resetForCharacter: (characterId) =>
        set((state) =>
          state.ownerCharacterId === characterId
            ? state
            : { ownerCharacterId: characterId, opened: [] }
        ),
      clear: () => set({ ownerCharacterId: null, opened: [] }),
    }),
    {
      name: 'amiyaeden.react.worktabs.v1',
      storage: createJSONStorage(() => localStorage),
      partialize: (state) => ({
        ownerCharacterId: state.ownerCharacterId,
        opened: state.opened,
      }),
    }
  )
)
