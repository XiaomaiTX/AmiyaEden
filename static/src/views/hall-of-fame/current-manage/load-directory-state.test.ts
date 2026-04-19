import assert from 'node:assert/strict'
import test from 'node:test'

import { ApiStatus } from '@/utils/http/status'

import {
  loadFuxiAdminDirectoryState,
  loadFuxiAdminPageDirectory,
  resolveManageAccess
} from './load-directory-state'

function createDirectory(): Api.FuxiAdmin.DirectoryResponse {
  return {
    config: {
      id: 1,
      base_font_size: 14,
      card_width: 240,
      page_background_color: '#10243a',
      card_background_color: '#1b324c',
      card_border_color: '#d9a441',
      tier_title_color: '#f8d26b',
      name_text_color: '#fff7d6',
      body_text_color: '#d7dfef',
      created_at: '2026-04-12T00:00:00Z',
      updated_at: '2026-04-12T00:00:00Z'
    },
    tiers: []
  }
}

function createManageDirectory(): Api.FuxiAdmin.ManageDirectoryResponse {
  return {
    ...createDirectory(),
    tiers: []
  }
}

test('loadFuxiAdminDirectoryState returns the directory on success', async () => {
  const directory = createDirectory()

  const result = await loadFuxiAdminDirectoryState(async () => directory, 'Failed to load')

  assert.equal(result.directory, directory)
  assert.equal(result.loadErrorMessage, null)
  assert.equal(result.showErrorToast, false)
  assert.equal(result.isAuthDenied, false)
})

test('loadFuxiAdminDirectoryState returns the localized fallback message on failure', async () => {
  const result = await loadFuxiAdminDirectoryState(async () => {
    throw new Error('database is closed')
  }, 'Failed to load the current Fuxi admin directory')

  assert.equal(result.directory, null)
  assert.equal(result.loadErrorMessage, 'Failed to load the current Fuxi admin directory')
  assert.equal(result.showErrorToast, true)
  assert.equal(result.isAuthDenied, false)
})

test('loadFuxiAdminDirectoryState suppresses the page-owned toast for unauthorized failures', async () => {
  const result = await loadFuxiAdminDirectoryState(async () => {
    throw { code: ApiStatus.unauthorized }
  }, 'Failed to load the current Fuxi admin directory')

  assert.equal(result.directory, null)
  assert.equal(result.loadErrorMessage, 'Failed to load the current Fuxi admin directory')
  assert.equal(result.showErrorToast, false)
  assert.equal(result.isAuthDenied, true)
})

test('loadFuxiAdminDirectoryState suppresses the page-owned toast for forbidden failures', async () => {
  const result = await loadFuxiAdminDirectoryState(async () => {
    throw { code: ApiStatus.forbidden }
  }, 'Failed to load the current Fuxi admin directory')

  assert.equal(result.directory, null)
  assert.equal(result.loadErrorMessage, 'Failed to load the current Fuxi admin directory')
  assert.equal(result.showErrorToast, false)
  assert.equal(result.isAuthDenied, true)
})

test('resolveManageAccess distinguishes role eligibility from confirmed manage access', () => {
  assert.equal(
    resolveManageAccess({
      hadAccess: true,
      hasRole: false,
      gotDirectory: false,
      isAuthDenied: false
    }),
    false
  )
  assert.equal(
    resolveManageAccess({
      hadAccess: false,
      hasRole: true,
      gotDirectory: true,
      isAuthDenied: false
    }),
    true
  )
  assert.equal(
    resolveManageAccess({
      hadAccess: true,
      hasRole: true,
      gotDirectory: false,
      isAuthDenied: true
    }),
    false
  )
  assert.equal(
    resolveManageAccess({
      hadAccess: true,
      hasRole: true,
      gotDirectory: false,
      isAuthDenied: false
    }),
    true
  )
})

test('loadFuxiAdminPageDirectory falls back to the public directory when manage access is denied', async () => {
  let manageCalls = 0
  let publicCalls = 0
  const publicDirectory = createDirectory()

  const result = await loadFuxiAdminPageDirectory({
    hadManageAccess: true,
    hasEditRole: true,
    loadFailedMessage: 'Failed to load the current Fuxi admin directory',
    loadManageDirectory: async () => {
      manageCalls += 1
      throw { code: ApiStatus.forbidden }
    },
    loadPublicDirectory: async () => {
      publicCalls += 1
      return publicDirectory
    }
  })

  assert.equal(manageCalls, 1)
  assert.equal(publicCalls, 1)
  assert.equal(result.directory, publicDirectory)
  assert.equal(result.hasManageAccess, false)
  assert.equal(result.loadErrorMessage, null)
  assert.equal(result.showErrorToast, false)
})

test('loadFuxiAdminPageDirectory keeps manage access when the manage directory loads successfully', async () => {
  let publicCalls = 0
  const manageDirectory = createManageDirectory()

  const result = await loadFuxiAdminPageDirectory({
    hadManageAccess: false,
    hasEditRole: true,
    loadFailedMessage: 'Failed to load the current Fuxi admin directory',
    loadManageDirectory: async () => manageDirectory,
    loadPublicDirectory: async () => {
      publicCalls += 1
      return createDirectory()
    }
  })

  assert.equal(publicCalls, 0)
  assert.equal(result.directory, manageDirectory)
  assert.equal(result.hasManageAccess, true)
})
