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

    expect(fetchSpy).toHaveBeenCalledTimes(1)
    const [, init] = fetchSpy.mock.calls[0]
    expect(init?.headers).toBeInstanceOf(Headers)
    expect((init?.headers as Headers).get('Authorization')).toBe('Bearer token-123')
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
