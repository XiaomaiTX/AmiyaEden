import { requestJson } from '@/api/http-client'
import { assertSuccess, type ApiResponse } from '@/api/response'
import type {
  AdjustParams,
  AnalyticsParams,
  LogSearchParams,
  TransactionSearchParams,
  Wallet,
  WalletAnalytics,
  WalletLogListResponse,
  WalletListResponse,
  WalletTransactionListResponse,
} from '@/types/api/sys-wallet'

export async function fetchMyWallet() {
  const response = await requestJson<ApiResponse<Wallet>>('/api/v1/shop/wallet/my', {
    method: 'POST',
  })
  return assertSuccess(response, 'fetch my wallet failed')
}

export async function fetchMyWalletTransactions(data?: TransactionSearchParams) {
  const response = await requestJson<ApiResponse<WalletTransactionListResponse>>(
    '/api/v1/shop/wallet/my/transactions',
    {
      method: 'POST',
      body: JSON.stringify(data ?? { current: 1, size: 20 }),
    }
  )
  return assertSuccess(response, 'fetch my wallet transactions failed')
}

export async function adminListWallets(data?: import('@/types/api/sys-wallet').WalletSearchParams) {
  const response = await requestJson<ApiResponse<WalletListResponse>>('/api/v1/system/wallet/list', {
    method: 'POST',
    body: JSON.stringify(data ?? { current: 1, size: 200 }),
  })
  return assertSuccess(response, 'list wallet failed')
}

export async function adminGetWallet(userId: number) {
  const response = await requestJson<ApiResponse<Wallet>>('/api/v1/system/wallet/detail', {
    method: 'POST',
    body: JSON.stringify({ user_id: userId }),
  })
  return assertSuccess(response, 'get wallet failed')
}

export async function adminAdjustWallet(data: AdjustParams) {
  const response = await requestJson<ApiResponse<Wallet>>('/api/v1/system/wallet/adjust', {
    method: 'POST',
    body: JSON.stringify(data),
  })
  return assertSuccess(response, 'adjust wallet failed')
}

export async function adminListTransactions(data?: TransactionSearchParams) {
  const response = await requestJson<ApiResponse<WalletTransactionListResponse>>(
    '/api/v1/system/wallet/transactions',
    {
      method: 'POST',
      body: JSON.stringify(data ?? { current: 1, size: 200 }),
    }
  )
  return assertSuccess(response, 'list wallet transactions failed')
}

export async function adminListWalletLogs(data?: LogSearchParams) {
  const response = await requestJson<ApiResponse<WalletLogListResponse>>('/api/v1/system/wallet/logs', {
    method: 'POST',
    body: JSON.stringify(data ?? { current: 1, size: 200 }),
  })
  return assertSuccess(response, 'list wallet logs failed')
}

export async function adminGetWalletAnalytics(data: AnalyticsParams) {
  const response = await requestJson<ApiResponse<WalletAnalytics>>('/api/v1/system/wallet/analytics', {
    method: 'POST',
    body: JSON.stringify(data),
  })
  return assertSuccess(response, 'get wallet analytics failed')
}
