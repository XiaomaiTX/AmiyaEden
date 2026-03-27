import assert from 'node:assert/strict'
import test from 'node:test'
import { formatTime } from './time'

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
