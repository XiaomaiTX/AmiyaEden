import { render, screen, waitFor } from '@testing-library/react'
import type { ReactNode } from 'react'
import { I18nProvider } from '@/i18n'
import { DashboardFuelOfficerStructuresPage } from '@/pages/dashboard-fuel-officer-structures-page'
import { DashboardGalaxyRegistryPage } from '@/pages/dashboard-galaxy-registry-page'
import { SystemQQGovernancePage } from '@/pages/system-qq-governance-page'
import { useSessionStore } from '@/stores'

vi.mock('@/api/corporation-structures', () => ({
  fetchMyAssignedCorporationStructures: vi.fn().mockResolvedValue({
    items: [
      {
        structure_id: 1,
        corporation_name: 'Amiya Corp',
        name: 'Eden Keepstar',
        system_name: 'Jita',
        state: 'shield_vulnerable',
        fuel_remaining: 24,
        fuel_per_hour: 10,
        fuel_to_month_end: 100,
        fuel_estimate_incomplete: false,
        state_timer_end: null,
        updated_at: 1_700_000_000,
      },
    ],
    total: 1,
    page: 1,
    page_size: 20,
  }),
}))

vi.mock('@/api/galaxy-registry', () => ({
  fetchGalaxyRegistrySystems: vi.fn().mockResolvedValue({
    summary: { idle_count: 1, busy_count: 0, overdue_count: 0 },
    items: [
      {
        system_config_id: 1,
        solar_system_id: 30_000_142,
        solar_system_name: 'Jita',
        region_name: 'The Forge',
        constellation_name: 'Kimotoro',
        security: 0.9,
        note: '',
        min_bounty_amount: 10_000_000,
        is_enabled: true,
        status: 'idle',
        active_entry: null,
      },
    ],
  }),
  fetchAdminGalaxyRegistryEntries: vi
    .fn()
    .mockResolvedValue({ list: [], total: 0, page: 1, pageSize: 20 }),
  createGalaxyRegistryEntry: vi.fn(),
  endGalaxyRegistryEntry: vi.fn(),
  forceEndAdminGalaxyRegistryEntry: vi.fn(),
  revalidateAdminGalaxyRegistryEntry: vi.fn(),
}))

vi.mock('@/api/qq-governance', () => ({
  fetchQQGovernanceGroups: vi.fn().mockResolvedValue([
    {
      group_id: 10001,
      group_name: 'Eden',
      enabled: true,
      member_count: 20,
      max_member_count: 200,
      bot_is_admin: true,
      valid_count: 20,
      review_count: 0,
      invalid_candidate_count: 0,
      invalid_confirmed_count: 0,
      snapshot_state: 'fresh',
      reconcile_run_status: 'completed',
      reconcile_expected: 20,
      reconcile_processed: 20,
      reconcile_failed: 0,
    },
  ]),
  fetchQQGovernancePolicies: vi.fn().mockResolvedValue([]),
  fetchQQGovernanceTasks: vi
    .fn()
    .mockResolvedValue({ list: [], total: 0, page: 1, page_size: 200 }),
  fetchQQGovernanceAlerts: vi
    .fn()
    .mockResolvedValue({ list: [], total: 0, page: 1, page_size: 200 }),
  fetchQQGovernanceMetrics: vi.fn().mockResolvedValue({
    window_minutes: 60,
    created: 1,
    succeeded: 1,
    failed: 0,
    dead: 0,
    failure_rate: 0,
    connected: true,
    risk_level: 0,
  }),
  fetchQQGovernanceConnection: vi.fn().mockResolvedValue({
    connected: true,
    risk_level: 0,
  }),
  fetchQQGovernanceSettings: vi.fn().mockResolvedValue({
    scan_interval_minutes: 60,
    mismatch_confirmations: 3,
    mismatch_observation_hours: 24,
  }),
  acknowledgeQQGovernanceAlert: vi.fn(),
  recoverQQGovernanceDisconnectedTasks: vi.fn(),
  resetQQGovernanceRisk: vi.fn(),
  retryQQGovernanceTask: vi.fn(),
  triggerQQGovernanceReconcile: vi.fn(),
  updateQQGovernanceSettings: vi.fn(),
}))

function renderPage(page: ReactNode) {
  return render(<I18nProvider>{page}</I18nProvider>)
}

describe('post-freeze migration pages', () => {
  beforeEach(() => {
    useSessionStore.getState().setSessionSnapshot({
      isLoggedIn: true,
      roles: ['super_admin'],
      corpCapabilities: ['menu.dashboard', 'system.manage'],
    })
  })

  test('renders the assigned fuel-officer structure list', async () => {
    renderPage(<DashboardFuelOfficerStructuresPage />)
    await waitFor(() => expect(screen.getByText('Eden Keepstar')).toBeInTheDocument())
  })

  test('renders galaxy registry status and admin entry area', async () => {
    renderPage(<DashboardGalaxyRegistryPage />)
    await waitFor(() => expect(screen.getByText('Jita')).toBeInTheDocument())
    expect(screen.getByText('管理员登记记录')).toBeInTheDocument()
  })

  test('renders QQ governance health and persisted group snapshot', async () => {
    renderPage(<SystemQQGovernancePage />)
    await waitFor(() => expect(screen.getByText('Eden')).toBeInTheDocument())
    expect(screen.getByText(/连接状态/)).toBeInTheDocument()
  })
})
