import assert from 'node:assert/strict'
import test from 'node:test'

import {
  buildNewCardPayload,
  clampCardCoordinate,
  patchCardById,
  toLayoutUpdates
} from './manage-canvas.helpers'

test('buildNewCardPayload creates a centered visible card with sensible defaults', () => {
  assert.deepEqual(buildNewCardPayload('Hero Candidate', 7), {
    name: 'Hero Candidate',
    pos_x: 50,
    pos_y: 50,
    width: 220,
    style_preset: 'gold',
    z_index: 8,
    visible: true
  })
})

test('clampCardCoordinate keeps drag coordinates inside the percentage canvas range', () => {
  assert.equal(clampCardCoordinate(-8), 0)
  assert.equal(clampCardCoordinate(36.5), 36.5)
  assert.equal(clampCardCoordinate(120), 100)
})

test('toLayoutUpdates extracts only id, coordinates, and z-index from cards', () => {
  assert.deepEqual(
    toLayoutUpdates([
      {
        id: 1,
        pos_x: 22,
        pos_y: 33,
        width: 220,
        height: 280,
        z_index: 4
      },
      {
        id: 2,
        pos_x: 65,
        pos_y: 75,
        width: 260,
        height: 0,
        z_index: 5
      }
    ]),
    [
      { id: 1, pos_x: 22, pos_y: 33, width: 220, height: 280, z_index: 4 },
      { id: 2, pos_x: 65, pos_y: 75, width: 260, height: 0, z_index: 5 }
    ]
  )
})

test('patchCardById merges changed fields without clobbering unrelated local state', () => {
  const cards: Api.HallOfFame.Card[] = [
    {
      id: 1,
      name: 'Hero Alpha',
      title: 'Strategist',
      description: 'Keeps the fleet together.',
      avatar: 'old-avatar',
      pos_x: 10,
      pos_y: 20,
      width: 220,
      height: 280,
      style_preset: 'gold',
      custom_bg_color: '',
      custom_text_color: '',
      custom_border_color: '',
      font_size: 14,
      z_index: 3,
      visible: true,
      created_at: '',
      updated_at: ''
    }
  ]

  const afterTitle = patchCardById(cards, 1, { title: 'Founder' })
  const afterAvatar = patchCardById(afterTitle, 1, { avatar: 'new-avatar' })

  assert.deepEqual(afterAvatar[0], {
    id: 1,
    name: 'Hero Alpha',
    title: 'Founder',
    description: 'Keeps the fleet together.',
    avatar: 'new-avatar',
    pos_x: 10,
    pos_y: 20,
    width: 220,
    height: 280,
    style_preset: 'gold',
    custom_bg_color: '',
    custom_text_color: '',
    custom_border_color: '',
    font_size: 14,
    z_index: 3,
    visible: true,
    created_at: '',
    updated_at: ''
  })
})
