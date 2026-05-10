import { createContext, useContext } from 'react'
import enUS from '@/i18n/messages/en-US'
import zhCN from '@/i18n/messages/zh-CN'

export const dictionaries = {
  'zh-CN': zhCN,
  'en-US': enUS,
} as const

export type I18nLocale = keyof typeof dictionaries

function getByPath(source: Record<string, unknown>, path: string): string | undefined {
  const value = path.split('.').reduce<unknown>((acc, segment) => {
    if (!acc || typeof acc !== 'object') {
      return undefined
    }
    return (acc as Record<string, unknown>)[segment]
  }, source)

  return typeof value === 'string' ? value : undefined
}

export function resolveLocaleText(locale: I18nLocale, key: string) {
  const dictionary = dictionaries[locale] as Record<string, unknown>
  return getByPath(dictionary, key) ?? key
}

interface I18nContextValue {
  locale: I18nLocale
  t: (key: string) => string
}

export const I18nContext = createContext<I18nContextValue>({
  locale: 'zh-CN',
  t: (key: string) => resolveLocaleText('zh-CN', key),
})

export function useI18n() {
  return useContext(I18nContext)
}
