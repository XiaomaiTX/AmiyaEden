import { useSessionStore } from '@/stores'
import { subscribeUnauthorized } from '@/auth'
import { HttpError } from './http-client'
import { unwrapContract } from './response'
import {
  completeEveRegistration,
  createAdmission,
  getAdmissionEvidence,
  getEveRegistration,
  uploadEvidence,
} from './generated/client'

const jsonResponse = (data: unknown, status = 200) =>
  new Response(JSON.stringify(data), { status, headers: { 'Content-Type': 'application/json' } })

beforeEach(() => {
  useSessionStore.getState().setSessionSnapshot({ isLoggedIn: true, accessToken: 'token' })
})

afterEach(() => {
  vi.restoreAllMocks()
  useSessionStore.getState().clearSession()
})

test('generated registration operations require and send the ticket header on both verbs', async () => {
  const fetch = vi.spyOn(globalThis, 'fetch').mockImplementation(async () => jsonResponse({}))
  const header = { 'X-Registration-Token': 'ticket' }
  const input = { nickname: 'Amiya', qq: '123456' }
  const options = { headers: new Headers({ 'x-registration-token': 'not-the-ticket', 'X-Caller': 'id' }) }

  // These annotations fail type checking if the header becomes optional again.
  // @ts-expect-error Registration GET must require the ticket header.
  const missingReadHeader: Parameters<typeof getEveRegistration>[0] = {}
  // @ts-expect-error Registration POST must require the ticket header.
  const missingWriteHeader: Parameters<typeof completeEveRegistration>[1] = {}
  void [missingReadHeader, missingWriteHeader]

  await getEveRegistration(header, options)
  await completeEveRegistration(input, header, options)

  for (const [url, init] of fetch.mock.calls) {
    expect(url).toBe('/api/v1/sso/eve/registration')
    const sentHeaders = new Headers(init?.headers)
    expect(sentHeaders.get('X-Registration-Token')).toBe('ticket')
    expect(sentHeaders.get('X-Caller')).toBe('id')
    expect(sentHeaders.get('Authorization')).toBe('Bearer token')
  }
  expect(fetch.mock.calls.map(([, request]) => request?.method)).toEqual(['GET', 'POST'])
  expect(JSON.parse(String(fetch.mock.calls[1][1]?.body))).toEqual(input)
})

test('generated evidence chooses binary mode without wrapper options and preserves 401 handling', async () => {
  const controller = new AbortController()
  const denied = { code: 404, msg: 'admissions.errors.evidence', data: null }
  const fetch = vi.spyOn(globalThis, 'fetch')
    .mockResolvedValueOnce(new Response(new Uint8Array([1, 2, 3]), { headers: { 'Content-Type': 'image/jpeg' } }))
    .mockResolvedValueOnce(jsonResponse(denied))
    .mockResolvedValueOnce(jsonResponse({ code: 401, msg: 'unauthorized', data: null }, 401))
    .mockResolvedValueOnce(jsonResponse({ code: 401, msg: 'session expired', data: null }))
  const unauthorized = vi.fn()
  const unsubscribe = subscribeUnauthorized(unauthorized)

  try {
    const result = await getAdmissionEvidence(17, 'launcher', { signal: controller.signal })
    expect(result).toMatchObject({ size: 3, type: 'image/jpeg' })
    const [url, init] = fetch.mock.calls[0]
    expect(url).toBe('/api/v1/admissions/applications/17/evidence/launcher')
    expect(init?.signal).toBe(controller.signal)
    expect(new Headers(init?.headers).get('Authorization')).toBe('Bearer token')
    await expect(getAdmissionEvidence(17, 'launcher')).resolves.toEqual(denied)
    await expect(getAdmissionEvidence(17, 'launcher')).rejects.toBeInstanceOf(HttpError)
    await expect(getAdmissionEvidence(17, 'launcher')).rejects.toThrow('session expired')
    expect(unauthorized).toHaveBeenCalledTimes(2)
    expect(unauthorized).toHaveBeenCalledWith({ reason: 'http_401' })
  } finally {
    unsubscribe()
  }
})

test('contact conflict remains a business response while notifying the existing dialog', async () => {
  const conflict = {
    code: 409,
    msg: 'contact_already_bound',
    data: { contact: 'qq', primary_character_id: 17, primary_character_name: 'Pilot' },
  }
  vi.spyOn(globalThis, 'fetch').mockImplementation(async () => jsonResponse(conflict))
  const listener = vi.fn()
  window.addEventListener('auth:contact-conflict', listener)
  const register = () => completeEveRegistration(
    { nickname: 'Amiya', qq: '123' }, { 'X-Registration-Token': 'ticket' },
  )
  try {
    await expect(register()).resolves.toEqual(conflict)
    expect(listener).toHaveBeenCalledTimes(1)
    expect(listener.mock.calls[0][0].detail).toEqual(conflict.data)
    await expect(unwrapContract(register())).rejects.toThrow('contact_already_bound')
  } finally {
    window.removeEventListener('auth:contact-conflict', listener)
  }
})

test('generated multipart request keeps caller headers and leaves the boundary to fetch', async () => {
  const fetch = vi.spyOn(globalThis, 'fetch').mockResolvedValueOnce(jsonResponse({}))
  await createAdmission({ payload: '{}' }, { headers: { 'X-Caller': 'id', 'Content-Type': 'application/json' } })
  const [, init] = fetch.mock.calls[0]
  expect(init?.body).toBeInstanceOf(FormData)
  const headers = new Headers(init?.headers)
  expect(headers.get('content-type')).toBeNull()
  expect(headers.get('x-caller')).toBe('id')
  expect(headers.get('authorization')).toBe('Bearer token')
})

test('generated welfare evidence upload uses the common response and error handling', async () => {
  const fetch = vi.spyOn(globalThis, 'fetch')
    .mockResolvedValueOnce(jsonResponse({ code: 200, msg: 'success', data: { url: 'data:image/png;base64,eA==' } }))
    .mockResolvedValueOnce(jsonResponse({ code: 400, msg: 'invalid file', data: null }))
    .mockResolvedValueOnce(jsonResponse({ code: 0, msg: '', data: { url: 'not-a-success' } }))
    .mockResolvedValueOnce(jsonResponse({ code: 403, msg: 'not permitted', data: null }, 403))
  const file = new File(['x'], 'evidence.png', { type: 'image/png' })
  await expect(unwrapContract(uploadEvidence({ file }), 'upload failed')).resolves.toEqual({ url: 'data:image/png;base64,eA==' })
  const [url, request] = fetch.mock.calls[0]
  expect(url).toBe('/api/v1/welfare/upload-evidence')
  expect(request?.method).toBe('POST')
  expect(request?.body).toBeInstanceOf(FormData)
  expect(new Headers(request?.headers).has('Content-Type')).toBe(false)
  await expect(unwrapContract(uploadEvidence({ file }), 'upload failed')).rejects.toMatchObject({
    status: 400, message: 'invalid file',
  })
  await expect(unwrapContract(uploadEvidence({ file }), 'upload failed')).rejects.toMatchObject({
    status: 0, message: 'upload failed',
  })
  await expect(unwrapContract(uploadEvidence({ file }), 'upload failed')).rejects.toMatchObject({
    status: 403, message: 'not permitted',
  })
})
