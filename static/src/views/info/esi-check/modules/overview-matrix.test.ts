import assert from 'node:assert/strict'
import test from 'node:test'
import { readFileSync } from 'node:fs'

const source = readFileSync(new URL('./overview-matrix.vue', import.meta.url), 'utf8')

test('overview matrix filters invalid characters by token_invalid or missing required scopes', () => {
  assert.match(source, /invalidCharacters/)
  assert.match(source, /token_invalid/)
  assert.match(source, /s\.required/)
})

test('overview matrix shows all-valid badge when no invalid characters', () => {
  assert.match(source, /esiCheckAllValid/)
})

test('overview matrix shows invalid count badge and character name badges', () => {
  assert.match(source, /esiCheckInvalidCount/)
  assert.match(source, /character_name/)
})

test('overview matrix emits select-character on badge click', () => {
  assert.match(source, /\$emit\('select-character'/)
  assert.match(source, /@click.*select-character/)
})
