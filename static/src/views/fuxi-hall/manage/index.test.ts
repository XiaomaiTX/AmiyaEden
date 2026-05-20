import assert from 'node:assert/strict'
import test from 'node:test'
import { readFileSync } from 'node:fs'

const source = readFileSync(new URL('./index.vue', import.meta.url), 'utf8')

test('manage view exposes live preview panel driven by editing state', () => {
  assert.match(source, /fuxiHall\.manage\.previewPanel/)
  assert.match(source, /const previewPage = computed\(/)
  assert.match(source, /const previewCards = computed\(/)
  assert.match(source, /v-html="previewPage\.description_html"/)
})

test('manage view preview updates from draft card form and visibility', () => {
  assert.match(source, /cardDialogOpen\.value/)
  assert.match(source, /cardForm\.nickname\.trim\(\)/)
  assert.match(source, /cardForm\.title_tags/)
  assert.match(source, /if \(!previewCard\.visible\)/)
})

test('manage view no longer edits main character id manually', () => {
  assert.doesNotMatch(source, /v-model="cardForm\.main_character_id"/)
  assert.doesNotMatch(source, /mainCharacterId/)
})

test('manage view supports manual reorder and responsive preview grid', () => {
  assert.match(source, /reorderFuxiHallCards/)
  assert.match(source, /grid-template-columns:\s*repeat\(auto-fit,\s*minmax\(300px,\s*1fr\)\)/)
  assert.match(source, /@media\s*\(max-width:\s*768px\)/)
})

test('manage view uses standard tabs, card-list add button, and page-level save button', () => {
  assert.match(source, /<ElTabs v-model="currentPageKey"/)
  assert.match(
    source,
    /<span>\{\{ t\('fuxiHall\.manage\.cardList'\) \}\}<\/span>[\s\S]*t\('fuxiHall\.manage\.addCard'\)/
  )
  assert.match(source, /<ElFormItem>[\s\S]*t\('common\.save'\)/)
})

test('manage view supports title tag editor and list rendering', () => {
  assert.match(source, /titleTagInput/)
  assert.match(source, /addTitleTag/)
  assert.match(source, /removeTitleTag/)
  assert.match(source, /fuxiHall\.manage\.titleTagPlaceholder/)
  assert.match(source, /fuxiHall\.manage\.addTitleTag/)
  assert.match(source, /row\.title_tags/)
})

test('manage view no longer exposes deprecated style preset, badge tone, and cover height fields', () => {
  assert.doesNotMatch(source, /fuxiHall\.manage\.stylePreset/)
  assert.doesNotMatch(source, /fuxiHall\.manage\.badgeTone/)
  assert.doesNotMatch(source, /fuxiHall\.manage\.coverHeight/)
  assert.doesNotMatch(source, /style_preset/)
  assert.doesNotMatch(source, /badge_tone/)
  assert.doesNotMatch(source, /cover_height/)
})

test('manage view removes cover-image editing and preview rendering', () => {
  assert.doesNotMatch(source, /cover_image/)
  assert.doesNotMatch(source, /fuxiHall\.manage\.coverImage/)
  assert.doesNotMatch(source, /fuxiHall\.manage\.uploadCover/)
  assert.doesNotMatch(source, /fuxi-hall-manage__preview-cover/)
})

test('manage view no longer binds legacy card.title rendering', () => {
  assert.doesNotMatch(source, /v-model="cardForm\.title"/)
  assert.doesNotMatch(source, /row\.title\b/)
})

test('manage view shows stats only when values are greater than zero', () => {
  assert.match(source, /row\.fleet_led_count/)
  assert.match(source, /row\.welfare_delivery_count/)
  assert.match(source, /\(row\.fleet_led_count \?\? 0\) > 0/)
  assert.match(source, /\(row\.welfare_delivery_count \?\? 0\) > 0/)
})

test('manage view exposes welfare delivery offset only for super admin in edit mode', () => {
  assert.match(source, /v-if="editingCardId && isSuperAdmin"/)
  assert.match(source, /welfare_delivery_offset/)
  assert.match(source, /roles\.includes\('super_admin'\)/)
})

test('manage view only sends welfare delivery offset in update payload for super admin', () => {
  assert.match(source, /if \(editingCardId\.value\)/)
  assert.match(source, /const updatePayload: Api\.FuxiHall\.UpdateCardParams = \{ \.\.\.payload \}/)
  assert.match(source, /if \(isSuperAdmin\.value\)/)
  assert.match(source, /updatePayload\.welfare_delivery_offset = cardForm\.welfare_delivery_offset/)
})
