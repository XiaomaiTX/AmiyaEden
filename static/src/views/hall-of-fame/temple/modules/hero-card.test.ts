import assert from 'node:assert/strict'
import test from 'node:test'
import { existsSync, readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'

const fileUrl = new URL('./hero-card.vue', import.meta.url)
const filePath = fileURLToPath(fileUrl)

test('hero card description is no longer hard-clamped to two lines', () => {
  assert.equal(existsSync(filePath), true)

  const source = readFileSync(fileUrl, 'utf8')

  assert.doesNotMatch(source, /-webkit-line-clamp:\s*2/)
  assert.match(source, /white-space:\s*pre-wrap/)
  assert.match(source, /overflow:\s*auto/)
  assert.match(source, /grid-template-columns:\s*72px minmax\(0, 1fr\)/)
  assert.match(source, /--hero-title-color/)
  assert.match(source, /card\.badge_image/)
})

test('hero card renders a decorative frame image that stretches with the card box', () => {
  const source = readFileSync(fileUrl, 'utf8')

  assert.match(
    source,
    /<img[\s\S]*v-if="rawStyle\.frameSrc"[\s\S]*:src="rawStyle\.frameSrc"[\s\S]*class="hero-card__frame"/
  )
  assert.match(source, /object-fit:\s*fill/)
  assert.match(
    source,
    /borderColor:\s*rawStyle\.value\.frameSrc\s*\?\s*'transparent'\s*:\s*rawStyle\.value\.borderColor/
  )
  assert.match(
    source,
    /backgroundClip:\s*rawStyle\.value\.frameSrc\s*\?\s*'border-box'\s*:\s*'padding-box'/
  )
})
