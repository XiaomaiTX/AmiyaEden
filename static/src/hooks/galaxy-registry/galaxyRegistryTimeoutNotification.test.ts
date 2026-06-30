import assert from 'node:assert/strict'
import test from 'node:test'

import {
  canMonitorGalaxyRegistryTimeout,
  findGalaxyRegistryTimeoutCandidates,
  hasGalaxyRegistryTimeoutNotified,
  markGalaxyRegistryTimeoutNotified,
  type GalaxyRegistryTimeoutSystemItem,
  type NotificationStorage
} from './galaxyRegistryTimeoutNotification'

const nowMs = new Date('2026-06-30T12:00:00Z').getTime()

test('findGalaxyRegistryTimeoutCandidates returns only mine overdue entries', () => {
  const systems: GalaxyRegistryTimeoutSystemItem[] = [
    {
      solar_system_name: 'A-1',
      active_entry: {
        entry_id: 1,
        actual_start_at: '2026-06-30T09:59:59Z',
        is_mine: true,
        is_overdue: false
      }
    },
    {
      solar_system_name: 'A-2',
      active_entry: {
        entry_id: 2,
        actual_start_at: '2026-06-30T10:30:00Z',
        is_mine: true,
        is_overdue: false
      }
    },
    {
      solar_system_name: 'A-3',
      active_entry: {
        entry_id: 3,
        actual_start_at: '2026-06-30T09:00:00Z',
        is_mine: false,
        is_overdue: true
      }
    },
    {
      solar_system_name: 'A-4',
      active_entry: null
    }
  ]

  assert.deepEqual(findGalaxyRegistryTimeoutCandidates(systems, nowMs), [
    {
      entryId: 1,
      systemName: 'A-1',
      actualStartAt: '2026-06-30T09:59:59Z'
    }
  ])
})

test('findGalaxyRegistryTimeoutCandidates accepts backend overdue signal', () => {
  const systems: GalaxyRegistryTimeoutSystemItem[] = [
    {
      solar_system_name: 'B-1',
      active_entry: {
        entry_id: 5,
        actual_start_at: 'not-a-date',
        is_mine: true,
        is_overdue: true
      }
    }
  ]

  assert.deepEqual(findGalaxyRegistryTimeoutCandidates(systems, nowMs), [
    {
      entryId: 5,
      systemName: 'B-1',
      actualStartAt: 'not-a-date'
    }
  ])
})

test('galaxy registry timeout notified storage deduplicates by entry id', () => {
  const values = new Map<string, string>()
  const storage: NotificationStorage = {
    getItem: (key) => values.get(key) ?? null,
    setItem: (key, value) => values.set(key, value)
  }

  assert.equal(hasGalaxyRegistryTimeoutNotified(storage, 42), false)
  markGalaxyRegistryTimeoutNotified(storage, 42)
  assert.equal(hasGalaxyRegistryTimeoutNotified(storage, 42), true)
  assert.equal(hasGalaxyRegistryTimeoutNotified(storage, 43), false)
})

test('canMonitorGalaxyRegistryTimeout follows galaxy registry roles', () => {
  assert.equal(canMonitorGalaxyRegistryTimeout(['captain']), true)
  assert.equal(canMonitorGalaxyRegistryTimeout(['admin']), true)
  assert.equal(canMonitorGalaxyRegistryTimeout(['super_admin']), true)
  assert.equal(canMonitorGalaxyRegistryTimeout(['user']), false)
})
