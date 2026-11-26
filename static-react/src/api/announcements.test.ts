import { beforeEach, describe, expect, it, vi } from 'vitest'
import { listAnnouncements, readAnnouncement, updateAnnouncement } from './generated/client'
import { unwrapContract } from './response'
import { HttpError, requestJson } from './http-client'

vi.mock('./http-client', async (importOriginal) => ({
  ...(await importOriginal<typeof import('./http-client')>()),
  requestJson: vi.fn(),
}))

const request = vi.mocked(requestJson)

describe('generated announcement client', () => {
  beforeEach(() => request.mockReset())

  it('serializes pagination and unwraps the list', async () => {
    const data = { list: [], total: 0 }
    request.mockResolvedValue({ code: 200, msg: 'success', data })
    await expect(unwrapContract(listAnnouncements({ page: 2 }), '')).resolves.toEqual(data)
    expect(request).toHaveBeenCalledWith(
      '/api/v1/announcements?page=2',
      expect.objectContaining({ method: 'GET' })
    )
  })

  it('uses PUT for an update and serializes the body once', async () => {
    const data = { title: 'Updated', body_html: '<p>Updated</p>', revision: 2, hidden: false }
    request.mockResolvedValue({ code: 200, msg: 'success', data })
    await expect(unwrapContract(updateAnnouncement(9, data), '')).resolves.toEqual(data)
    expect(request).toHaveBeenCalledWith(
      '/api/v1/announcements/9',
      expect.objectContaining({
        method: 'PUT',
        body: JSON.stringify(data),
      })
    )
  })

  it('preserves null success, business failures and HTTP errors', async () => {
    request.mockResolvedValueOnce({ code: 200, msg: 'success', data: null })
    await expect(unwrapContract(readAnnouncement(9, { version: 1 }), '')).resolves.toBeNull()
    request.mockResolvedValueOnce({ code: 400, msg: 'bad version', data: null })
    await expect(unwrapContract(readAnnouncement(9, { version: 0 }), '')).rejects.toThrow(
      'bad version'
    )
    const unauthorized = new HttpError(401, 'Unauthorized')
    request.mockRejectedValueOnce(unauthorized)
    await expect(unwrapContract(readAnnouncement(9, { version: 1 }), '')).rejects.toBe(unauthorized)
  })
})
