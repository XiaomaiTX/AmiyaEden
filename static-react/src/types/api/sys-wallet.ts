import type { PaginatedResponse, CommonSearchParams } from '@/types/api/common'

export interface Wallet {
  id: number
  user_id: number
  balance: number
  updated_at: string
  character_name?: string
}

export interface WalletTransaction {
  id: number
  user_id: number
  amount: number
  reason: string
  ref_type: string
  ref_id: string
  balance_after: number
  operator_id: number
  created_at: string
  character_name?: string
  nickname?: string
  operator_name?: string
}

export interface WalletLog {
  id: number
  operator_id: number
  target_uid: number
  action: 'add' | 'deduct' | 'set'
  amount: number
  before: number
  after: number
  reason: string
  created_at: string
  target_character_name?: string
  operator_character_name?: string
}

export interface AdjustParams {
  target_uid: number
  action: 'add' | 'deduct' | 'set'
  amount: number
  reason: string
}

export type WalletSearchParams = Partial<{
  current: number
  size: number
  user_keyword: string
}> &
  Partial<CommonSearchParams>

export type TransactionSearchParams = Partial<{
  current: number
  size: number
  user_id: number
  user_keyword: string
  ref_type: string
}> &
  Partial<CommonSearchParams>

export type LogSearchParams = Partial<{
  current: number
  size: number
  operator_id: number
  target_uid: number
  action: string
}> &
  Partial<CommonSearchParams>

export interface AnalyticsParams {
  start_date: string
  end_date: string
  ref_types?: string[]
  user_keyword?: string
  top_n?: number
}

export interface WalletAnalyticsSummary {
  wallet_count: number
  active_wallet_count: number
  total_balance: number
  income_total: number
  expense_total: number
  net_flow: number
}

export interface WalletAnalyticsDailySeriesItem {
  date: string
  income: number
  expense: number
  net_flow: number
}

export interface WalletAnalyticsRefTypeBreakdownItem {
  ref_type: string
  income: number
  expense: number
  count: number
}

export interface WalletAnalyticsTopUserItem {
  user_id: number
  character_name?: string
  amount: number
}

export interface WalletAnalyticsAdminAdjustOperatorItem {
  operator_id: number
  operator_name?: string
  count: number
  amount_total: number
}

export interface WalletAnalyticsAdminAdjustStats {
  count: number
  amount_total: number
  by_operator: WalletAnalyticsAdminAdjustOperatorItem[]
}

export interface WalletAnalyticsLargeTransactionItem {
  id: number
  user_id: number
  character_name?: string
  amount: number
  ref_type: string
  created_at: string
}

export interface WalletAnalyticsFrequentAdjustmentItem {
  target_uid: number
  character_name?: string
  adjust_count: number
  amount_total: number
  last_adjustment_time: string
}

export interface WalletAnalyticsOperatorConcentrationItem {
  operator_id: number
  operator_name?: string
  count: number
  amount_total: number
  ratio: number
}

export interface WalletAnalytics {
  summary: WalletAnalyticsSummary
  daily_series: WalletAnalyticsDailySeriesItem[]
  ref_type_breakdown: WalletAnalyticsRefTypeBreakdownItem[]
  top_inflow_users: WalletAnalyticsTopUserItem[]
  top_outflow_users: WalletAnalyticsTopUserItem[]
  admin_adjust_stats: WalletAnalyticsAdminAdjustStats
  anomalies: {
    large_transactions: WalletAnalyticsLargeTransactionItem[]
    frequent_adjustments: WalletAnalyticsFrequentAdjustmentItem[]
    operator_concentration: WalletAnalyticsOperatorConcentrationItem[]
  }
}

export type WalletListResponse = PaginatedResponse<Wallet>
export type WalletTransactionListResponse = PaginatedResponse<WalletTransaction>
export type WalletLogListResponse = PaginatedResponse<WalletLog>
