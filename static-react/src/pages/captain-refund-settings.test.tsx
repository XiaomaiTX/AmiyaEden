import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, test, vi } from 'vitest'
import { CaptainRefundSettingsPanel } from '@/pages/captain-refund-settings'
import { getRefundSettings, saveRefundSettings } from '@/api/captain-refund'
import type { RefundSettings } from '@/types/api/captain-refund'

vi.mock('@/i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))
vi.mock('@/api/captain-refund', () => ({
  getRefundSettings: vi.fn(),
  saveRefundSettings: vi.fn(),
}))

const settings: RefundSettings = {
  max_total_sp: 10_000_000,
  max_age_days: 90,
  tax_rate: 30,
  allowed_system_ids: [30_000_001],
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(getRefundSettings).mockResolvedValue(settings)
  vi.mocked(saveRefundSettings).mockImplementation(async (value) => value)
})

describe('captain refund settings', () => {
  test('rejects values outside the backend limits with a localized message', async () => {
    render(<CaptainRefundSettingsPanel />)
    const taxRate = await screen.findByLabelText('refund.tax_rate')
    fireEvent.change(taxRate, { target: { value: '101' } })
    fireEvent.click(screen.getByRole('button', { name: 'common.save' }))

    expect(await screen.findByRole('alert')).toHaveTextContent('refund.settingsInvalid')
    expect(saveRefundSettings).not.toHaveBeenCalled()
  })

  test('deduplicates configured solar systems before saving', async () => {
    render(<CaptainRefundSettingsPanel />)
    const allowedSystems = await screen.findByLabelText('refund.allowedSystems')
    fireEvent.change(allowedSystems, { target: { value: '30000001\n30000001' } })
    fireEvent.click(screen.getByRole('button', { name: 'common.save' }))

    await waitFor(() =>
      expect(saveRefundSettings).toHaveBeenCalledWith({
        ...settings,
        allowed_system_ids: [30_000_001],
      })
    )
  })
})
