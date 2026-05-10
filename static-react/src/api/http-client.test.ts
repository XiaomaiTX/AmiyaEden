import { subscribeUnauthorized } from '@/auth'
import { HttpError, requestJson } from '@/api/http-client'
import { useSessionStore } from '@/stores'

describe('http client', () => {
  beforeEach(() => {
    useSessionStore.getState().clearSession()
  })

  test('injects bearer token when accessToken exists', async () => {
    useSessionStore.getState().setSessionSnapshot({
      isLoggedIn: true,
      accessToken: 'token-123',
    })

    const fetchSpy = vi.spyOn(globalThis, 'fetch').mockResolvedValueOnce(
      new Response(JSON.stringify({ ok: true }), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      })
    )

    await requestJson('/api/v1/test')

    expect(fetchSpy).toHaveBeenCalledWith(
      '/api/v1/test',
      expect.objectContaining({
        headers: expect.objectContaining({
          Authorization: 'Bearer token-123',
        }),
      })
    )
  })

  test('dispatches unauthorized event and throws HttpError on 401', async () => {
    const events: Array<{ reason: string }> = []
    const unsubscribe = subscribeUnauthorized((event) => {
      events.push({ reason: event.reason })
    })

    vi.spyOn(globalThis, 'fetch').mockResolvedValueOnce(
      new Response('', {
        status: 401,
      })
    )

    await expect(requestJson('/api/v1/test')).rejects.toBeInstanceOf(HttpError)
    expect(events).toEqual([{ reason: 'http_401' }])

    unsubscribe()
  })
})
