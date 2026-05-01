import * as React from 'react'
import type { ThemeMode } from '@/stores'

type ResolvedTheme = 'light' | 'dark'

export type ThemeProviderState = {
  theme: ThemeMode
  resolvedTheme: ResolvedTheme
  setTheme: (theme: ThemeMode) => void
}

export const ThemeProviderContext = React.createContext<ThemeProviderState | undefined>(undefined)

export function useTheme() {
  const context = React.useContext(ThemeProviderContext)

  if (context === undefined) {
    return {
      theme: 'system',
      resolvedTheme: 'light',
      setTheme: () => {},
    }
  }

  return context
}

export type { ResolvedTheme }
