import assert from 'node:assert/strict'
import test from 'node:test'
import { buildFleetSearchForm } from './fleet-search-form'

test('buildFleetSearchForm uses the selected importance value', () => {
  assert.deepEqual(buildFleetSearchForm({ importance: 'cta' }, 'strat_op'), {
    importance: 'strat_op'
  })
})

test('buildFleetSearchForm keeps a cleared importance as undefined', () => {
  assert.deepEqual(buildFleetSearchForm({ importance: 'cta' }, undefined), {
    importance: undefined
  })
})
