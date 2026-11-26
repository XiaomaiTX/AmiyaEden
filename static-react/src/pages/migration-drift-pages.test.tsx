import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import type { ReactNode } from 'react'
import { I18nProvider } from '@/i18n'
import { DashboardFuelOfficerStructuresPage } from '@/pages/dashboard-fuel-officer-structures-page'
import { DashboardGalaxyRegistryPage } from '@/pages/dashboard-galaxy-registry-page'
import { useSessionStore } from '@/stores'

const success = vi.hoisted(() => <T,>(data: T) => ({ code: 200 as const, msg: '', data }))

vi.mock('@/api/generated/client', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/api/generated/client')>()),
  listMyAssignedCorpStructures: vi.fn().mockResolvedValue(
    success({
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
    })
  ),
  listSystems: vi.fn().mockResolvedValue(
    success({
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
    })
  ),
  listAdminSystems: vi.fn().mockResolvedValue(success([])),
  getAdminAnalytics: vi.fn().mockResolvedValue(
    success({
      range_start: '',
      range_end: '',
      current_snapshot: { idle_count: 1, busy_count: 0, overdue_count: 0 },
      recent_7d: {
        entry_count: 0,
        valid_count: 0,
        violation_count: 0,
        pending_count: 0,
        valid_rate: 0,
      },
      recent_30d: {
        entry_count: 0,
        valid_count: 0,
        violation_count: 0,
        pending_count: 0,
        valid_rate: 0,
      },
      top_systems: [],
      recent_violations: [],
    })
  ),
  createEntry: vi.fn(),
  endMyEntry: vi.fn(),
  forceEndAdminEntry: vi.fn(),
  revalidateAdminEntry: vi.fn(),
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
    const fetch = vi.spyOn(globalThis, 'fetch').mockImplementation(async (input) => {
      const url = new URL(String(input), window.location.href)
      if (url.pathname !== '/api/v1/dashboard/galaxy-registry/admin/entries') {
        throw new Error(`Unexpected API request: ${url.pathname}`)
      }
      return Response.json(success({ list: [], total: 0, page: 1, pageSize: 20 }))
    })
    renderPage(<DashboardGalaxyRegistryPage />)
    await waitFor(() => expect(screen.getByText('Jita')).toBeInTheDocument())
    fireEvent.click(screen.getByRole('tab', { name: '管理员' }))
    try {
      await waitFor(() => {
        expect(fetch).toHaveBeenCalledWith(
          '/api/v1/dashboard/galaxy-registry/admin/entries?current=1&size=20',
          expect.objectContaining({ method: 'GET' })
        )
      })
    } finally {
      fetch.mockRestore()
    }
    expect(screen.getByText('管理员登记记录')).toBeInTheDocument()
  })
})
