import { useCallback, useEffect, useState } from 'react'
import {
  fetchAdminMentorRelationships,
  fetchAdminMentorRewardDistributions,
  fetchMentorSettings,
  fetchMentorRewardStages,
  revokeMentorRelationship,
  runMentorRewardProcessing,
  updateMentorSettings,
} from '@/api/mentor'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { useI18n } from '@/i18n'
import { formatDateTime, formatNumber, getErrorMessage } from '@/pages/newbro-page-utils'
import type {
  MentorSettings,
  RelationshipView,
  RewardDistributionView,
  RewardStage,
  UpdateMentorSettingsParams,
} from '@/types/api/mentor'

export function NewbroMentorManagePage() {
  const { t } = useI18n()
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [relationships, setRelationships] = useState<RelationshipView[]>([])
  const [rewards, setRewards] = useState<RewardDistributionView[]>([])
  const [stages, setStages] = useState<RewardStage[]>([])
  const [settings, setSettings] = useState<MentorSettings | null>(null)
  const [revokeId, setRevokeId] = useState<number | null>(null)
  const [savingSettings, setSavingSettings] = useState(false)
  const [processingRewards, setProcessingRewards] = useState(false)
  const [refreshSeed, setRefreshSeed] = useState(0)

  const loadData = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const [relationshipData, rewardData, stageList, settingsData] = await Promise.all([
        fetchAdminMentorRelationships({ current: 1, size: 20 }),
        fetchAdminMentorRewardDistributions({ current: 1, size: 20 }),
        fetchMentorRewardStages(),
        fetchMentorSettings(),
      ])
      setRelationships(relationshipData.list ?? [])
      setRewards(rewardData.list ?? [])
      setStages(stageList)
      setSettings(settingsData)
    } catch (caughtError) {
      setRelationships([])
      setRewards([])
      setStages([])
      setSettings(null)
      setError(getErrorMessage(caughtError, t('newbroMentorManage.messages.loadFailed')))
    } finally {
      setLoading(false)
    }
  }, [t])

  useEffect(() => {
    void loadData()
  }, [loadData, refreshSeed])

  const handleRevoke = async (relationshipId: number) => {
    setRevokeId(relationshipId)
    try {
      await revokeMentorRelationship({ relationship_id: relationshipId })
      setRefreshSeed((current) => current + 1)
    } catch (caughtError) {
      setError(getErrorMessage(caughtError, t('newbroMentorManage.messages.revokeFailed')))
    } finally {
      setRevokeId(null)
    }
  }

  const handleSaveSettings = async () => {
    if (!settings) {
      return
    }

    setSavingSettings(true)
    try {
      const payload: UpdateMentorSettingsParams = {
        max_character_sp: settings.max_character_sp,
        max_account_age_days: settings.max_account_age_days,
      }
      setSettings(await updateMentorSettings(payload))
    } catch (caughtError) {
      setError(getErrorMessage(caughtError, t('newbroMentorManage.messages.saveFailed')))
    } finally {
      setSavingSettings(false)
    }
  }

  const handleProcessRewards = async () => {
    setProcessingRewards(true)
    try {
      await runMentorRewardProcessing()
      setRefreshSeed((current) => current + 1)
    } catch (caughtError) {
      setError(getErrorMessage(caughtError, t('newbroMentorManage.messages.processFailed')))
    } finally {
      setProcessingRewards(false)
    }
  }

  return (
    <section className="space-y-4">
      <div className="rounded-lg border bg-card p-5">
        <div className="flex flex-col gap-3 lg:flex-row lg:items-end lg:justify-between">
          <div>
            <h1 className="text-xl font-semibold">{t('newbroMentorManage.title')}</h1>
            <p className="mt-1 text-sm text-muted-foreground">{t('newbroMentorManage.subtitle')}</p>
          </div>
          <Button type="button" variant="outline" onClick={() => setRefreshSeed((v) => v + 1)}>
            {t('common.refresh')}
          </Button>
        </div>
      </div>

      {error ? <p className="text-sm text-destructive">{error}</p> : null}
      {loading ? <p className="text-sm text-muted-foreground">{t('newbroMentorManage.loading')}</p> : null}

      <div className="grid gap-4 xl:grid-cols-[1fr_0.9fr]">
        <div className="space-y-4">
          <div className="rounded-lg border bg-card p-5">
            <div className="flex items-center justify-between">
              <h2 className="text-base font-semibold">{t('newbroMentorManage.relationshipsSection')}</h2>
              <span className="text-sm text-muted-foreground">{formatNumber(relationships.length)}</span>
            </div>
            <div className="mt-4 space-y-3">
              {relationships.map((relationship) => (
                <div key={relationship.id} className="rounded-md border p-4">
                  <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
                    <div>
                      <div className="font-medium">{relationship.mentee_character_name}</div>
                      <div className="text-sm text-muted-foreground">{relationship.mentor_character_name}</div>
                      <div className="text-sm text-muted-foreground">
                        {t('newbroMentorManage.appliedAt')}: {formatDateTime(relationship.applied_at)}
                      </div>
                    </div>
                    <Button
                      type="button"
                      variant="destructive"
                      disabled={revokeId === relationship.id}
                      onClick={() => void handleRevoke(relationship.id)}
                    >
                      {revokeId === relationship.id
                        ? t('newbroMentorManage.revoking')
                        : t('newbroMentorManage.revoke')}
                    </Button>
                  </div>
                </div>
              ))}
              {!loading && relationships.length === 0 ? (
                <p className="rounded-md border border-dashed p-6 text-center text-sm text-muted-foreground">
                  {t('newbroMentorManage.noRelationships')}
                </p>
              ) : null}
            </div>
          </div>

          <div className="rounded-lg border bg-card p-5">
            <div className="flex items-center justify-between">
              <h2 className="text-base font-semibold">{t('newbroMentorManage.rewardsSection')}</h2>
              <span className="text-sm text-muted-foreground">{formatNumber(rewards.length)}</span>
            </div>
            <div className="mt-4 space-y-3">
              {rewards.map((item) => (
                <div key={item.id} className="rounded-md border p-4 text-sm">
                  <div className="flex items-center justify-between gap-3">
                    <div className="font-medium">{item.mentor_character_name}</div>
                    <div>{formatNumber(item.reward_amount)}</div>
                  </div>
                  <div className="mt-1 text-muted-foreground">
                    {t('newbroMentorManage.rewardMeta', { distributedAt: formatDateTime(item.distributed_at) })}
                  </div>
                </div>
              ))}
              {!loading && rewards.length === 0 ? (
                <p className="rounded-md border border-dashed p-6 text-center text-sm text-muted-foreground">
                  {t('newbroMentorManage.noRewards')}
                </p>
              ) : null}
            </div>
          </div>
        </div>

        <div className="space-y-4">
          <div className="rounded-lg border bg-card p-5">
            <h2 className="text-base font-semibold">{t('newbroMentorManage.settingsSection')}</h2>
            {settings ? (
              <div className="mt-4 space-y-4">
                <label className="block space-y-1">
                  <span className="text-sm text-muted-foreground">
                    {t('newbroMentorManage.fields.maxCharacterSp')}
                  </span>
                  <Input
                    type="number"
                    value={settings.max_character_sp}
                    onChange={(event) =>
                      setSettings({ ...settings, max_character_sp: Number(event.target.value) })
                    }
                  />
                </label>
                <label className="block space-y-1">
                  <span className="text-sm text-muted-foreground">
                    {t('newbroMentorManage.fields.maxAccountAgeDays')}
                  </span>
                  <Input
                    type="number"
                    value={settings.max_account_age_days}
                    onChange={(event) =>
                      setSettings({ ...settings, max_account_age_days: Number(event.target.value) })
                    }
                  />
                </label>
                <Button type="button" onClick={() => void handleSaveSettings()} disabled={savingSettings}>
                  {savingSettings ? t('newbroMentorManage.saving') : t('newbroMentorManage.save')}
                </Button>
              </div>
            ) : (
              <p className="mt-4 text-sm text-muted-foreground">{t('newbroMentorManage.noSettings')}</p>
            )}
          </div>

          <div className="rounded-lg border bg-card p-5">
            <div className="flex items-center justify-between">
              <h2 className="text-base font-semibold">{t('newbroMentorManage.rewardStagesSection')}</h2>
              <span className="text-sm text-muted-foreground">{formatNumber(stages.length)}</span>
            </div>
            <div className="mt-4 space-y-3">
              {stages.map((stage) => (
                <div key={stage.id} className="rounded-md border p-4 text-sm">
                  <div className="flex items-center justify-between gap-3">
                    <div className="font-medium">{stage.name}</div>
                    <div>{formatNumber(stage.reward_amount)}</div>
                  </div>
                  <div className="mt-1 text-muted-foreground">
                    {t('newbroMentorManage.stageMeta', {
                      order: stage.stage_order,
                      threshold: stage.threshold,
                    })}
                  </div>
                </div>
              ))}
              {!loading && stages.length === 0 ? (
                <p className="rounded-md border border-dashed p-6 text-center text-sm text-muted-foreground">
                  {t('newbroMentorManage.noStages')}
                </p>
              ) : null}
            </div>
            <div className="mt-4">
              <Button type="button" variant="outline" onClick={() => void handleProcessRewards()} disabled={processingRewards}>
                {processingRewards ? t('newbroMentorManage.processing') : t('newbroMentorManage.processRewards')}
              </Button>
            </div>
          </div>
        </div>
      </div>
    </section>
  )
}
