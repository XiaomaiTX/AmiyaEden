import { create } from 'zustand'
import { createJSONStorage, persist } from 'zustand/middleware'
import { PREFERENCE_STORE_KEY } from '@/stores/persistence-keys'

export type Locale = 'zh-CN' | 'en-US'
export type ThemeMode = 'light' | 'dark' | 'system'

interface PreferenceStoreState {
  locale: Locale
  sidebarCollapsed: boolean
  theme: ThemeMode
  setLocale: (locale: Locale) => void
  setSidebarCollapsed: (collapsed: boolean) => void
  setTheme: (theme: ThemeMode) => void
  toggleSidebar: () => void
}

export const usePreferenceStore = create<PreferenceStoreState>()(
  persist(
    (set) => ({
      locale: 'zh-CN',
      sidebarCollapsed: false,
      theme: 'system',
      setLocale: (locale) => {
        set({ locale })
      },
      setSidebarCollapsed: (collapsed) => {
        set({ sidebarCollapsed: collapsed })
      },
      setTheme: (theme) => {
        set({ theme })
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
        theme: state.theme,
      }),
    }
  )
)
