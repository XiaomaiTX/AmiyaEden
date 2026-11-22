export type LocaleMessageLoader<TLocale extends string, TMessages> = (
  locale: TLocale
) => Promise<TMessages>

export interface LazyLocaleManager<TLocale extends string> {
  isLoaded: (locale: TLocale) => boolean
  ensureLoaded: (locale: TLocale) => Promise<void>
  ensureWithFallback: (locale: TLocale, fallbackLocale: TLocale) => Promise<void>
  reset: () => void
}

export function createLazyLocaleManager<TLocale extends string, TMessages>(
  load: LocaleMessageLoader<TLocale, TMessages>,
  apply: (locale: TLocale, messages: TMessages) => void
): LazyLocaleManager<TLocale> {
  const loaded = new Set<TLocale>()
  const inFlight = new Map<TLocale, Promise<void>>()

  const isLoaded = (locale: TLocale) => loaded.has(locale)

  const ensureLoaded = (locale: TLocale): Promise<void> => {
    if (loaded.has(locale)) {
      return Promise.resolve()
    }

    const pending = inFlight.get(locale)
    if (pending) {
      return pending
    }

    const task = load(locale)
      .then((messages) => {
        apply(locale, messages)
        loaded.add(locale)
      })
      .finally(() => {
        inFlight.delete(locale)
      })

    inFlight.set(locale, task)
    return task
  }

  const ensureWithFallback = async (locale: TLocale, fallbackLocale: TLocale): Promise<void> => {
    if (locale === fallbackLocale) {
      await ensureLoaded(locale)
      return
    }
    await Promise.all([ensureLoaded(locale), ensureLoaded(fallbackLocale)])
  }

  const reset = () => {
    loaded.clear()
    inFlight.clear()
  }

  return {
    isLoaded,
    ensureLoaded,
    ensureWithFallback,
    reset
  }
}
