import { useEffect, useState } from 'react'
import { adminDeleteApplication, adminListApplications, adminReviewApplication } from '@/api/welfare'
import { Button } from '@/components/ui/button'
import { useI18n } from '@/i18n'
import type { AdminApplication } from '@/types/api/welfare'

function getErrorMessage(error: unknown, fallback: string) {
  return error instanceof Error && error.message ? error.message : fallback
}

function formatTime(value: string | null) {
  if (!value) return '-'
  const parsed = new Date(value)
  return Number.isNaN(parsed.getTime()) ? value : parsed.toLocaleString()
}

export function WelfareApprovalPage() {
  const { t } = useI18n()
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [activeTab, setActiveTab] = useState<'pending' | 'history'>('pending')
  const [keyword, setKeyword] = useState('')
  const [pending, setPending] = useState<AdminApplication[]>([])
  const [history, setHistory] = useState<AdminApplication[]>([])
  const [actionId, setActionId] = useState<number | null>(null)

  const loadPending = async () => {
    const response = await adminListApplications({ current: 1, size: 200, status: 'requested' })
    setPending(response.list ?? [])
  }

  const loadHistory = async () => {
    const response = await adminListApplications({
      current: 1,
      size: 200,
      status: 'delivered,rejected',
      keyword: keyword.trim() || undefined,
    })
    setHistory(response.list ?? [])
  }

  const loadData = async () => {
    setLoading(true)
    setError(null)
    try {
      await Promise.all([loadPending(), loadHistory()])
    } catch (caughtError) {
      setError(getErrorMessage(caughtError, t('welfareApproval.loadFailed')))
      setPending([])
      setHistory([])
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void loadData()
  }, [keyword, t])

  const review = async (id: number, action: 'deliver' | 'reject') => {
    setActionId(id)
    try {
      await adminReviewApplication({ id, action })
      await loadPending()
      await loadHistory()
    } catch (caughtError) {
      setError(getErrorMessage(caughtError, t('welfareApproval.actionFailed')))
    } finally {
      setActionId(null)
    }
  }

  const removeHistory = async (id: number) => {
    if (!window.confirm(t('welfareApproval.deleteConfirm'))) return

    setActionId(id)
    try {
      await adminDeleteApplication(id)
      await loadHistory()
    } catch (caughtError) {
      setError(getErrorMessage(caughtError, t('welfareApproval.deleteFailed')))
    } finally {
      setActionId(null)
    }
  }

  return (
    <section className="space-y-4">
      <div className="rounded-lg border bg-card p-5">
        <div className="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
          <div>
            <h1 className="text-xl font-semibold">{t('welfareApproval.title')}</h1>
            <p className="mt-1 text-sm text-muted-foreground">{t('welfareApproval.subtitle')}</p>
          </div>
          <div className="flex flex-wrap items-end gap-3">
            <input
              className="h-10 rounded-md border border-input bg-background px-3 text-sm"
              value={keyword}
              onChange={(event) => setKeyword(event.target.value)}
              placeholder={t('welfareApproval.keywordPlaceholder')}
            />
            <Button type="button" variant="outline" onClick={() => void loadData()}>
              {t('common.refresh')}
            </Button>
          </div>
        </div>
      </div>

      {error ? <p className="text-sm text-destructive">{error}</p> : null}
      {loading ? <p className="text-sm text-muted-foreground">{t('welfareApproval.loading')}</p> : null}

      <div className="rounded-lg border bg-card p-5">
        <div className="flex gap-2">
          <Button type="button" variant={activeTab === 'pending' ? 'default' : 'outline'} onClick={() => setActiveTab('pending')}>
            {t('welfareApproval.pendingTab')}
          </Button>
          <Button type="button" variant={activeTab === 'history' ? 'default' : 'outline'} onClick={() => setActiveTab('history')}>
            {t('welfareApproval.historyTab')}
          </Button>
        </div>

        <div className="mt-4 overflow-x-auto">
          <table className="min-w-full text-sm">
            <thead>
              <tr className="border-b bg-muted/40 text-left">
                <th className="px-3 py-2">{t('welfareApproval.columns.character')}</th>
                <th className="px-3 py-2">{t('welfareApproval.columns.welfare')}</th>
                <th className="px-3 py-2">{t('welfareApproval.columns.status')}</th>
                <th className="px-3 py-2">{t('welfareApproval.columns.reviewer')}</th>
                <th className="px-3 py-2">{t('welfareApproval.columns.appliedAt')}</th>
                <th className="px-3 py-2">{t('welfareApproval.columns.actions')}</th>
              </tr>
            </thead>
            <tbody>
              {(activeTab === 'pending' ? pending : history).map((item) => (
                <tr key={item.id} className="border-b">
                  <td className="px-3 py-2">{item.character_name}</td>
                  <td className="px-3 py-2">
                    <div className="font-medium">{item.welfare_name}</div>
                    <div className="text-xs text-muted-foreground">{item.welfare_description}</div>
                  </td>
                  <td className="px-3 py-2">{item.status}</td>
                  <td className="px-3 py-2">{item.reviewer_name || '-'}</td>
                  <td className="px-3 py-2">{formatTime(item.created_at)}</td>
                  <td className="px-3 py-2">
                    {activeTab === 'pending' ? (
                      <div className="flex flex-wrap gap-2">
                        <Button
                          type="button"
                          size="sm"
                          onClick={() => void review(item.id, 'deliver')}
                          disabled={actionId === item.id}
                        >
                          {t('welfareApproval.deliverBtn')}
                        </Button>
                        <Button
                          type="button"
                          size="sm"
                          variant="outline"
                          onClick={() => void review(item.id, 'reject')}
                          disabled={actionId === item.id}
                        >
                          {t('welfareApproval.rejectBtn')}
                        </Button>
                      </div>
                    ) : (
                      <Button
                        type="button"
                        size="sm"
                        variant="outline"
                        onClick={() => void removeHistory(item.id)}
                        disabled={actionId === item.id}
                      >
                        {t('common.delete')}
                      </Button>
                    )}
                  </td>
                </tr>
              ))}
              {!loading && (activeTab === 'pending' ? pending : history).length === 0 ? (
                <tr>
                  <td className="px-3 py-6 text-center text-muted-foreground" colSpan={6}>
                    {t('welfareApproval.empty')}
                  </td>
                </tr>
              ) : null}
            </tbody>
          </table>
        </div>
      </div>
    </section>
  )
}
