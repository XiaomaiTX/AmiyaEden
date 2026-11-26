import { beforeEach, describe, expect, test, vi } from 'vitest'

vi.mock('@/api/http-client', () => ({
  requestJson: vi.fn(),
}))

import { requestJson } from '@/api/http-client'
import {
  adminCreate,
  deleteToolBookmark,
  listToolBookmarks,
  listVisible,
  updateToolBookmark,
} from '@/api/generated/client'

import { unwrapContract } from '@/api/response'

const bookmark = {
  id: 1,
  name: 'EVE Tools',
  url: 'https://example.com',
  description: '',
  logo_url: '',
  logo_source: '',
  group_name: '',
  is_enabled: true,
  sort_order: 0,
  created_by: 1,
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
}

describe('tool bookmark api', () => {
  beforeEach(() => {
    vi.mocked(requestJson).mockReset()
  })

  test('wraps all existing tool bookmark endpoints', async () => {
    vi.mocked(requestJson)
      .mockResolvedValueOnce({ code: 200, msg: '', data: [bookmark] })
      .mockResolvedValueOnce({ code: 200, msg: '', data: [bookmark] })
      .mockResolvedValueOnce({ code: 200, msg: '', data: bookmark })
      .mockResolvedValueOnce({ code: 200, msg: '', data: bookmark })
      .mockResolvedValueOnce({ code: 200, msg: '', data: null })

    const payload = {
      name: 'EVE Tools',
      url: 'https://example.com',
      is_enabled: true,
      sort_order: 1,
    }
    await listVisible()
    await listToolBookmarks()
    await adminCreate(payload)
    await updateToolBookmark(1, payload)
    await deleteToolBookmark(1)

    expect(requestJson).toHaveBeenNthCalledWith(
      1,
      '/api/v1/info/tool-bookmarks',
      expect.objectContaining({ method: 'GET' })
    )
    expect(requestJson).toHaveBeenNthCalledWith(
      2,
      '/api/v1/system/tool-bookmarks',
      expect.objectContaining({ method: 'GET' })
    )
    expect(requestJson).toHaveBeenNthCalledWith(
      3,
      '/api/v1/system/tool-bookmarks',
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify(payload),
      })
    )
    expect(requestJson).toHaveBeenNthCalledWith(
      4,
      '/api/v1/system/tool-bookmarks/1',
      expect.objectContaining({
        method: 'PUT',
        body: JSON.stringify(payload),
      })
    )
    expect(requestJson).toHaveBeenNthCalledWith(
      5,
      '/api/v1/system/tool-bookmarks/1',
      expect.objectContaining({ method: 'DELETE' })
    )
    for (const index of [2, 3]) {
      const init = vi.mocked(requestJson).mock.calls[index]?.[1]
      expect(new Headers(init?.headers).get('Content-Type')).toBe('application/json')
    }
  })

  test('rejects failed business responses', async () => {
    vi.mocked(requestJson).mockResolvedValueOnce({
      code: 500,
      msg: 'bookmark service unavailable',
      data: [],
    })

    await expect(
      unwrapContract(listVisible(), 'fetch visible tool bookmarks failed')
    ).rejects.toThrow('bookmark service unavailable')
  })
})
