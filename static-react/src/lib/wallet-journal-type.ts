/** Shared labels for wallet filters, journal rows and analytics charts. */
export function formatJournalTypeLabel(value: string, t: (key: string) => string): string {
  const key = `infoWallet.refTypes.${value}`
  const translated = t(key)
  return translated !== key
    ? translated
    : value
        .split('_')
        .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
        .join(' ')
}
