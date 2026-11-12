import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { I18nProvider } from '@/i18n'
import { usePreferenceStore } from '@/stores'
import { fetchOperationSalaryRates } from '@/api/salary'
import { adminListApplications, adminReviewApplication } from '@/api/welfare'
import { adminListOrders, adminListOrderHistory, adminDeliverOrder } from '@/api/shop'
import {
  fetchApplicationList,
  fetchBatchPayoutSummary,
  batchPayoutAsFuxiCoin,
  batchPayoutByUser,
  payoutApplication,
} from '@/api/srp'
import { WelfareApprovalPage } from './welfare-approval-page'
import { ShopOrderManagePage } from './shop-order-manage-page'
import { SrpManagePage } from './srp-manage-page'

vi.mock('@/api/salary', () => ({ fetchOperationSalaryRates: vi.fn() }))
vi.mock('@/api/welfare', () => ({
  adminListApplications: vi.fn(),
  adminReviewApplication: vi.fn(),
  adminDeleteApplication: vi.fn(),
}))
vi.mock('@/api/shop', () => ({
  adminListOrders: vi.fn(),
  adminListOrderHistory: vi.fn(),
  adminDeliverOrder: vi.fn(),
  adminRejectOrder: vi.fn(),
}))
vi.mock('@/api/srp', () => ({
  fetchApplicationList: vi.fn(),
  fetchBatchPayoutSummary: vi.fn(),
  batchPayoutAsFuxiCoin: vi.fn(),
  batchPayoutByUser: vi.fn(),
  payoutApplication: vi.fn(),
  reviewApplication: vi.fn(),
  runFleetAutoApproval: vi.fn(),
}))
vi.mock('@/hooks/use-permission', () => ({ usePermission: () => true }))

beforeEach(() => {
  vi.resetAllMocks()
  usePreferenceStore.setState({ locale: 'zh-CN' })
  vi.mocked(fetchOperationSalaryRates).mockResolvedValue({
    recruitment: 4.25,
    welfare_delivery: 18,
    shop_delivery: 6.5,
    srp_payout: 2.5,
  })
})
afterEach(() => vi.restoreAllMocks())
const open = (element: React.ReactNode) => render(<I18nProvider>{element}</I18nProvider>)

test.each([false, true])(
  'welfare confirms current category salary before delivery: %s',
  async (consent) => {
    vi.mocked(adminListApplications).mockResolvedValue({
      list: [
        {
          id: 8,
          character_name: 'Pilot',
          welfare_name: 'Welfare',
          status: 'requested',
          created_at: '',
          reviewed_by: 0,
        },
      ],
      total: 1,
    } as Awaited<ReturnType<typeof adminListApplications>>)
    vi.mocked(adminReviewApplication).mockResolvedValue({ outcome: 'delivered' } as Awaited<
      ReturnType<typeof adminReviewApplication>
    >)
    const confirm = vi.spyOn(window, 'confirm').mockReturnValue(consent)
    open(<WelfareApprovalPage />)
    fireEvent.click(await screen.findByRole('button', { name: '通过' }))
    await waitFor(() =>
      expect(confirm).toHaveBeenCalledWith(expect.stringContaining('18.00 伏羲币工资'))
    )
    if (consent)
      await waitFor(() =>
        expect(adminReviewApplication).toHaveBeenCalledWith({ id: 8, action: 'deliver' })
      )
    else expect(adminReviewApplication).not.toHaveBeenCalled()
  }
)

test('shop delivery dialog shows one order salary and cancellation does not deliver', async () => {
  vi.mocked(adminListOrders).mockResolvedValue({
    list: [
      {
        id: 9,
        order_no: 'ORDER-9',
        main_character_name: 'Pilot',
        product_name: 'Ship',
        total_price: 100,
        quantity: 1,
        status: 'requested',
        created_at: '',
      },
    ],
    total: 1,
  } as Awaited<ReturnType<typeof adminListOrders>>)
  vi.mocked(adminListOrderHistory).mockResolvedValue({
    list: [],
    total: 0,
    page: 1,
    pageSize: 20,
  } as Awaited<ReturnType<typeof adminListOrderHistory>>)
  open(<ShopOrderManagePage />)
  fireEvent.click(await screen.findByRole('button', { name: '发放' }))
  await screen.findByText(/6.50 伏羲币工资/)
  fireEvent.click(screen.getByRole('button', { name: '取消' }))
  expect(adminDeliverOrder).not.toHaveBeenCalled()
})

test.each([false, true])(
  'batch SRP confirms one salary regardless of application count: %s',
  async (consent) => {
    vi.mocked(fetchApplicationList).mockResolvedValue({
      list: [],
      total: 0,
      page: 1,
      pageSize: 20,
    } as Awaited<ReturnType<typeof fetchApplicationList>>)
    vi.mocked(fetchBatchPayoutSummary).mockResolvedValue([
      {
        user_id: 3,
        main_character_id: 101,
        main_character_name: 'Pilot',
        application_count: 3,
        total_amount: 100000000,
      },
    ])
    vi.mocked(batchPayoutAsFuxiCoin).mockResolvedValue(
      {} as Awaited<ReturnType<typeof batchPayoutAsFuxiCoin>>
    )
    const confirm = vi.spyOn(window, 'confirm').mockReturnValue(consent)
    open(<SrpManagePage />)
    fireEvent.click(await screen.findByRole('button', { name: '批量发放' }))
    await waitFor(() =>
      expect(confirm).toHaveBeenCalledWith(
        expect.stringContaining('2.50 伏羲币工资（1 次 × 每次 2.50')
      )
    )
    if (consent) await waitFor(() => expect(batchPayoutAsFuxiCoin).toHaveBeenCalledOnce())
    else expect(batchPayoutAsFuxiCoin).not.toHaveBeenCalled()
    expect(batchPayoutByUser).not.toHaveBeenCalled()
    expect(payoutApplication).not.toHaveBeenCalled()
  }
)
