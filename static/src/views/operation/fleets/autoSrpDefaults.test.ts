import assert from 'node:assert/strict'
import test from 'node:test'
import { resolveAutoSrpModeOnFleetConfigChange } from './autoSrpDefaults'

test('defaults create form to auto approve when a fleet config is selected', () => {
  assert.equal(
    resolveAutoSrpModeOnFleetConfigChange({
      isEditing: false,
      selectedFleetConfigId: 7,
      currentMode: 'disabled',
      userTouchedMode: false
    }),
    'auto_approve'
  )
})

test('does not overwrite an explicit create-form mode choice', () => {
  assert.equal(
    resolveAutoSrpModeOnFleetConfigChange({
      isEditing: false,
      selectedFleetConfigId: 7,
      currentMode: 'submit_only',
      userTouchedMode: true
    }),
    'submit_only'
  )
})

test('does not change existing fleet mode while editing', () => {
  assert.equal(
    resolveAutoSrpModeOnFleetConfigChange({
      isEditing: true,
      selectedFleetConfigId: 7,
      currentMode: 'disabled',
      userTouchedMode: false
    }),
    'disabled'
  )
})
