import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { I18nProvider } from '@/i18n'
import { usePreferenceStore } from '@/stores'
import {
  operationRates,
  listSrpApplications,
  listSrpBatchPayoutSummary,
  payoutSrpApplication,
  batchPayoutSrpAsFuxiCoin,
  batchPayoutSrpByUser,
  type BatchPayoutSrpAsFuxiCoinResult,
  adminListWelfareApplications,
  adminReviewWelfareApplication,
  adminListShopOrders,
  adminDeliverShopOrder,
  type AdminListWelfareApplicationsResult,
  type AdminListShopOrdersResult,
  type AdminReviewWelfareApplicationResult,
} from '@/api/generated/client'
import { WelfareApprovalPage } from './welfare-approval-page'
import { ShopOrderManagePage } from './shop-order-manage-page'
import { SrpManagePage } from './srp-manage-page'

vi.mock('@/api/generated/client', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/api/generated/client')>()),
  operationRates: vi.fn(),
  listSrpApplications: vi.fn(),
  listSrpBatchPayoutSummary: vi.fn(),
  payoutSrpApplication: vi.fn(),
  runSrpFleetAutoApproval: vi.fn(),
  reviewSrpApplication: vi.fn(),
  batchPayoutSrpByUser: vi.fn(),
  batchPayoutSrpAsFuxiCoin: vi.fn(),
  adminListWelfareApplications: vi.fn(),
  adminReviewWelfareApplication: vi.fn(),
  adminDeleteWelfareApplication: vi.fn(),
  adminListShopOrders: vi.fn(),
  adminDeliverShopOrder: vi.fn(),
  adminRejectShopOrder: vi.fn(),
}))
vi.mock('@/hooks/use-permission', () => ({ usePermission: () => true }))

beforeEach(() => {
  vi.resetAllMocks()
  usePreferenceStore.setState({ locale: 'zh-CN' })
  vi.mocked(operationRates).mockResolvedValue({
    code: 200,
    msg: '',
    data: {
      recruitment: 4.25,
      welfare_delivery: 18,
      shop_delivery: 6.5,
      srp_payout: 2.5,
    },
  })
})
afterEach(() => vi.restoreAllMocks())
const success = <T,>(data: T) => ({ code: 200 as const, msg: '', data })
const open = (element: React.ReactNode) => render(<I18nProvider>{element}</I18nProvider>)

test.each([false, true])(
  'welfare confirms current category salary before delivery: %s',
  async (consent) => {
    vi.mocked(adminListWelfareApplications).mockResolvedValue(
      success({
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
      } as Extract<AdminListWelfareApplicationsResult, { code: 200 }>['data'])
    )
    vi.mocked(adminReviewWelfareApplication).mockResolvedValue(
      success({ outcome: 'delivered' } as Extract<
        AdminReviewWelfareApplicationResult,
        { code: 200 }
      >['data'])
    )
    const confirm = vi.spyOn(window, 'confirm').mockReturnValue(consent)
    open(<WelfareApprovalPage />)
    fireEvent.click(await screen.findByRole('button', { name: '通过' }))
    await waitFor(() =>
      expect(confirm).toHaveBeenCalledWith(expect.stringContaining('18.00 伏羲币工资'))
    )
    if (consent)
      await waitFor(() =>
        expect(adminReviewWelfareApplication).toHaveBeenCalledWith({ id: 8, action: 'deliver' })
      )
    else expect(adminReviewWelfareApplication).not.toHaveBeenCalled()
  }
)

test('shop delivery dialog shows one order salary and cancellation does not deliver', async () => {
  vi.mocked(adminListShopOrders).mockImplementation(async (input) =>
    input?.statuses?.includes('requested')
      ? success({
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
        } as Extract<AdminListShopOrdersResult, { code: 200 }>['data'])
      : success({
          list: [],
          total: 0,
          page: 1,
          pageSize: 20,
        })
  )
  open(<ShopOrderManagePage />)
  fireEvent.click(await screen.findByRole('button', { name: '发放' }))
  await screen.findByText(/6.50 伏羲币工资/)
  fireEvent.click(screen.getByRole('button', { name: '取消' }))
  expect(adminDeliverShopOrder).not.toHaveBeenCalled()
})

test.each([false, true])(
  'batch SRP confirms one salary regardless of application count: %s',
  async (consent) => {
    vi.mocked(listSrpApplications).mockResolvedValue(
      success({
        list: [],
        total: 0,
        page: 1,
        pageSize: 20,
      })
    )
    vi.mocked(listSrpBatchPayoutSummary).mockResolvedValue(
      success([
        {
          user_id: 3,
          main_character_id: 101,
          main_character_name: 'Pilot',
          application_count: 3,
          total_amount: 100000000,
        },
      ])
    )
    vi.mocked(batchPayoutSrpAsFuxiCoin).mockResolvedValue(
      success({} as Extract<BatchPayoutSrpAsFuxiCoinResult, { code: 200 }>['data'])
    )
    const confirm = vi.spyOn(window, 'confirm').mockReturnValue(consent)
    open(<SrpManagePage />)
    fireEvent.click(await screen.findByRole('button', { name: '批量发放' }))
    await waitFor(() =>
      expect(confirm).toHaveBeenCalledWith(expect.stringContaining('2.50 伏羲币工资。'))
    )
    if (consent) {
      await waitFor(() => expect(batchPayoutSrpAsFuxiCoin).toHaveBeenCalledExactlyOnceWith({}))
    } else expect(batchPayoutSrpAsFuxiCoin).not.toHaveBeenCalled()
    expect(batchPayoutSrpByUser).not.toHaveBeenCalled()
    expect(payoutSrpApplication).not.toHaveBeenCalled()
  }
)
