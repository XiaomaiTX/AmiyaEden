import { act, cleanup, fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import type { FleetCreateInput, FleetQQPingInput, FleetRecord } from '@/api/generated/client'
import { I18nProvider, ensureLocaleLoaded } from '@/i18n'
import { usePreferenceStore, useSessionStore } from '@/stores'
import { OperationFleetDetailPage } from './operation-fleet-detail-page'
import { FleetQQPingPanel } from './operation-fleet-qq-ping-panel'
import { OperationFleetsPage } from './operation-fleets-page'

const starts = '2026-11-26T19:00:00Z'
const ends = '2026-11-26T20:00:00Z'
const fleet: FleetRecord = {
  id: 'fleet-1',
  title: 'Test fleet',
  description: '',
  importance: 'cta',
  pap_count: 1,
  fc_user_id: 42,
  fc_character_id: 101,
  fc_character_name: 'Test Pilot',
  start_at: starts,
  end_at: ends,
  created_at: starts,
  updated_at: starts,
  auto_srp_mode: 'disabled',
}
const empty = { list: [], total: 0, page: 1, pageSize: 20 }
const qqPath = `/api/v1/operation/fleets/${fleet.id}/qq-ping`
const queued = { tasks: [{ task_id: 'task-1', status: 'queued', expires_at: ends }] }
const json = (data: unknown, httpStatus = 200) =>
  new Response(JSON.stringify({ code: 200, msg: '', data }), {
    status: httpStatus,
    headers: { 'Content-Type': 'application/json' },
  })
const rejected = (code: number, msg: string) =>
  new Response(JSON.stringify({ code, msg }), {
    status: code,
    headers: { 'Content-Type': 'application/json' },
  })

function mockFleetApi(
  overrides: (url: URL, request?: RequestInit) => Response | Promise<Response> | undefined
) {
  return vi.spyOn(globalThis, 'fetch').mockImplementation(async (input, request) => {
    const url = new URL(String(input), 'http://localhost')
    const response = overrides(url, request)
    if (response) return response
    switch (url.pathname) {
      case '/api/v1/operation/fleets':
      case '/api/v1/operation/fleet-configs':
      case `/api/v1/operation/fleets/${fleet.id}/members-pap`:
        return json(empty)
      case '/api/v1/sso/eve/characters':
        return json([{ character_id: 101, character_name: 'Test Pilot' }])
      case `/api/v1/operation/fleets/${fleet.id}/invites`:
        return json([])
      case `/api/v1/operation/fleets/${fleet.id}/refresh-esi`:
        return json(fleet)
      default:
        throw new Error(`Unexpected request: ${request?.method} ${url.pathname}`)
    }
  })
}

function openPage(detail = false) {
  return render(
    <I18nProvider>
      <MemoryRouter initialEntries={[detail ? `/fleets/${fleet.id}` : '/fleets']}>
        <Routes>
          <Route path="/fleets/:id" element={<OperationFleetDetailPage />} />
          <Route path="/fleets" element={<OperationFleetsPage />} />
        </Routes>
      </MemoryRouter>
    </I18nProvider>
  )
}

async function fillCreateDialog() {
  fireEvent.click(await screen.findByRole('button', { name: 'Create Fleet' }))
  const dialog = within(await screen.findByRole('dialog'))
  fireEvent.change(dialog.getByRole('textbox', { name: 'Title' }), {
    target: { value: fleet.title },
  })
  fireEvent.click(dialog.getByRole('button', { name: /FC/ }))
  fireEvent.click(await screen.findByRole('option', { name: 'Test Pilot' }))
  fireEvent.change(dialog.getByLabelText('Start At'), { target: { value: '2026-11-26T19:00' } })
  fireEvent.change(dialog.getByLabelText('End At'), { target: { value: '2026-11-26T20:00' } })
  return {
    dialog,
    existingPing: dialog.getByRole('checkbox', { name: 'Send ping after create' }),
    qqPing: dialog.getByRole('checkbox', { name: 'Queue QQ Ping on creation' }),
  }
}

beforeEach(async () => {
  await ensureLocaleLoaded('en-US')
  usePreferenceStore.setState({ locale: 'en-US' })
  useSessionStore.setState({ roles: ['admin'] })
})

afterEach(() => {
  cleanup()
  vi.restoreAllMocks()
  vi.useRealTimers()
  useSessionStore.getState().clearSession()
})

test('regular fleet creation leaves QQ off and preserves the existing Ping default', async () => {
  const requests: FleetCreateInput[] = []
  const fetch = mockFleetApi((url, init) => {
    if (url.pathname === '/api/v1/operation/fleets' && init?.method === 'POST') {
      requests.push(JSON.parse(String(init.body)))
      return json(fleet)
    }
  })
  openPage()
  const { dialog, existingPing, qqPing } = await fillCreateDialog()
  expect(existingPing).toBeChecked()
  expect(qqPing).not.toBeChecked()
  fireEvent.click(dialog.getByRole('button', { name: 'Confirm' }))
  await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())
  expect(requests).toEqual([expect.objectContaining({ send_ping: true, send_qq_ping: false })])
  expect(fetch.mock.calls.some(([url]) => String(url).includes('qq-ping'))).toBe(false)
})

test.each([
  [409, 'no_targets', 'No authorized QQ groups'],
  [429, 'queue_full', 'QQ operations are rate limited'],
  [503, 'disabled', 'QQ notifications are disabled'],
])(
  'keeps the form and does not fall back when requested QQ admission returns %s',
  async (status, reason, message) => {
    const requests: FleetCreateInput[] = []
    mockFleetApi((url, init) => {
      if (url.pathname === '/api/v1/operation/fleets' && init?.method === 'POST') {
        const payload = JSON.parse(String(init.body)) as FleetCreateInput
        requests.push(payload)
        return payload.send_qq_ping ? rejected(status, reason) : json(fleet)
      }
    })
    openPage()
    const { dialog, existingPing, qqPing } = await fillCreateDialog()
    fireEvent.click(existingPing)
    fireEvent.click(qqPing)
    fireEvent.click(dialog.getByRole('button', { name: 'Confirm' }))
    expect(await dialog.findByRole('alert')).toHaveTextContent(message)
    expect(qqPing).toBeChecked()
    expect(requests).toEqual([expect.objectContaining({ send_ping: false, send_qq_ping: true })])

    fireEvent.click(qqPing)
    fireEvent.click(dialog.getByRole('button', { name: 'Confirm' }))
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())
    expect(requests).toHaveLength(2)
    expect(requests[1]).toMatchObject({ send_ping: false, send_qq_ping: false })
  }
)

test('reports create-time QQ acceptance as queued, not delivered', async () => {
  mockFleetApi((url, init) => {
    if (url.pathname === '/api/v1/operation/fleets' && init?.method === 'POST') {
      return json({ ...fleet, qq_notification: { status: 'queued', task_count: 2 } })
    }
  })
  openPage()
  const { dialog, qqPing } = await fillCreateDialog()
  fireEvent.click(qqPing)
  fireEvent.click(dialog.getByRole('button', { name: 'Confirm' }))
  expect(await screen.findByText(/Fleet created. QQ Ping queued for 2/)).toBeInTheDocument()
})

test('manual QQ Ping is independent and retries the same UUID until explicitly reset', async () => {
  const bodies: FleetQQPingInput[] = []
  let rejectFirst: (reason: Error) => void = () => {
    throw new Error('The first request has not started')
  }
  const firstRequest = new Promise<Response>((_resolve, reject) => {
    rejectFirst = reject
  })
  let oldPings = 0
  mockFleetApi((url, init) => {
    if (url.pathname === `/api/v1/operation/fleets/${fleet.id}/ping` && init?.method === 'POST') {
      oldPings++
      return json(null)
    }
    if (url.pathname === qqPath) {
      if (init?.method !== 'POST') return json(empty)
      bodies.push(JSON.parse(String(init.body)))
      return bodies.length === 1 ? firstRequest : json(queued, 202)
    }
  })
  openPage(true)
  const queueButton = await screen.findByRole('button', { name: 'Queue QQ Ping' })
  fireEvent.click(screen.getByRole('button', { name: 'Send Ping' }))
  expect(oldPings).toBe(1)
  expect(bodies).toHaveLength(0)

  fireEvent.click(queueButton)
  fireEvent.click(queueButton)
  expect(bodies).toHaveLength(1)
  expect(bodies[0].idempotency_key).toMatch(/^[0-9a-f]{8}(?:-[0-9a-f]{4}){3}-[0-9a-f]{12}$/i)
  rejectFirst(new TypeError('connection lost'))
  fireEvent.click(await screen.findByRole('button', { name: 'Retry QQ Ping request' }))
  expect(await screen.findByText(/Queue acceptance does not confirm delivery/)).toBeInTheDocument()
  expect(bodies).toEqual([bodies[0], bodies[0]])
  expect(queueButton).toBeDisabled()
  expect(oldPings).toBe(1)

  fireEvent.click(screen.getByRole('button', { name: 'Prepare another QQ Ping' }))
  expect(bodies).toHaveLength(2)
  fireEvent.click(screen.getByRole('button', { name: 'Queue QQ Ping' }))
  expect(await screen.findByText(/Queue acceptance does not confirm delivery/)).toBeInTheDocument()
  expect(bodies).toHaveLength(3)
  expect(bodies[2].idempotency_key).not.toBe(bodies[0].idempotency_key)
})

test('does not load or expose QQ actions when existing fleet management is unavailable', async () => {
  useSessionStore.setState({ roles: [] })
  const fetch = mockFleetApi(() => undefined)
  openPage(true)
  await screen.findByRole('heading', { name: fleet.title })
  expect(screen.queryByRole('button', { name: 'Queue QQ Ping' })).not.toBeInTheDocument()
  expect(fetch.mock.calls.some(([url]) => String(url).includes(qqPath))).toBe(false)
  expect(screen.getByRole('button', { name: 'Send Ping' })).toBeDisabled()
})

test('polls active results to their actual states, pages history, and reloads without revealing raw data', async () => {
  vi.useFakeTimers()
  vi.spyOn(document, 'hidden', 'get').mockReturnValue(false)
  const pages: string[] = []
  mockFleetApi((url) => {
    if (url.pathname !== qqPath) return
    const page = url.searchParams.get('current')!
    pages.push(page)
    const rows = [
      {
        id: 'task-1',
        created_at: starts,
        expires_at: ends,
        status: pages.length === 1 ? 'queued' : page === '1' ? 'succeeded' : 'unknown',
        reason: page === '1' ? undefined : 'ambiguous',
        finished_at: pages.length === 1 ? undefined : ends,
        group_id: 'secret-group',
        payload: 'secret-payload',
      },
    ]
    return json({ list: rows, total: 101, page: Number(page), pageSize: 100 })
  })
  await act(async () => {
    render(
      <I18nProvider>
        <FleetQQPingPanel fleetId={fleet.id} />
      </I18nProvider>
    )
  })
  const panel = within(screen.getByRole('region', { name: 'QQ Ping Results' }))
  expect(panel.getByText('Queued', { exact: true })).toBeInTheDocument()
  await act(async () => vi.advanceTimersByTimeAsync(5000))
  expect(panel.getByText('Succeeded', { exact: true })).toBeInTheDocument()
  await act(async () => vi.advanceTimersByTimeAsync(10_000))
  expect(pages).toEqual(['1', '1'])
  await act(async () => fireEvent.click(panel.getByRole('button', { name: 'Next' })))
  expect(panel.getByText('Result uncertain')).toBeInTheDocument()
  expect(panel.queryByText(/secret-/)).not.toBeInTheDocument()
  await act(async () => fireEvent.click(panel.getByRole('button', { name: 'Reload QQ results' })))
  expect(pages).toEqual(['1', '1', '2', '2'])
})
