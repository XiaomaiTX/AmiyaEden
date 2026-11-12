import assert from 'node:assert/strict'
import test from 'node:test'
import { admissionMenuVisibility } from './admissions.helpers'
import type { AdmissionContext } from '../../types/api/admissions'
import type { AppRouteRecord } from '../../types/router'

const context: AdmissionContext = {
  can_create_new_member: true,
  can_create_alt: false,
  reviewable_corporation_ids: [],
  is_member: false,
  qq: '6666666',
  discord_id: '',
  targets: [],
  characters: []
}
const menu: AppRouteRecord[] = [
  {
    path: '/admissions',
    meta: { title: 'admissions' },
    children: [
      { path: 'new', meta: { title: 'new', admission: 'new_member' } },
      { path: 'alt', meta: { title: 'alt', admission: 'alt' } },
      { path: 'history', meta: { title: 'history' } }
    ]
  },
  { path: '/other', meta: { title: 'other' } }
]

test('unchanged admission permissions preserve menu identity across refreshes', () => {
  const visible = admissionMenuVisibility(menu, context)
  assert.equal(admissionMenuVisibility(visible, { ...context }), visible)
  assert.equal(visible[1], menu[1])
  assert.equal(visible[0].children?.[2], menu[0].children?.[2])
})

test('permission loss hides gated entries and preserves history and unrelated menus', () => {
  const visible = admissionMenuVisibility(menu, context)
  const closed = admissionMenuVisibility(visible, null)
  assert.equal(closed[0].children?.[0].meta.isHide, true)
  assert.equal(closed[0].children?.[1].meta.isHide, true)
  assert.equal(closed[0].children?.[2], visible[0].children?.[2])
  assert.equal(closed[1], visible[1])
  const member = admissionMenuVisibility(closed, {
    ...context,
    is_member: true,
    can_create_new_member: false,
    can_create_alt: true
  })
  assert.equal(member[0].children?.[0].meta.isHide, true)
  assert.equal(member[0].children?.[1].meta.isHide, false)
})
