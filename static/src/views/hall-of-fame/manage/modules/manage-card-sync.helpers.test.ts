import assert from 'node:assert/strict'
import test from 'node:test'

import {
  getMissingCardIdFromError,
  queueCardUpdateRequest,
  rebuildCardFromConfirmedState,
  settleCardUpdateRequest
} from './manage-card-sync.helpers'

test('getMissingCardIdFromError returns the card id from stale-card failures', () => {
  assert.equal(getMissingCardIdFromError(new Error('卡片 42 不存在')), 42)
  assert.equal(getMissingCardIdFromError(new Error('网络错误')), null)
})

test('serialized card updates keep newer queued fields when an earlier request settles', () => {
  const confirmedCard: Api.HallOfFame.Card = {
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

  const first = queueCardUpdateRequest({ active: null, queued: null }, { title: 'Founder' })

  assert.deepEqual(first.patchToSend, { title: 'Founder' })

  const second = queueCardUpdateRequest(first.state, {
    avatar: 'new-avatar',
    description: 'Final draft'
  })

  assert.equal(second.patchToSend, null)
  assert.deepEqual(second.state.queued, {
    avatar: 'new-avatar',
    description: 'Final draft'
  })

  const settled = settleCardUpdateRequest(second.state)

  assert.deepEqual(settled.patchToSend, {
    avatar: 'new-avatar',
    description: 'Final draft'
  })
  assert.deepEqual(
    rebuildCardFromConfirmedState(
      confirmedCard,
      {
        ...confirmedCard,
        title: 'Founder',
        avatar: 'new-avatar',
        description: 'Final draft',
        width: 260
      },
      settled.state.active
    ),
    {
      ...confirmedCard,
      avatar: 'new-avatar',
      description: 'Final draft',
      width: 260
    }
  )
})
