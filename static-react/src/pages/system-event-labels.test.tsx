import { fireEvent, render, screen } from '@testing-library/react'
import { fetchAuditEvents } from '@/api/audit'
import {
  fetchTasks,
  fetchTaskHistory,
  fetchESIRefreshTasks,
  fetchESIRefreshStatuses,
} from '@/api/task-manager'
import { I18nProvider } from '@/i18n'
import { usePreferenceStore } from '@/stores'
import { SystemAuditPage } from './system-audit-page'
import { SystemTaskManagerPage } from './system-task-manager-page'

vi.mock('@/api/audit', () => ({ fetchAuditEvents: vi.fn() }))
vi.mock('@/api/task-manager', () => ({
  fetchTasks: vi.fn(),
  fetchTaskHistory: vi.fn(),
  fetchESIRefreshTasks: vi.fn(),
  fetchESIRefreshStatuses: vi.fn(),
  runESIRefreshAll: vi.fn(),
  runESIRefreshTask: vi.fn(),
  runESIRefreshTaskByName: vi.fn(),
  runTask: vi.fn(),
  updateTaskSchedule: vi.fn(),
}))
vi.mock('@/hooks/use-corp-capability', () => ({
  useCorpCapability: () => ({ hasCapability: () => true }),
}))

beforeEach(() => {
  vi.resetAllMocks()
  usePreferenceStore.setState({ locale: 'zh-CN' })
})

test('task definitions, execution history and ESI statuses display Chinese labels', async () => {
  vi.mocked(fetchTasks).mockResolvedValue(
    ['admission_evidence_cleanup', 'audit_archive_daily', 'qq_group_governance_reconcile'].map(
      (name) => ({
        name,
        description: 'Backend description',
        category: 'system',
        type: 'recurring',
        runnable: true,
        cron_expr: '0 0 5 * * *',
        default_cron: '0 0 5 * * *',
      })
    )
  )
  vi.mocked(fetchTaskHistory).mockResolvedValue({
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
  })
  vi.mocked(fetchESIRefreshTasks).mockResolvedValue([])
  vi.mocked(fetchESIRefreshStatuses).mockResolvedValue({
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
  })
  render(
    <I18nProvider>
      <SystemTaskManagerPage />
    </I18nProvider>
  )
  expect(await screen.findByText('申请证据清理')).toBeInTheDocument()
  expect(screen.getByText('审计日志每日归档')).toBeInTheDocument()
  expect(screen.getByText('QQ 群成员巡检')).toBeInTheDocument()
  expect(screen.getByText('清理已过保留期的已结案入团申请证据')).toBeInTheDocument()
  fireEvent.click(screen.getByRole('button', { name: '执行历史' }))
  expect((await screen.findAllByText('审计日志每日归档')).length).toBeGreaterThan(0)
  fireEvent.click(screen.getByRole('button', { name: 'ESI 状态' }))
  expect(await screen.findByText('人物钱包')).toBeInTheDocument()
  expect(screen.getByText('刷新人物钱包信息')).toBeInTheDocument()
})

test('audit list and details localize actions and categories while retaining identifiers', async () => {
  vi.mocked(fetchAuditEvents).mockResolvedValue({
    list: [
      {
        id: 1,
        event_id: 'event-1',
        occurred_at: '2026-11-12T00:00:00Z',
        created_at: '',
        category: 'system',
        action: 'qq_governance_task_retry',
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
  })
  render(
    <I18nProvider>
      <SystemAuditPage />
    </I18nProvider>
  )
  expect(await screen.findByText('重试 QQ 群治理任务')).toBeInTheDocument()
  fireEvent.click(screen.getByRole('button', { name: '详情' }))
  expect(
    await screen.findByText('重试 QQ 群治理任务 (qq_governance_task_retry)')
  ).toBeInTheDocument()
  expect(screen.getByText('系统 (system)')).toBeInTheDocument()
})
