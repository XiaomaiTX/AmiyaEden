import assert from 'node:assert/strict'
import test from 'node:test'
import { readFileSync } from 'node:fs'

const source = readFileSync(new URL('./index.vue', import.meta.url), 'utf8')

test('contributors view binds contributors endpoint and fallback copy', () => {
  assert.match(source, /fetchFuxiHallContributors/)
  assert.match(source, /fuxiHall\.public\.defaultContributorsTitle/)
  assert.match(source, /fuxiHall\.public\.emptyTitle/)
  assert.match(source, /fuxiHall\.public\.emptySubtitle/)
})

test('contributors view keeps responsive card grid layout', () => {
  assert.match(source, /grid-template-columns:\s*repeat\(auto-fit,\s*minmax\(300px,\s*1fr\)\)/)
  assert.match(source, /buildEveCharacterPortraitUrl\(card\.main_character_id,\s*256\)/)
  assert.match(source, /@media\s*\(max-width:\s*768px\)/)
})

test('contributors view uses manage-style member card layout without cover section', () => {
  assert.match(source, /class="fuxi-hall-page__meta"/)
  assert.doesNotMatch(source, /fuxi-hall-page__cover/)
  assert.doesNotMatch(source, /card\.cover_image/)
})
