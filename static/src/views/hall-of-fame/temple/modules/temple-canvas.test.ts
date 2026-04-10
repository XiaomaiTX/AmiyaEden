import assert from 'node:assert/strict'
import test from 'node:test'
import { existsSync, readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'

const fileUrl = new URL('./temple-canvas.vue', import.meta.url)
const filePath = fileURLToPath(fileUrl)

test('temple canvas keeps the stage at full width without zoom transforms', () => {
  assert.equal(existsSync(filePath), true)

  const source = readFileSync(fileUrl, 'utf8')

  assert.match(source, /width:\s*max-content;/)
  assert.match(source, /min-width:\s*100%;/)
  assert.match(source, /width: `\$\{props\.config\.canvas_width\}px`/)
  assert.doesNotMatch(source, /zoomPercent = ref\(100\)/)
  assert.doesNotMatch(source, /transform: `scale\(/)
})
