import assert from 'node:assert/strict'
import test from 'node:test'
import { readFileSync } from 'node:fs'

const source = readFileSync(new URL('./index.vue', import.meta.url), 'utf8')

test('esi-check page fetches scopes and characters on mount', () => {
  assert.match(source, /fetchEveSSOScopes/)
  assert.match(source, /fetchMyCharacters/)
  assert.match(source, /Promise\.all/)
})

test('esi-check page passes data to OverviewMatrix and CharacterDetail', () => {
  assert.match(source, /:scopes="scopes"/)
  assert.match(source, /:characters="characters"/)
  assert.match(source, /OverviewMatrix/)
  assert.match(source, /CharacterDetail/)
})

test('esi-check page handles select-character event', () => {
  assert.match(source, /@select-character="onSelectCharacter"/)
  assert.match(source, /const onSelectCharacter/)
  assert.match(source, /selectedCharacterId\.value/)
})
