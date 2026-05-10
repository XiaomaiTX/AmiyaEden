import { useEffect, useMemo, useState, type Dispatch, type SetStateAction } from 'react'
import {
  adminAdjustWallet,
  adminGetWalletAnalytics,
  adminListTransactions,
  adminListWalletLogs,
  adminListWallets,
} from '@/api/sys-wallet'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { useI18n } from '@/i18n'
import type {
  AnalyticsParams,
  Wallet,
  WalletAnalytics,
  WalletLog,
  WalletTransaction,
} from '@/types/api/sys-wallet'
import { formatCoin, formatDateTime, formatSignedCoin, getErrorMessage, ShopDialog } from './shop-page-utils'

type WalletTab = 'wallets' | 'transactions' | 'logs' | 'analysis'
type AdjustAction = 'add' | 'deduct' | 'set'

type AdjustFormState = {
  targetUid: number | ''
  action: AdjustAction
  amount: number
  reason: string
}

const defaultAdjustForm: AdjustFormState = {
  targetUid: '',
  action: 'add',
  amount: 0,
  reason: '',
}

function refTypeLabel(t: ReturnType<typeof useI18n>['t'], value: string) {
  const key = `walletAdmin.refTypes.${value}`
  const translated = t(key)
  return translated === key ? value : translated
}

function refTypeTone(value: string) {
  switch (value) {
    case 'pap_reward':
    case 'pap_fc_salary':
    case 'welfare_payout':
      return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-300'
    case 'admin_adjust':
    case 'shop_refund':
      return 'bg-amber-100 text-amber-700 dark:bg-amber-500/10 dark:text-amber-300'
    case 'admin_award':
    case 'srp_payout':
    case 'newbro_captain_reward':
    case 'mentor_reward':
      return 'bg-sky-100 text-sky-700 dark:bg-sky-500/10 dark:text-sky-300'
    default:
      return 'bg-muted text-muted-foreground'
  }
}

function adjustActionTone(action: AdjustAction) {
  switch (action) {
    case 'add':
      return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-300'
    case 'deduct':
      return 'bg-rose-100 text-rose-700 dark:bg-rose-500/10 dark:text-rose-300'
    default:
      return 'bg-amber-100 text-amber-700 dark:bg-amber-500/10 dark:text-amber-300'
  }
}

export function SystemWalletPage() {
  const { t } = useI18n()
  const [activeTab, setActiveTab] = useState<WalletTab>('wallets')
  const [refreshSeed, setRefreshSeed] = useState(0)
  const [transactionsUserId, setTransactionsUserId] = useState<number | ''>('')
  const [adjustOpen, setAdjustOpen] = useState(false)
  const [adjustSaving, setAdjustSaving] = useState(false)
  const [adjustForm, setAdjustForm] = useState<AdjustFormState>(defaultAdjustForm)
  const [error, setError] = useState<string | null>(null)

  const openAdjustDialog = (userId: number, action: AdjustAction) => {
    setAdjustForm({
      targetUid: userId,
      action,
      amount: 0,
      reason: '',
    })
    setAdjustOpen(true)
  }

  const submitAdjust = async () => {
    if (!adjustForm.targetUid) {
      setError(t('walletAdmin.validation.targetUserId'))
      return
    }
    if (!adjustForm.reason.trim()) {
      setError(t('walletAdmin.validation.reason'))
      return
    }

    setAdjustSaving(true)
    setError(null)
    try {
      await adminAdjustWallet({
        target_uid: Number(adjustForm.targetUid),
        action: adjustForm.action,
        amount: adjustForm.amount,
        reason: adjustForm.reason.trim(),
      })
      setAdjustOpen(false)
      setRefreshSeed((current) => current + 1)
    } catch (caughtError) {
      setError(getErrorMessage(caughtError, t('walletAdmin.messages.actionFailed')))
    } finally {
      setAdjustSaving(false)
    }
  }

  return (
    <section className="space-y-4">
      <div className="rounded-lg border bg-card p-5">
        <div className="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
          <div>
            <h1 className="text-xl font-semibold">{t('walletAdmin.title')}</h1>
            <p className="mt-1 text-sm text-muted-foreground">{t('walletAdmin.subtitle')}</p>
          </div>
          <div className="flex flex-wrap gap-2">
            {([
              ['wallets', t('walletAdmin.tabs.wallets')],
              ['transactions', t('walletAdmin.tabs.transactions')],
              ['logs', t('walletAdmin.tabs.logs')],
              ['analysis', t('walletAdmin.tabs.analysis')],
            ] as const).map(([key, label]) => (
              <Button
                key={key}
                type="button"
                variant={activeTab === key ? 'default' : 'outline'}
                onClick={() => setActiveTab(key)}
              >
                {label}
              </Button>
            ))}
          </div>
        </div>
      </div>

      {error ? <p className="text-sm text-destructive">{error}</p> : null}

      {activeTab === 'wallets' ? (
        <WalletListPanel
          t={t}
          refreshSeed={refreshSeed}
          onAdjust={openAdjustDialog}
          onViewTransactions={(userId) => {
            setTransactionsUserId(userId)
            setActiveTab('transactions')
          }}
        />
      ) : null}
      {activeTab === 'transactions' ? (
        <WalletTransactionsPanel
          t={t}
          initialUserId={transactionsUserId}
          refreshSeed={refreshSeed}
        />
      ) : null}
      {activeTab === 'logs' ? <WalletLogsPanel t={t} refreshSeed={refreshSeed} /> : null}
      {activeTab === 'analysis' ? <WalletAnalysisPanel t={t} refreshSeed={refreshSeed} /> : null}

      <ShopDialog
        open={adjustOpen}
        title={t('walletAdmin.adjustTitle')}
        onClose={() => setAdjustOpen(false)}
        closeLabel={t('common.close')}
        widthClass="max-w-xl"
        footer={
          <>
            <Button type="button" variant="outline" onClick={() => setAdjustOpen(false)} disabled={adjustSaving}>
              {t('common.cancel')}
            </Button>
            <Button type="button" onClick={() => void submitAdjust()} disabled={adjustSaving}>
              {adjustSaving ? t('walletAdmin.messages.saving') : t('common.confirm')}
            </Button>
          </>
        }
      >
        <div className="grid gap-4 md:grid-cols-2">
          <label className="space-y-2">
            <span className="text-sm text-muted-foreground">{t('walletAdmin.fields.targetUserId')}</span>
            <Input
              type="number"
              min={1}
              value={String(adjustForm.targetUid)}
              onChange={(event) =>
                setAdjustForm((current) => ({ ...current, targetUid: Number(event.target.value) }))
              }
            />
          </label>
          <label className="space-y-2">
            <span className="text-sm text-muted-foreground">{t('walletAdmin.fields.action')}</span>
            <select
              className="h-10 rounded-md border border-input bg-background px-3 text-sm"
              value={adjustForm.action}
              onChange={(event) =>
                setAdjustForm((current) => ({ ...current, action: event.target.value as AdjustAction }))
              }
            >
              <option value="add">{t('walletAdmin.actions.add')}</option>
              <option value="deduct">{t('walletAdmin.actions.deduct')}</option>
              <option value="set">{t('walletAdmin.actions.set')}</option>
            </select>
          </label>
          <label className="space-y-2">
            <span className="text-sm text-muted-foreground">{t('walletAdmin.fields.amount')}</span>
            <Input
              type="number"
              min={0}
              step="0.01"
              value={String(adjustForm.amount)}
              onChange={(event) =>
                setAdjustForm((current) => ({ ...current, amount: Number(event.target.value) }))
              }
            />
          </label>
          <label className="space-y-2 md:col-span-2">
            <span className="text-sm text-muted-foreground">{t('walletAdmin.fields.reason')}</span>
            <textarea
              className="min-h-24 w-full rounded-lg border border-input bg-background px-3 py-2 text-sm outline-none"
              value={adjustForm.reason}
              onChange={(event) => setAdjustForm((current) => ({ ...current, reason: event.target.value }))}
              placeholder={t('walletAdmin.placeholders.reason')}
            />
          </label>
        </div>
      </ShopDialog>
    </section>
  )
}

function WalletListPanel({
  t,
  refreshSeed,
  onAdjust,
  onViewTransactions,
}: {
  t: ReturnType<typeof useI18n>['t']
  refreshSeed: number
  onAdjust: (userId: number, action: AdjustAction) => void
  onViewTransactions: (userId: number) => void
}) {
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [users, setUsers] = useState<Wallet[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)
  const [userKeyword, setUserKeyword] = useState('')
  const [searchState, setSearchState] = useState('')

  const pageCount = useMemo(() => Math.max(1, Math.ceil(total / pageSize) || 1), [pageSize, total])

  const loadData = async () => {
    setLoading(true)
    setError(null)
    try {
      const response = await adminListWallets({
        current: page,
        size: pageSize,
        user_keyword: searchState.trim() || undefined,
      })
      setUsers(response.list ?? [])
      setTotal(response.total ?? 0)
      setPage(response.page ?? page)
      setPageSize(response.pageSize ?? pageSize)
    } catch (caughtError) {
      setError(getErrorMessage(caughtError, t('walletAdmin.loadFailed')))
      setUsers([])
      setTotal(0)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    const timer = window.setTimeout(() => {
      void loadData()
    }, 0)
    return () => window.clearTimeout(timer)
  }, [page, pageSize, refreshSeed, searchState])

  return (
    <div className="space-y-4">
      <div className="rounded-lg border bg-card p-4">
        <div className="flex flex-wrap items-center gap-3">
          <Input
            className="w-60"
            value={userKeyword}
            onChange={(event) => setUserKeyword(event.target.value)}
            placeholder={t('walletAdmin.placeholders.userKeywordFilter')}
          />
          <Button
            type="button"
            variant="outline"
            onClick={() => {
              setSearchState(userKeyword)
              setPage(1)
            }}
          >
            {t('common.search')}
          </Button>
          <Button
            type="button"
            variant="outline"
            onClick={() => {
              setUserKeyword('')
              setSearchState('')
              setPage(1)
            }}
          >
            {t('common.reset')}
          </Button>
        </div>
      </div>

      <div className="overflow-hidden rounded-lg border bg-card">
        <div className="border-b px-4 py-3 text-sm font-medium">{t('walletAdmin.tabs.wallets')}</div>
        <div className="overflow-x-auto">
          {error ? <p className="px-4 py-3 text-sm text-destructive">{error}</p> : null}
          {loading ? <p className="px-4 py-3 text-sm text-muted-foreground">{t('walletAdmin.loading')}</p> : null}
          <table className="min-w-full text-sm">
            <thead>
              <tr className="border-b bg-muted/40 text-left">
                <th className="px-3 py-2">#</th>
                <th className="px-3 py-2">{t('walletAdmin.transactions.userId')}</th>
                <th className="px-3 py-2">{t('walletAdmin.transactions.characterName')}</th>
                <th className="px-3 py-2">{t('walletAdmin.wallets.balance')}</th>
                <th className="px-3 py-2">{t('common.updatedAt')}</th>
                <th className="px-3 py-2">{t('common.operation')}</th>
              </tr>
            </thead>
            <tbody>
              {users.map((wallet, index) => (
                <tr key={wallet.id} className="border-b align-top">
                  <td className="px-3 py-2">{index + 1}</td>
                  <td className="px-3 py-2">{wallet.user_id}</td>
                  <td className="px-3 py-2">{wallet.character_name || '-'}</td>
                  <td className="px-3 py-2 font-medium">
                    <span
                      className={
                        wallet.balance >= 0
                          ? 'text-emerald-600 dark:text-emerald-400'
                          : 'text-rose-600 dark:text-rose-400'
                      }
                    >
                      {formatCoin(wallet.balance)}
                    </span>
                  </td>
                  <td className="px-3 py-2">{formatDateTime(wallet.updated_at)}</td>
                  <td className="px-3 py-2">
                    <div className="flex flex-wrap gap-2">
                      <Button type="button" size="sm" variant="outline" onClick={() => onAdjust(wallet.user_id, 'add')}>
                        {t('walletAdmin.actions.add')}
                      </Button>
                      <Button type="button" size="sm" variant="outline" onClick={() => onAdjust(wallet.user_id, 'deduct')}>
                        {t('walletAdmin.actions.deduct')}
                      </Button>
                      <Button type="button" size="sm" variant="outline" onClick={() => onViewTransactions(wallet.user_id)}>
                        {t('walletAdmin.actions.transactions')}
                      </Button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      <Pager page={page} pageCount={pageCount} pageSize={pageSize} setPage={setPage} setPageSize={setPageSize} />
    </div>
  )
}

function WalletTransactionsPanel({
  t,
  initialUserId,
  refreshSeed,
}: {
  t: ReturnType<typeof useI18n>['t']
  initialUserId: number | ''
  refreshSeed: number
}) {
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [rows, setRows] = useState<WalletTransaction[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)
  const [userId, setUserId] = useState<number | ''>(initialUserId)
  const [userKeyword, setUserKeyword] = useState('')
  const [refType, setRefType] = useState('')
  const [searchState, setSearchState] = useState({ userId: initialUserId, userKeyword: '', refType: '' })

  useEffect(() => {
    setUserId(initialUserId)
    setSearchState((current) => ({ ...current, userId: initialUserId }))
    setPage(1)
  }, [initialUserId])

  const pageCount = useMemo(() => Math.max(1, Math.ceil(total / pageSize) || 1), [pageSize, total])

  const loadData = async () => {
    setLoading(true)
    setError(null)
    try {
      const response = await adminListTransactions({
        current: page,
        size: pageSize,
        user_id: searchState.userId === '' ? undefined : searchState.userId,
        user_keyword: searchState.userKeyword.trim() || undefined,
        ref_type: searchState.refType || undefined,
      })
      setRows(response.list ?? [])
      setTotal(response.total ?? 0)
      setPage(response.page ?? page)
      setPageSize(response.pageSize ?? pageSize)
    } catch (caughtError) {
      setError(getErrorMessage(caughtError, t('walletAdmin.loadFailed')))
      setRows([])
      setTotal(0)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    const timer = window.setTimeout(() => {
      void loadData()
    }, 0)
    return () => window.clearTimeout(timer)
  }, [page, pageSize, refreshSeed, searchState])

  return (
    <div className="space-y-4">
      <div className="rounded-lg border bg-card p-4">
        <div className="flex flex-wrap items-center gap-3">
          <Input
            className="w-60"
            value={userKeyword}
            onChange={(event) => setUserKeyword(event.target.value)}
            placeholder={t('walletAdmin.placeholders.userKeywordFilter')}
          />
          <Input
            className="w-40"
            type="number"
            value={String(userId)}
            onChange={(event) => setUserId(event.target.value ? Number(event.target.value) : '')}
            placeholder={t('walletAdmin.placeholders.targetUserId')}
          />
          <select
            className="h-10 rounded-md border border-input bg-background px-3 text-sm"
            value={refType}
            onChange={(event) => setRefType(event.target.value)}
          >
            <option value="">{t('walletAdmin.placeholders.refType')}</option>
            {[
              'pap_reward',
              'pap_fc_salary',
              'admin_adjust',
              'admin_award',
              'manual',
              'srp_payout',
              'welfare_payout',
              'shop_purchase',
              'shop_refund',
              'newbro_captain_reward',
              'mentor_reward',
            ].map((value) => (
              <option key={value} value={value}>
                {refTypeLabel(t, value)}
              </option>
            ))}
          </select>
          <Button
            type="button"
            variant="outline"
            onClick={() =>
              setSearchState({
                userId,
                userKeyword,
                refType,
              })
            }
          >
            {t('common.search')}
          </Button>
          <Button
            type="button"
            variant="outline"
            onClick={() => {
              setUserKeyword('')
              setUserId(initialUserId)
              setRefType('')
              setSearchState({ userId: initialUserId, userKeyword: '', refType: '' })
              setPage(1)
            }}
          >
            {t('common.reset')}
          </Button>
        </div>
      </div>

      <div className="overflow-hidden rounded-lg border bg-card">
        <div className="border-b px-4 py-3 text-sm font-medium">{t('walletAdmin.tabs.transactions')}</div>
        <div className="overflow-x-auto">
          {error ? <p className="px-4 py-3 text-sm text-destructive">{error}</p> : null}
          {loading ? <p className="px-4 py-3 text-sm text-muted-foreground">{t('walletAdmin.loading')}</p> : null}
          <table className="min-w-full text-sm">
            <thead>
              <tr className="border-b bg-muted/40 text-left">
                <th className="px-3 py-2">#</th>
                <th className="px-3 py-2">{t('walletAdmin.transactions.userId')}</th>
                <th className="px-3 py-2">{t('walletAdmin.transactions.characterName')}</th>
                <th className="px-3 py-2">{t('walletAdmin.transactions.amount')}</th>
                <th className="px-3 py-2">{t('walletAdmin.transactions.balanceAfter')}</th>
                <th className="px-3 py-2">{t('common.reason')}</th>
                <th className="px-3 py-2">{t('common.type')}</th>
                <th className="px-3 py-2">{t('walletAdmin.transactions.operator')}</th>
                <th className="px-3 py-2">{t('common.createdAt')}</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((row, index) => (
                <tr key={row.id} className="border-b align-top">
                  <td className="px-3 py-2">{index + 1}</td>
                  <td className="px-3 py-2">{row.user_id}</td>
                  <td className="px-3 py-2">{row.character_name || '-'}</td>
                  <td className="px-3 py-2 font-medium">
                    <span
                      className={row.amount >= 0 ? 'text-emerald-600 dark:text-emerald-400' : 'text-rose-600 dark:text-rose-400'}
                    >
                      {formatSignedCoin(row.amount)}
                    </span>
                  </td>
                  <td className="px-3 py-2">{formatCoin(row.balance_after)}</td>
                  <td className="px-3 py-2">{row.reason || '-'}</td>
                  <td className="px-3 py-2">
                    <span
                      className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${refTypeTone(row.ref_type)}`}
                    >
                      {refTypeLabel(t, row.ref_type)}
                    </span>
                  </td>
                  <td className="px-3 py-2">{row.operator_name || (row.operator_id === 0 ? t('walletAdmin.actions.system') : `#${row.operator_id}`)}</td>
                  <td className="px-3 py-2">{formatDateTime(row.created_at)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      <Pager page={page} pageCount={pageCount} pageSize={pageSize} setPage={setPage} setPageSize={setPageSize} />
    </div>
  )
}

function WalletLogsPanel({
  t,
  refreshSeed,
}: {
  t: ReturnType<typeof useI18n>['t']
  refreshSeed: number
}) {
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [rows, setRows] = useState<WalletLog[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)
  const [operatorId, setOperatorId] = useState('')
  const [targetUid, setTargetUid] = useState('')
  const [action, setAction] = useState('')
  const [searchState, setSearchState] = useState({ operatorId: '', targetUid: '', action: '' })

  const pageCount = useMemo(() => Math.max(1, Math.ceil(total / pageSize) || 1), [pageSize, total])

  const loadData = async () => {
    setLoading(true)
    setError(null)
    try {
      const response = await adminListWalletLogs({
        current: page,
        size: pageSize,
        operator_id: searchState.operatorId === '' ? undefined : Number(searchState.operatorId),
        target_uid: searchState.targetUid === '' ? undefined : Number(searchState.targetUid),
        action: searchState.action || undefined,
      })
      setRows(response.list ?? [])
      setTotal(response.total ?? 0)
      setPage(response.page ?? page)
      setPageSize(response.pageSize ?? pageSize)
    } catch (caughtError) {
      setError(getErrorMessage(caughtError, t('walletAdmin.loadFailed')))
      setRows([])
      setTotal(0)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    const timer = window.setTimeout(() => {
      void loadData()
    }, 0)
    return () => window.clearTimeout(timer)
  }, [page, pageSize, refreshSeed, searchState])

  return (
    <div className="space-y-4">
      <div className="rounded-lg border bg-card p-4">
        <div className="flex flex-wrap items-center gap-3">
          <Input
            className="w-40"
            type="number"
            value={operatorId}
            onChange={(event) => setOperatorId(event.target.value)}
            placeholder={t('walletAdmin.placeholders.operatorId')}
          />
          <Input
            className="w-40"
            type="number"
            value={targetUid}
            onChange={(event) => setTargetUid(event.target.value)}
            placeholder={t('walletAdmin.placeholders.targetUserId')}
          />
          <select
            className="h-10 rounded-md border border-input bg-background px-3 text-sm"
            value={action}
            onChange={(event) => setAction(event.target.value)}
          >
            <option value="">{t('walletAdmin.placeholders.action')}</option>
            <option value="add">{t('walletAdmin.actions.add')}</option>
            <option value="deduct">{t('walletAdmin.actions.deduct')}</option>
            <option value="set">{t('walletAdmin.actions.set')}</option>
          </select>
          <Button
            type="button"
            variant="outline"
            onClick={() =>
              setSearchState({
                operatorId,
                targetUid,
                action,
              })
            }
          >
            {t('common.search')}
          </Button>
          <Button
            type="button"
            variant="outline"
            onClick={() => {
              setOperatorId('')
              setTargetUid('')
              setAction('')
              setSearchState({ operatorId: '', targetUid: '', action: '' })
              setPage(1)
            }}
          >
            {t('common.reset')}
          </Button>
        </div>
      </div>

      <div className="overflow-hidden rounded-lg border bg-card">
        <div className="border-b px-4 py-3 text-sm font-medium">{t('walletAdmin.tabs.logs')}</div>
        <div className="overflow-x-auto">
          {error ? <p className="px-4 py-3 text-sm text-destructive">{error}</p> : null}
          {loading ? <p className="px-4 py-3 text-sm text-muted-foreground">{t('walletAdmin.loading')}</p> : null}
          <table className="min-w-full text-sm">
            <thead>
              <tr className="border-b bg-muted/40 text-left">
                <th className="px-3 py-2">#</th>
                <th className="px-3 py-2">{t('walletAdmin.logs.targetUser')}</th>
                <th className="px-3 py-2">{t('walletAdmin.transactions.characterName')}</th>
                <th className="px-3 py-2">{t('walletAdmin.logs.operator')}</th>
                <th className="px-3 py-2">{t('common.type')}</th>
                <th className="px-3 py-2">{t('common.amount')}</th>
                <th className="px-3 py-2">{t('walletAdmin.logs.before')}</th>
                <th className="px-3 py-2">{t('walletAdmin.logs.after')}</th>
                <th className="px-3 py-2">{t('common.reason')}</th>
                <th className="px-3 py-2">{t('common.createdAt')}</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((row, index) => (
                <tr key={row.id} className="border-b align-top">
                  <td className="px-3 py-2">{index + 1}</td>
                  <td className="px-3 py-2">{row.target_uid}</td>
                  <td className="px-3 py-2">{row.target_character_name || '-'}</td>
                  <td className="px-3 py-2">{row.operator_character_name || row.operator_id}</td>
                  <td className="px-3 py-2">
                    <span
                      className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${adjustActionTone(
                        row.action
                      )}`}
                    >
                      {t(`walletAdmin.actions.${row.action}`)}
                    </span>
                  </td>
                  <td className="px-3 py-2">{formatCoin(row.amount)}</td>
                  <td className="px-3 py-2">{formatCoin(row.before)}</td>
                  <td className="px-3 py-2">{formatCoin(row.after)}</td>
                  <td className="px-3 py-2">{row.reason || '-'}</td>
                  <td className="px-3 py-2">{formatDateTime(row.created_at)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      <Pager page={page} pageCount={pageCount} pageSize={pageSize} setPage={setPage} setPageSize={setPageSize} />
    </div>
  )
}

function WalletAnalysisPanel({
  t,
  refreshSeed,
}: {
  t: ReturnType<typeof useI18n>['t']
  refreshSeed: number
}) {
  const [dateRange, setDateRange] = useState<[string, string]>(() => {
    const end = new Date().toISOString().slice(0, 10)
    const start = new Date(Date.now() - 29 * 24 * 60 * 60 * 1000).toISOString().slice(0, 10)
    return [start, end]
  })
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [analytics, setAnalytics] = useState<WalletAnalytics | null>(null)
  const [refTypes, setRefTypes] = useState<string[]>([])
  const [userKeyword, setUserKeyword] = useState('')
  const [searchState, setSearchState] = useState<AnalyticsParams>({
    start_date: dateRange[0],
    end_date: dateRange[1],
    ref_types: [],
    user_keyword: '',
    top_n: 10,
  })

  const refTypeOptions = [
    'pap_reward',
    'pap_fc_salary',
    'admin_adjust',
    'admin_award',
    'manual',
    'srp_payout',
    'welfare_payout',
    'shop_purchase',
    'shop_refund',
    'newbro_captain_reward',
    'mentor_reward',
    'recruit_link_reward',
  ]

  const loadData = async () => {
    setLoading(true)
    setError(null)
    try {
      setAnalytics(
        await adminGetWalletAnalytics({
          start_date: searchState.start_date,
          end_date: searchState.end_date,
          ref_types: searchState.ref_types,
          user_keyword: searchState.user_keyword || undefined,
          top_n: searchState.top_n,
        })
      )
    } catch (caughtError) {
      setError(getErrorMessage(caughtError, t('walletAdmin.analysis.loadFailed')))
      setAnalytics(null)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    const timer = window.setTimeout(() => {
      void loadData()
    }, 0)
    return () => window.clearTimeout(timer)
  }, [refreshSeed, searchState])

  const summaryCards = [
    { key: 'wallet_count', label: t('walletAdmin.analysis.walletCount'), value: analytics?.summary.wallet_count ?? 0 },
    {
      key: 'active_wallet_count',
      label: t('walletAdmin.analysis.activeWalletCount'),
      value: analytics?.summary.active_wallet_count ?? 0,
    },
    {
      key: 'total_balance',
      label: t('walletAdmin.analysis.totalBalance'),
      value: formatCoin(analytics?.summary.total_balance ?? 0),
    },
    {
      key: 'income_total',
      label: t('walletAdmin.analysis.incomeTotal'),
      value: formatCoin(analytics?.summary.income_total ?? 0),
    },
    {
      key: 'expense_total',
      label: t('walletAdmin.analysis.expenseTotal'),
      value: formatCoin(analytics?.summary.expense_total ?? 0),
    },
    {
      key: 'net_flow',
      label: t('walletAdmin.analysis.netFlow'),
      value: formatCoin(analytics?.summary.net_flow ?? 0),
    },
  ]

  return (
    <div className="space-y-4">
      <div className="rounded-lg border bg-card p-4">
        <div className="flex flex-wrap items-center gap-3">
          <input
            className="h-10 rounded-md border border-input bg-background px-3 text-sm"
            type="date"
            value={dateRange[0]}
            onChange={(event) => setDateRange((current) => [event.target.value, current[1]])}
          />
          <input
            className="h-10 rounded-md border border-input bg-background px-3 text-sm"
            type="date"
            value={dateRange[1]}
            onChange={(event) => setDateRange((current) => [current[0], event.target.value])}
          />
          <select
            multiple
            className="min-h-10 rounded-md border border-input bg-background px-3 text-sm"
            value={refTypes}
            onChange={(event) =>
              setRefTypes(Array.from(event.target.selectedOptions).map((option) => option.value))
            }
          >
            {refTypeOptions.map((value) => (
              <option key={value} value={value}>
                {refTypeLabel(t, value)}
              </option>
            ))}
          </select>
          <Input
            className="w-60"
            value={userKeyword}
            onChange={(event) => setUserKeyword(event.target.value)}
            placeholder={t('walletAdmin.analysis.userKeyword')}
          />
          <Button
            type="button"
            variant="outline"
            onClick={() =>
              setSearchState({
                start_date: dateRange[0],
                end_date: dateRange[1],
                ref_types: refTypes,
                user_keyword: userKeyword,
                top_n: 10,
              })
            }
          >
            {t('common.search')}
          </Button>
        </div>
      </div>

      {error ? <p className="text-sm text-destructive">{error}</p> : null}
      {loading ? <p className="text-sm text-muted-foreground">{t('walletAdmin.loading')}</p> : null}

      <div className="grid gap-3 md:grid-cols-3 xl:grid-cols-6">
        {summaryCards.map((card) => (
          <div key={card.key} className="rounded-lg border bg-card p-4">
            <div className="text-sm text-muted-foreground">{card.label}</div>
            <div className="mt-2 text-xl font-semibold">{card.value}</div>
          </div>
        ))}
      </div>

      {analytics ? (
        <div className="space-y-4">
          <SimpleTableCard
            title={t('walletAdmin.analysis.dailyTrend')}
            rows={analytics.daily_series}
            columns={[
              ['date', t('common.createdAt')],
              ['income', t('walletAdmin.analysis.incomeTotal')],
              ['expense', t('walletAdmin.analysis.expenseTotal')],
              ['net_flow', t('walletAdmin.analysis.netFlow')],
            ]}
          />
          <div className="grid gap-4 xl:grid-cols-2">
            <SimpleTableCard
              title={t('walletAdmin.analysis.topInflowUsers')}
              rows={analytics.top_inflow_users}
              columns={[
                ['user_id', t('walletAdmin.transactions.userId')],
                ['character_name', t('walletAdmin.transactions.characterName')],
                ['amount', t('walletAdmin.analysis.amount')],
              ]}
            />
            <SimpleTableCard
              title={t('walletAdmin.analysis.topOutflowUsers')}
              rows={analytics.top_outflow_users}
              columns={[
                ['user_id', t('walletAdmin.transactions.userId')],
                ['character_name', t('walletAdmin.transactions.characterName')],
                ['amount', t('walletAdmin.analysis.amount')],
              ]}
            />
          </div>
          <div className="grid gap-4 xl:grid-cols-3">
            <SimpleTableCard
              title={t('walletAdmin.analysis.largeTransactions')}
              rows={analytics.anomalies.large_transactions}
              columns={[
                ['user_id', t('walletAdmin.transactions.userId')],
                ['character_name', t('walletAdmin.transactions.characterName')],
                ['amount', t('walletAdmin.analysis.amount')],
                ['ref_type', t('common.type')],
              ]}
            />
            <SimpleTableCard
              title={t('walletAdmin.analysis.frequentAdjustments')}
              rows={analytics.anomalies.frequent_adjustments}
              columns={[
                ['target_uid', t('walletAdmin.logs.targetUser')],
                ['character_name', t('walletAdmin.transactions.characterName')],
                ['adjust_count', t('walletAdmin.analysis.adjustCount')],
                ['amount_total', t('walletAdmin.analysis.amountTotal')],
              ]}
            />
            <SimpleTableCard
              title={t('walletAdmin.analysis.operatorConcentration')}
              rows={analytics.anomalies.operator_concentration}
              columns={[
                ['operator_id', t('walletAdmin.logs.operator')],
                ['operator_name', t('walletAdmin.analysis.operatorName')],
                ['count', t('walletAdmin.analysis.adjustCount')],
                ['ratio', t('walletAdmin.analysis.ratio')],
              ]}
            />
          </div>
        </div>
      ) : null}
    </div>
  )
}

function SimpleTableCard<T extends object>({
  title,
  rows,
  columns,
}: {
  title: string
  rows: T[]
  columns: Array<[string, string]>
}) {
  return (
    <div className="overflow-hidden rounded-lg border bg-card">
      <div className="border-b px-4 py-3 text-sm font-medium">{title}</div>
      <div className="overflow-x-auto">
        <table className="min-w-full text-sm">
          <thead>
            <tr className="border-b bg-muted/40 text-left">
              {columns.map(([_, label]) => (
                <th key={label} className="px-3 py-2">
                  {label}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {rows.map((row, index) => (
              <tr key={index} className="border-b">
                {columns.map(([key]) => (
                  <td key={key} className="px-3 py-2">
                    {(() => {
                      const value = (row as Record<string, unknown>)[key]
                      return value == null ? '-' : String(value)
                    })()}
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}

function Pager({
  page,
  pageCount,
  pageSize,
  setPage,
  setPageSize,
}: {
  page: number
  pageCount: number
  pageSize: number
  setPage: Dispatch<SetStateAction<number>>
  setPageSize: Dispatch<SetStateAction<number>>
}) {
  const { t } = useI18n()

  return (
    <div className="flex flex-wrap items-center gap-3 text-sm">
      <span>
        {page}/{pageCount}
      </span>
      <Button type="button" size="sm" variant="outline" onClick={() => setPage((current) => Math.max(1, current - 1))} disabled={page <= 1}>
        {t('welfareMy.pagination.prev')}
      </Button>
      <Button
        type="button"
        size="sm"
        variant="outline"
        onClick={() => setPage((current) => current + 1)}
        disabled={page >= pageCount}
      >
        {t('welfareMy.pagination.next')}
      </Button>
      <label className="flex items-center gap-2">
        <span>{t('welfareMy.pageSize')}</span>
        <select
          className="h-8 rounded-md border border-input bg-background px-2 text-sm"
          value={pageSize}
          onChange={(event) => {
            setPageSize(Number(event.target.value))
            setPage(1)
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
  )
}
