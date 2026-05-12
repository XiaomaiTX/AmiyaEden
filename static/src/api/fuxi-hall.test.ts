import assert from 'node:assert/strict'
import test from 'node:test'
import { readFileSync } from 'node:fs'

const source = readFileSync(new URL('./fuxi-hall.ts', import.meta.url), 'utf8')

test('fuxi hall api maps public pages to new login endpoints', () => {
  assert.match(source, /\/api\/v1\/fuxi-hall\/leadership/)
  assert.match(source, /\/api\/v1\/fuxi-hall\/contributors/)
})

test('fuxi hall manage api maps admin routes and reorder endpoint', () => {
  assert.match(source, /\/api\/v1\/system\/fuxi-hall\/pages\/\$\{pageKey\}/)
  assert.match(source, /\/api\/v1\/system\/fuxi-hall\/cards\/\$\{pageKey\}/)
  assert.match(source, /\/api\/v1\/system\/fuxi-hall\/cards\/reorder/)
})
