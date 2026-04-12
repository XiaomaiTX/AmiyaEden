import assert from 'node:assert/strict'
import test from 'node:test'
import { readFileSync } from 'node:fs'

const source = readFileSync(new URL('./overview-matrix.vue', import.meta.url), 'utf8')

test('overview matrix groups scopes by module', () => {
  assert.match(source, /scope\.module/)
  assert.match(source, /groupedScopes/)
})

test('overview matrix shows token invalid warning for characters', () => {
  assert.match(source, /token_invalid/)
  assert.match(source, /esiCheckTokenInvalid/)
})

test('overview matrix computes coverage per character', () => {
  assert.match(source, /formatCoverage/)
  assert.match(source, /esiCheckCoverage/)
  assert.match(source, /props\.scopes\.filter/)
})

test('overview matrix emits select-character on click', () => {
  assert.match(source, /\$emit\('select-character'/)
  assert.match(source, /@click.*select-character/)
})
