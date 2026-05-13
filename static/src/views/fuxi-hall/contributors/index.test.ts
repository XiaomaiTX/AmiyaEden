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
  assert.match(source, /FuxiHallMemberCard/)
  assert.match(source, /@media\s*\(max-width:\s*768px\)/)
})

test('contributors view uses shared member card component and no legacy title field binding', () => {
  assert.match(source, /components\/FuxiHallMemberCard\.vue/)
  assert.doesNotMatch(source, /card\.title/)
})
