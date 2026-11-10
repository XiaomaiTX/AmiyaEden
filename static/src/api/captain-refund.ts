import request from '@/utils/http'
import type {
  RefundSettings,
  RefundRequest,
  RefundCalculation,
  RefundApplication,
  RefundOptions,
  RefundList
} from '@/types/api/captain-refund'
const mine = '/api/v1/newbro/captain/refunds'
const admin = '/api/v1/system/newbro/refunds'
export const refundOptions = () => request.get<RefundOptions>({ url: mine + '/options' })
export const previewRefund = (data: RefundRequest) =>
  request.post<RefundCalculation>({ url: mine + '/preview', data })
export const createRefund = (data: RefundRequest) =>
  request.post<RefundApplication>({ url: mine, data })
export const listRefunds = async (
  isAdmin: boolean,
  params: Api.Common.CommonSearchParams
): Promise<Api.Common.PaginatedResponse<RefundApplication>> => {
  const response = await request.get<RefundList>({ url: isAdmin ? admin : mine, params })
  return {
    list: response.list,
    total: response.total,
    page: response.page,
    pageSize: response.page_size
  }
}
export const getRefund = (isAdmin: boolean, id: number) =>
  request.get<RefundApplication>({ url: (isAdmin ? admin : mine) + '/' + id })
export const reviewRefund = (id: number, approve: boolean, reason: string) =>
  request.post({ url: admin + '/' + id + (approve ? '/approve' : '/reject'), data: { reason } })
export const getRefundSettings = () =>
  request.get<RefundSettings>({ url: '/api/v1/system/config/newbro-refund' })
export const saveRefundSettings = (data: RefundSettings) =>
  request.put<RefundSettings>({ url: '/api/v1/system/config/newbro-refund', data })
