import { Input } from '@/components/ui/input'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '@/components/ui/table'
import { useCallback, useEffect, useState } from 'react'
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

  const loadPending = useCallback(async () => {
    const response = await adminListApplications({ current: 1, size: 200, status: 'requested' })
    setPending(response.list ?? [])
  }, [])

  const loadHistory = useCallback(async () => {
    const response = await adminListApplications({
      current: 1,
      size: 200,
      status: 'delivered,rejected',
      keyword: keyword.trim() || undefined,
    })
    setHistory(response.list ?? [])
  }, [keyword])

  const loadData = useCallback(async () => {
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
  }, [loadHistory, loadPending, t])

  useEffect(() => {
    const timer = window.setTimeout(() => {
      void loadData()
    }, 0)
    return () => window.clearTimeout(timer)
  }, [loadData, keyword])

  const review = async (id: number, action: 'deliver' | 'reject') => {
    setActionId(id)
    try {
      const result = await adminReviewApplication({ id, action })
      if (result?.outcome === 'auto_rejected') {
        const reason = result.eligibility_reason
          ? t(`welfareApproval.eligibilityReasons.${result.eligibility_reason}`)
          : t('welfareApproval.eligibilityReasons.unknown')
        setError(t('welfareApproval.autoRejected', { reason }))
      }
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
            <Input
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
          <Table className="min-w-full text-sm">
            <TableHeader>
              <TableRow className="border-b bg-muted/40 text-left">
                <TableHead className="px-3 py-2">{t('welfareApproval.columns.character')}</TableHead>
                <TableHead className="px-3 py-2">{t('welfareApproval.columns.welfare')}</TableHead>
                <TableHead className="px-3 py-2">{t('welfareApproval.columns.status')}</TableHead>
                <TableHead className="px-3 py-2">{t('welfareApproval.columns.reviewer')}</TableHead>
                <TableHead className="px-3 py-2">{t('welfareApproval.columns.appliedAt')}</TableHead>
                <TableHead className="px-3 py-2">{t('welfareApproval.columns.actions')}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {(activeTab === 'pending' ? pending : history).map((item) => (
                <TableRow key={item.id} className="border-b">
                  <TableCell className="px-3 py-2">{item.character_name}</TableCell>
                  <TableCell className="px-3 py-2">
                    <div className="font-medium">{item.welfare_name}</div>
                    <div className="text-xs text-muted-foreground">{item.welfare_description}</div>
                  </TableCell>
                  <TableCell className="px-3 py-2">{item.status}</TableCell>
                  <TableCell className="px-3 py-2">{item.reviewer_name || '-'}</TableCell>
                  <TableCell className="px-3 py-2">{formatTime(item.created_at)}</TableCell>
                  <TableCell className="px-3 py-2">
                    {activeTab === 'pending' ? (
                      <div className="flex flex-wrap gap-2">
                        <Button
                          type="button"
                          size="sm"
                          onClick={() => void review(item.id, 'deliver')}
                          isDisabled={actionId === item.id}
                        >
                          {t('welfareApproval.deliverBtn')}
                        </Button>
                        <Button
                          type="button"
                          size="sm"
                          variant="outline"
                          onClick={() => void review(item.id, 'reject')}
                          isDisabled={actionId === item.id}
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
                        isDisabled={actionId === item.id}
                      >
                        {t('common.delete')}
                      </Button>
                    )}
                  </TableCell>
                </TableRow>
              ))}
              {!loading && (activeTab === 'pending' ? pending : history).length === 0 ? (
                <TableRow>
                  <TableCell className="px-3 py-6 text-center text-muted-foreground" colSpan={6}>
                    {t('welfareApproval.empty')}
                  </TableCell>
                </TableRow>
              ) : null}
            </TableBody>
          </Table>
        </div>
      </div>
    </section>
  )
}
