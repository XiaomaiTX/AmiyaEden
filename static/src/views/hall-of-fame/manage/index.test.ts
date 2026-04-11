import assert from 'node:assert/strict'
import test from 'node:test'
import { existsSync, readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'

const fileUrl = new URL('./index.vue', import.meta.url)
const filePath = fileURLToPath(fileUrl)

test('hall of fame manage page exists and wires admin APIs into the editor layout', () => {
  assert.equal(existsSync(filePath), true)

  const source = readFileSync(fileUrl, 'utf8')

  assert.match(source, /fetchHofConfig/)
  assert.match(source, /fetchHofCards/)
  assert.match(source, /createHofCard/)
  assert.match(source, /batchUpdateHofLayout/)
  assert.match(source, /uploadHofBackground/)
  assert.match(source, /uploadHofBadgeImage/)
  assert.match(source, /import CanvasToolbar from '\.\/modules\/canvas-toolbar\.vue'/)
  assert.match(source, /import ManageCanvas from '\.\/modules\/manage-canvas\.vue'/)
  assert.match(source, /import CardEditor from '\.\/modules\/card-editor\.vue'/)
  assert.match(source, /canvasZoom = ref\(100\)/)
  assert.match(source, /@upload-badge-image="handleBadgeImageUpload"/)
  assert.doesNotMatch(source, /uploadImageAsDataUrl/)
  assert.doesNotMatch(source, /@upload-avatar/)
  assert.match(source, /selectedCardId/)
  assert.match(source, /dirty/)
})

test('hall of fame manage page does not render a standalone title card', () => {
  const source = readFileSync(fileUrl, 'utf8')

  assert.doesNotMatch(source, /hallOfFame\.manage\.eyebrow/)
  assert.doesNotMatch(source, /hallOfFame\.manage\.summary/)
  assert.doesNotMatch(source, /<section class="hall-of-fame-manage__hero">/)
})

test('hall of fame manage page flushes editor drafts into a dedicated preview snapshot', () => {
  const source = readFileSync(fileUrl, 'utf8')

  assert.match(source, /ref="cardEditorRef"/)
  assert.match(source, /cardEditorRef\.value\?\.flushPendingUpdates\(\)/)
  assert.match(source, /saveHallOfFamePreviewDraft\(/)
  assert.match(source, /cards:\s*cards\.value\.filter\(\(card\) => card\.visible\)/)
  assert.doesNotMatch(source, /@update-z-index="handleCardZIndexUpdate"/)
})

test('hall of fame manage page reconciles stale-card errors by removing the missing card locally', () => {
  const source = readFileSync(fileUrl, 'utf8')

  assert.match(source, /getMissingCardIdFromError\(error\)/)
  assert.match(source, /removeCardLocally\(staleCardId\)/)
})
