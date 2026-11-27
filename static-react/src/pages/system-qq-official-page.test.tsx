import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type {
  QQOfficialGroup,
  QQOfficialJoinRequest,
  QQOfficialStatus,
  QQOfficialTaskDetail,
} from '@/api/generated/client'
import { ensureLocaleLoaded, I18nContext, resolveLocaleText } from '@/i18n/context'
import { formatTime } from '@/lib/format'
import { SystemQQOfficialPage } from './system-qq-official-page'

vi.mock('@/hooks/use-corp-capability', () => ({
  useCorpCapability: () => ({ hasCapability: (cap: string) => cap === 'system.manage' }),
}))

const root = '/api/v1/system/qq-official'
const group: QQOfficialGroup = {
  id: '998358e6-13ca-4964-a1bd-f205983aaa0e',
  alias: 'Local group',
  present: true,
  authorized: true,
  notification_enabled: true,
  test_verified: false,
  approval_enabled: true,
  push_state: 'allowed',
  approval_capability: 'available',
}
const task: QQOfficialTaskDetail = {
  id: 'd9f346cd-184a-478e-a0a9-ccc9e4074920',
  group_id: group.id,
  status: 'queued',
  kind: 'send_text',
  source: 'test',
  created_at: '2026-11-26T00:00:00Z',
  next_run_at: '2026-11-26T00:00:00Z',
  expires_at: '2099-01-01T00:00:00Z',
  attempts: [],
}
const join: QQOfficialJoinRequest = {
  id: 'bf44effb-df4b-4186-9a88-95d31f87da0b',
  group_id: group.id,
  username: 'Applicant Pilot',
  status: 'pending',
  applied_at: '2026-11-26T00:00:00Z',
  expires_at: '2099-01-01T00:00:00Z',
}
const baseline: QQOfficialStatus = {
  app_id: '10001',
  has_app_secret: true,
  enabled: true,
  paused: false,
  configured: true,
  certified: false,
  approval_writes_enabled: true,
  token_state: 'ready',
  gateway_state: 'connecting',
  queue_depth: 0,
  attempts_last_minute: 0,
  messages_last_minute: 0,
  message_budget: 24,
  cooldowns: [],
}
const ok = (data: unknown, status = 200) =>
  Response.json({ code: 200, msg: 'success', data }, { status })
const page = (list: unknown[], pageSize: number) => ({
  list,
  total: list.length,
  page: 1,
  pageSize,
})
const receipt = () =>
  ok({ task_id: task.id, status: task.status, expires_at: task.expires_at }, 202)

function open(
  status = baseline,
  handle?: (url: URL, init: RequestInit) => Response | Promise<Response> | undefined
) {
  const mock = vi.fn(async (input: RequestInfo | URL, init: RequestInit = {}) => {
    const url = new URL(String(input), 'https://local.invalid')
    const result = await handle?.(url, init)
    if (result) return result
    if (url.pathname === `${root}/status`) return ok(status)
    if (url.pathname === `${root}/groups`) return ok(page([group], 20))
    if (url.pathname === `${root}/tasks/${task.id}`) return ok(task)
    if (url.pathname === `${root}/groups/${group.id}/join-requests`) return ok(page([join], 200))
    throw new Error(`Unexpected request: ${url.pathname}`)
  })
  vi.stubGlobal('fetch', mock)
  render(
    <I18nContext.Provider
      value={{ locale: 'en-US', t: (key, vars) => resolveLocaleText('en-US', key, vars) }}
    >
      <SystemQQOfficialPage />
    </I18nContext.Provider>
  )
  return mock
}

beforeEach(async () => {
  await ensureLocaleLoaded('en-US')
})

afterEach(() => vi.unstubAllGlobals())

test('shows gateway health separately from token state and refreshes the last group event', async () => {
  const lastEvent = '2026-11-26T11:00:00Z'
  let status = baseline
  open(baseline, (url) => {
    if (url.pathname === `${root}/status`) return ok(status)
  })
  const runtime = within(screen.getByRole('region', { name: 'Runtime status' }))
  const gateway = await runtime.findByText('QQ WebSocket gateway')
  const event = runtime.getByText('Last recorded group event')
  expect(gateway.nextElementSibling).toHaveTextContent('Connecting')
  expect(event.nextElementSibling).toHaveTextContent(/^-$/u)
  expect(runtime.getByText('Token state').nextElementSibling).toHaveTextContent('Ready')
  expect(runtime.queryByText('Last callback')).not.toBeInTheDocument()

  status = { ...baseline, gateway_state: 'connected', last_event_at: lastEvent }
  await userEvent.setup().click(runtime.getByRole('button', { name: 'Reload local data' }))
  await waitFor(() => expect(gateway.nextElementSibling).toHaveTextContent('Connected'))
  expect(event.nextElementSibling).toHaveTextContent(formatTime(lastEvent))
})

test('one-click pause sends only paused without changing enabled or certified', async () => {
  let paused = false
  const mock = open(baseline, (url, init) => {
    if (url.pathname === `${root}/settings` && init.method === 'PUT') {
      paused = true
      return ok({ ...baseline, paused })
    }
    if (url.pathname === `${root}/status`) return ok({ ...baseline, paused })
  })
  const user = userEvent.setup()
  await user.click(await screen.findByRole('button', { name: 'Pause dispatch now' }))
  await screen.findByRole('button', { name: 'Resume dispatch' })
  const settings = mock.mock.calls.filter(([url]) => String(url) === `${root}/settings`)
  expect(settings).toHaveLength(1)
  expect(JSON.parse(String(settings[0][1]?.body))).toEqual({ paused: true })
})

function settingsBodies(mock: ReturnType<typeof open>) {
  return mock.mock.calls
    .filter(([url]) => String(url) === `${root}/settings`)
    .map(([, init]) => JSON.parse(String(init?.body)))
}

test('switching apps requires fresh credentials and confirmation, clears old groups, and requires separate enable', async () => {
  const nextID = '10002'
  const secret = 'fixture-new-secret'
  let current = baseline
  const mock = open(baseline, (url, init) => {
    if (url.pathname === `${root}/status`) {
      return ok({ ...current, app_secret: 'unexpected-server-value' })
    }
    if (url.pathname === `${root}/settings`) {
      const patch = JSON.parse(String(init.body))
      current = patch.app_secret
        ? { ...baseline, app_id: nextID, enabled: false, approval_writes_enabled: false }
        : { ...current, enabled: true }
      return ok(current)
    }
    if (url.pathname === `${root}/groups` && current.app_id === nextID) {
      return ok(page([{ ...group, alias: 'New app group' }], 20))
    }
  })
  const user = userEvent.setup()
  const appID = await screen.findByLabelText('QQ AppID')
  const password = () => screen.getByLabelText('QQ AppSecret')
  const save = () => screen.getByRole('button', { name: 'Save credentials' })
  expect(appID).toHaveValue(baseline.app_id)
  expect(password()).toHaveAttribute('type', 'password')
  expect(password()).toHaveValue('')
  expect(save()).toBeDisabled()
  await user.click(await screen.findByRole('button', { name: 'Join requests' }))
  await screen.findByText(join.username)
  await user.clear(appID)
  await user.type(appID, nextID)
  expect(save()).toBeDisabled()
  expect(screen.getByText(/Enter a new AppSecret/)).toBeInTheDocument()

  await user.type(password(), secret)
  await user.click(save())
  let dialog = screen.getByRole('dialog', { name: 'Save credentials' })
  expect(dialog).not.toHaveTextContent(secret)
  expect(settingsBodies(mock)).toEqual([])
  await user.click(within(dialog).getByRole('button', { name: 'Cancel' }))
  expect(password()).toHaveValue('')

  await user.type(password(), secret)
  await user.click(save())
  dialog = screen.getByRole('dialog', { name: 'Save credentials' })
  await user.click(within(dialog).getByRole('button', { name: 'Confirm' }))
  await screen.findByText('New app group')
  expect(screen.queryByText(group.alias)).not.toBeInTheDocument()
  expect(settingsBodies(mock)).toEqual([{ app_id: nextID, app_secret: secret }])
  expect(password()).toHaveValue('')
  expect(screen.getByRole('switch', { name: 'Allow join approval writes' })).not.toBeChecked()
  const enabled = screen.getByRole('checkbox', { name: 'Enabled' })
  expect(enabled).not.toBeChecked()

  await user.click(screen.getByRole('tab', { name: 'Join requests' }))
  expect(screen.getByText(/Open join requests from a group/)).toBeInTheDocument()
  expect(screen.queryByText(join.username)).not.toBeInTheDocument()
  await user.click(enabled)
  await waitFor(() => expect(enabled).toBeChecked())
  expect(settingsBodies(mock)).toEqual([{ app_id: nextID, app_secret: secret }, { enabled: true }])
})

test('reports unreadable stored credentials and blocks enabling until a confirmed replacement', async () => {
  const secret = 'fixture-replacement'
  const mock = open(
    {
      ...baseline,
      configured: false,
      enabled: false,
      approval_writes_enabled: false,
      token_state: 'credentials_unavailable',
    },
    (url) => {
      if (url.pathname === `${root}/settings`) {
        return ok({ ...baseline, enabled: false, approval_writes_enabled: false })
      }
    }
  )
  const user = userEvent.setup()
  expect(await screen.findByText('Credentials unavailable')).toBeInTheDocument()
  expect(screen.getByText(/The stored AppSecret cannot be read/)).toBeInTheDocument()
  const enabled = () => screen.getByRole('checkbox', { name: 'Enabled' })
  expect(enabled()).toBeDisabled()
  await user.type(screen.getByLabelText('QQ AppSecret'), secret)
  await user.click(screen.getByRole('button', { name: 'Save credentials' }))
  const confirm = within(screen.getByRole('dialog', { name: 'Save credentials' }))
  await user.click(confirm.getByRole('button', { name: 'Confirm' }))
  await waitFor(() => expect(enabled()).toBeEnabled())
  expect(enabled()).not.toBeChecked()
  expect(settingsBodies(mock)).toEqual([{ app_id: baseline.app_id, app_secret: secret }])
  expect(screen.getByLabelText('QQ AppSecret')).toHaveValue('')
})

test('global approval writes require separate consent and turning off refreshes the task ledger', async () => {
  let current = { ...baseline, approval_writes_enabled: false }
  const mock = open(current, (url, init) => {
    if (url.pathname === `${root}/settings`) {
      current = { ...current, ...JSON.parse(String(init.body)) }
      return ok(current)
    }
    if (url.pathname === `${root}/tasks`) return ok(page([task], 25))
  })
  const user = userEvent.setup()
  const approval = () => screen.getByRole('switch', { name: 'Allow join approval writes' })
  const ledgerReads = () =>
    mock.mock.calls.filter(([input]) => String(input).startsWith(`${root}/tasks?`))
  await user.click(await screen.findByRole('tab', { name: 'Task ledger' }))
  await waitFor(() => expect(ledgerReads()).toHaveLength(1))
  await user.click(approval())
  const dialog = within(screen.getByRole('dialog', { name: 'Enable approval writes' }))
  expect(settingsBodies(mock)).toEqual([])
  await user.click(dialog.getByRole('button', { name: 'Confirm' }))
  await waitFor(() => expect(approval()).toBeChecked())
  await waitFor(() => expect(ledgerReads()).toHaveLength(2))
  await user.click(approval())
  await waitFor(() => expect(approval()).not.toBeChecked())
  await waitFor(() => expect(ledgerReads()).toHaveLength(3))
  expect(settingsBodies(mock)).toEqual([
    { approval_writes_enabled: true },
    { approval_writes_enabled: false },
  ])
})

test('reuses idempotency key after a lost receipt and opens the accepted task', async () => {
  let attempt = 0
  const mock = open(baseline, (url) => {
    if (url.pathname === `${root}/groups/${group.id}/test-message`) {
      if (++attempt === 1) throw new TypeError('local test transport failure')
      return receipt()
    }
  })
  const user = userEvent.setup()
  const testButton = () => screen.getByRole('button', { name: 'Queue test message' })
  const submissions = () => mock.mock.calls.filter(([url]) => String(url).endsWith('/test-message'))
  await waitFor(() => expect(testButton()).toBeEnabled())
  await user.click(testButton())
  await waitFor(() => expect(submissions()).toHaveLength(1))
  await waitFor(() => expect(testButton()).toBeEnabled())
  await user.click(testButton())
  await waitFor(() => expect(submissions()).toHaveLength(2))
  const detail = await screen.findByRole('dialog', { name: 'View task' })
  expect(await within(detail).findByText(task.id)).toBeInTheDocument()
  const bodies = submissions().map(([, init]) => JSON.parse(String(init?.body)))
  expect(bodies).toHaveLength(2)
  expect(bodies[0]).toEqual({ idempotency_key: expect.stringMatching(/^[0-9a-f-]{36}$/) })
  expect(bodies[1]).toEqual(bodies[0])
})

test('queues a join decision only after confirmation, with no operator text', async () => {
  const mock = open(baseline, (url) => {
    if (url.pathname === `${root}/join-requests/${join.id}/decision`) return receipt()
  })
  const user = userEvent.setup()
  const groups = await screen.findByRole('grid', { name: 'Data table' })
  await user.click(await within(groups).findByRole('button', { name: 'Join requests' }))
  await user.click(await screen.findByRole('button', { name: 'Approve' }))
  const confirm = await screen.findByRole('dialog', { name: 'Approve' })
  expect(within(confirm).getByText(/Applicant Pilot/)).toBeInTheDocument()
  expect(mock.mock.calls.some(([url]) => String(url).endsWith('/decision'))).toBe(false)
  await user.click(within(confirm).getByRole('button', { name: 'Confirm' }))
  await screen.findByRole('dialog', { name: 'View task' })
  const [call] = mock.mock.calls.filter(([url]) => String(url).endsWith('/decision'))
  expect(JSON.parse(String(call[1]?.body))).toEqual({
    decision: 'approve',
    idempotency_key: expect.any(String),
  })
  expect(mock.mock.calls.some(([url]) => String(url).includes('size=200'))).toBe(true)
})
