import assert from 'node:assert/strict'
import test from 'node:test'
import { existsSync, readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'

const fileUrl = new URL('./card-editor.vue', import.meta.url)
const filePath = fileURLToPath(fileUrl)

test('card editor routes z-index changes through the layout save path instead of immediate card updates', () => {
  assert.equal(existsSync(filePath), true)

  const source = readFileSync(fileUrl, 'utf8')

  assert.match(source, /'update-z-index': \[id: number, value: number\]/)
  assert.match(source, /emit\('update-z-index', props\.card\.id, value\)/)
  assert.doesNotMatch(source, /queueUpdate\(\{ z_index: value \}\)/)
})
