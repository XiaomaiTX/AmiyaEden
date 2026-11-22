import type { PropsWithChildren } from 'react'
import { useEffect, useMemo, useState } from 'react'
import {
  ensureLocaleLoaded,
  I18nContext,
  isLocaleLoaded,
  resolveLocaleText,
} from '@/i18n/context'
import { usePreferenceStore } from '@/stores'

export function I18nProvider({ children }: PropsWithChildren) {
  const locale = usePreferenceStore((state) => state.locale)
  const [loadedVersion, setLoadedVersion] = useState(0)

  useEffect(() => {
    if (isLocaleLoaded(locale)) {
      return
    }
    let cancelled = false
    void ensureLocaleLoaded(locale).then(() => {
      if (!cancelled) {
        setLoadedVersion((version) => version + 1)
      }
    })
    return () => {
      cancelled = true
    }
  }, [locale])

  const value = useMemo(
    () => ({
      locale,
      t: (key: string, vars?: Record<string, string | number>) =>
        resolveLocaleText(locale, key, vars),
    }),
    // loadedVersion ensures consumers re-render once an on-demand locale chunk arrives.
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [locale, loadedVersion]
  )

  return <I18nContext.Provider value={value}>{children}</I18nContext.Provider>
}
