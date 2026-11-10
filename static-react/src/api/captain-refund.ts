import { requestJson } from '@/api/http-client'
import { assertSuccess, type ApiResponse } from '@/api/response'
import type {
  RefundSettings,
  RefundRequest,
  RefundCalculation,
  RefundApplication,
  RefundOptions,
  RefundList,
} from '@/types/api/captain-refund'
async function call<T>(path: string, method = 'GET', body?: unknown): Promise<T> {
  const response = await requestJson<ApiResponse<T>>('/api/v1' + path, {
    method,
    ...(body === undefined ? {} : { body: JSON.stringify(body) }),
  })
  return assertSuccess(response, 'Request failed')
}
const mine = '/newbro/captain/refunds'
const admin = '/system/newbro/refunds'
export const refundOptions = () => call<RefundOptions>(mine + '/options')
export const previewRefund = (data: RefundRequest) =>
  call<RefundCalculation>(mine + '/preview', 'POST', data)
export const createRefund = (data: RefundRequest) => call<RefundApplication>(mine, 'POST', data)
export const listRefunds = (isAdmin: boolean, page: number, size: number) =>
  call<RefundList>((isAdmin ? admin : mine) + '?current=' + page + '&size=' + size)
export const getRefund = (isAdmin: boolean, id: number) =>
  call<RefundApplication>((isAdmin ? admin : mine) + '/' + id)
export const reviewRefund = (id: number, approve: boolean, reason: string) =>
  call<unknown>(admin + '/' + id + (approve ? '/approve' : '/reject'), 'POST', { reason })
export const getRefundSettings = () => call<RefundSettings>('/system/config/newbro-refund')
export const saveRefundSettings = (data: RefundSettings) =>
  call<RefundSettings>('/system/config/newbro-refund', 'PUT', data)
