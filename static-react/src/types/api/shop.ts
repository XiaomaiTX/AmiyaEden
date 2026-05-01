import type { CommonSearchParams, PaginatedResponse } from '@/types/api/common'

export interface Product {
  id: number
  name: string
  description: string
  image: string
  price: number
  stock: number
  max_per_user: number
  limit_period: 'forever' | 'daily' | 'weekly' | 'monthly'
  type: 'normal'
  status: number
  sort_order: number
  created_at: string
  updated_at: string
}

export interface Order {
  id: number
  order_no: string
  user_id: number
  main_character_name: string
  nickname: string
  qq: string
  discord_id: string
  product_id: number
  product_name: string
  product_type: string
  quantity: number
  unit_price: number
  total_price: number
  status: string
  transaction_id: number | null
  remark: string
  reviewed_by: number | null
  reviewed_at: string | null
  reviewer_name?: string
  review_remark: string
  created_at: string
  updated_at: string
}

export type ProductListResponse = PaginatedResponse<Product>
export type OrderListResponse = PaginatedResponse<Order>

export interface BuyParams {
  product_id: number
  quantity: number
  remark?: string
}

export interface ProductCreateParams {
  name: string
  description?: string
  image?: string
  price: number
  stock?: number
  max_per_user?: number
  limit_period?: 'forever' | 'daily' | 'weekly' | 'monthly'
  type: 'normal'
  status?: number
  sort_order?: number
}

export interface ProductUpdateParams {
  id: number
  name?: string
  description?: string
  image?: string
  price?: number
  stock?: number
  max_per_user?: number
  limit_period?: 'forever' | 'daily' | 'weekly' | 'monthly'
  type?: string
  status?: number
  sort_order?: number
}

export type ProductSearchParams = Partial<{
  current: number
  size: number
  status: number
  type: string
  name: string
}>

export type OrderSearchParams = Partial<{
  current: number
  size: number
  keyword: string
  statuses: string[]
  status: string
}>

export interface OrderReviewParams {
  order_id: number
  remark?: string
}

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

export type WalletTransactionListResponse = PaginatedResponse<WalletTransaction>

export type WalletSearchParams = Partial<{
  current: number
  size: number
  user_keyword: string
}>

export type WalletTransactionSearchParams = Partial<{
  current: number
  size: number
  user_id: number
  user_keyword: string
  ref_type: string
}>

export interface OrderActionResult extends Order {
  mail_id?: number
  mail_url?: string
}

export type ShopSearchParams = Partial<CommonSearchParams>
