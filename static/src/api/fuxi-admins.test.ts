import assert from 'node:assert/strict'
import test from 'node:test'
import { readFileSync } from 'node:fs'

const source = readFileSync(new URL('./fuxi-admins.ts', import.meta.url), 'utf8')

test('directory fetch suppresses shared HTTP error toasts because the page owns load failure UI', () => {
  const directoryFetchBlock = source.slice(
    source.indexOf('export function fetchFuxiAdminDirectory()'),
    source.indexOf('// ─── Admin: Config ───')
  )

  assert.match(directoryFetchBlock, /showErrorMessage:\s*false/)
})
