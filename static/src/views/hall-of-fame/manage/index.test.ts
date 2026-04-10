import assert from 'node:assert/strict'
import test from 'node:test'
import { existsSync, readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'

const fileUrl = new URL('./index.vue', import.meta.url)
const filePath = fileURLToPath(fileUrl)

test('hall of fame manage page exists and wires admin APIs into the editor layout', () => {
  assert.equal(existsSync(filePath), true)

  const source = readFileSync(fileUrl, 'utf8')

  assert.match(source, /fetchHofConfig/)
  assert.match(source, /fetchHofCards/)
  assert.match(source, /createHofCard/)
  assert.match(source, /batchUpdateHofLayout/)
  assert.match(source, /uploadHofBackground/)
  assert.match(source, /import CanvasToolbar from '\.\/modules\/canvas-toolbar\.vue'/)
  assert.match(source, /import ManageCanvas from '\.\/modules\/manage-canvas\.vue'/)
  assert.match(source, /import CardEditor from '\.\/modules\/card-editor\.vue'/)
  assert.match(source, /selectedCardId/)
  assert.match(source, /dirty/)
})

test('hall of fame manage page treats z-index changes as dirty layout updates', () => {
  const source = readFileSync(fileUrl, 'utf8')

  assert.match(source, /@update-z-index="handleCardZIndexUpdate"/)
  assert.match(source, /function handleCardZIndexUpdate\(id: number, zIndex: number\)/)
  assert.match(source, /patchCardById\(cards\.value, id, \{\s*z_index: zIndex/s)
  assert.match(source, /dirty\.value = true/)
})

test('hall of fame manage page reconciles stale-card errors by removing the missing card locally', () => {
  const source = readFileSync(fileUrl, 'utf8')

  assert.match(source, /getMissingCardIdFromError\(error\)/)
  assert.match(source, /removeCardLocally\(staleCardId\)/)
})
