import { act, renderHook, waitFor } from '@testing-library/react'
import { useSessionStore } from '@/stores'
import { useAdmissionContext } from './use-admission-context'
import type { AdmissionContext } from '@/types/api/admissions'

vi.unmock('@/hooks/use-admission-context')
const fetchMock = vi.fn<typeof fetch>()

beforeEach(() => {
  vi.stubGlobal('fetch', fetchMock)
})

const context: AdmissionContext = {
  is_member: true,
  can_create_new_member: false,
  can_create_alt: true,
  qq: '',
  discord_id: 'discord',
  characters: [],
  targets: [],
  reviewable_corporation_ids: [98185110],
}

const success = (data: AdmissionContext) => Response.json({ code: 200, msg: '', data })

afterEach(() => {
  act(() => useSessionStore.getState().clearSession())
  fetchMock.mockReset()
  vi.unstubAllGlobals()
})

it('keeps eligibility closed on failure, retries, refreshes membership, and clears on logout', async () => {
  fetchMock.mockRejectedValueOnce(new Error('offline')).mockResolvedValueOnce(success(context))
  act(() =>
    useSessionStore.setState({ isLoggedIn: true, accessToken: 'context-retry', hydratedAt: '1' })
  )
  const { result } = renderHook(() => useAdmissionContext())
  expect(result.current.context).toBeNull()
  await waitFor(() => expect(result.current.error).toBe(true))
  await act(() => result.current.reload())
  expect(result.current.context?.can_create_alt).toBe(true)
  fetchMock.mockResolvedValueOnce(
    success({ ...context, is_member: false, can_create_alt: false, can_create_new_member: true })
  )
  act(() => useSessionStore.setState({ hydratedAt: '2' }))
  await waitFor(() => expect(result.current.context?.can_create_new_member).toBe(true))
  act(() => useSessionStore.getState().clearSession())
  expect(result.current.context).toBeNull()
})

it.each(['characters', 'targets', 'reviewable_corporation_ids'] as const)(
  'keeps eligibility closed when context has invalid %s',
  async (field) => {
    fetchMock.mockResolvedValueOnce(success({ ...context, [field]: null }))
    act(() =>
      useSessionStore.setState({
        isLoggedIn: true,
        accessToken: `invalid-${field}`,
        hydratedAt: '1',
      })
    )
    const { result } = renderHook(() => useAdmissionContext())
    await waitFor(() => expect(result.current.error).toBe(true))
    expect(result.current.context).toBeNull()
  }
)

it('does not let an older session response restore permissions after logout', async () => {
  let finish!: (value: ReturnType<typeof success>) => void
  fetchMock.mockReturnValue(
    new Promise((resolve) => {
      finish = resolve
    })
  )
  act(() =>
    useSessionStore.setState({ isLoggedIn: true, accessToken: 'context-stale', hydratedAt: '3' })
  )
  const { result } = renderHook(() => useAdmissionContext())
  await waitFor(() => expect(fetchMock).toHaveBeenCalled())
  act(() => useSessionStore.getState().clearSession())
  await act(async () => {
    finish(success(context))
  })
  expect(result.current.context).toBeNull()
  expect(result.current.loading).toBe(false)
})
