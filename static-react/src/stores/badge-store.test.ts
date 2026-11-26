import type { BadgeCounts } from '@/api/generated/client'
import { useBadgeStore } from '@/stores/badge-store'
import { useSessionStore } from '@/stores/session-store'

const fetchMock = vi.fn<typeof fetch>()
const success = (data: BadgeCounts) => Response.json({ code: 200, msg: '', data })

describe('badge store', () => {
  beforeEach(() => {
    useBadgeStore.getState().clear()
    fetchMock.mockReset()
    vi.stubGlobal('fetch', fetchMock)
    useSessionStore.setState({ roles: ['user'] })
  })

  afterEach(() => vi.unstubAllGlobals())

  test.each([[['guest']], [['recruiter']], [['guest', 'recruiter']]])(
    'skips the Login-only request for %j accounts',
    async (roles) => {
      useSessionStore.setState({ roles })

      await useBadgeStore.getState().load(1001)

      expect(fetchMock).not.toHaveBeenCalled()
      expect(useBadgeStore.getState().counts).toEqual({})
      expect(useBadgeStore.getState().loadedForCharacterId).toBe(1001)
    }
  )

  test('loads badge counts once for the active character', async () => {
    fetchMock.mockResolvedValue(success({ order_pending: 2 }))

    await Promise.all([useBadgeStore.getState().load(1001), useBadgeStore.getState().load(1001)])
    await useBadgeStore.getState().load(1001)

    expect(fetchMock).toHaveBeenCalledTimes(1)
    expect(fetchMock).toHaveBeenCalledWith(
      '/api/v1/badge-counts',
      expect.objectContaining({ method: 'GET' })
    )
    expect(useBadgeStore.getState().counts).toEqual({ order_pending: 2 })
  })

  test('refreshes pending admissions after a mutation without clearing other badges', async () => {
    fetchMock.mockResolvedValueOnce(success({ admission_pending: 2, order_pending: 3 }))
    await useBadgeStore.getState().load(1001)
    fetchMock.mockResolvedValueOnce(success({ order_pending: 3 }))
    const request = useBadgeStore.getState().load(1001, true)
    expect(useBadgeStore.getState().counts).toEqual({ admission_pending: 2, order_pending: 3 })
    await request
    expect(useBadgeStore.getState().counts).toEqual({ order_pending: 3 })
  })

  test('discards a response after the session badge state is cleared', async () => {
    let resolveRequest: (response: Response) => void = () => undefined
    fetchMock.mockReturnValue(
      new Promise((resolve) => {
        resolveRequest = resolve
      })
    )

    const request = useBadgeStore.getState().load(1001)
    useBadgeStore.getState().clear()
    resolveRequest(success({ ticket_attention: 3 }))
    await request

    expect(useBadgeStore.getState().counts).toEqual({})
    expect(useBadgeStore.getState().loadedForCharacterId).toBeNull()
  })

  test('forced refresh supersedes an in-flight load and ignores its stale response', async () => {
    let resolveStale: (response: Response) => void = () => undefined
    fetchMock
      .mockReturnValueOnce(
        new Promise((resolve) => {
          resolveStale = resolve
        })
      )
      .mockResolvedValueOnce(success({ announcement_unread: 0 }))

    const staleRequest = useBadgeStore.getState().load(1001)
    const forcedRequest = useBadgeStore.getState().load(1001, true)
    await forcedRequest
    expect(useBadgeStore.getState().counts).toEqual({ announcement_unread: 0 })

    resolveStale(success({ announcement_unread: 2 }))
    await staleRequest
    expect(fetchMock).toHaveBeenCalledTimes(2)
    expect(useBadgeStore.getState().counts).toEqual({ announcement_unread: 0 })
  })
})
