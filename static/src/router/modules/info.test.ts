import assert from 'node:assert/strict'
import test from 'node:test'
import { readFileSync } from 'node:fs'

const source = readFileSync(new URL('./info.ts', import.meta.url), 'utf8')

test('info route tree includes tool bookmarks page for login users', () => {
  const block = source.slice(source.indexOf("path: 'tool-bookmarks'"), source.length)
  assert.match(block, /name:\s*'EveInfoToolBookmarks'/)
  assert.match(block, /component:\s*'\/info\/tool-bookmarks'/)
  assert.match(block, /title:\s*'menus\.info\.toolBookmarks'/)
  assert.match(block, /login:\s*true/)
})
