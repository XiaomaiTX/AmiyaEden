import assert from 'node:assert/strict'
import test from 'node:test'
import { formatTime, fuelExpiryMonthOffset, toLocalOffsetISO } from './time'

test('formatTime returns a localized string for valid timestamps', () => {
  const originalToLocaleString = Date.prototype.toLocaleString
  Date.prototype.toLocaleString = function () {
    return 'localized time'
  }

  try {
    assert.equal(formatTime('2026-03-28T00:00:00.000Z'), 'localized time')
  } finally {
    Date.prototype.toLocaleString = originalToLocaleString
  }
})

test('formatTime returns a dash for empty timestamps', () => {
  assert.equal(formatTime(''), '-')
})

test('toLocalOffsetISO produces an RFC3339 string with the browser offset', () => {
  // Lock the timezone offset to +08:00 so the assertion is deterministic.
  const originalGetTimezoneOffset = Date.prototype.getTimezoneOffset
  Date.prototype.getTimezoneOffset = function () {
    return -480
  }

  try {
    const iso = toLocalOffsetISO('2026-07-06 14:00:00')
    assert.equal(iso, '2026-07-06T14:00:00+08:00')
  } finally {
    Date.prototype.getTimezoneOffset = originalGetTimezoneOffset
  }
})

test('fuelExpiryMonthOffset computes whole-month difference in UTC', () => {
  const now = new Date('2026-08-15T12:00:00Z')
  assert.equal(fuelExpiryMonthOffset('2026-08-20T00:00:00Z', now), 0)
  assert.equal(fuelExpiryMonthOffset('2026-09-01T00:00:00Z', now), 1)
  assert.equal(fuelExpiryMonthOffset('2026-11-15T00:00:00Z', now), 3)
})

test('fuelExpiryMonthOffset rolls over across the year boundary', () => {
  const now = new Date('2026-12-31T23:00:00Z')
  assert.equal(fuelExpiryMonthOffset('2027-01-01T00:00:00Z', now), 1)
})

test('fuelExpiryMonthOffset returns null for invalid input', () => {
  assert.equal(fuelExpiryMonthOffset('not-a-date', new Date('2026-08-15T00:00:00Z')), null)
})

test('toLocalOffsetISO handles a negative (west of UTC) offset', () => {
  const originalGetTimezoneOffset = Date.prototype.getTimezoneOffset
  Date.prototype.getTimezoneOffset = function () {
    return 300 // UTC-05:00
  }

  try {
    const iso = toLocalOffsetISO('2026-07-06 09:30:15')
    assert.equal(iso, '2026-07-06T09:30:15-05:00')
  } finally {
    Date.prototype.getTimezoneOffset = originalGetTimezoneOffset
  }
})
