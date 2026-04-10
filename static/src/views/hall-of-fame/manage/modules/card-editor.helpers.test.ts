import assert from 'node:assert/strict'
import test from 'node:test'

import { mergePendingCardUpdates } from './card-editor.helpers'

test('mergePendingCardUpdates keeps earlier field edits when a later field change arrives in the same save window', () => {
  assert.deepEqual(
    mergePendingCardUpdates(
      { name: 'Hero Alpha', description: 'First draft' },
      { title: 'Founder', description: 'Final draft' }
    ),
    {
      name: 'Hero Alpha',
      title: 'Founder',
      description: 'Final draft'
    }
  )
})
