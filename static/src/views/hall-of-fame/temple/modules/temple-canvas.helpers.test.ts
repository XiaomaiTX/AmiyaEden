import assert from 'node:assert/strict'
import test from 'node:test'

import { buildHeroCardStyle, getTempleScale } from './temple-canvas.helpers'

test('getTempleScale clamps scale to 1 and keeps wrapper height aligned to scaled canvas', () => {
  assert.deepEqual(getTempleScale(960, 1920, 1080), {
    ratio: 0.5,
    wrapperHeight: 540
  })

  assert.deepEqual(getTempleScale(2400, 1920, 1080), {
    ratio: 1,
    wrapperHeight: 1080
  })
})

test('buildHeroCardStyle returns preset gradients and falls back to custom colors', () => {
  assert.deepEqual(
    buildHeroCardStyle({
      width: 220,
      height: 0,
      style_preset: 'gold',
      custom_bg_color: '',
      custom_text_color: '',
      custom_border_color: ''
    }),
    {
      width: '220px',
      minHeight: '220px',
      background: 'linear-gradient(180deg, #3a2f0b 0%, #1a1505 100%)',
      color: '#fff7d6',
      borderColor: '#ffd700',
      boxShadowColor: 'rgba(255, 215, 0, 0.35)'
    }
  )

  assert.deepEqual(
    buildHeroCardStyle({
      width: 240,
      height: 280,
      style_preset: 'custom',
      custom_bg_color: '#101820',
      custom_text_color: '#f7f7f7',
      custom_border_color: '#4ad295'
    }),
    {
      width: '240px',
      minHeight: '280px',
      background: '#101820',
      color: '#f7f7f7',
      borderColor: '#4ad295',
      boxShadowColor: 'rgba(74, 210, 149, 0.35)'
    }
  )
})
