import { useCallback, useEffect, useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  fetchEndAffiliation,
  fetchMyAffiliationHistory,
  fetchMyNewbroAffiliation,
  fetchNewbroCaptains,
  fetchSelectCaptain,
} from '@/api/newbro'
import { Button } from '@/components/ui/button'
import { buildEveCharacterPortraitUrl } from '@/lib/eve-image'
import { useI18n } from '@/i18n'
import { formatDateTime, formatNumber, getErrorMessage } from '@/pages/newbro-page-utils'
import type { CaptainCandidate, MyAffiliationResponse, AffiliationSummary } from '@/types/api/newbro'

export function NewbroSelectCaptainPage() {
  const { t } = useI18n()
  const navigate = useNavigate()
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [state, setState] = useState<MyAffiliationResponse | null>(null)
  const [captains, setCaptains] = useState<CaptainCandidate[]>([])
  const [history, setHistory] = useState<AffiliationSummary[]>([])
  const [historyTotal, setHistoryTotal] = useState(0)
  const [submittingCaptainId, setSubmittingCaptainId] = useState<number | null>(null)
  const [endingAffiliation, setEndingAffiliation] = useState(false)
  const [refreshSeed, setRefreshSeed] = useState(0)

  const canChooseCaptain = useMemo(() => state?.is_currently_newbro === true, [state])

  const loadData = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const [affiliation, captainList, historyResponse] = await Promise.all([
        fetchMyNewbroAffiliation(),
        fetchNewbroCaptains(),
        fetchMyAffiliationHistory({ current: 1, size: 10 }),
      ])
      setState(affiliation)
      setCaptains(captainList)
      setHistory(historyResponse.list ?? [])
      setHistoryTotal(historyResponse.total ?? 0)
    } catch (caughtError) {
      setState(null)
      setCaptains([])
      setHistory([])
      setHistoryTotal(0)
      setError(getErrorMessage(caughtError, t('newbroSelectCaptain.messages.loadFailed')))
    } finally {
      setLoading(false)
    }
  }, [t])

  useEffect(() => {
    void loadData()
  }, [loadData, refreshSeed])

  const handleSelectCaptain = async (captainUserId: number) => {
    setSubmittingCaptainId(captainUserId)
    try {
      await fetchSelectCaptain({ captain_user_id: captainUserId })
      setRefreshSeed((current) => current + 1)
    } catch (caughtError) {
      setError(getErrorMessage(caughtError, t('newbroSelectCaptain.messages.chooseFailed')))
    } finally {
      setSubmittingCaptainId(null)
    }
  }

  const handleEndAffiliation = async () => {
    setEndingAffiliation(true)
    try {
      await fetchEndAffiliation()
      setRefreshSeed((current) => current + 1)
    } catch (caughtError) {
      setError(getErrorMessage(caughtError, t('newbroSelectCaptain.messages.endFailed')))
    } finally {
      setEndingAffiliation(false)
    }
  }

  return (
    <section className="space-y-4">
      <div className="rounded-lg border bg-card p-5">
        <div className="flex flex-col gap-3 lg:flex-row lg:items-end lg:justify-between">
          <div>
            <h1 className="text-xl font-semibold">{t('newbroSelectCaptain.title')}</h1>
            <p className="mt-1 text-sm text-muted-foreground">{t('newbroSelectCaptain.subtitle')}</p>
          </div>
          <Button type="button" variant="outline" onClick={() => setRefreshSeed((v) => v + 1)}>
            {t('common.refresh')}
          </Button>
        </div>
      </div>

      {error ? <p className="text-sm text-destructive">{error}</p> : null}
      {loading ? <p className="text-sm text-muted-foreground">{t('newbroSelectCaptain.loading')}</p> : null}

      <div className="grid gap-4 xl:grid-cols-[1.2fr_0.8fr]">
        <div className="space-y-4">
          <div className="rounded-lg border bg-card p-5">
            <div className="flex items-center justify-between gap-3">
              <div>
                <h2 className="text-base font-semibold">{t('newbroSelectCaptain.currentSection')}</h2>
                <p className="mt-1 text-sm text-muted-foreground">
                  {state?.current_affiliation
                    ? t('newbroSelectCaptain.currentAffiliationHint')
                    : t('newbroSelectCaptain.noCurrentAffiliation')}
                </p>
              </div>
              {state?.current_affiliation ? (
                <Button type="button" variant="destructive" onClick={handleEndAffiliation} disabled={endingAffiliation}>
                  {endingAffiliation
                    ? t('newbroSelectCaptain.endingAffiliation')
                    : t('newbroSelectCaptain.endAffiliation')}
                </Button>
              ) : null}
            </div>

            {state?.current_affiliation ? (
              <div className="mt-4 grid gap-3 sm:grid-cols-2">
                <div className="rounded-md border p-4">
                  <div className="text-sm text-muted-foreground">{t('newbroSelectCaptain.columns.captain')}</div>
                  <div className="mt-1 font-medium">
                    {state.current_affiliation.captain_character_name}
                  </div>
                  <div className="mt-1 text-sm text-muted-foreground">
                    {state.current_affiliation.captain_nickname || '-'}
                  </div>
                </div>
                <div className="rounded-md border p-4">
                  <div className="text-sm text-muted-foreground">{t('newbroSelectCaptain.columns.startedAt')}</div>
                  <div className="mt-1 font-medium">{formatDateTime(state.current_affiliation.started_at)}</div>
                </div>
              </div>
            ) : null}
          </div>

          <div className="rounded-lg border bg-card p-5">
            <div className="flex items-center justify-between">
              <h2 className="text-base font-semibold">{t('newbroSelectCaptain.captainList')}</h2>
              <span className="text-sm text-muted-foreground">{formatNumber(captains.length)}</span>
            </div>

            <div className="mt-4 grid gap-3">
              {captains.map((captain) => (
                <div key={captain.captain_user_id} className="rounded-md border p-4">
                  <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
                    <div className="flex items-center gap-3">
                      <img
                        alt={captain.captain_character_name}
                        src={buildEveCharacterPortraitUrl(captain.captain_character_id, 64)}
                        className="h-16 w-16 rounded-full border object-cover"
                      />
                      <div>
                        <div className="font-medium">{captain.captain_character_name}</div>
                        <div className="text-sm text-muted-foreground">{captain.captain_nickname || '-'}</div>
                        <div className="text-sm text-muted-foreground">
                          {t('newbroSelectCaptain.activeNewbros')}: {captain.active_newbro_count}
                        </div>
                      </div>
                    </div>
                    <Button
                      type="button"
                      onClick={() => void handleSelectCaptain(captain.captain_user_id)}
                      disabled={!canChooseCaptain || submittingCaptainId === captain.captain_user_id}
                    >
                      {submittingCaptainId === captain.captain_user_id
                        ? t('newbroSelectCaptain.choosingCaptain')
                        : t('newbroSelectCaptain.chooseCaptain')}
                    </Button>
                  </div>
                </div>
              ))}
              {!loading && captains.length === 0 ? (
                <p className="rounded-md border border-dashed p-6 text-center text-sm text-muted-foreground">
                  {t('newbroSelectCaptain.noCaptains')}
                </p>
              ) : null}
            </div>
          </div>
        </div>

        <div className="rounded-lg border bg-card p-5">
          <div className="flex items-center justify-between">
            <h2 className="text-base font-semibold">{t('newbroSelectCaptain.historySection')}</h2>
            <span className="text-sm text-muted-foreground">{historyTotal}</span>
          </div>

          <div className="mt-4 overflow-x-auto">
            <table className="min-w-full text-sm">
              <thead>
                <tr className="border-b bg-muted/40 text-left">
                  <th className="px-3 py-2">{t('newbroSelectCaptain.columns.captain')}</th>
                  <th className="px-3 py-2">{t('newbroSelectCaptain.columns.startedAt')}</th>
                  <th className="px-3 py-2">{t('newbroSelectCaptain.columns.endedAt')}</th>
                </tr>
              </thead>
              <tbody>
                {history.map((item) => (
                  <tr key={item.affiliation_id} className="border-b">
                    <td className="px-3 py-2">
                      <div className="font-medium">{item.captain_character_name}</div>
                      <div className="text-xs text-muted-foreground">#{item.captain_user_id}</div>
                    </td>
                    <td className="px-3 py-2">{formatDateTime(item.started_at)}</td>
                    <td className="px-3 py-2">{formatDateTime(item.ended_at)}</td>
                  </tr>
                ))}
                {!loading && history.length === 0 ? (
                  <tr>
                    <td className="px-3 py-6 text-center text-muted-foreground" colSpan={3}>
                      {t('newbroSelectCaptain.historyEmpty')}
                    </td>
                  </tr>
                ) : null}
              </tbody>
            </table>
          </div>

          <div className="mt-4">
            <Button type="button" variant="outline" onClick={() => navigate('/newbro/recruit-link')}>
              {t('newbroSelectCaptain.goRecruitLink')}
            </Button>
          </div>
        </div>
      </div>
    </section>
  )
}
