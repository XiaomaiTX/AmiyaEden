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
import { fetchMyWallet, fetchMyWalletTransactions } from '@/api/shop'
import { Button } from '@/components/ui/button'
import { useI18n } from '@/i18n'
import type { Wallet, WalletTransaction } from '@/types/api/shop'
import {
  formatCoin,
  formatDateTime,
  formatSignedCoin,
  getErrorMessage,
  refTypeClass,
  ShopBadge,
} from './shop-page-utils'

function refTypeLabel(t: ReturnType<typeof useI18n>['t'], value: string) {
  const key = `walletAdmin.refTypes.${value}`
  const translated = t(key)
  return translated === key ? value : translated
}

export function ShopWalletPage() {
  const { t } = useI18n()
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [wallet, setWallet] = useState<Wallet | null>(null)
  const [transactions, setTransactions] = useState<WalletTransaction[]>([])
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)
  const [total, setTotal] = useState(0)
  const [refreshSeed, setRefreshSeed] = useState(0)

  const pageCount = useMemo(() => Math.max(1, Math.ceil(total / pageSize) || 1), [pageSize, total])

  const loadData = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const [walletResponse, transactionResponse] = await Promise.all([
        fetchMyWallet(),
        fetchMyWalletTransactions({ current: page, size: pageSize }),
      ])
      setWallet(walletResponse)
      setTransactions(transactionResponse.list ?? [])
      setTotal(transactionResponse.total ?? 0)
      setPage(transactionResponse.page ?? page)
      setPageSize(transactionResponse.pageSize ?? pageSize)
    } catch (caughtError) {
      setError(getErrorMessage(caughtError, t('shopWallet.loadFailed')))
      setWallet(null)
      setTransactions([])
      setTotal(0)
    } finally {
      setLoading(false)
    }
  }, [page, pageSize, t])

  useEffect(() => {
    const timer = window.setTimeout(() => {
      void loadData()
    }, 0)
    return () => window.clearTimeout(timer)
  }, [loadData, refreshSeed])

  return (
    <section className="space-y-4">
      <div className="rounded-lg border bg-card p-5">
        <div className="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
          <div>
            <h1 className="text-xl font-semibold">{t('shopWallet.title')}</h1>
            <p className="mt-1 text-sm text-muted-foreground">{t('shopWallet.subtitle')}</p>
          </div>
          <Button
            type="button"
            variant="outline"
            onClick={() => setRefreshSeed((current) => current + 1)}
          >
            {t('common.refresh')}
          </Button>
        </div>
      </div>

      {error ? <p className="text-sm text-destructive">{error}</p> : null}
      {loading ? <p className="text-sm text-muted-foreground">{t('shopWallet.loading')}</p> : null}

      <div className="rounded-lg border bg-card p-5">
        <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
          <div>
            <p className="text-sm text-muted-foreground">{t('shop.myBalance')}</p>
            <p
              className={`mt-1 text-3xl font-semibold ${wallet && wallet.balance < 0 ? 'text-rose-600' : 'text-emerald-600'}`}
            >
              {wallet ? `${formatCoin(wallet.balance)} ${t('shop.currency')}` : '-'}
            </p>
            {wallet?.character_name ? (
              <p className="mt-2 text-sm text-muted-foreground">
                {t('shopWallet.characterName')}: {wallet.character_name}
              </p>
            ) : null}
            {wallet?.updated_at ? (
              <p className="mt-1 text-xs text-muted-foreground">
                {t('common.updatedAt')}: {formatDateTime(wallet.updated_at)}
              </p>
            ) : null}
          </div>
        </div>
      </div>

      <div className="overflow-hidden rounded-lg border bg-card">
        <div className="border-b px-4 py-3 text-sm font-medium">
          {t('shopWallet.transactionsTitle')} ({total})
        </div>
        <div className="overflow-x-auto">
          <Table className="min-w-full text-sm">
            <TableHeader>
              <TableRow className="border-b bg-muted/40 text-left">
                <TableHead className="px-3 py-2">{t('shopWallet.columns.createdAt')}</TableHead>
                <TableHead className="px-3 py-2">{t('shopWallet.columns.refType')}</TableHead>
                <TableHead className="px-3 py-2">{t('shopWallet.columns.amount')}</TableHead>
                <TableHead className="px-3 py-2">{t('shopWallet.columns.balanceAfter')}</TableHead>
                <TableHead className="px-3 py-2">{t('shopWallet.columns.reason')}</TableHead>
                <TableHead className="px-3 py-2">{t('shopWallet.columns.operator')}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {transactions.map((row) => (
                <TableRow key={row.id} className="border-b">
                  <TableCell className="px-3 py-2">{formatDateTime(row.created_at)}</TableCell>
                  <TableCell className="px-3 py-2">
                    <ShopBadge className={refTypeClass(row.ref_type)}>
                      {refTypeLabel(t, row.ref_type)}
                    </ShopBadge>
                  </TableCell>
                  <TableCell className="px-3 py-2 font-medium">
                    <span className={row.amount >= 0 ? 'text-emerald-600' : 'text-rose-600'}>
                      {formatSignedCoin(row.amount)} {t('shop.currency')}
                    </span>
                  </TableCell>
                  <TableCell className="px-3 py-2">
                    {formatCoin(row.balance_after)} {t('shop.currency')}
                  </TableCell>
                  <TableCell className="px-3 py-2">{row.reason || '-'}</TableCell>
                  <TableCell className="px-3 py-2">
                    {row.operator_name || row.operator_id || '-'}
                  </TableCell>
                </TableRow>
              ))}
              {!loading && transactions.length === 0 ? (
                <TableRow>
                  <TableCell className="px-3 py-6 text-center text-muted-foreground" colSpan={6}>
                    {t('shopWallet.empty')}
                  </TableCell>
                </TableRow>
              ) : null}
            </TableBody>
          </Table>
        </div>
      </div>

      <div className="flex flex-wrap items-center gap-3 text-sm">
        <span>
          {page}/{pageCount}
        </span>
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={() => setPage((current) => Math.max(1, current - 1))}
          isDisabled={page <= 1}
        >
          {t('welfareMy.pagination.prev')}
        </Button>
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={() => setPage((current) => current + 1)}
          isDisabled={transactions.length < pageSize || page * pageSize >= total}
        >
          {t('welfareMy.pagination.next')}
        </Button>
        <label className="flex items-center gap-2">
          <span>{t('welfareMy.pageSize')}</span>
          <Select
            selectedKey={String(pageSize ?? '')}
            onSelectionChange={(key) => ((value) => {
              setPageSize(Number(value))
              setPage(1)
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
    </section>
  )
}
