import assert from 'node:assert/strict'
import test from 'node:test'
import { readFileSync } from 'node:fs'

const source = readFileSync(new URL('./index.vue', import.meta.url), 'utf8')

test('tool bookmarks page supports admin management and public view', () => {
  assert.match(source, /fetchVisibleToolBookmarks/)
  assert.match(source, /fetchAdminToolBookmarks/)
  assert.match(source, /createToolBookmark/)
  assert.match(source, /updateToolBookmark/)
  assert.match(source, /deleteToolBookmark/)
  assert.match(source, /const isAdmin = computed/)
  assert.match(source, /target="_blank"/)
})
