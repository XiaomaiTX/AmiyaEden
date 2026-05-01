import { useCallback, useEffect, useMemo, useState } from 'react'
import { buyProduct, fetchMyOrders, fetchMyWallet, fetchProducts } from '@/api/shop'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { useI18n } from '@/i18n'
import type { Order, Product, Wallet } from '@/types/api/shop'
import {
  formatCoin,
  formatDateTime,
  getErrorMessage,
  getLimitPeriodLabel,
  orderStatusClass,
  ShopBadge,
  ShopDialog,
  productStatusClass,
} from './shop-page-utils'

type ActiveTab = 'products' | 'orders'

function orderStatusLabel(
  t: ReturnType<typeof useI18n>['t'],
  status: string
) {
  const key = `shopAdmin.orders.status.${status}`
  const translated = t(key)
  return translated === key ? status : translated
}

export function ShopBrowsePage() {
  const { t } = useI18n()
  const [activeTab, setActiveTab] = useState<ActiveTab>('products')
  const [error, setError] = useState<string | null>(null)
  const [wallet, setWallet] = useState<Wallet | null>(null)
  const [walletLoading, setWalletLoading] = useState(true)
  const [products, setProducts] = useState<Product[]>([])
  const [productLoading, setProductLoading] = useState(true)
  const [productPage, setProductPage] = useState(1)
  const [productPageSize, setProductPageSize] = useState(12)
  const [productTotal, setProductTotal] = useState(0)
  const [orders, setOrders] = useState<Order[]>([])
  const [orderLoading, setOrderLoading] = useState(false)
  const [orderPage, setOrderPage] = useState(1)
  const [orderPageSize, setOrderPageSize] = useState(20)
  const [orderTotal, setOrderTotal] = useState(0)
  const [orderStatus, setOrderStatus] = useState('')
  const [selectedProduct, setSelectedProduct] = useState<Product | null>(null)
  const [buyQuantity, setBuyQuantity] = useState(1)
  const [buyRemark, setBuyRemark] = useState('')
  const [buyLoading, setBuyLoading] = useState(false)

  const buyMaxQty = useMemo(() => {
    if (!selectedProduct) return 1

    const stockLimit = selectedProduct.stock < 0 ? Number.POSITIVE_INFINITY : selectedProduct.stock
    const perUserLimit = selectedProduct.max_per_user > 0 ? selectedProduct.max_per_user : Number.POSITIVE_INFINITY
    const maxQty = Math.min(stockLimit, perUserLimit)
    return Number.isFinite(maxQty) ? Math.max(1, maxQty) : 999
  }, [selectedProduct])

  const loadWallet = useCallback(async () => {
    setWalletLoading(true)
    try {
      setWallet(await fetchMyWallet())
    } catch (caughtError) {
      setError(getErrorMessage(caughtError, t('shopBrowse.loadWalletFailed')))
      setWallet(null)
    } finally {
      setWalletLoading(false)
    }
  }, [t])

  const loadProducts = useCallback(
    async (nextPage = productPage, nextPageSize = productPageSize) => {
      setProductLoading(true)
      setError(null)
      try {
        const response = await fetchProducts({ current: nextPage, size: nextPageSize })
        setProducts(response.list ?? [])
        setProductTotal(response.total ?? 0)
        setProductPage(response.page ?? nextPage)
        setProductPageSize(response.pageSize ?? nextPageSize)
      } catch (caughtError) {
        setError(getErrorMessage(caughtError, t('shopBrowse.loadProductsFailed')))
        setProducts([])
        setProductTotal(0)
      } finally {
        setProductLoading(false)
      }
    },
    [productPage, productPageSize, t]
  )

  const loadOrders = useCallback(
    async (nextPage = orderPage, nextPageSize = orderPageSize) => {
      setOrderLoading(true)
      setError(null)
      try {
        const response = await fetchMyOrders({
          current: nextPage,
          size: nextPageSize,
          status: orderStatus || undefined,
        })
        setOrders(response.list ?? [])
        setOrderTotal(response.total ?? 0)
        setOrderPage(response.page ?? nextPage)
        setOrderPageSize(response.pageSize ?? nextPageSize)
      } catch (caughtError) {
        setError(getErrorMessage(caughtError, t('shopBrowse.loadOrdersFailed')))
        setOrders([])
        setOrderTotal(0)
      } finally {
        setOrderLoading(false)
      }
    },
    [orderPage, orderPageSize, orderStatus, t]
  )

  useEffect(() => {
    const timer = window.setTimeout(() => {
      void loadWallet()
    }, 0)
    return () => window.clearTimeout(timer)
  }, [loadWallet])

  useEffect(() => {
    const timer = window.setTimeout(() => {
      void loadProducts(productPage, productPageSize)
    }, 0)
    return () => window.clearTimeout(timer)
  }, [loadProducts, productPage, productPageSize])

  useEffect(() => {
    if (activeTab !== 'orders') {
      return
    }

    const timer = window.setTimeout(() => {
      void loadOrders(orderPage, orderPageSize)
    }, 0)
    return () => window.clearTimeout(timer)
  }, [activeTab, loadOrders, orderPage, orderPageSize])

  const openBuyDialog = (product: Product) => {
    setSelectedProduct(product)
    setBuyQuantity(1)
    setBuyRemark('')
  }

  const closeBuyDialog = () => {
    setSelectedProduct(null)
  }

  const confirmBuy = async () => {
    if (!selectedProduct) return

    setBuyLoading(true)
    setError(null)
    try {
      await buyProduct({
        product_id: selectedProduct.id,
        quantity: buyQuantity,
        remark: buyRemark.trim() || undefined,
      })
      setSelectedProduct(null)
      await Promise.all([
        loadWallet(),
        loadProducts(productPage, productPageSize),
        activeTab === 'orders' ? loadOrders(orderPage, orderPageSize) : Promise.resolve(),
      ])
    } catch (caughtError) {
      setError(getErrorMessage(caughtError, t('shopBrowse.purchaseFailed')))
    } finally {
      setBuyLoading(false)
    }
  }

  const pageCount = Math.max(1, Math.ceil(productTotal / productPageSize) || 1)
  const orderPageCount = Math.max(1, Math.ceil(orderTotal / orderPageSize) || 1)

  return (
    <section className="space-y-4">
      <div className="rounded-lg border bg-card p-5">
        <div className="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
          <div>
            <h1 className="text-xl font-semibold">{t('shopBrowse.title')}</h1>
            <p className="mt-1 text-sm text-muted-foreground">{t('shopBrowse.subtitle')}</p>
          </div>
          <div className="flex flex-wrap gap-2">
            <Button
              type="button"
              variant={activeTab === 'products' ? 'default' : 'outline'}
              onClick={() => setActiveTab('products')}
            >
              {t('shop.products')}
            </Button>
            <Button
              type="button"
              variant={activeTab === 'orders' ? 'default' : 'outline'}
              onClick={() => setActiveTab('orders')}
            >
              {t('shop.myOrders')}
            </Button>
          </div>
        </div>
      </div>

      <div className="rounded-lg border bg-card p-5">
        <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <p className="text-sm text-muted-foreground">{t('shop.myBalance')}</p>
            <p className="mt-1 text-2xl font-semibold">
              {wallet ? `${formatCoin(wallet.balance)} ${t('shop.currency')}` : '-'}
            </p>
            {wallet?.updated_at ? (
              <p className="mt-1 text-xs text-muted-foreground">
                {t('common.updatedAt')}: {formatDateTime(wallet.updated_at)}
              </p>
            ) : null}
          </div>
          <Button type="button" variant="outline" onClick={() => void loadWallet()} disabled={walletLoading}>
            {t('common.refresh')}
          </Button>
        </div>
      </div>

      {error ? <p className="text-sm text-destructive">{error}</p> : null}
      {productLoading && activeTab === 'products' ? (
        <p className="text-sm text-muted-foreground">{t('shopBrowse.loadingProducts')}</p>
      ) : null}
      {orderLoading && activeTab === 'orders' ? (
        <p className="text-sm text-muted-foreground">{t('shopBrowse.loadingOrders')}</p>
      ) : null}

      {activeTab === 'products' ? (
        <div className="space-y-4">
          <div className="rounded-lg border bg-card p-4">
            <div className="flex flex-wrap items-center gap-3">
              <Button type="button" variant="outline" onClick={() => void loadProducts()} disabled={productLoading}>
                {t('common.refresh')}
              </Button>
              <span className="text-sm text-muted-foreground">
                {t('shopBrowse.productCount', { count: productTotal })}
              </span>
            </div>
          </div>

          <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
            {products.map((item) => (
              <article key={item.id} className="overflow-hidden rounded-lg border bg-card shadow-sm">
                <div className="flex aspect-square items-center justify-center border-b bg-muted/30">
                  {item.image ? (
                    <img alt={item.name} className="h-full w-full object-contain p-4" src={item.image} />
                  ) : (
                    <div className="text-sm text-muted-foreground">{t('shop.products')}</div>
                  )}
                </div>
                <div className="space-y-3 p-4">
                  <div>
                    <h3 className="font-semibold">{item.name}</h3>
                    {item.description ? (
                      <p className="mt-1 line-clamp-2 text-sm text-muted-foreground">{item.description}</p>
                    ) : null}
                  </div>
                  <div className="flex flex-wrap gap-2">
                    <ShopBadge className={productStatusClass(item.status)}>
                      {item.status === 1 ? t('shopManage.statusOnSale') : t('shopManage.statusOffSale')}
                    </ShopBadge>
                    {item.stock < 0 ? (
                      <ShopBadge className="bg-slate-100 text-slate-700 dark:bg-slate-500/10 dark:text-slate-300">
                        {t('shop.unlimitedStock')}
                      </ShopBadge>
                    ) : (
                      <ShopBadge className="bg-slate-100 text-slate-700 dark:bg-slate-500/10 dark:text-slate-300">
                        {t('shop.stockRemaining', { n: item.stock })}
                      </ShopBadge>
                    )}
                  </div>
                  <div className="space-y-1 text-sm">
                    <div className="font-semibold text-orange-600">
                      {formatCoin(item.price)} {t('shop.currency')}
                    </div>
                    {item.max_per_user > 0 ? (
                      <div className="text-muted-foreground">
                        {t('shop.limitPerUser', { n: item.max_per_user })}{' '}
                        {item.limit_period !== 'forever'
                          ? `(${getLimitPeriodLabel(t, item.limit_period)})`
                          : null}
                      </div>
                    ) : null}
                  </div>
                  <Button
                    type="button"
                    className="w-full"
                    disabled={item.stock === 0 || item.status !== 1}
                    onClick={() => openBuyDialog(item)}
                  >
                    {item.stock === 0 ? t('shop.soldOut') : t('shop.buy')}
                  </Button>
                </div>
              </article>
            ))}
          </div>

          {!productLoading && products.length === 0 ? (
            <p className="rounded-lg border bg-card p-4 text-sm text-muted-foreground">
              {t('shop.noProducts')}
            </p>
          ) : null}

          <div className="flex flex-wrap items-center gap-3 text-sm">
            <span>
              {productPage}/{pageCount}
            </span>
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => {
                const next = Math.max(1, productPage - 1)
                setProductPage(next)
              }}
              disabled={productPage <= 1}
            >
              {t('welfareMy.pagination.prev')}
            </Button>
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => {
                const next = productPage + 1
                setProductPage(next)
              }}
              disabled={products.length < productPageSize || productPage * productPageSize >= productTotal}
            >
              {t('welfareMy.pagination.next')}
            </Button>
            <label className="flex items-center gap-2">
              <span>{t('welfareMy.pageSize')}</span>
              <select
                className="h-8 rounded-md border border-input bg-background px-2 text-sm"
                value={productPageSize}
                onChange={(event) => {
                  const nextSize = Number(event.target.value)
                  setProductPageSize(nextSize)
                  setProductPage(1)
                }}
              >
                {[12, 24, 36, 48].map((size) => (
                  <option key={size} value={size}>
                    {size}
                  </option>
                ))}
              </select>
            </label>
          </div>
        </div>
      ) : null}

      {activeTab === 'orders' ? (
        <div className="space-y-4">
          <div className="rounded-lg border bg-card p-4">
            <div className="flex flex-wrap items-end gap-3">
              <label className="space-y-1">
                <span className="text-sm text-muted-foreground">{t('shop.allStatuses')}</span>
                <select
                  className="h-10 rounded-md border border-input bg-background px-3 text-sm"
                  value={orderStatus}
                  onChange={(event) => {
                    setOrderStatus(event.target.value)
                    setOrderPage(1)
                  }}
                >
                  <option value="">{t('shop.allStatuses')}</option>
                  <option value="requested">{orderStatusLabel(t, 'requested')}</option>
                  <option value="delivered">{orderStatusLabel(t, 'delivered')}</option>
                  <option value="rejected">{orderStatusLabel(t, 'rejected')}</option>
                </select>
              </label>
              <Button
                type="button"
                variant="outline"
                onClick={() => void loadOrders()}
                disabled={orderLoading}
              >
                {t('common.refresh')}
              </Button>
            </div>
          </div>

          <div className="overflow-hidden rounded-lg border bg-card">
            <div className="border-b px-4 py-3 text-sm font-medium">
              {t('shop.myOrders')} ({orderTotal})
            </div>
            <div className="overflow-x-auto">
              <table className="min-w-full text-sm">
                <thead>
                  <tr className="border-b bg-muted/40 text-left">
                    <th className="px-3 py-2">{t('shop.orderNo')}</th>
                    <th className="px-3 py-2">{t('shop.productName')}</th>
                    <th className="px-3 py-2">{t('shop.quantity')}</th>
                    <th className="px-3 py-2">{t('shop.unitPrice')}</th>
                    <th className="px-3 py-2">{t('shop.totalPrice')}</th>
                    <th className="px-3 py-2">{t('shop.status')}</th>
                    <th className="px-3 py-2">{t('shop.reviewerName')}</th>
                    <th className="px-3 py-2">{t('shop.submitterRemark')}</th>
                    <th className="px-3 py-2">{t('shop.reviewRemark')}</th>
                    <th className="px-3 py-2">{t('shop.orderTime')}</th>
                  </tr>
                </thead>
                <tbody>
                  {orders.map((order) => (
                    <tr key={order.id} className="border-b">
                      <td className="px-3 py-2 font-medium">{order.order_no}</td>
                      <td className="px-3 py-2">{order.product_name}</td>
                      <td className="px-3 py-2">{order.quantity}</td>
                      <td className="px-3 py-2">
                        {formatCoin(order.unit_price)} {t('shop.currency')}
                      </td>
                      <td className="px-3 py-2 font-medium text-red-600">
                        {formatCoin(order.total_price)} {t('shop.currency')}
                      </td>
                      <td className="px-3 py-2">
                        <ShopBadge className={orderStatusClass(order.status)}>
                          {orderStatusLabel(t, order.status)}
                        </ShopBadge>
                      </td>
                      <td className="px-3 py-2">{order.reviewer_name || '-'}</td>
                      <td className="px-3 py-2">{order.remark || '-'}</td>
                      <td className="px-3 py-2">{order.review_remark || '-'}</td>
                      <td className="px-3 py-2">{formatDateTime(order.created_at)}</td>
                    </tr>
                  ))}
                  {!orderLoading && orders.length === 0 ? (
                    <tr>
                      <td className="px-3 py-6 text-center text-muted-foreground" colSpan={10}>
                        {t('shop.noOrders')}
                      </td>
                    </tr>
                  ) : null}
                </tbody>
              </table>
            </div>
          </div>

          <div className="flex flex-wrap items-center gap-3 text-sm">
            <span>
              {orderPage}/{orderPageCount}
            </span>
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => {
                const next = Math.max(1, orderPage - 1)
                setOrderPage(next)
              }}
              disabled={orderPage <= 1}
            >
              {t('welfareMy.pagination.prev')}
            </Button>
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => {
                const next = orderPage + 1
                setOrderPage(next)
              }}
              disabled={orders.length < orderPageSize || orderPage * orderPageSize >= orderTotal}
            >
              {t('welfareMy.pagination.next')}
            </Button>
            <label className="flex items-center gap-2">
              <span>{t('welfareMy.pageSize')}</span>
              <select
                className="h-8 rounded-md border border-input bg-background px-2 text-sm"
                value={orderPageSize}
                onChange={(event) => {
                  const nextSize = Number(event.target.value)
                  setOrderPageSize(nextSize)
                  setOrderPage(1)
                }}
              >
                {[10, 20, 50].map((size) => (
                  <option key={size} value={size}>
                    {size}
                  </option>
                ))}
              </select>
            </label>
          </div>
        </div>
      ) : null}

      <ShopDialog
        open={selectedProduct !== null}
        title={t('shop.buyTitle')}
        onClose={closeBuyDialog}
        closeLabel={t('common.close')}
        footer={
          <>
            <Button type="button" variant="outline" onClick={closeBuyDialog} disabled={buyLoading}>
              {t('common.cancel')}
            </Button>
            <Button type="button" onClick={() => void confirmBuy()} disabled={buyLoading || !selectedProduct}>
              {buyLoading ? t('shopBrowse.buying') : t('shop.confirmBuy')}
            </Button>
          </>
        }
      >
        {selectedProduct ? (
          <div className="space-y-4 text-sm">
            <div className="grid gap-3 sm:grid-cols-2">
              <label className="space-y-2">
                <span className="text-muted-foreground">{t('shop.productName')}</span>
                <Input value={selectedProduct.name} disabled />
              </label>
              <label className="space-y-2">
                <span className="text-muted-foreground">{t('shop.unitPrice')}</span>
                <Input value={`${formatCoin(selectedProduct.price)} ${t('shop.currency')}`} disabled />
              </label>
              <label className="space-y-2">
                <span className="text-muted-foreground">{t('shop.quantity')}</span>
                <Input
                  type="number"
                  min={1}
                  max={buyMaxQty}
                  value={buyQuantity}
                  onChange={(event) => setBuyQuantity(Number(event.target.value))}
                />
              </label>
              <label className="space-y-2">
                <span className="text-muted-foreground">{t('shop.totalPrice')}</span>
                <Input
                  value={`${formatCoin(selectedProduct.price * buyQuantity)} ${t('shop.currency')}`}
                  disabled
                />
              </label>
            </div>
            <label className="space-y-2 block">
              <span className="text-muted-foreground">{t('shop.remark')}</span>
              <textarea
                className="min-h-24 w-full rounded-lg border border-input bg-background px-3 py-2 text-sm outline-none"
                value={buyRemark}
                placeholder={t('shop.remarkPlaceholder')}
                onChange={(event) => setBuyRemark(event.target.value)}
              />
            </label>
          </div>
        ) : null}
      </ShopDialog>
    </section>
  )
}
