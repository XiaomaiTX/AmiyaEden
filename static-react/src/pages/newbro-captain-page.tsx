import { useCallback, useEffect, useState } from 'react'
import {
  fetchCaptainAttributions,
  fetchCaptainEligiblePlayers,
  fetchCaptainEnrollPlayer,
  fetchCaptainEndAffiliation,
  fetchCaptainOverview,
  fetchCaptainPlayers,
  fetchCaptainRewardSettlements,
} from '@/api/newbro'
import { Button } from '@/components/ui/button'
import { useI18n } from '@/i18n'
import { formatDateTime, formatNumber, getErrorMessage } from '@/pages/newbro-page-utils'
import type {
  CaptainAttributionItem,
  CaptainEligiblePlayerListItem,
  CaptainOverview,
  CaptainPlayerListItem,
  CaptainPlayerStatus,
  CaptainRewardSettlementItem,
} from '@/types/api/newbro'

export function NewbroCaptainPage() {
  const { t } = useI18n()
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [overview, setOverview] = useState<CaptainOverview | null>(null)
  const [players, setPlayers] = useState<CaptainPlayerListItem[]>([])
  const [eligiblePlayers, setEligiblePlayers] = useState<CaptainEligiblePlayerListItem[]>([])
  const [attributions, setAttributions] = useState<CaptainAttributionItem[]>([])
  const [rewards, setRewards] = useState<CaptainRewardSettlementItem[]>([])
  const [playerStatus, setPlayerStatus] = useState<CaptainPlayerStatus>('all')
  const [enrollingPlayerId, setEnrollingPlayerId] = useState<number | null>(null)
  const [endingPlayerId, setEndingPlayerId] = useState<number | null>(null)
  const [refreshSeed, setRefreshSeed] = useState(0)

  const loadData = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const [overviewData, playerList, eligibleList, attributionData, rewardData] = await Promise.all([
        fetchCaptainOverview(),
        fetchCaptainPlayers({ current: 1, size: 20, status: playerStatus }),
        fetchCaptainEligiblePlayers({ current: 1, size: 20 }),
        fetchCaptainAttributions({ current: 1, size: 20 }),
        fetchCaptainRewardSettlements({ current: 1, size: 20 }),
      ])

      setOverview(overviewData)
      setPlayers(playerList.list ?? [])
      setEligiblePlayers(eligibleList.list ?? [])
      setAttributions(attributionData.list ?? [])
      setRewards(rewardData.list ?? [])
    } catch (caughtError) {
      setOverview(null)
      setPlayers([])
      setEligiblePlayers([])
      setAttributions([])
      setRewards([])
      setError(getErrorMessage(caughtError, t('newbroCaptain.messages.loadFailed')))
    } finally {
      setLoading(false)
    }
  }, [playerStatus, t])

  useEffect(() => {
    void loadData()
  }, [loadData, refreshSeed])

  const handleEnroll = async (playerUserId: number) => {
    setEnrollingPlayerId(playerUserId)
    try {
      await fetchCaptainEnrollPlayer({ player_user_id: playerUserId })
      setRefreshSeed((current) => current + 1)
    } catch (caughtError) {
      setError(getErrorMessage(caughtError, t('newbroCaptain.messages.enrollFailed')))
    } finally {
      setEnrollingPlayerId(null)
    }
  }

  const handleEndAffiliation = async (playerUserId: number) => {
    setEndingPlayerId(playerUserId)
    try {
      await fetchCaptainEndAffiliation({ player_user_id: playerUserId })
      setRefreshSeed((current) => current + 1)
    } catch (caughtError) {
      setError(getErrorMessage(caughtError, t('newbroCaptain.messages.endFailed')))
    } finally {
      setEndingPlayerId(null)
    }
  }

  return (
    <section className="space-y-4">
      <div className="rounded-lg border bg-card p-5">
        <div className="flex flex-col gap-3 lg:flex-row lg:items-end lg:justify-between">
          <div>
            <h1 className="text-xl font-semibold">{t('newbroCaptain.title')}</h1>
            <p className="mt-1 text-sm text-muted-foreground">{t('newbroCaptain.subtitle')}</p>
          </div>
          <Button type="button" variant="outline" onClick={() => setRefreshSeed((v) => v + 1)}>
            {t('common.refresh')}
          </Button>
        </div>
      </div>

      {error ? <p className="text-sm text-destructive">{error}</p> : null}
      {loading ? <p className="text-sm text-muted-foreground">{t('newbroCaptain.loading')}</p> : null}

      <div className="grid gap-4 xl:grid-cols-4">
        <div className="rounded-lg border bg-card p-4">
          <div className="text-sm text-muted-foreground">{t('newbroCaptain.metrics.activePlayers')}</div>
          <div className="mt-1 text-2xl font-semibold">{formatNumber(overview?.active_player_count)}</div>
        </div>
        <div className="rounded-lg border bg-card p-4">
          <div className="text-sm text-muted-foreground">{t('newbroCaptain.metrics.historicalPlayers')}</div>
          <div className="mt-1 text-2xl font-semibold">{formatNumber(overview?.historical_player_count)}</div>
        </div>
        <div className="rounded-lg border bg-card p-4">
          <div className="text-sm text-muted-foreground">{t('newbroCaptain.metrics.bountyTotal')}</div>
          <div className="mt-1 text-2xl font-semibold">{formatNumber(overview?.attributed_bounty_total)}</div>
        </div>
        <div className="rounded-lg border bg-card p-4">
          <div className="text-sm text-muted-foreground">{t('newbroCaptain.metrics.recordCount')}</div>
          <div className="mt-1 text-2xl font-semibold">{formatNumber(overview?.attribution_record_count)}</div>
        </div>
      </div>

      <div className="grid gap-4 xl:grid-cols-2">
        <div className="rounded-lg border bg-card p-5">
          <div className="flex items-center justify-between gap-3">
            <h2 className="text-base font-semibold">{t('newbroCaptain.playersSection')}</h2>
            <select
              className="h-9 rounded-md border border-input bg-background px-2 text-sm"
              value={playerStatus}
              onChange={(event) => {
                setPlayerStatus(event.target.value as CaptainPlayerStatus)
                setRefreshSeed((current) => current + 1)
              }}
            >
              <option value="all">{t('newbroCaptain.playerStatus.all')}</option>
              <option value="active">{t('newbroCaptain.playerStatus.active')}</option>
              <option value="historical">{t('newbroCaptain.playerStatus.historical')}</option>
            </select>
          </div>

          <div className="mt-4 space-y-3">
            {players.map((player) => (
              <div key={player.player_user_id} className="rounded-md border p-4">
                <div className="flex flex-col gap-2 md:flex-row md:items-center md:justify-between">
                  <div>
                    <div className="font-medium">{player.player_character_name}</div>
                    <div className="text-sm text-muted-foreground">{player.player_nickname || '-'}</div>
                    <div className="text-sm text-muted-foreground">
                      {t('newbroCaptain.startedAt')}: {formatDateTime(player.started_at)}
                    </div>
                    <div className="text-sm text-muted-foreground">
                      {t('newbroCaptain.endedAt')}: {formatDateTime(player.ended_at)}
                    </div>
                  </div>
                  {player.ended_at ? null : (
                    <Button
                      type="button"
                      variant="destructive"
                      disabled={endingPlayerId === player.player_user_id}
                      onClick={() => void handleEndAffiliation(player.player_user_id)}
                    >
                      {endingPlayerId === player.player_user_id
                        ? t('newbroCaptain.ending')
                        : t('newbroCaptain.endAffiliation')}
                    </Button>
                  )}
                </div>
              </div>
            ))}
            {!loading && players.length === 0 ? (
              <p className="rounded-md border border-dashed p-6 text-center text-sm text-muted-foreground">
                {t('newbroCaptain.noPlayers')}
              </p>
            ) : null}
          </div>
        </div>

        <div className="space-y-4">
          <div className="rounded-lg border bg-card p-5">
            <div className="flex items-center justify-between">
              <h2 className="text-base font-semibold">{t('newbroCaptain.eligibleSection')}</h2>
              <span className="text-sm text-muted-foreground">{formatNumber(eligiblePlayers.length)}</span>
            </div>
            <div className="mt-4 space-y-3">
              {eligiblePlayers.map((player) => (
                <div key={player.player_user_id} className="rounded-md border p-4">
                  <div className="flex flex-col gap-2 md:flex-row md:items-center md:justify-between">
                    <div>
                      <div className="font-medium">{player.player_character_name}</div>
                      <div className="text-sm text-muted-foreground">{player.player_nickname || '-'}</div>
                      <div className="text-sm text-muted-foreground">
                        {player.current_affiliation
                          ? t('newbroCaptain.currentCaptain', {
                              captain: player.current_affiliation.captain_character_name,
                            })
                          : t('newbroCaptain.noCurrentCaptain')}
                      </div>
                    </div>
                    <Button
                      type="button"
                      disabled={enrollingPlayerId === player.player_user_id}
                      onClick={() => void handleEnroll(player.player_user_id)}
                    >
                      {enrollingPlayerId === player.player_user_id
                        ? t('newbroCaptain.enrolling')
                        : t('newbroCaptain.enroll')}
                    </Button>
                  </div>
                </div>
              ))}
              {!loading && eligiblePlayers.length === 0 ? (
                <p className="rounded-md border border-dashed p-6 text-center text-sm text-muted-foreground">
                  {t('newbroCaptain.noEligiblePlayers')}
                </p>
              ) : null}
            </div>
          </div>

          <div className="rounded-lg border bg-card p-5">
            <h2 className="text-base font-semibold">{t('newbroCaptain.attributionsSection')}</h2>
            <div className="mt-4 space-y-3">
              {attributions.map((item) => (
                <div key={item.id} className="rounded-md border p-4 text-sm">
                  <div className="flex items-center justify-between gap-3">
                    <div className="font-medium">{item.player_character_name}</div>
                    <div>{formatNumber(item.amount)}</div>
                  </div>
                  <div className="mt-1 text-muted-foreground">
                    {t('newbroCaptain.attributionMeta', {
                      captain: item.captain_character_name,
                      system: item.system_id,
                    })}
                  </div>
                </div>
              ))}
              {!loading && attributions.length === 0 ? (
                <p className="rounded-md border border-dashed p-6 text-center text-sm text-muted-foreground">
                  {t('newbroCaptain.noAttributions')}
                </p>
              ) : null}
            </div>
          </div>

          <div className="rounded-lg border bg-card p-5">
            <h2 className="text-base font-semibold">{t('newbroCaptain.rewardSection')}</h2>
            <div className="mt-4 space-y-3">
              {rewards.map((item) => (
                <div key={item.id} className="rounded-md border p-4 text-sm">
                  <div className="flex items-center justify-between gap-3">
                    <div className="font-medium">{item.captain_character_name}</div>
                    <div>{formatNumber(item.credited_value)}</div>
                  </div>
                  <div className="mt-1 text-muted-foreground">
                    {t('newbroCaptain.rewardMeta', {
                      count: item.attribution_count,
                      rate: item.bonus_rate,
                    })}
                  </div>
                </div>
              ))}
              {!loading && rewards.length === 0 ? (
                <p className="rounded-md border border-dashed p-6 text-center text-sm text-muted-foreground">
                  {t('newbroCaptain.noRewards')}
                </p>
              ) : null}
            </div>
          </div>
        </div>
      </div>
    </section>
  )
}
