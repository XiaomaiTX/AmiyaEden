import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

const source = readFileSync(new URL('./useCorpCapability.ts', import.meta.url), 'utf8')

test('useCorpCapability hook exposes three predicates', () => {
  assert.match(source, /export const useCorpCapability/)
  assert.match(source, /hasCapability,/)
  assert.match(source, /hasAllCapabilities,/)
  assert.match(source, /hasAnyCapability/)
})

test('useCorpCapability short-circuits to true for super_admin', () => {
  assert.match(
    source,
    /const isSuperAdmin = \(\):\s*boolean => roles\(\)\.includes\('super_admin'\)/
  )
  assert.match(source, /if \(isSuperAdmin\(\)\) return true/)
})

test('useCorpCapability reads capabilities from user store', () => {
  assert.match(source, /userStore\.info\?\.corpCapabilities/)
})
