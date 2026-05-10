import { requestJson } from '@/api/http-client'
import { assertSuccess, type ApiResponse } from '@/api/response'
import type {
  BuyParams,
  Order,
  OrderActionResult,
  OrderListResponse,
  OrderReviewParams,
  OrderSearchParams,
  Product,
  ProductCreateParams,
  ProductListResponse,
  ProductSearchParams,
  ProductUpdateParams,
  Wallet,
  WalletTransactionListResponse,
  WalletTransactionSearchParams,
} from '@/types/api/shop'

type ApiResult<T> = ApiResponse<T>

export async function fetchProducts(data?: ProductSearchParams) {
  const response = await requestJson<ApiResult<ProductListResponse>>('/api/v1/shop/products', {
    method: 'POST',
    body: JSON.stringify(data ?? { current: 1, size: 20 }),
  })
  return assertSuccess(response, 'fetch shop products failed')
}

export async function fetchProductDetail(productId: number) {
  const response = await requestJson<ApiResult<Product>>('/api/v1/shop/product/detail', {
    method: 'POST',
    body: JSON.stringify({ product_id: productId }),
  })
  return assertSuccess(response, 'fetch shop product detail failed')
}

export async function buyProduct(data: BuyParams) {
  const response = await requestJson<ApiResult<Order>>('/api/v1/shop/buy', {
    method: 'POST',
    body: JSON.stringify(data),
  })
  return assertSuccess(response, 'buy product failed')
}

export async function fetchMyOrders(data?: OrderSearchParams) {
  const response = await requestJson<ApiResult<OrderListResponse>>('/api/v1/shop/orders', {
    method: 'POST',
    body: JSON.stringify(data ?? { current: 1, size: 20 }),
  })
  return assertSuccess(response, 'fetch shop orders failed')
}

export async function adminListProducts(data?: ProductSearchParams) {
  const response = await requestJson<ApiResult<ProductListResponse>>(
    '/api/v1/system/shop/product/list',
    {
      method: 'POST',
      body: JSON.stringify(data ?? { current: 1, size: 20 }),
    }
  )
  return assertSuccess(response, 'list shop products failed')
}

export async function adminCreateProduct(data: ProductCreateParams) {
  const response = await requestJson<ApiResult<Product>>('/api/v1/system/shop/product/add', {
    method: 'POST',
    body: JSON.stringify(data),
  })
  return assertSuccess(response, 'create shop product failed')
}

export async function adminUpdateProduct(data: ProductUpdateParams) {
  const response = await requestJson<ApiResult<Product>>('/api/v1/system/shop/product/edit', {
    method: 'POST',
    body: JSON.stringify(data),
  })
  return assertSuccess(response, 'update shop product failed')
}

export async function adminDeleteProduct(id: number) {
  const response = await requestJson<ApiResult<null>>('/api/v1/system/shop/product/delete', {
    method: 'POST',
    body: JSON.stringify({ id }),
  })
  return assertSuccess(response, 'delete shop product failed')
}

export async function adminListOrders(data?: OrderSearchParams) {
  const response = await requestJson<ApiResult<OrderListResponse>>('/api/v1/system/shop/order/list', {
    method: 'POST',
    body: JSON.stringify(data ?? { current: 1, size: 20, statuses: ['requested'] }),
  })
  return assertSuccess(response, 'list shop orders failed')
}

export async function adminListOrderHistory(data?: OrderSearchParams) {
  const response = await requestJson<ApiResult<OrderListResponse>>('/api/v1/system/shop/order/list', {
    method: 'POST',
    body: JSON.stringify({ ...data, statuses: ['delivered', 'rejected'] }),
  })
  return assertSuccess(response, 'list shop order history failed')
}

export async function adminDeliverOrder(data: OrderReviewParams) {
  const response = await requestJson<ApiResult<OrderActionResult>>(
    '/api/v1/system/shop/order/deliver',
    {
      method: 'POST',
      body: JSON.stringify(data),
    }
  )
  return assertSuccess(response, 'deliver shop order failed')
}

export async function adminRejectOrder(data: OrderReviewParams) {
  const response = await requestJson<ApiResult<Order>>('/api/v1/system/shop/order/reject', {
    method: 'POST',
    body: JSON.stringify(data),
  })
  return assertSuccess(response, 'reject shop order failed')
}

export async function fetchMyWallet() {
  const response = await requestJson<ApiResult<Wallet>>('/api/v1/shop/wallet/my', {
    method: 'POST',
  })
  return assertSuccess(response, 'fetch shop wallet failed')
}

export async function fetchMyWalletTransactions(data?: WalletTransactionSearchParams) {
  const response = await requestJson<ApiResult<WalletTransactionListResponse>>(
    '/api/v1/shop/wallet/my/transactions',
    {
      method: 'POST',
      body: JSON.stringify(data ?? { current: 1, size: 20 }),
    }
  )
  return assertSuccess(response, 'fetch shop wallet transactions failed')
}
