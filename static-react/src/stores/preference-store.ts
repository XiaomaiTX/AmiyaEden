import { create } from 'zustand'
import { createJSONStorage, persist } from 'zustand/middleware'
import { PREFERENCE_STORE_KEY } from '@/stores/persistence-keys'

export type Locale = 'zh-CN' | 'en-US'

interface PreferenceStoreState {
  locale: Locale
  sidebarCollapsed: boolean
  setLocale: (locale: Locale) => void
  setSidebarCollapsed: (collapsed: boolean) => void
  toggleSidebar: () => void
}

export const usePreferenceStore = create<PreferenceStoreState>()(
  persist(
    (set) => ({
      locale: 'zh-CN',
      sidebarCollapsed: false,
      setLocale: (locale) => {
        set({ locale })
      },
      setSidebarCollapsed: (collapsed) => {
        set({ sidebarCollapsed: collapsed })
      },
      toggleSidebar: () => {
        set((state) => ({ sidebarCollapsed: !state.sidebarCollapsed }))
      },
    }),
    {
      name: PREFERENCE_STORE_KEY,
      storage: createJSONStorage(() => localStorage),
      partialize: (state) => ({
        locale: state.locale,
        sidebarCollapsed: state.sidebarCollapsed,
      }),
    }
  )
)
