import assert from 'node:assert/strict'
import test from 'node:test'
import { createLazyLocaleManager } from './loader'

test('lazy locale manager loads once, deduplicates concurrent requests, and caches loaded locales', async () => {
  let loadCalls = 0
  const applied: Array<{ locale: string; messages: Record<string, string> }> = []

  const manager = createLazyLocaleManager<'zh' | 'en', Record<string, string>>(
    async (locale) => {
      loadCalls += 1
      await Promise.resolve()
      return { greeting: locale === 'zh' ? '你好' : 'Hello' }
    },
    (locale, messages) => {
      applied.push({ locale, messages })
    }
  )

  assert.equal(manager.isLoaded('zh'), false)

  await Promise.all([manager.ensureLoaded('zh'), manager.ensureLoaded('zh')])
  assert.equal(loadCalls, 1)
  assert.equal(manager.isLoaded('zh'), true)
  assert.equal(manager.isLoaded('en'), false)

  await manager.ensureLoaded('zh')
  assert.equal(loadCalls, 1)

  await manager.ensureLoaded('en')
  assert.equal(loadCalls, 2)
  assert.equal(manager.isLoaded('en'), true)
  assert.deepEqual(
    applied.map((entry) => entry.locale),
    ['zh', 'en']
  )
})

test('lazy locale manager retries after a transient load failure', async () => {
  let attempts = 0
  const manager = createLazyLocaleManager<'en', Record<string, string>>(
    async () => {
      attempts += 1
      if (attempts === 1) {
        throw new Error('chunk failed')
      }
      return { ok: 'yes' }
    },
    () => {}
  )

  await assert.rejects(() => manager.ensureLoaded('en'), /chunk failed/)
  assert.equal(manager.isLoaded('en'), false)

  await manager.ensureLoaded('en')
  assert.equal(attempts, 2)
  assert.equal(manager.isLoaded('en'), true)
})

test('ensureWithFallback loads fallback locale alongside non-default active locale', async () => {
  const loadedLocales: string[] = []
  const manager = createLazyLocaleManager<'zh' | 'en', Record<string, string>>(
    async (locale) => {
      loadedLocales.push(locale)
      return { key: locale }
    },
    () => {}
  )

  await manager.ensureWithFallback('en', 'zh')
  assert.deepEqual(loadedLocales.sort(), ['en', 'zh'])
  assert.equal(manager.isLoaded('en'), true)
  assert.equal(manager.isLoaded('zh'), true)
})

test('ensureWithFallback loads only once when active locale equals fallback locale', async () => {
  const loadedLocales: string[] = []
  const manager = createLazyLocaleManager<'zh' | 'en', Record<string, string>>(
    async (locale) => {
      loadedLocales.push(locale)
      return { key: locale }
    },
    () => {}
  )

  await manager.ensureWithFallback('zh', 'zh')
  assert.deepEqual(loadedLocales, ['zh'])
})
