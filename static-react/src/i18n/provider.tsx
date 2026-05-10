import type { PropsWithChildren } from 'react'
import { useMemo } from 'react'
import { I18nContext, resolveLocaleText } from '@/i18n/context'
import { usePreferenceStore } from '@/stores'

export function I18nProvider({ children }: PropsWithChildren) {
  const locale = usePreferenceStore((state) => state.locale)

  const value = useMemo(
    () => ({
      locale,
      t: (key: string, vars?: Record<string, string | number>) =>
        resolveLocaleText(locale, key, vars),
    }),
    [locale]
  )

  return <I18nContext.Provider value={value}>{children}</I18nContext.Provider>
}
