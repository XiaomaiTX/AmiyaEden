import assert from 'node:assert/strict'
import { test } from 'node:test'
import type { FleetConfigFittingItem } from '@/api/generated/client'
import { toItemSettingsEdit, toItemSettingsUpdate } from './item-settings-edit'

test('fitting settings preserve stored values without offering them as selectable options', () => {
  const item: FleetConfigFittingItem = {
    id: 42,
    type_id: 12,
    type_name: 'Module',
    quantity: 1,
    flag: 'LowSlot0',
    flag_group: 'Low',
    importance: 'replaceable',
    penalty: 'legacy',
    replacement_penalty: 'none',
    replacements: [{ id: 19, type_id: 33, type_name: 'Replacement' }]
  }

  const edit = toItemSettingsEdit(item)
  assert.deepEqual(edit, {
    importance: 'replaceable',
    penalty: undefined,
    replacementPenalty: 'none',
    replacements: [33]
  })
  assert.deepEqual(toItemSettingsUpdate(item, edit), {
    id: 42,
    importance: 'replaceable',
    penalty: 'legacy',
    replacement_penalty: 'none',
    replacements: [33]
  })
  assert.equal(toItemSettingsUpdate(item, { ...edit, penalty: 'half' }).penalty, 'half')
})
