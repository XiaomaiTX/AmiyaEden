import assert from 'node:assert/strict'
import test from 'node:test'
import { readFileSync } from 'node:fs'

const source = readFileSync(new URL('./index.vue', import.meta.url), 'utf8')

test('fuxi directory page uses correct i18n key for page title', () => {
  assert.match(source, /menus\.hallOfFame\.currentManage/)
  assert.match(source, /hallOfFame\.currentManage\.title/)
})

test('fuxi directory page renders tier sections and empty state', () => {
  assert.match(source, /hallOfFame\.currentManage\.emptyTitle/)
  assert.match(source, /TierSection/)
})

test('fuxi directory page shows admin controls only for editors', () => {
  assert.match(source, /canEdit/)
  assert.match(source, /hallOfFame\.currentManage\.addTier/)
  assert.match(source, /hallOfFame\.currentManage\.baseFontSize/)
})

test('fuxi directory page includes tier and admin dialogs', () => {
  assert.match(source, /TierDialog/)
  assert.match(source, /AdminCardDialog/)
})
