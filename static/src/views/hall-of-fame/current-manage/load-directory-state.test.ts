import assert from 'node:assert/strict'
import test from 'node:test'

import { ApiStatus } from '@/utils/http/status'

import { loadFuxiAdminDirectoryState } from './load-directory-state'

test('loadFuxiAdminDirectoryState returns the directory on success', async () => {
  const directory: Api.FuxiAdmin.DirectoryResponse = {
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

  const result = await loadFuxiAdminDirectoryState(async () => directory, 'Failed to load')

  assert.equal(result.directory, directory)
  assert.equal(result.loadErrorMessage, null)
  assert.equal(result.showErrorToast, false)
})

test('loadFuxiAdminDirectoryState returns the localized fallback message on failure', async () => {
  const result = await loadFuxiAdminDirectoryState(async () => {
    throw new Error('database is closed')
  }, 'Failed to load the current Fuxi admin directory')

  assert.equal(result.directory, null)
  assert.equal(result.loadErrorMessage, 'Failed to load the current Fuxi admin directory')
  assert.equal(result.showErrorToast, true)
})

test('loadFuxiAdminDirectoryState suppresses the page-owned toast for unauthorized failures', async () => {
  const result = await loadFuxiAdminDirectoryState(async () => {
    throw { code: ApiStatus.unauthorized }
  }, 'Failed to load the current Fuxi admin directory')

  assert.equal(result.directory, null)
  assert.equal(result.loadErrorMessage, 'Failed to load the current Fuxi admin directory')
  assert.equal(result.showErrorToast, false)
})
