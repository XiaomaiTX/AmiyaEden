import assert from 'node:assert/strict'
import test from 'node:test'
import { readFileSync } from 'node:fs'

const indexSource = readFileSync(new URL('./index.vue', import.meta.url), 'utf8')
const adminCardSource = readFileSync(new URL('./modules/admin-card.vue', import.meta.url), 'utf8')
const adminDialogSource = readFileSync(
  new URL('./modules/admin-card-dialog.vue', import.meta.url),
  'utf8'
)

test('current-manage page removes the title banner and exposes color controls', () => {
  assert.doesNotMatch(indexSource, /hallOfFame\.currentManage\.title/)
  assert.match(indexSource, /ElColorPicker/)
  assert.match(
    indexSource,
    /pageBackgroundColor|cardBackgroundColor|cardBorderColor|tierTitleColor|nameTextColor|bodyTextColor/
  )
  assert.match(indexSource, /cardWidth/)
})

test('current-manage page serializes config saves to avoid stale responses overriding newer choices', () => {
  assert.match(indexSource, /pendingConfigSnapshot/)
  assert.match(indexSource, /configSaveInFlight/)
  assert.match(indexSource, /flushConfigSaveQueue/)
  assert.match(indexSource, /buildConfigUpdateSnapshot/)
})

test('admin-card dialog accepts a default tier id for add flows', () => {
  assert.match(adminDialogSource, /defaultTierId/)
  assert.match(adminDialogSource, /props\.admin\?\.tier_id \?\? props\.defaultTierId/)
})

test('admin cards render description and distinct typography variables', () => {
  assert.match(adminCardSource, /admin\.description/)
  assert.match(adminCardSource, /--card-name-font-size/)
  assert.match(adminCardSource, /--card-title-font-size/)
  assert.match(adminCardSource, /--card-description-font-size/)
  assert.match(adminCardSource, /--card-contact-font-size/)
  assert.match(adminCardSource, /--card-width/)
  assert.match(adminCardSource, /--card-border-color/)
  assert.match(adminCardSource, /--card-name-color/)
  assert.match(adminCardSource, /--card-body-color/)
  assert.match(adminCardSource, /overflow-wrap:\s*anywhere/)
  assert.doesNotMatch(adminCardSource, /Math\.max\(props\.styleConfig\.base_font_size, 14\)/)
})
