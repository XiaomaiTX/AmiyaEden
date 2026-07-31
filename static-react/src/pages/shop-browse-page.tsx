import { Textarea } from '@/components/ui/textarea'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Table,
  TableHeader,
  TableBody,
  TableRow,
  TableHead,
  TableCell,
} from '@/components/ui/table'
import { useCallback, useEffect, useMemo, useState } from 'react'
import { buyProduct, fetchMyOrders, fetchMyWallet, fetchProducts } from '@/api/shop'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { useCorpCapability } from '@/hooks/use-corp-capability'
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

function orderStatusLabel(t: ReturnType<typeof useI18n>['t'], status: string) {
  const key = `shopAdmin.orders.status.${status}`
  const translated = t(key)
  return translated === key ? status : translated
}

export function ShopBrowsePage() {
  const { t } = useI18n()
  const { hasCapability } = useCorpCapability()
  // Creating an order requires `shop.order.create`; the route enforces only
  // `shop.order.read_self`, so the buy button must gate separately.
  const canCreateOrder = hasCapability('shop.order.create')
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
    const perUserLimit =
      selectedProduct.max_per_user > 0 ? selectedProduct.max_per_user : Number.POSITIVE_INFINITY
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
          <Button
            type="button"
            variant="outline"
            onClick={() => void loadWallet()}
            isDisabled={walletLoading}
          >
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
              <Button
                type="button"
                variant="outline"
                onClick={() => void loadProducts()}
                isDisabled={productLoading}
              >
                {t('common.refresh')}
              </Button>
              <span className="text-sm text-muted-foreground">
                {t('shopBrowse.productCount', { count: productTotal })}
              </span>
            </div>
          </div>

          <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
            {products.map((item) => (
              <article
                key={item.id}
                className="overflow-hidden rounded-lg border bg-card shadow-sm"
              >
                <div className="flex aspect-square items-center justify-center border-b bg-muted/30">
                  {item.image ? (
                    <img
                      alt={item.name}
                      className="h-full w-full object-contain p-4"
                      src={item.image}
                    />
                  ) : (
                    <div className="text-sm text-muted-foreground">{t('shop.products')}</div>
                  )}
                </div>
                <div className="space-y-3 p-4">
                  <div>
                    <h3 className="font-semibold">{item.name}</h3>
                    {item.description ? (
                      <p className="mt-1 line-clamp-2 text-sm text-muted-foreground">
                        {item.description}
                      </p>
                    ) : null}
                  </div>
                  <div className="flex flex-wrap gap-2">
                    <ShopBadge className={productStatusClass(item.status)}>
                      {item.status === 1
                        ? t('shopManage.statusOnSale')
                        : t('shopManage.statusOffSale')}
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
                    isDisabled={item.stock === 0 || item.status !== 1 || !canCreateOrder}
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
              isDisabled={productPage <= 1}
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
              isDisabled={
                products.length < productPageSize || productPage * productPageSize >= productTotal
              }
            >
              {t('welfareMy.pagination.next')}
            </Button>
            <label className="flex items-center gap-2">
              <span>{t('welfareMy.pageSize')}</span>
              <Select
                selectedKey={String(productPageSize ?? '')}
                onSelectionChange={(key) => ((value) => {
                  const nextSize = Number(value)
                  setProductPageSize(nextSize)
                  setProductPage(1)
                })(String(key))}
              >
                <SelectTrigger className="h-8">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {[12, 24, 36, 48].map((size) => (
                    <SelectItem key={size} id={String(size ?? '')}>
                      {size}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
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
                <Select
                  selectedKey={String(orderStatus ?? '')}
                  onSelectionChange={(key) => ((value) => {
                    setOrderStatus(value)
                    setOrderPage(1)
                  })(String(key))}
                >
                  <SelectTrigger className="h-10">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem id="">{t('shop.allStatuses')}</SelectItem>
                    <SelectItem id="requested">{orderStatusLabel(t, 'requested')}</SelectItem>
                    <SelectItem id="delivered">{orderStatusLabel(t, 'delivered')}</SelectItem>
                    <SelectItem id="rejected">{orderStatusLabel(t, 'rejected')}</SelectItem>
                  </SelectContent>
                </Select>
              </label>
              <Button
                type="button"
                variant="outline"
                onClick={() => void loadOrders()}
                isDisabled={orderLoading}
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
              <Table className="min-w-full text-sm">
                <TableHeader>
                  <TableRow className="border-b bg-muted/40 text-left">
                    <TableHead className="px-3 py-2">{t('shop.orderNo')}</TableHead>
                    <TableHead className="px-3 py-2">{t('shop.productName')}</TableHead>
                    <TableHead className="px-3 py-2">{t('shop.quantity')}</TableHead>
                    <TableHead className="px-3 py-2">{t('shop.unitPrice')}</TableHead>
                    <TableHead className="px-3 py-2">{t('shop.totalPrice')}</TableHead>
                    <TableHead className="px-3 py-2">{t('shop.status')}</TableHead>
                    <TableHead className="px-3 py-2">{t('shop.reviewerName')}</TableHead>
                    <TableHead className="px-3 py-2">{t('shop.submitterRemark')}</TableHead>
                    <TableHead className="px-3 py-2">{t('shop.reviewRemark')}</TableHead>
                    <TableHead className="px-3 py-2">{t('shop.orderTime')}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {orders.map((order) => (
                    <TableRow key={order.id} className="border-b">
                      <TableCell className="px-3 py-2 font-medium">{order.order_no}</TableCell>
                      <TableCell className="px-3 py-2">{order.product_name}</TableCell>
                      <TableCell className="px-3 py-2">{order.quantity}</TableCell>
                      <TableCell className="px-3 py-2">
                        {formatCoin(order.unit_price)} {t('shop.currency')}
                      </TableCell>
                      <TableCell className="px-3 py-2 font-medium text-red-600">
                        {formatCoin(order.total_price)} {t('shop.currency')}
                      </TableCell>
                      <TableCell className="px-3 py-2">
                        <ShopBadge className={orderStatusClass(order.status)}>
                          {orderStatusLabel(t, order.status)}
                        </ShopBadge>
                      </TableCell>
                      <TableCell className="px-3 py-2">{order.reviewer_name || '-'}</TableCell>
                      <TableCell className="px-3 py-2">{order.remark || '-'}</TableCell>
                      <TableCell className="px-3 py-2">{order.review_remark || '-'}</TableCell>
                      <TableCell className="px-3 py-2">
                        {formatDateTime(order.created_at)}
                      </TableCell>
                    </TableRow>
                  ))}
                  {!orderLoading && orders.length === 0 ? (
                    <TableRow>
                      <TableCell
                        className="px-3 py-6 text-center text-muted-foreground"
                        colSpan={10}
                      >
                        {t('shop.noOrders')}
                      </TableCell>
                    </TableRow>
                  ) : null}
                </TableBody>
              </Table>
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
              isDisabled={orderPage <= 1}
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
              isDisabled={orders.length < orderPageSize || orderPage * orderPageSize >= orderTotal}
            >
              {t('welfareMy.pagination.next')}
            </Button>
            <label className="flex items-center gap-2">
              <span>{t('welfareMy.pageSize')}</span>
              <Select
                selectedKey={String(orderPageSize ?? '')}
                onSelectionChange={(key) => ((value) => {
                  const nextSize = Number(value)
                  setOrderPageSize(nextSize)
                  setOrderPage(1)
                })(String(key))}
              >
                <SelectTrigger className="h-8">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {[10, 20, 50].map((size) => (
                    <SelectItem key={size} id={String(size ?? '')}>
                      {size}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
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
            <Button type="button" variant="outline" onClick={closeBuyDialog} isDisabled={buyLoading}>
              {t('common.cancel')}
            </Button>
            <Button
              type="button"
              onClick={() => void confirmBuy()}
              isDisabled={buyLoading || !selectedProduct || !canCreateOrder}
            >
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
                <Input
                  value={`${formatCoin(selectedProduct.price)} ${t('shop.currency')}`}
                  disabled
                />
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
              <Textarea
                className="min-h-24"
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
