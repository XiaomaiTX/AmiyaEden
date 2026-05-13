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
  assert.match(source, /cardForm\.main_character_id/)
  assert.match(source, /if \(!previewCard\.visible\)/)
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

test('manage view no longer exposes deprecated style preset, badge tone, and cover height fields', () => {
  assert.doesNotMatch(source, /fuxiHall\.manage\.stylePreset/)
  assert.doesNotMatch(source, /fuxiHall\.manage\.badgeTone/)
  assert.doesNotMatch(source, /fuxiHall\.manage\.coverHeight/)
  assert.doesNotMatch(source, /style_preset/)
  assert.doesNotMatch(source, /badge_tone/)
  assert.doesNotMatch(source, /cover_height/)
})
