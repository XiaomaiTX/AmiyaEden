import assert from 'node:assert/strict'
import { createRequire } from 'node:module'
import test from 'node:test'

const require = createRequire(import.meta.url)

require.extensions['.svg'] = (module, filename) => {
  module.exports = filename
}

const { buildHeroCardStyle, clampHallOfFameZoom } =
  require('./temple-canvas.helpers') as typeof import('./temple-canvas.helpers')
const gildedBorder = require('@imgs/borders/gilded.svg') as string

test('clampHallOfFameZoom keeps canvas zoom inside the supported range', () => {
  assert.equal(clampHallOfFameZoom(15), 40)
  assert.equal(clampHallOfFameZoom(97.4), 97)
  assert.equal(clampHallOfFameZoom(300), 160)
  assert.equal(clampHallOfFameZoom(Number.NaN), 100)
})

test('buildHeroCardStyle returns preset gradients and falls back to custom colors', () => {
  assert.deepEqual(
    buildHeroCardStyle({
      width: 220,
      height: 0,
      style_preset: 'gold',
      custom_bg_color: '',
      custom_text_color: '',
      custom_border_color: '',
      title_color: ''
    }),
    {
      width: '220px',
      minHeight: '148px',
      background: 'linear-gradient(180deg, #3a2f0b 0%, #1a1505 100%)',
      color: '#fff7d6',
      borderColor: '#ffd700',
      borderWidth: '2px',
      boxShadowColor: 'rgba(255, 215, 0, 0.35)',
      titleColor: '#ffe89a'
    }
  )

  assert.deepEqual(
    buildHeroCardStyle({
      width: 220,
      height: 260,
      style_preset: 'rose',
      custom_bg_color: '',
      custom_text_color: '',
      custom_border_color: '',
      title_color: '#ffd4eb'
    }),
    {
      width: '220px',
      minHeight: '260px',
      background: 'linear-gradient(180deg, #4b2238 0%, #24111e 100%)',
      color: '#ffe7f2',
      borderColor: '#ff8fc7',
      borderWidth: '2px',
      boxShadowColor: 'rgba(255, 143, 199, 0.35)',
      titleColor: '#ffd4eb'
    }
  )

  assert.deepEqual(
    buildHeroCardStyle({
      width: 220,
      height: 240,
      style_preset: 'jade',
      custom_bg_color: '',
      custom_text_color: '',
      custom_border_color: '',
      title_color: ''
    }),
    {
      width: '220px',
      minHeight: '240px',
      background: 'linear-gradient(180deg, #12332d 0%, #081a17 100%)',
      color: '#e8fff7',
      borderColor: '#53d7ad',
      borderWidth: '2px',
      boxShadowColor: 'rgba(83, 215, 173, 0.34)',
      titleColor: '#aef1d9'
    }
  )

  assert.deepEqual(
    buildHeroCardStyle({
      width: 220,
      height: 240,
      style_preset: 'midnight',
      custom_bg_color: '',
      custom_text_color: '',
      custom_border_color: '',
      title_color: ''
    }),
    {
      width: '220px',
      minHeight: '240px',
      background: 'linear-gradient(180deg, #182645 0%, #0a1123 100%)',
      color: '#edf3ff',
      borderColor: '#7ea8ff',
      borderWidth: '2px',
      boxShadowColor: 'rgba(126, 168, 255, 0.34)',
      titleColor: '#c4d7ff'
    }
  )

  assert.deepEqual(
    buildHeroCardStyle({
      width: 240,
      height: 280,
      style_preset: 'custom',
      custom_bg_color: '#101820',
      custom_text_color: '#f7f7f7',
      custom_border_color: '#4ad295',
      title_color: ''
    }),
    {
      width: '240px',
      minHeight: '280px',
      background: '#101820',
      color: '#f7f7f7',
      borderColor: '#4ad295',
      borderWidth: '2px',
      boxShadowColor: 'rgba(74, 210, 149, 0.35)',
      titleColor: '#4ad295'
    }
  )
})

test('buildHeroCardStyle returns a frame overlay src when border_style is set to gilded', () => {
  const result = buildHeroCardStyle({
    width: 220,
    height: 240,
    style_preset: 'gold',
    custom_bg_color: '',
    custom_text_color: '',
    custom_border_color: '',
    title_color: '',
    border_style: 'gilded'
  })

  assert.equal(result.frameSrc, gildedBorder)
  assert.equal(result.borderWidth, '2px')
})

test('buildHeroCardStyle omits frameSrc when border_style is none', () => {
  const result = buildHeroCardStyle({
    width: 220,
    height: 240,
    style_preset: 'gold',
    custom_bg_color: '',
    custom_text_color: '',
    custom_border_color: '',
    title_color: '',
    border_style: 'none'
  })

  assert.equal(result.frameSrc, undefined)
})

test('buildHeroCardStyle omits frameSrc when border_style is missing', () => {
  const result = buildHeroCardStyle({
    width: 220,
    height: 240,
    style_preset: 'gold',
    custom_bg_color: '',
    custom_text_color: '',
    custom_border_color: '',
    title_color: ''
  })

  assert.equal(result.frameSrc, undefined)
})
