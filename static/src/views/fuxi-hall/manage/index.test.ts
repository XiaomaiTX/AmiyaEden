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
  assert.match(source, /if \(!previewCard\.visible\)/)
})

test('manage view supports manual reorder and responsive preview grid', () => {
  assert.match(source, /reorderFuxiHallCards/)
  assert.match(source, /grid-template-columns:\s*repeat\(auto-fit,\s*minmax\(260px,\s*1fr\)\)/)
  assert.match(source, /@media\s*\(max-width:\s*768px\)/)
})
