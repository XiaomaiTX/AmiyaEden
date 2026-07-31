import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '@/components/ui/table'
import { useCallback, useEffect, useMemo, useState } from 'react'
import {
  batchPayoutAsFuxiCoin,
  batchPayoutByUser,
  fetchApplicationList,
  fetchBatchPayoutSummary,
  payoutApplication,
  reviewApplication,
  runFleetAutoApproval,
} from '@/api/srp'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { useI18n } from '@/i18n'
import type { Application, BatchPayoutSummary } from '@/types/api/srp'

function getErrorMessage(error: unknown, fallback: string) {
  return error instanceof Error && error.message ? error.message : fallback
}

export function SrpManagePage() {
  const { t } = useI18n()
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [activeTab, setActiveTab] = useState<'pending' | 'history'>('pending')
  const [filter, setFilter] = useState({ keyword: '', status: '' })
  const [items, setItems] = useState<Application[]>([])
  const [summary, setSummary] = useState<BatchPayoutSummary[]>([])
  const [actionId, setActionId] = useState<number | null>(null)

  const loadData = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const [applications, batchSummary] = await Promise.all([
        fetchApplicationList({
          current: 1,
          size: 200,
          review_status:
            activeTab === 'pending' ? 'submitted' : filter.status || undefined,
          keyword: filter.keyword.trim() || undefined,
        }),
        fetchBatchPayoutSummary(),
      ])
      setItems(applications.list ?? [])
      setSummary(batchSummary ?? [])
    } catch (caughtError) {
      setError(getErrorMessage(caughtError, t('srpManage.loadFailed')))
      setItems([])
      setSummary([])
    } finally {
      setLoading(false)
    }
  }, [activeTab, filter.keyword, filter.status, t])

  useEffect(() => {
    const timer = window.setTimeout(() => {
      void loadData()
    }, 0)
    return () => window.clearTimeout(timer)
  }, [loadData])

  const filteredItems = useMemo(() => {
    if (activeTab === 'pending') {
      return items.filter((item) => item.review_status === 'submitted')
    }
    return items.filter((item) => item.review_status !== 'submitted')
  }, [activeTab, items])

  const approve = async (item: Application) => {
    setActionId(item.id)
    try {
      await reviewApplication(item.id, { action: 'approve', final_amount: item.final_amount || item.recommended_amount })
      await loadData()
    } catch (caughtError) {
      setError(getErrorMessage(caughtError, t('srpManage.actionFailed')))
    } finally {
      setActionId(null)
    }
  }

  const reject = async (item: Application) => {
    setActionId(item.id)
    try {
      await reviewApplication(item.id, { action: 'reject', review_note: 'Rejected in React' })
      await loadData()
    } catch (caughtError) {
      setError(getErrorMessage(caughtError, t('srpManage.actionFailed')))
    } finally {
      setActionId(null)
    }
  }

  const payout = async (item: Application) => {
    setActionId(item.id)
    try {
      await payoutApplication(item.id, { final_amount: item.final_amount || item.recommended_amount })
      await loadData()
    } catch (caughtError) {
      setError(getErrorMessage(caughtError, t('srpManage.actionFailed')))
    } finally {
      setActionId(null)
    }
  }

  const autoApprove = async () => {
    const fleetId = window.prompt(t('srpManage.autoApprovePrompt')) ?? ''
    if (!fleetId.trim()) return
    await runFleetAutoApproval({ fleet_id: fleetId.trim() })
    await loadData()
  }

  const batchPayout = async () => {
    if (!window.confirm(t('srpManage.batchPayoutConfirm'))) return
    await batchPayoutAsFuxiCoin()
    await loadData()
  }

  const batchPayoutBySelected = async (userId: number) => {
    await batchPayoutByUser(userId)
    await loadData()
  }

  return (
    <section className="space-y-4">
      <div className="rounded-lg border bg-card p-5">
        <div className="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
          <div>
            <h1 className="text-xl font-semibold">{t('srpManage.title')}</h1>
            <p className="mt-1 text-sm text-muted-foreground">{t('srpManage.subtitle')}</p>
          </div>
          <div className="flex flex-wrap items-end gap-3">
            <Input
              value={filter.keyword}
              onChange={(event) => setFilter((current) => ({ ...current, keyword: event.target.value }))}
              placeholder={t('srpManage.keywordPlaceholder')}
            />
            <Button type="button" variant="outline" onClick={() => void loadData()}>
              {t('common.refresh')}
            </Button>
          </div>
        </div>
      </div>

      {error ? <p className="text-sm text-destructive">{error}</p> : null}
      {loading ? <p className="text-sm text-muted-foreground">{t('srpManage.loading')}</p> : null}

      <div className="rounded-lg border bg-card p-5">
        <div className="flex gap-2">
          <Button type="button" variant={activeTab === 'pending' ? 'default' : 'outline'} onClick={() => setActiveTab('pending')}>
            {t('srpManage.pendingTab')}
          </Button>
          <Button type="button" variant={activeTab === 'history' ? 'default' : 'outline'} onClick={() => setActiveTab('history')}>
            {t('srpManage.historyTab')}
          </Button>
          {activeTab === 'pending' ? (
            <>
              <Button type="button" variant="outline" onClick={() => void autoApprove()}>
                {t('srpManage.autoApproveBtn')}
              </Button>
              <Button type="button" variant="outline" onClick={() => void batchPayout()}>
                {t('srpManage.batchPayoutBtn')}
              </Button>
            </>
          ) : null}
        </div>

        <div className="mt-4 overflow-x-auto">
          <Table className="min-w-full text-sm">
            <TableHeader>
              <TableRow className="border-b bg-muted/40 text-left">
                <TableHead className="px-3 py-2">{t('srpManage.columns.killmail')}</TableHead>
                <TableHead className="px-3 py-2">{t('srpManage.columns.character')}</TableHead>
                <TableHead className="px-3 py-2">{t('srpManage.columns.ship')}</TableHead>
                <TableHead className="px-3 py-2">{t('srpManage.columns.reviewStatus')}</TableHead>
                <TableHead className="px-3 py-2">{t('srpManage.columns.finalAmount')}</TableHead>
                <TableHead className="px-3 py-2">{t('srpManage.columns.actions')}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {filteredItems.map((item) => (
                <TableRow key={item.id} className="border-b">
                  <TableCell className="px-3 py-2">{item.killmail_id}</TableCell>
                  <TableCell className="px-3 py-2">
                    <div className="font-medium">{item.character_name}</div>
                    <div className="text-xs text-muted-foreground">{item.nickname || '-'}</div>
                  </TableCell>
                  <TableCell className="px-3 py-2">{item.ship_name}</TableCell>
                  <TableCell className="px-3 py-2">{item.review_status}</TableCell>
                  <TableCell className="px-3 py-2">{item.final_amount || item.recommended_amount}</TableCell>
                  <TableCell className="px-3 py-2">
                    {activeTab === 'pending' ? (
                      <div className="flex flex-wrap gap-2">
                        <Button type="button" size="sm" onClick={() => void approve(item)} isDisabled={actionId === item.id}>
                          {t('srpManage.approveBtn')}
                        </Button>
                        <Button type="button" size="sm" variant="outline" onClick={() => void reject(item)} isDisabled={actionId === item.id}>
                          {t('srpManage.rejectBtn')}
                        </Button>
                      </div>
                    ) : (
                      <div className="flex flex-wrap gap-2">
                        <Button type="button" size="sm" variant="outline" onClick={() => void payout(item)} isDisabled={actionId === item.id}>
                          {t('srpManage.payoutBtn')}
                        </Button>
                        <Button type="button" size="sm" variant="outline" onClick={() => void batchPayoutBySelected(item.user_id)} isDisabled={actionId === item.id}>
                          {t('srpManage.batchByUserBtn')}
                        </Button>
                      </div>
                    )}
                  </TableCell>
                </TableRow>
              ))}
              {!loading && filteredItems.length === 0 ? (
                <TableRow>
                  <TableCell className="px-3 py-6 text-center text-muted-foreground" colSpan={6}>
                    {t('srpManage.empty')}
                  </TableCell>
                </TableRow>
              ) : null}
            </TableBody>
          </Table>
        </div>
      </div>

      <div className="rounded-lg border bg-card p-5">
        <h2 className="text-lg font-semibold">{t('srpManage.batchSummaryTitle')}</h2>
        <div className="mt-4 overflow-x-auto">
          <Table className="min-w-full text-sm">
            <TableHeader>
              <TableRow className="border-b bg-muted/40 text-left">
                <TableHead className="px-3 py-2">{t('srpManage.columns.user')}</TableHead>
                <TableHead className="px-3 py-2">{t('srpManage.columns.amount')}</TableHead>
                <TableHead className="px-3 py-2">{t('srpManage.columns.count')}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {summary.map((row) => (
                <TableRow key={row.user_id} className="border-b">
                  <TableCell className="px-3 py-2">{row.main_character_name}</TableCell>
                  <TableCell className="px-3 py-2">{row.total_amount}</TableCell>
                  <TableCell className="px-3 py-2">{row.application_count}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      </div>
    </section>
  )
}
