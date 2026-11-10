import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { I18nProvider } from '@/i18n'
import { QQGroupBusiness } from './qq-group-business'
import { saveQQGroupRule } from '@/api/qqbot'
vi.mock('@/api/qqbot', () => ({
  fetchQQGroupBusiness: vi
    .fn()
    .mockResolvedValue({
      rules: [],
      applications: [],
      members: [],
      inspections: [],
      judgements: [],
    }),
  saveQQGroupRule: vi.fn().mockResolvedValue(null),
}))
test('defaults to record-only and submits a typed rule with the confirmed safety defaults', async () => {
  render(
    <I18nProvider>
      <QQGroupBusiness />
    </I18nProvider>
  )
  fireEvent.change(screen.getByLabelText('目标群'), { target: { value: '8' } })
  fireEvent.change(screen.getByLabelText('允许的军团 ID（逗号分隔）'), { target: { value: '200' } })
  fireEvent.click(screen.getByRole('button', { name: '保存配置' }))
  await waitFor(() =>
    expect(saveQQGroupRule).toHaveBeenCalledWith(
      expect.objectContaining({
        group_id: 8,
        corporation_ids: [200],
        auto_reject: false,
        auto_kick: false,
        inspection_enabled: false,
        card_enabled: false,
        mismatch_confirmations: 3,
        observation_hours: 24,
      })
    )
  )
  expect(screen.queryByRole('button', { name: '批准' })).not.toBeInTheDocument()
})
