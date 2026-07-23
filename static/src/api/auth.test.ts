import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

const source = readFileSync(new URL('./auth.ts', import.meta.url), 'utf8')

test('mapLoginResult exposes corp capabilities from /me payload', () => {
  assert.match(source, /corpCapabilities:\s*Array\.isArray\(data\.corp_capabilities\)/)
})

test('mapLoginResult coerces missing corp capabilities to empty array', () => {
  assert.match(source, /\?\s*data\.corp_capabilities\s*: \[\]/)
})
