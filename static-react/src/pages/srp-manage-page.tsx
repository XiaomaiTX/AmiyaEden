import { useEffect, useMemo, useState } from 'react'
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

  const loadData = async () => {
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
  }

  useEffect(() => {
    void loadData()
  }, [activeTab])

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
          <table className="min-w-full text-sm">
            <thead>
              <tr className="border-b bg-muted/40 text-left">
                <th className="px-3 py-2">{t('srpManage.columns.killmail')}</th>
                <th className="px-3 py-2">{t('srpManage.columns.character')}</th>
                <th className="px-3 py-2">{t('srpManage.columns.ship')}</th>
                <th className="px-3 py-2">{t('srpManage.columns.reviewStatus')}</th>
                <th className="px-3 py-2">{t('srpManage.columns.finalAmount')}</th>
                <th className="px-3 py-2">{t('srpManage.columns.actions')}</th>
              </tr>
            </thead>
            <tbody>
              {filteredItems.map((item) => (
                <tr key={item.id} className="border-b">
                  <td className="px-3 py-2">{item.killmail_id}</td>
                  <td className="px-3 py-2">
                    <div className="font-medium">{item.character_name}</div>
                    <div className="text-xs text-muted-foreground">{item.nickname || '-'}</div>
                  </td>
                  <td className="px-3 py-2">{item.ship_name}</td>
                  <td className="px-3 py-2">{item.review_status}</td>
                  <td className="px-3 py-2">{item.final_amount || item.recommended_amount}</td>
                  <td className="px-3 py-2">
                    {activeTab === 'pending' ? (
                      <div className="flex flex-wrap gap-2">
                        <Button type="button" size="sm" onClick={() => void approve(item)} disabled={actionId === item.id}>
                          {t('srpManage.approveBtn')}
                        </Button>
                        <Button type="button" size="sm" variant="outline" onClick={() => void reject(item)} disabled={actionId === item.id}>
                          {t('srpManage.rejectBtn')}
                        </Button>
                      </div>
                    ) : (
                      <div className="flex flex-wrap gap-2">
                        <Button type="button" size="sm" variant="outline" onClick={() => void payout(item)} disabled={actionId === item.id}>
                          {t('srpManage.payoutBtn')}
                        </Button>
                        <Button type="button" size="sm" variant="outline" onClick={() => void batchPayoutBySelected(item.user_id)} disabled={actionId === item.id}>
                          {t('srpManage.batchByUserBtn')}
                        </Button>
                      </div>
                    )}
                  </td>
                </tr>
              ))}
              {!loading && filteredItems.length === 0 ? (
                <tr>
                  <td className="px-3 py-6 text-center text-muted-foreground" colSpan={6}>
                    {t('srpManage.empty')}
                  </td>
                </tr>
              ) : null}
            </tbody>
          </table>
        </div>
      </div>

      <div className="rounded-lg border bg-card p-5">
        <h2 className="text-lg font-semibold">{t('srpManage.batchSummaryTitle')}</h2>
        <div className="mt-4 overflow-x-auto">
          <table className="min-w-full text-sm">
            <thead>
              <tr className="border-b bg-muted/40 text-left">
                <th className="px-3 py-2">{t('srpManage.columns.user')}</th>
                <th className="px-3 py-2">{t('srpManage.columns.amount')}</th>
                <th className="px-3 py-2">{t('srpManage.columns.count')}</th>
              </tr>
            </thead>
            <tbody>
              {summary.map((row) => (
                <tr key={row.user_id} className="border-b">
                  <td className="px-3 py-2">{row.main_character_name}</td>
                  <td className="px-3 py-2">{row.total_amount}</td>
                  <td className="px-3 py-2">{row.application_count}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </section>
  )
}
