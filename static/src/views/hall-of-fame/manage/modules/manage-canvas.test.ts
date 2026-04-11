import assert from 'node:assert/strict'
import test from 'node:test'
import { existsSync, readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'

const fileUrl = new URL('./manage-canvas.vue', import.meta.url)
const filePath = fileURLToPath(fileUrl)

test('manage canvas exposes a synced top scrollbar for wide canvases', () => {
  assert.equal(existsSync(filePath), true)

  const source = readFileSync(fileUrl, 'utf8')

  assert.match(source, /manage-canvas-shell__top-scrollbar/)
  assert.match(source, /manage-canvas-shell__stage/)
  assert.match(source, /transform: `scale\(\$\{props\.zoomRatio\}\)`/)
  assert.match(source, /handleTopScrollbarScroll/)
  assert.match(source, /handleViewportScroll/)
  assert.match(source, /target\.scrollLeft = source\.scrollLeft/)
  assert.match(source, /min-width:\s*0/)
  assert.match(source, /overflow-x:\s*auto/)
  assert.match(source, /overflow-y:\s*auto/)
})
