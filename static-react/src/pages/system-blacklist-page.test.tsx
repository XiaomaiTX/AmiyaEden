import userEvent from '@testing-library/user-event'
import { render, screen, waitFor, within } from '@testing-library/react'
import { latestRefresh, listBlacklist, refresh as refreshBlacklist } from '@/api/generated/client'
import { notifyError, notifySuccess } from '@/feedback'
import { I18nProvider } from '@/i18n'
import { usePreferenceStore } from '@/stores'
import type { BlacklistEntry, BlacklistRefreshSummary } from '@/api/generated/client'
import { SystemBlacklistPage } from './system-blacklist-page'

vi.mock('@/api/generated/client', () => ({
  listBlacklist: vi.fn(),
  latestRefresh: vi.fn(),
  refresh: vi.fn(),
}))
vi.mock('@/feedback', () => ({ notifyError: vi.fn(), notifySuccess: vi.fn() }))

const entries: BlacklistEntry[] = [
  {
    kind: 'character',
    value: '90000001',
    display: 'Test Pilot',
    reason: 'Scam\nSecond line',
    updated_at: '2026-11-12T00:00:00Z',
  },
  { kind: 'qq', value: '10000001', display: '10000001', reason: 'Spy', updated_at: '' },
  { kind: 'discord', value: 'tester#0001', display: 'Tester#0001', reason: '', updated_at: '' },
]

const summary: BlacklistRefreshSummary = {
  finished_at: '2026-11-12T08:00:00Z',
  operator_user_id: 1,
  operator_name: 'Admin',
  characters: 1,
  qqs: 1,
  discords: 1,
  added: 3,
  updated: 0,
  removed: 2,
  new_unresolved: 1,
  skipped_unresolved: 4,
  lookup_failed: 0,
  next_refresh_at: '2020-01-01T00:00:00Z',
}

function success<T>(data: T) {
  return { code: 200 as const, msg: 'ok', data }
}

function failure(msg: string) {
  return { code: 400 as const, msg, data: null }
}

function page(list: BlacklistEntry[], total = list.length) {
  return { list, total, page: 1, pageSize: 20 }
}

function open() {
  return render(
    <I18nProvider>
      <SystemBlacklistPage />
    </I18nProvider>
  )
}

beforeEach(() => {
  vi.resetAllMocks()
  usePreferenceStore.setState({ locale: 'zh-CN' })
  vi.mocked(listBlacklist).mockResolvedValue(success(page(entries)))
  vi.mocked(latestRefresh).mockResolvedValue(success(null))
})

test('renders all three kinds and the never-refreshed state', async () => {
  open()

  expect(await screen.findByText('Test Pilot（90000001）')).toBeInTheDocument()
  expect(screen.getByText('Scam\nSecond line', { normalizer: (value) => value })).toHaveClass(
    'whitespace-pre-line'
  )
  const table = screen.getByRole('grid', { name: 'Data table' })
  expect(within(table).getByText('人物')).toBeInTheDocument()
  expect(within(table).getByText('QQ')).toBeInTheDocument()
  expect(within(table).getByText('Discord')).toBeInTheDocument()
  expect(within(table).getByText('10000001')).toBeInTheDocument()
  expect(within(table).getByText('Tester#0001')).toBeInTheDocument()
  expect(
    await screen.findByText('尚未刷新过黑名单，点击「更新黑名单」开始导入。')
  ).toBeInTheDocument()
  expect(listBlacklist).toHaveBeenCalledWith({ current: 1, size: 20 })
})

test('sends the kind filter and trimmed keyword when searching', async () => {
  const user = userEvent.setup()
  open()
  await screen.findByText('Test Pilot（90000001）')

  await user.click(screen.getByRole('button', { name: /全部类型/ }))
  await user.click(screen.getByRole('option', { name: 'QQ' }))
  await user.type(screen.getByRole('textbox', { name: /搜索人物名/ }), '  10000001  ')
  await user.click(screen.getByRole('button', { name: '搜索' }))

  await waitFor(() =>
    expect(listBlacklist).toHaveBeenLastCalledWith({
      current: 1,
      size: 20,
      kind: 'qq',
      keyword: '10000001',
    })
  )
})

test('shows loading while refreshing, then the summary and a reloaded list', async () => {
  const user = userEvent.setup()
  let finish: (value: BlacklistRefreshSummary) => void = () => {}
  vi.mocked(refreshBlacklist).mockReturnValue(
    new Promise((resolve) => {
      finish = (value) => resolve(success(value))
    })
  )
  open()
  await screen.findByText('Test Pilot（90000001）')
  expect(listBlacklist).toHaveBeenCalledTimes(1)

  await user.click(screen.getByRole('button', { name: '更新黑名单' }))
  expect(screen.getByRole('button', { name: '正在更新…' })).toBeDisabled()

  expect(refreshBlacklist).toHaveBeenCalledWith({ signal: expect.any(AbortSignal) })
  finish(summary)
  await waitFor(() =>
    expect(notifySuccess).toHaveBeenCalledWith('黑名单已更新：新增 3，更新 0，移除 2')
  )
  expect(await screen.findByText('人物 1，QQ 1，Discord 1')).toBeInTheDocument()
  expect(screen.getByText('新发现 1，已跳过 4')).toBeInTheDocument()
  expect(screen.getByText('Admin')).toBeInTheDocument()
  expect(screen.queryByText('本次查询失败')).not.toBeInTheDocument()
  expect(screen.getByRole('button', { name: '更新黑名单' })).toBeEnabled()
  await waitFor(() => expect(listBlacklist).toHaveBeenCalledTimes(2))
})

test.each([
  {
    locale: 'zh-CN' as const,
    button: '更新黑名单',
    failure: '7 个人名本次查询失败，下次刷新重试',
    success: '黑名单已更新：新增 3，更新 0，移除 2',
  },
  {
    locale: 'en-US' as const,
    button: 'Update blacklist',
    failure: '7 names could not be looked up this time and will be retried on the next refresh',
    success: 'Blacklist updated: 3 added, 0 updated, 2 removed',
  },
])('shows deferred names in the saved summary and success message ($locale)', async (copy) => {
  const user = userEvent.setup()
  usePreferenceStore.setState({ locale: copy.locale })
  const deferredSummary = { ...summary, lookup_failed: 7, next_refresh_at: null }
  vi.mocked(latestRefresh).mockResolvedValue(success(deferredSummary))
  vi.mocked(refreshBlacklist).mockResolvedValue(success(deferredSummary))
  open()

  expect(await screen.findByText(copy.failure)).toBeInTheDocument()
  await user.click(screen.getByRole('button', { name: copy.button }))
  await waitFor(() =>
    expect(notifySuccess).toHaveBeenCalledWith(`${copy.success}\n${copy.failure}`)
  )
  expect(screen.getByText(copy.failure)).toBeInTheDocument()

  // A later successful refresh clears the previous failed-name warning.
  vi.mocked(refreshBlacklist).mockResolvedValue(success(summary))
  await user.click(screen.getByRole('button', { name: copy.button }))
  await waitFor(() => expect(notifySuccess).toHaveBeenLastCalledWith(copy.success))
  expect(screen.queryByText(copy.failure)).not.toBeInTheDocument()
})

test('disables refresh during the complete-refresh cooldown and shows its end time', async () => {
  vi.mocked(latestRefresh).mockResolvedValue(
    success({
      ...summary,
      next_refresh_at: new Date(Date.now() + 24 * 60 * 60 * 1000).toISOString(),
    })
  )
  open()
  expect(await screen.findByText('下次可刷新（完整刷新间隔 24 小时）')).toBeInTheDocument()
  expect(screen.getByRole('button', { name: '更新黑名单' })).toBeDisabled()
  expect(refreshBlacklist).not.toHaveBeenCalled()
})

test('localizes backend error keys and falls back for unknown errors', async () => {
  const user = userEvent.setup()
  vi.mocked(listBlacklist).mockResolvedValue(failure('blacklist.errors.invalidQuery'))
  vi.mocked(refreshBlacklist)
    .mockResolvedValueOnce(failure('blacklist.errors.refreshInProgress'))
    .mockResolvedValueOnce(failure('blacklist.errors.refreshCooldown'))
    .mockRejectedValueOnce(new Error('Failed to fetch'))
  open()

  expect(await screen.findByText('查询参数无效。')).toBeInTheDocument()

  await user.click(screen.getByRole('button', { name: '更新黑名单' }))
  await waitFor(() => expect(notifyError).toHaveBeenCalledWith('黑名单正在刷新中，请稍后再试。'))
  await user.click(screen.getByRole('button', { name: '更新黑名单' }))
  await waitFor(() =>
    expect(notifyError).toHaveBeenLastCalledWith('黑名单已完整刷新，请在上次完成 24 小时后再试。')
  )
  await user.click(screen.getByRole('button', { name: '更新黑名单' }))
  await waitFor(() => expect(notifyError).toHaveBeenLastCalledWith('黑名单操作失败，请重试。'))
})
