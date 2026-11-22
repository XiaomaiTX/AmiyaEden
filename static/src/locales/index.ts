/**
 * 国际化配置
 *
 * 基于 vue-i18n 实现的多语言国际化解决方案。
 * 支持中文和英文切换，自动从本地存储恢复用户的语言偏好。
 *
 * ## 主要功能
 *
 * - 多语言支持 - 支持中文（简体）和英文两种语言
 * - 语言切换 - 运行时动态切换语言，无需刷新页面
 * - 持久化存储 - 自动保存和恢复用户的语言偏好
 * - 全局注入 - 在任何组件中都可以使用 $t 函数进行翻译
 * - 类型安全 - 提供 TypeScript 类型支持
 *
 * ## 支持的语言
 *
 * - zh: 简体中文
 * - en: English
 *
 * @module locales
 * @author Art Design Pro Team
 */

import { createI18n } from 'vue-i18n'
import type { I18n, I18nOptions } from 'vue-i18n'
import { LanguageEnum } from '@/enums/appEnum'
import { getSystemStorage } from '@/utils/storage'
import { StorageKeyManager } from '@/utils/storage/storage-key-manager'
import { createLazyLocaleManager } from './loader'

/**
 * 存储键管理器实例
 */
const storageKeyManager = new StorageKeyManager()

type LocaleMessageSchema = Record<string, unknown>

const localeLoaders: Record<LanguageEnum, () => Promise<{ default: LocaleMessageSchema }>> = {
  [LanguageEnum.ZH]: () => import('./langs/zh.json') as Promise<{ default: LocaleMessageSchema }>,
  [LanguageEnum.EN]: () => import('./langs/en.json') as Promise<{ default: LocaleMessageSchema }>
}

/**
 * 语言选项列表
 * 用于语言切换下拉框
 */
export const languageOptions = [
  { value: LanguageEnum.ZH, label: '简体中文' },
  { value: LanguageEnum.EN, label: 'English' }
]

/**
 * 从存储中获取语言设置
 * @returns 语言设置，如果获取失败则返回默认语言
 */
const getDefaultLanguage = (): LanguageEnum => {
  // 尝试从版本化的存储中获取语言设置
  try {
    const storageKey = storageKeyManager.getStorageKey('user')
    const userStore = localStorage.getItem(storageKey)

    if (userStore) {
      const { language } = JSON.parse(userStore)
      if (language && Object.values(LanguageEnum).includes(language)) {
        return language
      }
    }
  } catch (error) {
    console.warn('[i18n] 从版本化存储获取语言设置失败:', error)
  }

  // 尝试从系统存储中获取语言设置
  try {
    const sys = getSystemStorage()
    if (sys) {
      const { user } = JSON.parse(sys)
      if (user?.language && Object.values(LanguageEnum).includes(user.language)) {
        return user.language
      }
    }
  } catch (error) {
    console.warn('[i18n] 从系统存储获取语言设置失败:', error)
  }

  // 返回默认语言
  console.debug('[i18n] 使用默认语言:', LanguageEnum.ZH)
  return LanguageEnum.ZH
}

export const defaultLanguage = getDefaultLanguage()

/**
 * i18n 配置选项（语言包按需动态加载，同时保持 zh 回退语言可用）
 */
const i18nOptions: I18nOptions = {
  locale: defaultLanguage,
  legacy: false,
  globalInjection: true,
  fallbackLocale: LanguageEnum.ZH,
  messages: {}
}

/**
 * i18n 实例
 */
const i18n: I18n = createI18n(i18nOptions)

const localeManager = createLazyLocaleManager<LanguageEnum, LocaleMessageSchema>(
  async (lang) => {
    const mod = await localeLoaders[lang]()
    return mod.default
  },
  (lang, loadedMessages) => {
    i18n.global.setLocaleMessage(lang, loadedMessages)
  }
)

export function loadLocaleMessages(lang: LanguageEnum): Promise<void> {
  return localeManager.ensureWithFallback(lang, LanguageEnum.ZH)
}

export async function setAppLocale(lang: LanguageEnum): Promise<void> {
  await loadLocaleMessages(lang)
  ;(i18n.global.locale as { value: string }).value = lang
}

/**
 * 翻译函数类型
 */
interface Translation {
  (key: string, values?: Record<string, string | number>): string
}

/**
 * 全局翻译函数
 * 可在任何地方使用，无需导入 useI18n
 */
export const $t = i18n.global.t as Translation

export default i18n
