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

function resolveReference(locale: I18nLocale, key: string, visited: Set<string>): string | undefined {
  if (visited.has(key)) {
    return undefined
  }

  visited.add(key)

  const dictionary = dictionaries[locale] as Record<string, unknown>
  const value = getByPath(dictionary, key)
  if (value === undefined) {
    return undefined
  }

  if (value.startsWith('@:')) {
    return resolveReference(locale, value.slice(2), visited)
  }

  return value
}

function interpolateText(value: string, vars?: Record<string, string | number>) {
  if (!vars) {
    return value
  }

  return value.replace(/\{(\w+)\}/g, (match, key: string) => {
    const replacement = vars[key]
    return replacement === undefined ? match : String(replacement)
  })
}

export function resolveLocaleText(
  locale: I18nLocale,
  key: string,
  vars?: Record<string, string | number>
) {
  const value = resolveReference(locale, key, new Set())
  return value === undefined ? key : interpolateText(value, vars)
}

type TranslateFunction = (key: string, vars?: Record<string, string | number>) => string

interface I18nContextValue {
  locale: I18nLocale
  t: TranslateFunction
}

export const I18nContext = createContext<I18nContextValue>({
  locale: 'zh-CN',
  t: (key: string, vars?: Record<string, string | number>) =>
    resolveLocaleText('zh-CN', key, vars),
})

export function useI18n() {
  return useContext(I18nContext)
}
