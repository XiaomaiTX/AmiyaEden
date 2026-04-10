import assert from 'node:assert/strict'
import test from 'node:test'
import { existsSync, readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'

const fileUrl = new URL('./draggable-card.vue', import.meta.url)
const filePath = fileURLToPath(fileUrl)

test('draggable card exists and uses interactjs around the shared hero card', () => {
  assert.equal(existsSync(filePath), true)

  const source = readFileSync(fileUrl, 'utf8')

  assert.match(source, /import interact from 'interactjs'/)
  assert.match(source, /import HeroCard from '\.\.\/\.\.\/temple\/modules\/hero-card\.vue'/)
  assert.match(source, /interact\(/)
  assert.match(source, /draggable\(/)
  assert.match(source, /resizable\(/)
})
