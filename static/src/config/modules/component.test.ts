import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

const source = readFileSync(new URL('./component.ts', import.meta.url), 'utf8')

test('global components include enabled browser notification host', () => {
  assert.match(source, /key:\s*'browser-notifications'/)
  assert.match(source, /art-browser-notifications\/index\.vue/)
  assert.match(
    source,
    /key:\s*'browser-notifications'[\s\S]*?art-browser-notifications\/index\.vue[\s\S]*?enabled:\s*true/
  )
})
