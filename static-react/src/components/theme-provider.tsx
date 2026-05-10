import * as React from 'react'
import { ThemeProviderContext, type ResolvedTheme } from '@/components/theme-context'
import { usePreferenceStore } from '@/stores'

function getSystemTheme(): ResolvedTheme {
  if (typeof window === 'undefined' || typeof window.matchMedia !== 'function') {
    return 'light'
  }

  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
}

function resolveTheme(theme: 'light' | 'dark' | 'system', systemTheme: ResolvedTheme) {
  return theme === 'system' ? systemTheme : theme
}

function applyThemeClass(theme: ResolvedTheme) {
  const root = window.document.documentElement

  root.classList.remove('light', 'dark')
  root.classList.add(theme)
  root.style.colorScheme = theme
}

export function ThemeProvider({ children }: { children: React.ReactNode }) {
  const theme = usePreferenceStore((state) => state.theme)
  const setTheme = usePreferenceStore((state) => state.setTheme)
  const [systemTheme, setSystemTheme] = React.useState<ResolvedTheme>(() => getSystemTheme())
  const resolvedTheme = resolveTheme(theme, systemTheme)

  React.useLayoutEffect(() => {
    applyThemeClass(resolvedTheme)
  }, [resolvedTheme])

  React.useEffect(() => {
    if (theme !== 'system' || typeof window.matchMedia !== 'function') {
      return
    }

    const media = window.matchMedia('(prefers-color-scheme: dark)')
    const handleChange = () => {
      setSystemTheme(getSystemTheme())
    }

    if (typeof media.addEventListener === 'function') {
      media.addEventListener('change', handleChange)
      return () => {
        media.removeEventListener('change', handleChange)
      }
    }

    media.addListener(handleChange)
    return () => {
      media.removeListener(handleChange)
    }
  }, [theme])

  const value = React.useMemo(
    () => ({
      theme,
      resolvedTheme,
      setTheme,
    }),
    [theme, resolvedTheme, setTheme]
  )

  return <ThemeProviderContext.Provider value={value}>{children}</ThemeProviderContext.Provider>
}
