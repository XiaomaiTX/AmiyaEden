import assert from 'node:assert/strict'
import test from 'node:test'
import { existsSync, readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'

const fileUrl = new URL('./card-editor.vue', import.meta.url)
const filePath = fileURLToPath(fileUrl)

test('card editor uses a plain character id input and exposes badge-image controls', () => {
  assert.equal(existsSync(filePath), true)

  const source = readFileSync(fileUrl, 'utf8')

  assert.match(source, /hallOfFame\.manage\.fontSize/)
  assert.match(source, /hallOfFame\.manage\.characterId/)
  assert.match(source, /hallOfFame\.manage\.titleColor/)
  assert.match(source, /hallOfFame\.manage\.badgeImage/)
  assert.match(source, /upload-badge-image/)
  assert.match(source, /handleBadgeImageChange/)
  assert.match(source, /inputmode="numeric"/)
  assert.doesNotMatch(source, /ElInputNumber/)
  assert.match(source, /@update:model-value="\(value\) => handleFontSizeChange\(value\)"/)
  assert.match(source, /function handleFontSizeChange\(value: number \| number\[\]\)/)
  assert.doesNotMatch(source, /hallOfFame\.manage\.zIndex/)
  assert.doesNotMatch(source, /hallOfFame\.manage\.visible/)
  assert.doesNotMatch(source, /ElSwitch/)
  assert.doesNotMatch(source, /upload-avatar/)
  assert.doesNotMatch(source, /changeAvatar/)
})

test('card editor exposes the new rose, jade, and midnight style preset options', () => {
  const source = readFileSync(fileUrl, 'utf8')

  assert.match(source, /ElOption value="rose" :label="t\('hallOfFame\.manage\.rose'\)"/)
  assert.match(source, /ElOption value="jade" :label="t\('hallOfFame\.manage\.jade'\)"/)
  assert.match(source, /ElOption value="midnight" :label="t\('hallOfFame\.manage\.midnight'\)"/)
  assert.doesNotMatch(source, /ElOption value="rainbow"/)
})

test('card editor exposes a localized border style selector with none plus eight frame options', () => {
  const source = readFileSync(fileUrl, 'utf8')

  assert.match(source, /hallOfFame\.manage\.stylePreset[\s\S]*hallOfFame\.manage\.borderStyle/)
  assert.match(source, /:model-value="card\.border_style \|\| 'none'"/)
  assert.match(source, /@update:model-value="handleBorderStyleChange"/)
  assert.match(
    source,
    /function handleBorderStyleChange\(value: Api\.HallOfFame\.CardBorderStyle\)[\s\S]*queueUpdate\(\{ border_style: value \}\)/
  )
  assert.match(source, /ElOption value="none" :label="t\('hallOfFame\.manage\.borderNone'\)"/)
  assert.match(source, /ElOption value="gilded" :label="t\('hallOfFame\.manage\.borderGilded'\)"/)
  assert.match(
    source,
    /ElOption value="imperial" :label="t\('hallOfFame\.manage\.borderImperial'\)"/
  )
  assert.match(
    source,
    /ElOption value="neon-circuit" :label="t\('hallOfFame\.manage\.borderNeonCircuit'\)"/
  )
  assert.match(
    source,
    /ElOption value="void-rift" :label="t\('hallOfFame\.manage\.borderVoidRift'\)"/
  )
  assert.match(source, /ElOption value="amarr" :label="t\('hallOfFame\.manage\.borderAmarr'\)"/)
  assert.match(source, /ElOption value="caldari" :label="t\('hallOfFame\.manage\.borderCaldari'\)"/)
  assert.match(
    source,
    /ElOption value="minmatar" :label="t\('hallOfFame\.manage\.borderMinmatar'\)"/
  )
  assert.match(
    source,
    /ElOption value="gallente" :label="t\('hallOfFame\.manage\.borderGallente'\)"/
  )
})
