import assert from 'node:assert/strict'
import test from 'node:test'
import { canAccessAdmission, compressAdmissionImage } from './admissions'
import type { AdmissionContext } from '../types/api/admissions'

test('creation and review fail closed while owner history remains available', () => {
  for (const kind of ['new_member', 'alt', 'review'] as const)
    assert.equal(canAccessAdmission(null, kind), false)
  assert.equal(canAccessAdmission(null, 'history'), true)
  const context: AdmissionContext = {
    can_create_new_member: true,
    can_create_alt: false,
    reviewable_corporation_ids: [],
    is_member: false,
    qq: '',
    discord_id: 'discord',
    targets: [],
    characters: []
  }
  assert.equal(canAccessAdmission(context, 'new_member'), true)
  assert.equal(canAccessAdmission(context, 'alt'), false)
  assert.equal(canAccessAdmission(context, 'review'), false)
  assert.equal(
    canAccessAdmission(
      {
        ...context,
        can_create_new_member: false,
        can_create_alt: true,
        reviewable_corporation_ids: [98824507]
      },
      'review'
    ),
    true
  )
})
test('evidence rejects unsupported images before attempting browser decode', async () => {
  await assert.rejects(
    compressAdmissionImage(new File(['bad'], 'bad.svg', { type: 'image/svg+xml' })),
    /admissions.errors.evidence/
  )
})
