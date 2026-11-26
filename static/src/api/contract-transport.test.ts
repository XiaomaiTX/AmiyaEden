import assert from 'node:assert/strict'
import { fileURLToPath } from 'node:url'
import { test } from 'node:test'
import axios, { type AxiosAdapter, type InternalAxiosRequestConfig } from 'axios'
import { createServer } from 'vite'
import type { completeEveRegistration, getEveRegistration } from './generated/client'
import { unwrapContract } from './response'
import { decodeBlobResponse } from '../utils/http/response-body'
import { defaultResponseAdapter } from '../utils/table/tableUtils'

// These annotations fail type checking if generation no longer requires the registration header.
const readHeaders: Parameters<typeof getEveRegistration>[0] = { 'X-Registration-Token': 'ticket' }
const writeHeaders: Parameters<typeof completeEveRegistration>[1] = readHeaders
// @ts-expect-error X-Registration-Token is mandatory on the generated GET.
const missingReadHeader: Parameters<typeof getEveRegistration>[0] = {}
// @ts-expect-error X-Registration-Token is mandatory on the generated POST.
const missingWriteHeader: Parameters<typeof completeEveRegistration>[1] = {}
void [writeHeaders, missingReadHeader, missingWriteHeader]

test('binary response decoding preserves a file and reads JSON from the same response mode', async () => {
  const jpeg = new Blob(['image'], { type: 'image/jpeg' })
  assert.equal(decodeBlobResponse(jpeg), jpeg)

  const problem = { code: 404, msg: 'admissions.errors.evidence', data: null } as const
  const json = new Blob([JSON.stringify(problem)], { type: 'application/json;charset=utf-8' })
  assert.deepEqual(await decodeBlobResponse(json), problem)
  await assert.rejects(unwrapContract(Promise.resolve(problem)), /admissions\.errors\.evidence/)
})

test('shared envelope unwrapping preserves successful null payloads', async () => {
  assert.equal(await unwrapContract(Promise.resolve({ code: 200, msg: 'ok', data: null })), null)
})

test('generated operations use the real Vue HTTP and Axios adapters', async (t) => {
  // Use Vite only to load the production modules and import.meta.env without browser UI or build plugins.
  const root = fileURLToPath(new URL('../../', import.meta.url))
  const stubs = new Map([
    ['\0test-user', 'export const useUserStore = () => ({ accessToken: "access" })'],
    ['\0test-locales', 'export const $t = (key) => key']
  ])
  const server = await createServer({
    root,
    configFile: false,
    envFile: false,
    appType: 'custom',
    server: { middlewareMode: true, ws: false, watch: null },
    resolve: {
      alias: {
        '@/store/modules/user': '\0test-user',
        '@/locales': '\0test-locales',
        '@': fileURLToPath(new URL('../', import.meta.url))
      }
    },
    plugins: [
      {
        name: 'contract-test-ui-stubs',
        resolveId: (id) => (stubs.has(id) ? id : undefined),
        load: (id) => stubs.get(id)
      }
    ]
  })
  t.after(() => server.close())

  const requests: InternalAxiosRequestConfig[] = []
  const success = { code: 200, msg: 'ok', data: null }
  let body: string | Blob = JSON.stringify(success)
  const adapter: AxiosAdapter = async (config) => {
    requests.push(config)
    return {
      config,
      status: 200,
      statusText: 'OK',
      headers: { 'content-type': body instanceof Blob ? body.type : 'application/json' },
      data: body
    }
  }
  const originalAdapter = axios.defaults.adapter
  axios.defaults.adapter = adapter
  const {
    completeEveRegistration,
    createAdmission,
    getAdmissionEvidence,
    listFleets,
    runSkillPlanCompletionCheck
  } = await (async () => {
    try {
      return (await server.ssrLoadModule(
        '/src/api/generated/client.ts'
      )) as typeof import('./generated/client')
    } finally {
      axios.defaults.adapter = originalAdapter
    }
  })()

  const registrationBody = { nickname: 'Pilot', qq: '123' }
  const registration = { code: 200, msg: 'ok', data: { token: 'new-access' } }
  body = JSON.stringify(registration)
  assert.deepEqual(
    await completeEveRegistration(registrationBody, writeHeaders, {
      headers: { 'X-Registration-Token': 'ignored', 'X-Custom': 'retained' }
    }),
    registration
  )
  const [first] = requests
  assert.equal(first.method, 'post')
  assert.equal(first.url, '/api/v1/sso/eve/registration')
  assert.equal(first.headers.get('X-Registration-Token'), 'ticket')
  assert.equal(first.headers.get('X-Custom'), 'retained')
  assert.equal(first.headers.getAuthorization(), 'Bearer access')
  assert.equal(first.headers.getContentType(), 'application/json')
  assert.equal(first.data, JSON.stringify(registrationBody))

  const page = { list: null, total: 0, page: 2, pageSize: 20 }
  body = JSON.stringify({ ...success, data: page })
  const fleetPage = await unwrapContract(listFleets({ current: 2, size: 20 }))
  assert.deepEqual(fleetPage, page)
  assert.deepEqual(defaultResponseAdapter(fleetPage), { list: [], total: 0, current: 2, size: 20 })
  assert.deepEqual(requests[1].params, { current: 2, size: 20 })
  assert.equal(requests[1].data, undefined)

  const launcher = new Blob(['launcher'], { type: 'image/jpeg' })
  body = JSON.stringify(success)
  await createAdmission({ payload: '{}', launcher })
  const form: unknown = requests[2].data
  assert.ok(form instanceof FormData)
  assert.equal(form.get('payload'), '{}')
  const uploaded = form.get('launcher')
  assert.ok(uploaded instanceof Blob)
  assert.equal(await uploaded.text(), 'launcher')
  assert.notEqual(requests[2].headers.getContentType(), 'multipart/form-data')

  body = launcher
  assert.equal(await getAdmissionEvidence(42, 'launcher'), launcher)
  assert.equal(requests[3].responseType, 'blob')
  assert.equal(requests[3].url, '/api/v1/admissions/applications/42/evidence/launcher')

  body = JSON.stringify(success)
  await runSkillPlanCompletionCheck(undefined, { lang: 'en' })
  assert.equal(requests[4].method, 'post')
  assert.deepEqual(requests[4].params, { lang: 'en' })
  assert.equal(requests[4].data, undefined)

  body = new Blob([JSON.stringify({ code: 404, msg: 'missing evidence', data: null })], {
    type: 'application/problem+json;charset=utf-8'
  })
  t.mock.method(console, 'error', () => {})
  assert.deepEqual(await getAdmissionEvidence(42, 'launcher', { showErrorMessage: false }), {
    code: 404,
    msg: 'missing evidence',
    data: null
  })

  const failure = { code: 409, msg: 'fleet busy', data: null }
  body = JSON.stringify(failure)
  assert.deepEqual(await listFleets({ current: 2, size: 20 }, { showErrorMessage: false }), failure)
  await assert.rejects(
    unwrapContract(listFleets({ current: 2, size: 20 }, { showErrorMessage: false })),
    { name: 'HttpError', code: 409, message: 'fleet busy' }
  )
})
