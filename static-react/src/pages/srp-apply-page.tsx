import { useEffect, useMemo, useState } from 'react'
import { fetchKillmailDetail, fetchMyApplications, fetchMyKillmails, submitApplication } from '@/api/srp'
import { Button } from '@/components/ui/button'
import { useI18n } from '@/i18n'
import type { Application, FleetKillmailItem, KillmailDetailResponse } from '@/types/api/srp'

function getErrorMessage(error: unknown, fallback: string) {
  return error instanceof Error && error.message ? error.message : fallback
}

function formatTime(value: string) {
  const parsed = new Date(value)
  return Number.isNaN(parsed.getTime()) ? value : parsed.toLocaleString()
}

export function SrpApplyPage() {
  const { t } = useI18n()
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [killmails, setKillmails] = useState<FleetKillmailItem[]>([])
  const [applications, setApplications] = useState<Application[]>([])
  const [selectedKillmailId, setSelectedKillmailId] = useState<number>(0)
  const [note, setNote] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [detail, setDetail] = useState<KillmailDetailResponse | null>(null)
  const [detailLoading, setDetailLoading] = useState(false)

  const loadData = async () => {
    setLoading(true)
    setError(null)
    try {
      const [killmailList, applicationList] = await Promise.all([
        fetchMyKillmails({ limit: 50, exclude_submitted: true }),
        fetchMyApplications({ current: 1, size: 20 }),
      ])
      setKillmails(killmailList ?? [])
      setApplications(applicationList.list ?? [])
      setSelectedKillmailId((current) => current || killmailList?.[0]?.killmail_id || 0)
    } catch (caughtError) {
      setError(getErrorMessage(caughtError, t('srpApply.loadFailed')))
      setKillmails([])
      setApplications([])
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void loadData()
  }, [t])

  const selectedKillmail = useMemo(
    () => killmails.find((killmail) => killmail.killmail_id === selectedKillmailId) ?? null,
    [killmails, selectedKillmailId]
  )

  useEffect(() => {
    if (!selectedKillmailId) {
      setDetail(null)
      return
    }

    let cancelled = false
    const loadDetail = async () => {
      setDetailLoading(true)
      try {
        const response = await fetchKillmailDetail({ killmail_id: selectedKillmailId })
        if (!cancelled) setDetail(response)
      } catch (caughtError) {
        if (!cancelled) setError(getErrorMessage(caughtError, t('srpApply.detailLoadFailed')))
      } finally {
        if (!cancelled) setDetailLoading(false)
      }
    }

    void loadDetail()
    return () => {
      cancelled = true
    }
  }, [selectedKillmailId, t])

  const submit = async () => {
    if (!selectedKillmail) return
    setSubmitting(true)
    try {
      await submitApplication({
        character_id: selectedKillmail.character_id,
        killmail_id: selectedKillmail.killmail_id,
        note: note.trim() || undefined,
      })
      setNote('')
      await loadData()
    } catch (caughtError) {
      setError(getErrorMessage(caughtError, t('srpApply.submitFailed')))
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <section className="space-y-4">
      <div className="rounded-lg border bg-card p-5">
        <h1 className="text-xl font-semibold">{t('srpApply.title')}</h1>
        <p className="mt-1 text-sm text-muted-foreground">{t('srpApply.subtitle')}</p>
      </div>

      {error ? <p className="text-sm text-destructive">{error}</p> : null}
      {loading ? <p className="text-sm text-muted-foreground">{t('srpApply.loading')}</p> : null}

      <div className="grid gap-4 xl:grid-cols-2">
        <div className="rounded-lg border bg-card p-5">
          <h2 className="text-lg font-semibold">{t('srpApply.formTitle')}</h2>
          <p className="mt-1 text-sm text-muted-foreground">{t('srpApply.formHint')}</p>

          <div className="mt-4 space-y-4">
            <label className="space-y-2 block">
              <span className="text-sm text-muted-foreground">{t('srpApply.killmail')}</span>
              <select
                className="h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
                value={selectedKillmailId}
                onChange={(event) => setSelectedKillmailId(Number(event.target.value))}
              >
                {killmails.map((killmail) => (
                  <option key={killmail.killmail_id} value={killmail.killmail_id}>
                    {killmail.killmail_id} - {killmail.victim_name} - {formatTime(killmail.killmail_time)}
                  </option>
                ))}
              </select>
            </label>

            <label className="space-y-2 block">
              <span className="text-sm text-muted-foreground">{t('srpApply.note')}</span>
              <textarea
                className="min-h-24 w-full rounded-md border border-input bg-background px-3 py-2 text-sm outline-none"
                value={note}
                onChange={(event) => setNote(event.target.value)}
                placeholder={t('srpApply.notePlaceholder')}
              />
            </label>

            <Button type="button" onClick={() => void submit()} disabled={submitting || !selectedKillmail}>
              {submitting ? t('srpApply.submitting') : t('srpApply.submitBtn')}
            </Button>
          </div>
        </div>

        <div className="rounded-lg border bg-card p-5">
          <div className="flex items-center justify-between gap-3">
            <h2 className="text-lg font-semibold">{t('srpApply.previewTitle')}</h2>
            <Button type="button" variant="outline" onClick={() => void loadData()}>
              {t('common.refresh')}
            </Button>
          </div>

          <div className="mt-4 space-y-3 text-sm">
            {detailLoading ? <p className="text-muted-foreground">{t('srpApply.detailLoading')}</p> : null}
            {detail ? (
              <div className="rounded-lg border bg-background p-4 space-y-2">
                <p>
                  <span className="font-medium">{t('srpApply.killmail')}:</span> {detail.killmail_id}
                </p>
                <p>
                  <span className="font-medium">{t('srpApply.ship')}:</span> {detail.ship_name}
                </p>
                <p>
                  <span className="font-medium">{t('srpApply.system')}:</span> {detail.system_name}
                </p>
                <p>
                  <span className="font-medium">{t('srpApply.character')}:</span> {detail.character_name}
                </p>
                <p>
                  <span className="font-medium">{t('srpApply.janiceAmount')}:</span>{' '}
                  {detail.janice_amount ?? '-'}
                </p>
              </div>
            ) : (
              <p className="text-muted-foreground">{t('srpApply.detailEmpty')}</p>
            )}
          </div>
        </div>
      </div>

      <div className="rounded-lg border bg-card p-5">
        <h2 className="text-lg font-semibold">{t('srpApply.myApplications')}</h2>
        <div className="mt-4 overflow-x-auto">
          <table className="min-w-full text-sm">
            <thead>
              <tr className="border-b bg-muted/40 text-left">
                <th className="px-3 py-2">{t('srpApply.columns.killmail')}</th>
                <th className="px-3 py-2">{t('srpApply.columns.character')}</th>
                <th className="px-3 py-2">{t('srpApply.columns.ship')}</th>
                <th className="px-3 py-2">{t('srpApply.columns.reviewStatus')}</th>
                <th className="px-3 py-2">{t('srpApply.columns.payoutStatus')}</th>
              </tr>
            </thead>
            <tbody>
              {applications.map((application) => (
                <tr key={application.id} className="border-b">
                  <td className="px-3 py-2">{application.killmail_id}</td>
                  <td className="px-3 py-2">{application.character_name}</td>
                  <td className="px-3 py-2">{application.ship_name}</td>
                  <td className="px-3 py-2">{application.review_status}</td>
                  <td className="px-3 py-2">{application.payout_status}</td>
                </tr>
              ))}
              {!loading && applications.length === 0 ? (
                <tr>
                  <td className="px-3 py-6 text-center text-muted-foreground" colSpan={5}>
                    {t('srpApply.empty')}
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
