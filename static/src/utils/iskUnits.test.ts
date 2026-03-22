import assert from 'node:assert/strict'
import test from 'node:test'
import { fromMillionISKInput, toMillionISKInput } from './iskUnits'

test('toMillionISKInput converts raw isk to million-based editor values', () => {
  assert.equal(toMillionISKInput(0), 0)
  assert.equal(toMillionISKInput(14_500_000), 14.5)
  assert.equal(toMillionISKInput(125_000_000), 125)
})

test('fromMillionISKInput converts million-based editor values back to raw isk', () => {
  assert.equal(fromMillionISKInput(0), 0)
  assert.equal(fromMillionISKInput(14.5), 14_500_000)
  assert.equal(fromMillionISKInput(125), 125_000_000)
})
