import assert from 'node:assert/strict'
import test from 'node:test'
import type { AppRouteRecord } from '@/types/router'
import { RoutePermissionValidator } from './RoutePermissionValidator'

const sampleMenu: AppRouteRecord[] = [
  {
    name: 'Characters',
    path: '/characters',
    meta: { title: 'menus.system.userCenter', isHide: true }
  },
  {
    name: 'MumbleCredentials',
    path: '/mumble',
    meta: { title: 'menus.mumble.title' }
  },
  {
    name: 'Admissions',
    path: '/admissions',
    meta: { title: 'menus.admissions.title' },
    children: [
      {
        name: 'AdmissionsNew',
        path: '/admissions/new',
        meta: { title: 'menus.admissions.new', isHide: true }
      }
    ]
  }
]

test('RoutePermissionValidator allows hidden /characters route for reload and gate redirects', () => {
  assert.deepEqual(RoutePermissionValidator.validatePath('/characters', sampleMenu, '/mumble'), {
    path: '/characters',
    hasPermission: true
  })
})

test('RoutePermissionValidator still rejects hidden non-characters routes without visible parent', () => {
  const hiddenOnlyMenu: AppRouteRecord[] = [
    {
      name: 'Secret',
      path: '/secret',
      meta: { title: 'secret', isHide: true }
    }
  ]

  assert.deepEqual(RoutePermissionValidator.validatePath('/secret', hiddenOnlyMenu, '/mumble'), {
    path: '/mumble',
    hasPermission: false
  })
})
