import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { I18nProvider } from '@/i18n'
import { DashboardNpcKillsPage } from './dashboard-npc-kills-page'

describe('dashboard npc kills page', () => {
  afterEach(() => {
    vi.useRealTimers()
    vi.restoreAllMocks()
  })

  test('loads the current month by default and restores it on reset', async () => {
    vi.useFakeTimers({ toFake: ['Date'] })
    vi.setSystemTime(new Date(2028, 1, 1))
    const makeResponse = () =>
      new Response(
        JSON.stringify({
          code: 200,
          msg: 'ok',
          data: {
            summary: {
              total_bounty: 1234567.89,
              total_ess: 0,
              total_incursion: 0,
              total_mission: 0,
              total_tax: 12345,
              actual_income: 1222222.89,
              total_records: 3,
            },
            members: [
              {
                user_id: 31,
                display_name: 'Amiya Commander',
                character_count: 2,
                total_bounty: 40001000,
                total_ess: 0,
                total_incursion: 40000000,
                total_mission: 0,
                total_tax: 100,
                actual_income: 40000900,
                record_count: 3,
              },
            ],
            by_system: [
              {
                solar_system_id: 30000142,
                solar_system_name: 'Jita',
                count: 2,
                amount: 1000,
              },
            ],
            trend: [
              {
                date: '2028-02-01',
                amount: 1000,
                count: 2,
              },
            ],
          },
        }),
        { status: 200, headers: { 'Content-Type': 'application/json' } }
      )
    const fetchSpy = vi.spyOn(globalThis, 'fetch').mockImplementation(async () => makeResponse())

    render(
      <I18nProvider>
        <DashboardNpcKillsPage />
      </I18nProvider>
    )

    await waitFor(() => {
      expect(screen.getByText('1,234,567.89')).toBeInTheDocument()
    })

    expect(screen.getByText('Amiya Commander')).toBeInTheDocument()
    expect(screen.getByText('Jita')).toBeInTheDocument()

    const startDate = screen.getByLabelText('开始日期')
    const endDate = screen.getByLabelText('结束日期')
    const monthPayload = { start_date: '2028-02-01', end_date: '2028-02-29' }
    const reportBodies = () =>
      fetchSpy.mock.calls
        .filter(([url]) => url === '/api/v1/system/npc-kills')
        .map(([, init]) => JSON.parse(String(init?.body)))

    expect(startDate).toHaveValue(monthPayload.start_date)
    expect(endDate).toHaveValue(monthPayload.end_date)
    expect(reportBodies()).toEqual([monthPayload])

    fireEvent.change(startDate, { target: { value: '2028-01-01' } })
    fireEvent.click(screen.getByRole('button', { name: '搜索' }))
    await waitFor(() => {
      expect(reportBodies()).toEqual([monthPayload, { ...monthPayload, start_date: '2028-01-01' }])
    })

    fireEvent.click(screen.getByRole('button', { name: '重置' }))
    await waitFor(() => {
      expect(reportBodies()).toEqual([
        monthPayload,
        { ...monthPayload, start_date: '2028-01-01' },
        monthPayload,
      ])
    })
    expect(startDate).toHaveValue(monthPayload.start_date)
    expect(endDate).toHaveValue(monthPayload.end_date)
  })
})
