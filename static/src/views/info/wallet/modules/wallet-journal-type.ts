/**
 * 钱包流水类型（ref_type）展示名解析
 *
 * 流水表格与收支分析图共用同一条 i18n 回退链：
 * npcKill.refTypes.* -> walletAdmin.refTypes.* -> info.wallet.refTypes.* -> 下划线转 Title Case
 *
 * @module views/info/wallet/modules/wallet-journal-type
 */

/** i18n 翻译函数，仅需要支持点分 key */
export type TranslateFn = (key: string) => string

const REF_TYPE_I18N_PREFIXES = ['npcKill.refTypes', 'walletAdmin.refTypes', 'info.wallet.refTypes']

/** 下划线命名转 Title Case，作为全部翻译层未命中时的兜底展示名 */
export const toTitleCaseRefType = (value: string): string =>
  value
    .split('_')
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(' ')

/** 解析钱包流水类型展示名：命中回退链中任意一层翻译即返回，否则退回 Title Case */
export const formatJournalTypeLabel = (value: string, t: TranslateFn): string => {
  for (const prefix of REF_TYPE_I18N_PREFIXES) {
    const key = `${prefix}.${value}`
    const translated = t(key)
    if (translated !== key) {
      return translated
    }
  }
  return toTitleCaseRefType(value)
}
