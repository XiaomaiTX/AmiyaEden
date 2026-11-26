import { saveAdmission, getAdmissionEvidence } from './admissions'
import { inviteAdmission, listAdmissionReviews } from './generated/client'
import { unwrapContract } from './response'
import { useSessionStore } from '@/stores'

beforeEach(() => {
  useSessionStore
    .getState()
    .setSessionSnapshot({ isLoggedIn: true, accessToken: 'admission-token' })
})
afterEach(() => {
  vi.restoreAllMocks()
  useSessionStore.getState().clearSession()
})

it('sends a versioned whole-user multipart resubmission without overriding its boundary', async () => {
  const fetch = vi
    .spyOn(globalThis, 'fetch')
    .mockResolvedValue(
      new Response(JSON.stringify({ code: 200, data: { id: 17 } }), {
        headers: { 'Content-Type': 'application/json' },
      })
    )
  const input = {
    kind: 'new_member' as const,
    target_corporation_id: 98824507,
    character_ids: [101, 102],
    expected_version: 3,
  }
  await saveAdmission(
    input,
    { launcher: new Blob(['jpeg']), seat_profile: new Blob(['jpeg']) },
    17,
    true
  )
  const [url, init] = fetch.mock.calls[0]
  expect(url).toBe('/api/v1/admissions/applications/17/resubmit')
  expect(init?.method).toBe('POST')
  expect((init?.headers as Headers).get('Content-Type')).toBeNull()
  expect((init?.headers as Headers).get('Authorization')).toBe('Bearer admission-token')
  const body = init?.body as FormData
  expect(JSON.parse(body.get('payload') as string)).toEqual(input)
  expect([...body.keys()]).toEqual(['payload', 'launcher', 'seat_profile'])
  expect(body.get('launcher')).toMatchObject({ name: 'launcher.jpg', size: 4 })
  expect(body.get('seat_profile')).toMatchObject({ name: 'seat_profile.jpg', size: 4 })
})

it('authenticates private evidence and preserves localized processing failures', async () => {
  const fetch = vi
    .spyOn(globalThis, 'fetch')
    .mockResolvedValueOnce(
      new Response(new Blob(['jpeg']), { headers: { 'Content-Type': 'image/jpeg' } })
    )
  await getAdmissionEvidence(17, 'seat_profile')
  expect(new Headers(fetch.mock.calls[0][1]?.headers).get('Authorization')).toBe(
    'Bearer admission-token'
  )
  fetch.mockResolvedValueOnce(
    new Response(JSON.stringify({ code: 500, msg: 'admissions.errors.evidence', data: null }), {
      headers: { 'Content-Type': 'application/json' },
    })
  )
  await expect(getAdmissionEvidence(17, 'seat_profile')).rejects.toThrow(
    'admissions.errors.evidence'
  )
  fetch.mockResolvedValueOnce(
    new Response(JSON.stringify({ msg: 'admissions.errors.walletDisabled' }), { status: 403 })
  )
  await expect(
    unwrapContract(inviteAdmission(17, { expected_version: 3, reason: '' }), '')
  ).rejects.toThrow('admissions.errors.walletDisabled')
})

it('sends reviewer status, kind, and target filters to the scoped endpoint', async () => {
  const fetch = vi
    .spyOn(globalThis, 'fetch')
    .mockResolvedValue(
      new Response(JSON.stringify({ code: 200, data: { list: [], total: 0 } }), {
        headers: { 'Content-Type': 'application/json' },
      })
    )
  await listAdmissionReviews({
    current: 2,
    size: 20,
    status: 'processed',
    target_corporation_id: '98185110',
    kind: 'alt',
  })
  const url = new URL(String(fetch.mock.calls[0][0]), 'https://local.test')
  expect(url.pathname).toBe('/api/v1/system/admissions/applications')
  expect(Object.fromEntries(url.searchParams)).toEqual({
    current: '2',
    size: '20',
    status: 'processed',
    target_corporation_id: '98185110',
    kind: 'alt',
  })
})
