import { canAccessAdmission, compressAdmissionImage } from './admissions'
import type { AdmissionContext } from '@/types/api/admissions'
test('eligibility is fail closed and follows refreshed user context', () => {
  expect(canAccessAdmission(null, 'new_member')).toBe(false)
  expect(canAccessAdmission(null, 'alt')).toBe(false)
  expect(canAccessAdmission(null, 'review')).toBe(false)
  expect(canAccessAdmission(null, 'history')).toBe(true)
  const context: AdmissionContext = {
    can_create_new_member: true,
    can_create_alt: false,
    reviewable_corporation_ids: [],
    is_member: false,
    qq: '',
    discord_id: 'discord',
    targets: [],
    characters: [],
  }
  expect(canAccessAdmission(context, 'new_member')).toBe(true)
  expect(canAccessAdmission(context, 'alt')).toBe(false)
  expect(
    canAccessAdmission({ ...context, can_create_new_member: false, can_create_alt: true }, 'alt')
  ).toBe(true)
})
test('compression rejects unsupported files before decoding', async () => {
  await expect(
    compressAdmissionImage(new File(['bad'], 'bad.svg', { type: 'image/svg+xml' }))
  ).rejects.toThrow('admissions.errors.evidence')
})
