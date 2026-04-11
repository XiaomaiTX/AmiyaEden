import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

const source = readFileSync(new URL('./ArtUserMenu.vue', import.meta.url), 'utf8')

test('ArtUserMenu shows localized role labels instead of raw role codes', () => {
  assert.doesNotMatch(source, /userInfo\.roles\?\.\[0\]\s*\|\|\s*''/)
  assert.match(source, /getRoleNames/)
})

test('ArtUserMenu allows the localized role list to wrap instead of truncating', () => {
  assert.doesNotMatch(source, /localizedRoleNames\s*}}<\/span>/)
  assert.match(source, /localizedRoleNames/)
  assert.doesNotMatch(source, /text-xs text-g-500 truncate/)
})
