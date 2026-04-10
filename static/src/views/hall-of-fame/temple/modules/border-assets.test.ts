import assert from 'node:assert/strict'
import test from 'node:test'
import { readFileSync } from 'node:fs'

const borderAssetPaths = [
  new URL('../../../../assets/images/borders/amarr.svg', import.meta.url),
  new URL('../../../../assets/images/borders/caldari.svg', import.meta.url),
  new URL('../../../../assets/images/borders/gallente.svg', import.meta.url),
  new URL('../../../../assets/images/borders/gilded.svg', import.meta.url),
  new URL('../../../../assets/images/borders/imperial.svg', import.meta.url),
  new URL('../../../../assets/images/borders/minmatar.svg', import.meta.url),
  new URL('../../../../assets/images/borders/neon-circuit.svg', import.meta.url),
  new URL('../../../../assets/images/borders/void-rift.svg', import.meta.url)
]

test('hall of fame border SVG assets opt into non-uniform stretching', () => {
  for (const assetPath of borderAssetPaths) {
    const source = readFileSync(assetPath, 'utf8')
    assert.match(source, /preserveAspectRatio="none"/)
  }
})
