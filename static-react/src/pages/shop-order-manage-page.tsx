import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { adminDeliverOrder, adminListOrderHistory, adminListOrders, adminRejectOrder } from '@/api/shop'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { useI18n } from '@/i18n'
import { useSessionStore } from '@/stores'
import type { Order } from '@/types/api/shop'
import {
  formatCoin,
  formatDateTime,
  formatContact,
  getErrorMessage,
  orderStatusClass,
  ShopBadge,
  ShopDialog,
} from './shop-page-utils'

type ActiveTab = 'orders' | 'history'

function orderStatusLabel(t: ReturnType<typeof useI18n>['t'], status: string) {
  const key = `shopAdmin.orders.status.${status}`
  const translated = t(key)
  return translated === key ? status : translated
}

export function ShopOrderManagePage() {
  const { t } = useI18n()
  const authList = useSessionStore((state) => state.authList)
  const [activeTab, setActiveTab] = useState<ActiveTab>('orders')
  const [error, setError] = useState<string | null>(null)
  const [keyword, setKeyword] = useState('')
  const [pendingOrders, setPendingOrders] = useState<Order[]>([])
  const [pendingTotal, setPendingTotal] = useState(0)
  const [pendingPage, setPendingPage] = useState(1)
  const [pendingPageSize, setPendingPageSize] = useState(20)
  const [historyOrders, setHistoryOrders] = useState<Order[]>([])
  const [historyTotal, setHistoryTotal] = useState(0)
  const [historyPage, setHistoryPage] = useState(1)
  const [historyPageSize, setHistoryPageSize] = useState(20)
  const [loading, setLoading] = useState(true)
  const [reviewing, setReviewing] = useState(false)
  const [reviewDialogOpen, setReviewDialogOpen] = useState(false)
  const [reviewAction, setReviewAction] = useState<'deliver' | 'reject'>('deliver')
  const [reviewOrder, setReviewOrder] = useState<Order | null>(null)
  const [reviewRemark, setReviewRemark] = useState('')
  const didMountRef = useRef(false)

  const canApprove = authList.includes('approve_order')

  const pendingPageCount = useMemo(
    () => Math.max(1, Math.ceil(pendingTotal / pendingPageSize) || 1),
    [pendingPageSize, pendingTotal]
  )
  const historyPageCount = useMemo(
    () => Math.max(1, Math.ceil(historyTotal / historyPageSize) || 1),
    [historyPageSize, historyTotal]
  )

  const loadPending = useCallback(
    async (nextPage = pendingPage, nextPageSize = pendingPageSize) => {
      const response = await adminListOrders({
        current: nextPage,
        size: nextPageSize,
        keyword: keyword.trim() || undefined,
        statuses: ['requested'],
      })
      setPendingOrders(response.list ?? [])
      setPendingTotal(response.total ?? 0)
      setPendingPage(response.page ?? nextPage)
      setPendingPageSize(response.pageSize ?? nextPageSize)
    },
    [keyword, pendingPage, pendingPageSize]
  )

  const loadHistory = useCallback(
    async (nextPage = historyPage, nextPageSize = historyPageSize) => {
      const response = await adminListOrderHistory({
        current: nextPage,
        size: nextPageSize,
        keyword: keyword.trim() || undefined,
      })
      setHistoryOrders(response.list ?? [])
      setHistoryTotal(response.total ?? 0)
      setHistoryPage(response.page ?? nextPage)
      setHistoryPageSize(response.pageSize ?? nextPageSize)
    },
    [historyPage, historyPageSize, keyword]
  )

  const loadData = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      await Promise.all([loadPending(), loadHistory()])
    } catch (caughtError) {
      setError(getErrorMessage(caughtError, t('shopAdmin.orders.messages.actionFailed')))
      setPendingOrders([])
      setHistoryOrders([])
      setPendingTotal(0)
      setHistoryTotal(0)
    } finally {
      setLoading(false)
    }
  }, [loadHistory, loadPending, t])

  useEffect(() => {
    const timer = window.setTimeout(() => {
      void loadData()
    }, 0)
    return () => window.clearTimeout(timer)
  }, [loadData])

  useEffect(() => {
    if (!didMountRef.current) {
      didMountRef.current = true
      return
    }

    const timer = window.setTimeout(() => {
      if (activeTab === 'orders') {
        void loadPending(pendingPage, pendingPageSize)
        return
      }

      void loadHistory(historyPage, historyPageSize)
    }, 0)
    return () => window.clearTimeout(timer)
  }, [activeTab, historyPage, historyPageSize, loadHistory, loadPending, pendingPage, pendingPageSize])

  const openReviewDialog = (order: Order, action: 'deliver' | 'reject') => {
    setReviewOrder(order)
    setReviewAction(action)
    setReviewRemark('')
    setReviewDialogOpen(true)
  }

  const submitReview = async () => {
    if (!reviewOrder) return

    setReviewing(true)
    setError(null)
    try {
      const payload = {
        order_id: reviewOrder.id,
        remark: reviewRemark.trim() || undefined,
      }

      if (reviewAction === 'deliver') {
        await adminDeliverOrder(payload)
      } else {
        await adminRejectOrder(payload)
      }

      setReviewDialogOpen(false)
      await loadPending()
      await loadHistory()
    } catch (caughtError) {
      setError(getErrorMessage(caughtError, t('shopAdmin.orders.messages.actionFailed')))
    } finally {
      setReviewing(false)
    }
  }

  const currentRows = activeTab === 'orders' ? pendingOrders : historyOrders

  return (
    <section className="space-y-4">
      <div className="rounded-lg border bg-card p-5">
        <div className="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
          <div>
            <h1 className="text-xl font-semibold">{t('shopAdmin.tabs.orders')}</h1>
            <p className="mt-1 text-sm text-muted-foreground">{t('shopAdmin.tabs.ordersSubtitle')}</p>
          </div>
          <div className="flex flex-wrap items-end gap-3">
            <label className="space-y-1">
              <span className="text-sm text-muted-foreground">{t('shopAdmin.orders.keywordPlaceholder')}</span>
              <Input value={keyword} onChange={(event) => setKeyword(event.target.value)} />
            </label>
            <Button type="button" variant="outline" onClick={() => void loadData()} disabled={loading}>
              {t('common.refresh')}
            </Button>
          </div>
        </div>
      </div>

      {error ? <p className="text-sm text-destructive">{error}</p> : null}
      {loading ? <p className="text-sm text-muted-foreground">{t('shopAdmin.tabs.orders')}</p> : null}

      <div className="rounded-lg border bg-card p-5">
        <div className="flex gap-2">
          <Button
            type="button"
            variant={activeTab === 'orders' ? 'default' : 'outline'}
            onClick={() => setActiveTab('orders')}
          >
            {t('shopAdmin.tabs.orders')}
          </Button>
          <Button
            type="button"
            variant={activeTab === 'history' ? 'default' : 'outline'}
            onClick={() => setActiveTab('history')}
          >
            {t('shopAdmin.tabs.orderHistory')}
          </Button>
        </div>

        <div className="mt-4 overflow-x-auto">
          <table className="min-w-full text-sm">
            <thead>
              <tr className="border-b bg-muted/40 text-left">
                <th className="px-3 py-2">{t('shopAdmin.orders.table.orderNo')}</th>
                <th className="px-3 py-2">{t('shopAdmin.orders.table.mainCharacter')}</th>
                <th className="px-3 py-2">{t('shopAdmin.orders.table.nickname')}</th>
                <th className="px-3 py-2">{t('shopAdmin.orders.table.contact')}</th>
                <th className="px-3 py-2">{t('shopAdmin.orders.table.product')}</th>
                <th className="px-3 py-2">{t('shopAdmin.orders.table.quantity')}</th>
                <th className="px-3 py-2">{t('shopAdmin.orders.table.totalPrice')}</th>
                <th className="px-3 py-2">{t('shopAdmin.orders.table.status')}</th>
                <th className="px-3 py-2">{t('shopAdmin.orders.table.reviewerName')}</th>
                <th className="px-3 py-2">{t('shopAdmin.orders.table.userRemark')}</th>
                <th className="px-3 py-2">{t('shopAdmin.orders.fields.deliverRemark')}</th>
                <th className="px-3 py-2">{t('shopAdmin.orders.table.createdAt')}</th>
                <th className="px-3 py-2">{t('common.operation')}</th>
              </tr>
            </thead>
            <tbody>
              {currentRows.map((order) => (
                <tr key={order.id} className="border-b">
                  <td className="px-3 py-2 font-medium">{order.order_no}</td>
                  <td className="px-3 py-2">{order.main_character_name}</td>
                  <td className="px-3 py-2">{order.nickname || '-'}</td>
                  <td className="px-3 py-2">{formatContact(t, order.qq, order.discord_id)}</td>
                  <td className="px-3 py-2">{order.product_name}</td>
                  <td className="px-3 py-2">{order.quantity}</td>
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
                  <td className="px-3 py-2">
                    {activeTab === 'orders' ? (
                      <div className="flex flex-wrap gap-2">
                        <Button type="button" size="sm" onClick={() => openReviewDialog(order, 'deliver')} disabled={!canApprove}>
                          {t('shopAdmin.orders.deliverButton')}
                        </Button>
                        <Button
                          type="button"
                          size="sm"
                          variant="outline"
                          onClick={() => openReviewDialog(order, 'reject')}
                          disabled={!canApprove}
                        >
                          {t('shopAdmin.orders.rejectButton')}
                        </Button>
                      </div>
                    ) : (
                      <ShopBadge className={orderStatusClass(order.status)}>
                        {orderStatusLabel(t, order.status)}
                      </ShopBadge>
                    )}
                  </td>
                </tr>
              ))}
              {!loading && currentRows.length === 0 ? (
                <tr>
                  <td className="px-3 py-6 text-center text-muted-foreground" colSpan={13}>
                    {t('shopAdmin.orders.empty')}
                  </td>
                </tr>
              ) : null}
            </tbody>
          </table>
        </div>
      </div>

      <div className="flex flex-wrap items-center gap-3 text-sm">
        <span>
          {activeTab === 'orders' ? pendingPage : historyPage}/{activeTab === 'orders' ? pendingPageCount : historyPageCount}
        </span>
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={() => {
            if (activeTab === 'orders') {
              const next = Math.max(1, pendingPage - 1)
              setPendingPage(next)
              return
            }

            const next = Math.max(1, historyPage - 1)
            setHistoryPage(next)
          }}
          disabled={activeTab === 'orders' ? pendingPage <= 1 : historyPage <= 1}
        >
          {t('welfareMy.pagination.prev')}
        </Button>
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={() => {
            if (activeTab === 'orders') {
              const next = pendingPage + 1
              setPendingPage(next)
              return
            }

            const next = historyPage + 1
            setHistoryPage(next)
          }}
          disabled={
            activeTab === 'orders'
              ? pendingOrders.length < pendingPageSize || pendingPage * pendingPageSize >= pendingTotal
              : historyOrders.length < historyPageSize || historyPage * historyPageSize >= historyTotal
          }
        >
          {t('welfareMy.pagination.next')}
        </Button>
        <label className="flex items-center gap-2">
          <span>{t('welfareMy.pageSize')}</span>
          <select
            className="h-8 rounded-md border border-input bg-background px-2 text-sm"
            value={activeTab === 'orders' ? pendingPageSize : historyPageSize}
            onChange={(event) => {
              const nextSize = Number(event.target.value)
              if (activeTab === 'orders') {
                setPendingPageSize(nextSize)
                setPendingPage(1)
                return
              }

              setHistoryPageSize(nextSize)
              setHistoryPage(1)
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

      <ShopDialog
        open={reviewDialogOpen}
        title={reviewAction === 'deliver' ? t('shopAdmin.orders.dialogDeliver') : t('shopAdmin.orders.dialogReject')}
        onClose={() => setReviewDialogOpen(false)}
        closeLabel={t('common.close')}
        footer={
          <>
            <Button type="button" variant="outline" onClick={() => setReviewDialogOpen(false)} disabled={reviewing}>
              {t('common.cancel')}
            </Button>
            <Button type="button" onClick={() => void submitReview()} disabled={reviewing || !reviewOrder}>
              {reviewAction === 'deliver' ? t('shopAdmin.orders.deliverConfirm') : t('shopAdmin.orders.rejectConfirm')}
            </Button>
          </>
        }
      >
        {reviewOrder ? (
          <div className="space-y-4 text-sm">
            <div className="grid gap-3 sm:grid-cols-2">
              <label className="space-y-2">
                <span className="text-muted-foreground">{t('shopAdmin.orders.fields.orderNo')}</span>
                <Input value={reviewOrder.order_no} disabled />
              </label>
              <label className="space-y-2">
                <span className="text-muted-foreground">{t('shopAdmin.orders.table.mainCharacter')}</span>
                <Input value={reviewOrder.main_character_name} disabled />
              </label>
            </div>
            <label className="space-y-2 block">
              <span className="text-muted-foreground">{t('shopAdmin.orders.fields.deliverRemark')}</span>
              <textarea
                className="min-h-24 w-full rounded-lg border border-input bg-background px-3 py-2 text-sm outline-none"
                value={reviewRemark}
                placeholder={t('shopAdmin.orders.placeholders.deliverRemark')}
                onChange={(event) => setReviewRemark(event.target.value)}
              />
            </label>
          </div>
        ) : null}
      </ShopDialog>
    </section>
  )
}
