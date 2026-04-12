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

test('character detail shows reauth tip only for missing required scopes', () => {
  assert.match(source, /hasMissingRequiredScopes/)
  assert.match(source, /r\.required/)
  assert.match(source, /esiCheckReauthTip/)
})

test('character detail reauth button calls getEveBindURL', () => {
  assert.match(source, /getEveBindURL/)
  assert.match(source, /window\.location\.href/)
  assert.match(source, /esiCheckReauthFailed/)
})

test('character detail coverage summary counts only required scopes', () => {
  assert.match(source, /formatCoverage/)
  assert.match(source, /esiCheckCoverage/)
  assert.match(source, /s\.required/)
})
