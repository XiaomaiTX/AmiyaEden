import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, test, vi } from 'vitest'
import { CaptainRefundPage } from '@/pages/captain-refund-page'
import type {
  CaptainRefundCalculation as RefundCalculation,
  CaptainRefundApplication as RefundApplication,
} from '@/api/generated/client'
import {
  getCaptainRefundOptions,
  previewCaptainRefund,
  createCaptainRefund,
  listAdminCaptainRefunds,
  listMyCaptainRefunds,
  getAdminCaptainRefund,
  rejectCaptainRefund,
} from '@/api/generated/client'
const success = <T,>(data: T) => ({ code: 200 as const, msg: 'ok', data })
const translate = (key: string) => key
vi.mock('@/i18n', () => ({ useI18n: () => ({ t: translate }) }))
vi.mock('@/api/generated/client', () => ({
  getCaptainRefundOptions: vi.fn(),
  previewCaptainRefund: vi.fn(),
  createCaptainRefund: vi.fn(),
  listAdminCaptainRefunds: vi.fn(),
  listMyCaptainRefunds: vi.fn(),
  getAdminCaptainRefund: vi.fn(),
  getMyCaptainRefund: vi.fn(),
  rejectCaptainRefund: vi.fn(),
  approveCaptainRefund: vi.fn(),
}))
const calculation: RefundCalculation = {
  request: {
    captain_character_id: 1,
    member_names: ['New Member'],
    system_id: 30000001,
    start: '2026-11-10T10:00:00Z',
    end: '2026-11-10T11:00:00Z',
  },
  settings: {
    max_total_sp: 10_000_000,
    max_age_days: 90,
    tax_rate: 30,
    allowed_system_ids: [30000001],
  },
  members: [],
  ignored_names: ['Unknown'],
  journals: [],
  conflicts: [],
  bounty: 2_000_000,
  reward: 0.6,
  has_newbro: true,
  can_submit: true,
}
const application: RefundApplication = {
  id: 1,
  user_id: 1,
  status: 'pending',
  reward: 0.6,
  created_at: '2026-11-10T11:00:00Z',
  reviewed_at: null,
  reviewed_by: 0,
  reject_reason: '',
  wallet_ref_id: '',
  calculation,
}
beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(getCaptainRefundOptions).mockResolvedValue(
    success({
      characters: [{ character_id: 1, character_name: 'Captain' }],
      settings: calculation.settings,
    })
  )
  vi.mocked(listMyCaptainRefunds).mockResolvedValue(
    success({ list: [], total: 0, page: 1, pageSize: 200 })
  )
  vi.mocked(previewCaptainRefund).mockResolvedValue(success(calculation))
  vi.mocked(createCaptainRefund).mockResolvedValue(success(application))
  vi.mocked(getAdminCaptainRefund).mockResolvedValue(success(application))
  vi.mocked(rejectCaptainRefund).mockResolvedValue(success({}))
})
describe('captain refund UI', () => {
  test('sends UTC request, invalidates changed preview and submits without client amounts', async () => {
    render(<CaptainRefundPage />)
    await screen.findByRole('option', { name: 'Captain' })
    await waitFor(() =>
      expect(listMyCaptainRefunds).toHaveBeenCalledWith({ current: 1, size: 200 })
    )
    fireEvent.change(screen.getByLabelText('refund.members'), {
      target: { value: 'New Member\nUnknown' },
    })
    fireEvent.change(screen.getByLabelText('refund.start'), {
      target: { value: '2026-11-10T10:00:00' },
    })
    fireEvent.change(screen.getByLabelText('refund.end'), {
      target: { value: '2026-11-10T11:00:00' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'refund.preview' }))
    const submit = await screen.findByRole('button', { name: 'refund.submit' })
    await waitFor(() => expect(submit).toBeEnabled())
    expect(previewCaptainRefund).toHaveBeenCalledWith({
      captain_character_id: 1,
      system_id: 30000001,
      member_names: ['New Member', 'Unknown'],
      start: '2026-11-10T10:00:00.000Z',
      end: '2026-11-10T11:00:00.000Z',
    })
    fireEvent.change(screen.getByLabelText('refund.members'), { target: { value: 'New Member' } })
    expect(screen.queryByRole('button', { name: 'refund.submit' })).toBeNull()
    fireEvent.click(screen.getByRole('button', { name: 'refund.preview' }))
    const refreshedSubmit = await screen.findByRole('button', { name: 'refund.submit' })
    await waitFor(() => expect(refreshedSubmit).toBeEnabled())
    fireEvent.click(refreshedSubmit)
    await waitFor(() => expect(createCaptainRefund).toHaveBeenCalledOnce())
    expect(createCaptainRefund).toHaveBeenCalledWith(
      expect.not.objectContaining({ reward: expect.anything() })
    )
  })
  test('disables submission for conflict and requires reason for rejection', async () => {
    vi.mocked(listAdminCaptainRefunds).mockResolvedValue(
      success({
        list: [application],
        total: 1,
        page: 1,
        pageSize: 200,
      })
    )
    render(<CaptainRefundPage admin />)
    const detailButton = await screen.findByRole('button', { name: 'refund.details' })
    fireEvent.click(detailButton)
    const reject = await screen.findByRole('button', { name: 'refund.reject' })
    expect(reject).toBeDisabled()
    fireEvent.change(screen.getByLabelText('refund.rejectReason'), {
      target: { value: 'incorrect members' },
    })
    fireEvent.click(reject)
    await waitFor(() =>
      expect(rejectCaptainRefund).toHaveBeenCalledWith(1, { reason: 'incorrect members' })
    )
    expect(getCaptainRefundOptions).not.toHaveBeenCalled()
  })

  test('refreshes the calculation after a concurrent claim conflict', async () => {
    vi.mocked(previewCaptainRefund)
      .mockResolvedValueOnce(success(calculation))
      .mockResolvedValueOnce(
        success({
          ...calculation,
          can_submit: false,
          conflicts: [
            {
              character_id: 2,
              journal_id: 42,
              journal_at: '2026-11-10T10:30:00Z',
              refund_id: 9,
            },
          ],
        })
      )
    vi.mocked(createCaptainRefund).mockRejectedValueOnce(new Error('claim conflict'))
    render(<CaptainRefundPage />)
    await screen.findByRole('option', { name: 'Captain' })
    fireEvent.change(screen.getByLabelText('refund.members'), {
      target: { value: 'New Member' },
    })
    fireEvent.change(screen.getByLabelText('refund.start'), {
      target: { value: '2026-11-10T10:00:00' },
    })
    fireEvent.change(screen.getByLabelText('refund.end'), {
      target: { value: '2026-11-10T11:00:00' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'refund.preview' }))
    const submit = await screen.findByRole('button', { name: 'refund.submit' })
    await waitFor(() => expect(submit).toBeEnabled())
    fireEvent.click(submit)
    await screen.findByText(/refund\.conflict: 2/)
    expect(screen.getByRole('button', { name: 'refund.submit' })).toBeDisabled()
    expect(previewCaptainRefund).toHaveBeenCalledTimes(2)
  })
})
