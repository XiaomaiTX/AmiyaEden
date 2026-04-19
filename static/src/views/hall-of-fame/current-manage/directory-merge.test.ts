import assert from 'node:assert/strict'
import test from 'node:test'

import { mergeSavedAdminIntoDirectory } from './directory-merge'

function createManageDirectory(): Api.FuxiAdmin.ManageDirectoryResponse {
  return {
    config: {
      id: 1,
      base_font_size: 14,
      card_width: 240,
      page_background_color: '#10243a',
      card_background_color: '#1b324c',
      card_border_color: '#d9a441',
      tier_title_color: '#f8d26b',
      name_text_color: '#fff7d6',
      body_text_color: '#d7dfef',
      created_at: '2026-04-12T00:00:00Z',
      updated_at: '2026-04-12T00:00:00Z'
    },
    tiers: [
      {
        id: 1,
        name: 'Ops',
        sort_order: 0,
        created_at: '2026-04-12T00:00:00Z',
        updated_at: '2026-04-12T00:00:00Z',
        admins: [
          {
            id: 11,
            tier_id: 1,
            nickname: 'Alpha',
            character_name: 'Commander',
            description: 'Original',
            contact_qq: '',
            contact_discord: '',
            character_id: 1001,
            welfare_delivery_offset: 2,
            fleet_led_count: 1,
            welfare_delivery_count: 4,
            created_at: '2026-04-12T00:00:00Z',
            updated_at: '2026-04-12T00:00:00Z'
          }
        ]
      },
      {
        id: 2,
        name: 'Support',
        sort_order: 1,
        created_at: '2026-04-12T00:00:00Z',
        updated_at: '2026-04-12T00:00:00Z',
        admins: []
      }
    ]
  }
}

test('mergeSavedAdminIntoDirectory replaces an existing admin in place', () => {
  const directory = createManageDirectory()
  const merged = mergeSavedAdminIntoDirectory(directory, {
    ...directory.tiers[0].admins[0],
    description: 'Updated',
    fleet_led_count: 3,
    welfare_delivery_count: 6
  })

  assert.notEqual(merged, directory)
  assert.equal(merged.tiers[0].admins[0].description, 'Updated')
  assert.equal(merged.tiers[0].admins[0].fleet_led_count, 3)
  assert.equal(merged.tiers[0].admins[0].welfare_delivery_count, 6)
  assert.equal(directory.tiers[0].admins[0].description, 'Original')
})

test('mergeSavedAdminIntoDirectory moves an updated admin into a new tier', () => {
  const directory = createManageDirectory()
  const merged = mergeSavedAdminIntoDirectory(directory, {
    ...directory.tiers[0].admins[0],
    tier_id: 2,
    nickname: 'Moved'
  })

  assert.equal(merged.tiers[0].admins.length, 0)
  assert.equal(merged.tiers[1].admins.length, 1)
  assert.equal(merged.tiers[1].admins[0].nickname, 'Moved')
})

test('mergeSavedAdminIntoDirectory appends a newly created admin to its tier', () => {
  const directory = createManageDirectory()
  const merged = mergeSavedAdminIntoDirectory(directory, {
    id: 22,
    tier_id: 2,
    nickname: 'Bravo',
    character_name: '',
    description: '',
    contact_qq: '',
    contact_discord: '',
    character_id: 2002,
    welfare_delivery_offset: 0,
    fleet_led_count: 0,
    welfare_delivery_count: 0,
    created_at: '2026-04-12T00:00:00Z',
    updated_at: '2026-04-12T00:00:00Z'
  })

  assert.equal(merged.tiers[1].admins.length, 1)
  assert.equal(merged.tiers[1].admins[0].id, 22)
})
