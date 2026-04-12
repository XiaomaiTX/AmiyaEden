import assert from 'node:assert/strict'
import test from 'node:test'
import { readFileSync } from 'node:fs'

const source = readFileSync(new URL('./character-detail.vue', import.meta.url), 'utf8')

test('character detail renders scope authorization status from character.scopes', () => {
  assert.match(source, /parseScopeSet/)
  assert.match(source, /scopeSet\.has\(s\.scope\)/)
  assert.match(source, /authorized/)
})

test('character detail shows alert when token is invalid', () => {
  assert.match(source, /token_invalid/)
  assert.match(source, /ElAlert/)
  assert.match(source, /esiCheckTokenInvalidTip/)
})

test('character detail shows reauth tip for missing scopes', () => {
  assert.match(source, /hasMissingScopes/)
  assert.match(source, /esiCheckReauthTip/)
})

test('character detail computes coverage summary', () => {
  assert.match(source, /formatCoverage/)
  assert.match(source, /esiCheckCoverage/)
})
