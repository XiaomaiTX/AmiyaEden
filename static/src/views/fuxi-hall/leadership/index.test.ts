import assert from 'node:assert/strict'
import test from 'node:test'
import { readFileSync } from 'node:fs'

const source = readFileSync(new URL('./index.vue', import.meta.url), 'utf8')

test('leadership view binds leadership endpoint and fallback copy', () => {
  assert.match(source, /fetchFuxiHallLeadership/)
  assert.match(source, /fuxiHall\.public\.defaultLeadershipTitle/)
  assert.match(source, /fuxiHall\.public\.emptyTitle/)
  assert.match(source, /fuxiHall\.public\.emptySubtitle/)
})

test('leadership view renders sanitized rich-text payload fields', () => {
  assert.match(source, /v-html="page\.description_html"/)
  assert.match(source, /FuxiHallMemberCard/)
  assert.match(source, /:card="card"/)
})

test('leadership view uses shared member card component and no legacy title field binding', () => {
  assert.match(source, /components\/FuxiHallMemberCard\.vue/)
  assert.doesNotMatch(source, /card\.title/)
})
