import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, test, vi } from 'vitest'
import { CaptainRefundSettingsPanel } from '@/pages/captain-refund-settings'
import { getCaptainRefundSettings, updateCaptainRefundSettings } from '@/api/generated/client'
import type { CaptainRefundSettings as RefundSettings } from '@/api/generated/client'

vi.mock('@/i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))
vi.mock('@/api/generated/client', () => ({
  getCaptainRefundSettings: vi.fn(),
  updateCaptainRefundSettings: vi.fn(),
}))

const success = <T,>(data: T) => ({ code: 200 as const, msg: 'ok', data })
const settings: RefundSettings = {
  max_total_sp: 10_000_000,
  max_age_days: 90,
  tax_rate: 30,
  allowed_system_ids: [30_000_001],
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(getCaptainRefundSettings).mockResolvedValue(success(settings))
  vi.mocked(updateCaptainRefundSettings).mockImplementation(async (value) =>
    success({ ...settings, ...value })
  )
})

describe('captain refund settings', () => {
  test('rejects values outside the backend limits with a localized message', async () => {
    render(<CaptainRefundSettingsPanel />)
    const taxRate = await screen.findByLabelText('refund.tax_rate')
    fireEvent.change(taxRate, { target: { value: '101' } })
    fireEvent.click(screen.getByRole('button', { name: 'common.save' }))

    expect(await screen.findByRole('alert')).toHaveTextContent('refund.settingsInvalid')
    expect(updateCaptainRefundSettings).not.toHaveBeenCalled()
  })

  test('deduplicates configured solar systems before saving', async () => {
    render(<CaptainRefundSettingsPanel />)
    const allowedSystems = await screen.findByLabelText('refund.allowedSystems')
    fireEvent.change(allowedSystems, { target: { value: '30000001\n30000001' } })
    fireEvent.click(screen.getByRole('button', { name: 'common.save' }))

    await waitFor(() =>
      expect(updateCaptainRefundSettings).toHaveBeenCalledWith({
        ...settings,
        allowed_system_ids: [30_000_001],
      })
    )
  })
})
