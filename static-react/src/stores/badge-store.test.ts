import { fetchBadgeCounts } from '@/api/badge'
import { useBadgeStore } from '@/stores/badge-store'
import { useSessionStore } from '@/stores/session-store'

vi.mock('@/api/badge', () => ({
  fetchBadgeCounts: vi.fn(),
}))

describe('badge store', () => {
  beforeEach(() => {
    useBadgeStore.getState().clear()
    vi.mocked(fetchBadgeCounts).mockReset()
    useSessionStore.setState({ roles: ['user'] })
  })

  test.each([[['guest']], [['recruiter']], [['guest', 'recruiter']]])(
    'skips the Login-only request for %j accounts',
    async (roles) => {
      useSessionStore.setState({ roles })

      await useBadgeStore.getState().load(1001)

      expect(fetchBadgeCounts).not.toHaveBeenCalled()
      expect(useBadgeStore.getState().counts).toEqual({})
      expect(useBadgeStore.getState().loadedForCharacterId).toBe(1001)
    }
  )

  test('loads badge counts once for the active character', async () => {
    vi.mocked(fetchBadgeCounts).mockResolvedValue({ order_pending: 2 })

    await Promise.all([useBadgeStore.getState().load(1001), useBadgeStore.getState().load(1001)])
    await useBadgeStore.getState().load(1001)

    expect(fetchBadgeCounts).toHaveBeenCalledTimes(1)
    expect(useBadgeStore.getState().counts).toEqual({ order_pending: 2 })
  })

  test('refreshes pending admissions after a mutation without clearing other badges', async () => {
    vi.mocked(fetchBadgeCounts).mockResolvedValueOnce({ admission_pending: 2, order_pending: 3 })
    await useBadgeStore.getState().load(1001)
    vi.mocked(fetchBadgeCounts).mockResolvedValueOnce({ order_pending: 3 })
    const request = useBadgeStore.getState().load(1001, true)
    expect(useBadgeStore.getState().counts).toEqual({ admission_pending: 2, order_pending: 3 })
    await request
    expect(useBadgeStore.getState().counts).toEqual({ order_pending: 3 })
  })

  test('discards a response after the session badge state is cleared', async () => {
    let resolveRequest: (counts: { ticket_attention: number }) => void = () => undefined
    vi.mocked(fetchBadgeCounts).mockReturnValue(
      new Promise((resolve) => {
        resolveRequest = resolve
      })
    )

    const request = useBadgeStore.getState().load(1001)
    useBadgeStore.getState().clear()
    resolveRequest({ ticket_attention: 3 })
    await request

    expect(useBadgeStore.getState().counts).toEqual({})
    expect(useBadgeStore.getState().loadedForCharacterId).toBeNull()
  })
})
