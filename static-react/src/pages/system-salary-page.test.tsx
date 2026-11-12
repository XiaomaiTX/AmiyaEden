import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { I18nProvider } from '@/i18n'
import { usePreferenceStore } from '@/stores'
import { SystemSalaryPage } from './system-salary-page'
import { fetchSalarySettings, updateSalarySettings } from '@/api/salary'
vi.mock('@/api/salary', () => ({ fetchSalarySettings: vi.fn(), updateSalarySettings: vi.fn() }))
vi.mock('@/feedback', () => ({ notifySuccess: vi.fn() }))
const settings = {
  recruitment: 10,
  welfare_delivery: 18,
  shop_delivery: 6.5,
  srp_payout: 10,
  fc_salary: 400,
  fc_salary_monthly_limit: 5,
}
beforeEach(() => {
  vi.resetAllMocks()
  usePreferenceStore.setState({ locale: 'zh-CN' })
  vi.mocked(fetchSalarySettings).mockResolvedValue(settings)
})
function open() {
  return render(
    <I18nProvider>
      <SystemSalaryPage />
    </I18nProvider>
  )
}
test('loads independent values and preserves explicit zero when saving', async () => {
  vi.mocked(updateSalarySettings).mockImplementation(async (value) => value)
  open()
  const input = await screen.findByLabelText('招新（标记已邀请）')
  fireEvent.change(input, { target: { value: '0' } })
  fireEvent.click(screen.getByRole('button', { name: '保存' }))
  await waitFor(() =>
    expect(updateSalarySettings).toHaveBeenCalledWith({ ...settings, recruitment: 0 })
  )
  expect(screen.getByLabelText('商城发货')).toHaveValue(6.5)
})
test('failed loading cannot save fallback defaults and can be retried', async () => {
  vi.mocked(fetchSalarySettings).mockRejectedValueOnce(new Error('offline'))
  open()
  await screen.findByRole('alert')
  expect(screen.queryByRole('button', { name: '保存' })).not.toBeInTheDocument()
  fireEvent.click(screen.getByRole('button', { name: '刷新' }))
  await screen.findByLabelText('福利发放')
  expect(updateSalarySettings).not.toHaveBeenCalled()
})
