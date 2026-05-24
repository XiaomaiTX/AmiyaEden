import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

const source = readFileSync(new URL('./index.vue', import.meta.url), 'utf8')

test('captain selection page allows captain cards to grow with content', () => {
  assert.match(source, /grid grid-cols-1 xl:grid-cols-2 gap-4/)
  assert.doesNotMatch(source, /<template>\s*<div class="[^"]*\bart-full-height\b[^"]*">/)
})

test('captain selection page shows captain name first in current captain tag', () => {
  assert.match(source, /newbro\.select\.currentCaptainTag/)
  assert.match(source, /name: state\.current_affiliation\.captain_character_name/)
  assert.match(source, /id: state\.current_affiliation\.captain_user_id/)
})
