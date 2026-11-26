import { fireEvent, render, screen } from '@testing-library/react'
import {
  listAuditEvents,
  getManagedTasks,
  listManagedTaskHistory,
  listEsiRefreshTasks,
  listEsiRefreshStatuses,
} from '@/api/generated/client'
import { I18nProvider } from '@/i18n'
import { usePreferenceStore } from '@/stores'
import { SystemAuditPage } from './system-audit-page'
import { SystemTaskManagerPage } from './system-task-manager-page'

vi.mock('@/api/generated/client', () => ({
  listAuditEvents: vi.fn(),
  getManagedTasks: vi.fn(),
  listManagedTaskHistory: vi.fn(),
  listEsiRefreshTasks: vi.fn(),
  listEsiRefreshStatuses: vi.fn(),
  runEsiRefreshAll: vi.fn(),
  runEsiRefreshTask: vi.fn(),
  runEsiRefreshTaskByName: vi.fn(),
  runManagedTask: vi.fn(),
  updateManagedTaskSchedule: vi.fn(),
}))
vi.mock('@/hooks/use-corp-capability', () => ({
  useCorpCapability: () => ({ hasCapability: () => true }),
}))

beforeEach(() => {
  vi.resetAllMocks()
  usePreferenceStore.setState({ locale: 'zh-CN' })
})

test('task definitions, execution history and ESI statuses display Chinese labels', async () => {
  vi.mocked(getManagedTasks).mockResolvedValue({
    code: 200,
    msg: 'ok',
    data: ['admission_evidence_cleanup', 'audit_archive_daily'].map((name) => ({
      name,
      description: 'Backend description',
      category: 'system',
      type: 'recurring',
      runnable: true,
      cron_expr: '0 0 5 * * *',
      default_cron: '0 0 5 * * *',
      last_execution: null,
    })),
  })
  vi.mocked(listManagedTaskHistory).mockResolvedValue({
    code: 200,
    msg: 'ok',
    data: {
      list: [
        {
          id: 1,
          task_name: 'audit_archive_daily',
          trigger: 'cron',
          status: 'success',
          started_at: '2026-11-12T00:00:00Z',
        },
      ],
      total: 1,
      page: 1,
      pageSize: 20,
    },
  })
  vi.mocked(listEsiRefreshTasks).mockResolvedValue({ code: 200, msg: 'ok', data: [] })
  vi.mocked(listEsiRefreshStatuses).mockResolvedValue({
    code: 200,
    msg: 'ok',
    data: {
      list: [
        {
          task_name: 'character_wallet',
          description: 'Backend wallet description',
          character_id: 123,
          character_name: 'Pilot',
          priority: 50,
          status: 'success',
          last_run: '',
          next_run: '',
          error: '',
        },
      ],
      total: 1,
      page: 1,
      pageSize: 20,
    },
  })
  render(
    <I18nProvider>
      <SystemTaskManagerPage />
    </I18nProvider>
  )
  expect(await screen.findByText('申请证据清理')).toBeInTheDocument()
  expect(screen.getByText('审计日志每日归档')).toBeInTheDocument()
  expect(screen.getByText('清理已过保留期的已结案入团申请证据')).toBeInTheDocument()
  fireEvent.click(screen.getByRole('button', { name: '执行历史' }))
  expect((await screen.findAllByText('审计日志每日归档')).length).toBeGreaterThan(0)
  fireEvent.click(screen.getByRole('button', { name: 'ESI 状态' }))
  expect(await screen.findByText('人物钱包')).toBeInTheDocument()
  expect(screen.getByText('刷新人物钱包信息')).toBeInTheDocument()
})

test('audit list and details localize actions and categories while retaining identifiers', async () => {
  vi.mocked(listAuditEvents).mockResolvedValue({
    code: 200,
    msg: 'success',
    data: {
      list: [
        {
          id: 1,
          event_id: 'event-1',
          occurred_at: '2026-11-12T00:00:00Z',
          created_at: '',
          category: 'system',
          action: 'audit_archive_daily',
          actor_user_id: 1,
          target_user_id: 2,
          resource_type: 'task',
          resource_id: '3',
          result: 'success',
          request_id: '',
          ip: '',
          user_agent: '',
          details_json: '{}',
        },
      ],
      total: 1,
      page: 1,
      pageSize: 20,
    },
  })
  render(
    <I18nProvider>
      <SystemAuditPage />
    </I18nProvider>
  )
  expect(await screen.findByText('审计日志每日归档')).toBeInTheDocument()
  fireEvent.click(screen.getByRole('button', { name: '详情' }))
  expect(await screen.findByText('审计日志每日归档 (audit_archive_daily)')).toBeInTheDocument()
  expect(screen.getByText('系统 (system)')).toBeInTheDocument()
})
