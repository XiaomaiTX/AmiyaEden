import { useCallback, useEffect, useState } from 'react'
import {
  fetchAdminAffiliationHistory,
  fetchAdminCaptainDetail,
  fetchAdminCaptainList,
  fetchAdminRewardSettlements,
} from '@/api/newbro'
import { Button } from '@/components/ui/button'
import { useI18n } from '@/i18n'
import { formatDateTime, formatNumber, getErrorMessage } from '@/pages/newbro-page-utils'
import type {
  AdminAffiliationHistoryItem,
  AdminCaptainDetail,
  CaptainOverview,
  RewardDistributionView,
} from '@/types/api/newbro'

export function NewbroManagePage() {
  const { t } = useI18n()
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [captains, setCaptains] = useState<CaptainOverview[]>([])
  const [selectedCaptainId, setSelectedCaptainId] = useState<number | null>(null)
  const [detail, setDetail] = useState<AdminCaptainDetail | null>(null)
  const [history, setHistory] = useState<AdminAffiliationHistoryItem[]>([])
  const [rewards, setRewards] = useState<RewardDistributionView[]>([])
  const [refreshSeed, setRefreshSeed] = useState(0)

  const loadData = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const [captainList, historyData, rewardData] = await Promise.all([
        fetchAdminCaptainList({ current: 1, size: 20 }),
        fetchAdminAffiliationHistory({ current: 1, size: 20 }),
        fetchAdminRewardSettlements({ current: 1, size: 20 }),
      ])
      const captainRows = captainList.list ?? []
      setCaptains(captainRows)
      setHistory(historyData.list ?? [])
      setRewards(rewardData.list ?? [])
      const nextSelected = selectedCaptainId ?? captainRows[0]?.captain_user_id ?? null
      setSelectedCaptainId(nextSelected)
      setDetail(nextSelected ? await fetchAdminCaptainDetail(nextSelected) : null)
    } catch (caughtError) {
      setCaptains([])
      setDetail(null)
      setHistory([])
      setRewards([])
      setError(getErrorMessage(caughtError, t('newbroManage.messages.loadFailed')))
    } finally {
      setLoading(false)
    }
  }, [selectedCaptainId, t])

  useEffect(() => {
    void loadData()
  }, [loadData, refreshSeed])

  const handleSelectCaptain = async (captainUserId: number) => {
    setSelectedCaptainId(captainUserId)
    try {
      setDetail(await fetchAdminCaptainDetail(captainUserId))
    } catch (caughtError) {
      setError(getErrorMessage(caughtError, t('newbroManage.messages.detailFailed')))
    }
  }

  return (
    <section className="space-y-4">
      <div className="rounded-lg border bg-card p-5">
        <div className="flex flex-col gap-3 lg:flex-row lg:items-end lg:justify-between">
          <div>
            <h1 className="text-xl font-semibold">{t('newbroManage.title')}</h1>
            <p className="mt-1 text-sm text-muted-foreground">{t('newbroManage.subtitle')}</p>
          </div>
          <Button type="button" variant="outline" onClick={() => setRefreshSeed((v) => v + 1)}>
            {t('common.refresh')}
          </Button>
        </div>
      </div>

      {error ? <p className="text-sm text-destructive">{error}</p> : null}
      {loading ? <p className="text-sm text-muted-foreground">{t('newbroManage.loading')}</p> : null}

      <div className="grid gap-4 xl:grid-cols-[0.8fr_1.2fr]">
        <div className="rounded-lg border bg-card p-5">
          <div className="flex items-center justify-between">
            <h2 className="text-base font-semibold">{t('newbroManage.captainList')}</h2>
            <span className="text-sm text-muted-foreground">{formatNumber(captains.length)}</span>
          </div>
          <div className="mt-4 space-y-3">
            {captains.map((captain) => (
              <button
                key={captain.captain_user_id}
                type="button"
                className={`w-full rounded-md border p-4 text-left ${
                  selectedCaptainId === captain.captain_user_id ? 'border-primary' : ''
                }`}
                onClick={() => void handleSelectCaptain(captain.captain_user_id)}
              >
                <div className="font-medium">{captain.captain_character_name}</div>
                <div className="text-sm text-muted-foreground">{captain.captain_nickname || '-'}</div>
                <div className="text-sm text-muted-foreground">
                  {t('newbroManage.activePlayers')}: {captain.active_player_count}
                </div>
              </button>
            ))}
            {!loading && captains.length === 0 ? (
              <p className="rounded-md border border-dashed p-6 text-center text-sm text-muted-foreground">
                {t('newbroManage.noCaptains')}
              </p>
            ) : null}
          </div>
        </div>

        <div className="space-y-4">
          <div className="rounded-lg border bg-card p-5">
            <h2 className="text-base font-semibold">{t('newbroManage.detailTitle')}</h2>
            {detail?.overview ? (
              <div className="mt-4 grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
                <div className="rounded-md border p-4">
                  <div className="text-sm text-muted-foreground">{t('newbroManage.activePlayers')}</div>
                  <div className="mt-1 text-xl font-semibold">{formatNumber(detail.overview.active_player_count)}</div>
                </div>
                <div className="rounded-md border p-4">
                  <div className="text-sm text-muted-foreground">{t('newbroManage.historicalPlayers')}</div>
                  <div className="mt-1 text-xl font-semibold">{formatNumber(detail.overview.historical_player_count)}</div>
                </div>
                <div className="rounded-md border p-4">
                  <div className="text-sm text-muted-foreground">{t('newbroManage.bountyTotal')}</div>
                  <div className="mt-1 text-xl font-semibold">{formatNumber(detail.overview.attributed_bounty_total)}</div>
                </div>
                <div className="rounded-md border p-4">
                  <div className="text-sm text-muted-foreground">{t('newbroManage.recordCount')}</div>
                  <div className="mt-1 text-xl font-semibold">{formatNumber(detail.overview.attribution_record_count)}</div>
                </div>
              </div>
            ) : (
              <p className="mt-4 text-sm text-muted-foreground">{t('newbroManage.noDetail')}</p>
            )}
          </div>

          <div className="rounded-lg border bg-card p-5">
            <h2 className="text-base font-semibold">{t('newbroManage.historySection')}</h2>
            <div className="mt-4 overflow-x-auto">
              <table className="min-w-full text-sm">
                <thead>
                  <tr className="border-b bg-muted/40 text-left">
                    <th className="px-3 py-2">{t('newbroManage.columns.player')}</th>
                    <th className="px-3 py-2">{t('newbroManage.columns.captain')}</th>
                    <th className="px-3 py-2">{t('newbroManage.columns.startedAt')}</th>
                  </tr>
                </thead>
                <tbody>
                  {history.map((item) => (
                    <tr key={item.affiliation_id} className="border-b">
                      <td className="px-3 py-2">{item.player_character_name}</td>
                      <td className="px-3 py-2">{item.captain_character_name}</td>
                      <td className="px-3 py-2">{formatDateTime(item.started_at)}</td>
                    </tr>
                  ))}
                  {!loading && history.length === 0 ? (
                    <tr>
                      <td className="px-3 py-6 text-center text-muted-foreground" colSpan={3}>
                        {t('newbroManage.noHistory')}
                      </td>
                    </tr>
                  ) : null}
                </tbody>
              </table>
            </div>
          </div>

          <div className="rounded-lg border bg-card p-5">
            <h2 className="text-base font-semibold">{t('newbroManage.rewardsSection')}</h2>
            <div className="mt-4 space-y-3">
              {rewards.map((item) => (
                <div key={item.id} className="rounded-md border p-4 text-sm">
                  <div className="flex items-center justify-between">
                    <div className="font-medium">{item.mentor_character_name}</div>
                    <div>{formatNumber(item.reward_amount)}</div>
                  </div>
                  <div className="mt-1 text-muted-foreground">{formatDateTime(item.distributed_at)}</div>
                </div>
              ))}
              {!loading && rewards.length === 0 ? (
                <p className="rounded-md border border-dashed p-6 text-center text-sm text-muted-foreground">
                  {t('newbroManage.noRewards')}
                </p>
              ) : null}
            </div>
          </div>
        </div>
      </div>
    </section>
  )
}
