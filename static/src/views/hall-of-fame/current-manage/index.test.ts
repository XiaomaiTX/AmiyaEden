import assert from 'node:assert/strict'
import test from 'node:test'
import { readFileSync } from 'node:fs'

const source = readFileSync(new URL('./index.vue', import.meta.url), 'utf8')

test('fuxi directory page renders tier sections and empty state', () => {
  assert.match(source, /hallOfFame\.currentManage\.emptyTitle/)
  assert.match(source, /TierSection/)
})

test('fuxi directory page shows editor-only settings for layout and grouped colors', () => {
  assert.match(source, /canEdit/)
  assert.match(source, /hallOfFame\.currentManage\.addTier/)
  assert.match(source, /hallOfFame\.currentManage\.baseFontSize/)
  assert.match(source, /hallOfFame\.currentManage\.cardWidth/)
  assert.match(source, /hallOfFame\.currentManage\.pageBackgroundColor/)
  assert.match(source, /hallOfFame\.currentManage\.cardBackgroundColor/)
  assert.match(source, /hallOfFame\.currentManage\.cardBorderColor/)
  assert.match(source, /hallOfFame\.currentManage\.tierTitleColor/)
  assert.match(source, /hallOfFame\.currentManage\.nameTextColor/)
  assert.match(source, /hallOfFame\.currentManage\.bodyTextColor/)
  assert.match(source, /ElColorPicker/)
  assert.match(source, /ElInputNumber/)
})

test('fuxi directory page removes the title banner and forwards tier defaults into the dialog', () => {
  assert.doesNotMatch(source, /hallOfFame\.currentManage\.title/)
  assert.match(source, /default-tier-id="addingAdminToTierId"/)
})

test('fuxi directory page includes tier and admin dialogs', () => {
  assert.match(source, /TierDialog/)
  assert.match(source, /AdminCardDialog/)
})
