import assert from 'node:assert/strict'
import test from 'node:test'
import { getNewbroIneligibilityReasonKey } from './newbroEligibility'

test('getNewbroIneligibilityReasonKey maps the skill threshold reason', () => {
  assert.equal(
    getNewbroIneligibilityReasonKey('skill_point_threshold_reached'),
    'newbro.select.ineligibleBecauseSkillThreshold'
  )
})

test('getNewbroIneligibilityReasonKey maps the multi-character reason', () => {
  assert.equal(
    getNewbroIneligibilityReasonKey('multi_character_skill_point_threshold_reached'),
    'newbro.select.ineligibleBecauseMultiCharacterThreshold'
  )
})

test('getNewbroIneligibilityReasonKey falls back for unknown reasons', () => {
  assert.equal(
    getNewbroIneligibilityReasonKey('something_else'),
    'newbro.select.currentlyIneligible'
  )
  assert.equal(getNewbroIneligibilityReasonKey(''), 'newbro.select.currentlyIneligible')
})
