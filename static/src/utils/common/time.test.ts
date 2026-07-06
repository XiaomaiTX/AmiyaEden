import assert from 'node:assert/strict'
import test from 'node:test'
import { formatTime, toLocalOffsetISO } from './time'

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
