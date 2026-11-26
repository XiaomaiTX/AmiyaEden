import type { ToolBookmark } from '@/api/generated/client'
import assert from 'node:assert/strict'
import { test } from 'node:test'
import { groupToolBookmarks } from './tool-bookmark-groups'

test('groups sites in API order, retaining ungrouped sites and relative order', () => {
  const site = (id: number, group_name: string): ToolBookmark => ({
    id,
    group_name,
    name: String(id),
    url: 'https://example.com',
    description: '',
    logo_url: '',
    logo_source: '',
    is_enabled: true,
    sort_order: id,
    created_by: 1,
    created_at: '',
    updated_at: ''
  })
  const groups = groupToolBookmarks([
    site(1, 'Tools'),
    site(2, 'Resources'),
    site(3, 'Tools'),
    site(4, '')
  ])
  assert.deepEqual(
    groups.map((group) => [group.name, group.items.map((item) => item.id)]),
    [
      ['Tools', [1, 3]],
      ['Resources', [2]],
      ['', [4]]
    ]
  )
  assert.deepEqual(groupToolBookmarks([]), [])
})
