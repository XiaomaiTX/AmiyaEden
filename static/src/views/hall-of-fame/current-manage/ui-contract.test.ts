import assert from 'node:assert/strict'
import test from 'node:test'
import { readFileSync } from 'node:fs'

const indexSource = readFileSync(new URL('./index.vue', import.meta.url), 'utf8')
const adminCardSource = readFileSync(new URL('./modules/admin-card.vue', import.meta.url), 'utf8')
const adminDialogSource = readFileSync(
  new URL('./modules/admin-card-dialog.vue', import.meta.url),
  'utf8'
)
const enMessages = JSON.parse(
  readFileSync(new URL('../../../locales/langs/en.json', import.meta.url), 'utf8')
)
const zhMessages = JSON.parse(
  readFileSync(new URL('../../../locales/langs/zh.json', import.meta.url), 'utf8')
)

test('current-manage page removes the title banner and exposes color controls', () => {
  assert.doesNotMatch(indexSource, /hallOfFame\.currentManage\.title/)
  assert.match(indexSource, /ElColorPicker/)
  assert.match(indexSource, /v-if="canEdit && directory"/)
  assert.match(
    indexSource,
    /pageBackgroundColor|cardBackgroundColor|cardBorderColor|tierTitleColor|nameTextColor|bodyTextColor/
  )
  assert.match(indexSource, /cardWidth/)
})

test('current-manage page serializes config saves to avoid stale responses overriding newer choices', () => {
  assert.match(indexSource, /pendingConfigSnapshot/)
  assert.match(indexSource, /const configSaveInFlight = ref\(false\)/)
  assert.match(indexSource, /configSaveInFlight\.value/)
  assert.doesNotMatch(indexSource, /let configSaveInFlight = false/)
  assert.match(indexSource, /flushConfigSaveQueue/)
  assert.match(indexSource, /buildConfigUpdateSnapshot/)
})

test('current-manage page disables alpha on all six color pickers', () => {
  assert.equal(indexSource.match(/ElColorPicker/g)?.length ?? 0, 6)
  assert.equal(indexSource.match(/show-alpha="false"/g)?.length ?? 0, 6)
})

test('current-manage page exposes an explicit load failure state with dedicated i18n', () => {
  assert.match(indexSource, /loadErrorMessage/)
  assert.match(indexSource, /hallOfFame\.currentManage\.loadFailed/)
  assert.match(indexSource, /v-else-if="loadErrorMessage/)
  assert.equal(typeof enMessages.hallOfFame.currentManage.loadFailed, 'string')
  assert.equal(typeof zhMessages.hallOfFame.currentManage.loadFailed, 'string')
})

test('admin-card dialog accepts a default tier id for add flows', () => {
  assert.match(adminDialogSource, /defaultTierId/)
  assert.match(adminDialogSource, /props\.admin\?\.tier_id \?\? props\.defaultTierId/)
})

test('admin-card dialog uses the renamed position, nickname, and character name labels', () => {
  assert.match(adminDialogSource, /hallOfFame\.currentManage\.tierLabel/)
  assert.match(adminDialogSource, /hallOfFame\.currentManage\.nameLabel/)
  assert.match(adminDialogSource, /hallOfFame\.currentManage\.titleLabel/)
  assert.match(adminDialogSource, /hallOfFame\.currentManage\.tierRequired/)
  assert.match(adminDialogSource, /form\.nickname/)
  assert.match(adminDialogSource, /form\.characterName/)
  assert.match(adminDialogSource, /character_name/)
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

test('admin cards include shared copy buttons for visible contact values', () => {
  assert.match(adminCardSource, /ArtCopyButton/)
  assert.match(adminCardSource, /admin\.contact_qq/)
  assert.match(adminCardSource, /admin\.contact_discord/)
  assert.match(adminCardSource, /fuxi-admin-card__contact-value/)
  assert.match(adminCardSource, /admin\.nickname/)
  assert.match(adminCardSource, /admin\.character_name/)
})
