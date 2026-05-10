import { useCallback, useEffect, useState } from 'react'
import {
  acceptMentorApplication,
  fetchMentorApplications,
  fetchMentorMentees,
  fetchMentorRewardStages,
  fetchMyMentorStatus,
  rejectMentorApplication,
} from '@/api/mentor'
import { Button } from '@/components/ui/button'
import { useI18n } from '@/i18n'
import { formatDateTime, formatNumber, getErrorMessage } from '@/pages/newbro-page-utils'
import type {
  MenteeListItem,
  MyStatusResponse,
  RelationshipView,
  RewardStage,
} from '@/types/api/mentor'

export function NewbroMentorPage() {
  const { t } = useI18n()
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [status, setStatus] = useState<MyStatusResponse | null>(null)
  const [applications, setApplications] = useState<RelationshipView[]>([])
  const [mentees, setMentees] = useState<MenteeListItem[]>([])
  const [rewardStages, setRewardStages] = useState<RewardStage[]>([])
  const [statusFilter, setStatusFilter] = useState<'active' | 'pending' | 'rejected' | 'revoked' | 'graduated' | 'all'>('active')
  const [actioningRelationshipId, setActioningRelationshipId] = useState<number | null>(null)
  const [refreshSeed, setRefreshSeed] = useState(0)

  const loadData = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const [mentorStatus, applicationList, menteeList, stageList] = await Promise.all([
        fetchMyMentorStatus(),
        fetchMentorApplications(),
        fetchMentorMentees({ current: 1, size: 20, status: statusFilter }),
        fetchMentorRewardStages(),
      ])
      setStatus(mentorStatus)
      setApplications(applicationList)
      setMentees(menteeList.list ?? [])
      setRewardStages(stageList)
    } catch (caughtError) {
      setStatus(null)
      setApplications([])
      setMentees([])
      setRewardStages([])
      setError(getErrorMessage(caughtError, t('newbroMentor.messages.loadFailed')))
    } finally {
      setLoading(false)
    }
  }, [statusFilter, t])

  useEffect(() => {
    void loadData()
  }, [loadData, refreshSeed])

  const handleApplicationAction = async (action: 'accept' | 'reject', relationshipId: number) => {
    setActioningRelationshipId(relationshipId)
    try {
      if (action === 'accept') {
        await acceptMentorApplication({ relationship_id: relationshipId })
      } else {
        await rejectMentorApplication({ relationship_id: relationshipId })
      }
      setRefreshSeed((current) => current + 1)
    } catch (caughtError) {
      setError(getErrorMessage(caughtError, t('newbroMentor.messages.actionFailed')))
    } finally {
      setActioningRelationshipId(null)
    }
  }

  return (
    <section className="space-y-4">
      <div className="rounded-lg border bg-card p-5">
        <div className="flex flex-col gap-3 lg:flex-row lg:items-end lg:justify-between">
          <div>
            <h1 className="text-xl font-semibold">{t('newbroMentor.title')}</h1>
            <p className="mt-1 text-sm text-muted-foreground">{t('newbroMentor.subtitle')}</p>
          </div>
          <Button type="button" variant="outline" onClick={() => setRefreshSeed((v) => v + 1)}>
            {t('common.refresh')}
          </Button>
        </div>
      </div>

      {error ? <p className="text-sm text-destructive">{error}</p> : null}
      {loading ? <p className="text-sm text-muted-foreground">{t('newbroMentor.loading')}</p> : null}

      <div className="grid gap-4 xl:grid-cols-2">
        <div className="rounded-lg border bg-card p-5">
          <h2 className="text-base font-semibold">{t('newbroMentor.statusSection')}</h2>
          <div className="mt-4 rounded-md border p-4">
            <div className="text-sm text-muted-foreground">{t('newbroMentor.eligibility')}</div>
            <div className="mt-1 font-medium">
              {status?.is_eligible ? t('newbroMentor.eligible') : t('newbroMentor.ineligible')}
            </div>
            {!status?.is_eligible ? (
              <div className="mt-1 text-sm text-muted-foreground">
                {t('newbroMentor.disqualifiedReason', { reason: status?.disqualified_reason || '-' })}
              </div>
            ) : null}
          </div>

          <div className="mt-4 rounded-md border p-4">
            <div className="text-sm text-muted-foreground">{t('newbroMentor.currentRelationship')}</div>
            {status?.current_relationship ? (
              <div className="mt-2 space-y-1">
                <div className="font-medium">{status.current_relationship.mentor_character_name}</div>
                <div className="text-sm text-muted-foreground">{status.current_relationship.mentor_nickname || '-'}</div>
                <div className="text-sm text-muted-foreground">
                  {t('newbroMentor.appliedAt')}: {formatDateTime(status.current_relationship.applied_at)}
                </div>
              </div>
            ) : (
              <div className="mt-1 text-sm text-muted-foreground">{t('newbroMentor.noCurrentRelationship')}</div>
            )}
          </div>
        </div>

        <div className="rounded-lg border bg-card p-5">
          <div className="flex items-center justify-between gap-3">
            <h2 className="text-base font-semibold">{t('newbroMentor.applicationsSection')}</h2>
            <span className="text-sm text-muted-foreground">{formatNumber(applications.length)}</span>
          </div>
          <div className="mt-4 space-y-3">
            {applications.map((application) => (
              <div key={application.id} className="rounded-md border p-4">
                <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
                  <div>
                    <div className="font-medium">{application.mentee_character_name}</div>
                    <div className="text-sm text-muted-foreground">{application.mentee_nickname || '-'}</div>
                    <div className="text-sm text-muted-foreground">
                      {t('newbroMentor.appliedAt')}: {formatDateTime(application.applied_at)}
                    </div>
                  </div>
                  <div className="flex gap-2">
                    <Button
                      type="button"
                      variant="outline"
                      disabled={actioningRelationshipId === application.id}
                      onClick={() => void handleApplicationAction('reject', application.id)}
                    >
                      {t('newbroMentor.reject')}
                    </Button>
                    <Button
                      type="button"
                      disabled={actioningRelationshipId === application.id}
                      onClick={() => void handleApplicationAction('accept', application.id)}
                    >
                      {t('newbroMentor.accept')}
                    </Button>
                  </div>
                </div>
              </div>
            ))}
            {!loading && applications.length === 0 ? (
              <p className="rounded-md border border-dashed p-6 text-center text-sm text-muted-foreground">
                {t('newbroMentor.noApplications')}
              </p>
            ) : null}
          </div>
        </div>
      </div>

      <div className="grid gap-4 xl:grid-cols-[0.9fr_1.1fr]">
        <div className="rounded-lg border bg-card p-5">
          <div className="flex items-center justify-between gap-3">
            <h2 className="text-base font-semibold">{t('newbroMentor.menteesSection')}</h2>
            <select
              className="h-9 rounded-md border border-input bg-background px-2 text-sm"
              value={statusFilter}
              onChange={(event) => {
                setStatusFilter(event.target.value as typeof statusFilter)
                setRefreshSeed((current) => current + 1)
              }}
            >
              <option value="active">{t('newbroMentor.status.active')}</option>
              <option value="pending">{t('newbroMentor.status.pending')}</option>
              <option value="rejected">{t('newbroMentor.status.rejected')}</option>
              <option value="revoked">{t('newbroMentor.status.revoked')}</option>
              <option value="graduated">{t('newbroMentor.status.graduated')}</option>
              <option value="all">{t('newbroMentor.status.all')}</option>
            </select>
          </div>
          <div className="mt-4 overflow-x-auto">
            <table className="min-w-full text-sm">
              <thead>
                <tr className="border-b bg-muted/40 text-left">
                  <th className="px-3 py-2">{t('newbroMentor.columns.mentee')}</th>
                  <th className="px-3 py-2">{t('newbroMentor.columns.status')}</th>
                  <th className="px-3 py-2">{t('newbroMentor.columns.sp')}</th>
                </tr>
              </thead>
              <tbody>
                {mentees.map((mentee) => (
                  <tr key={mentee.relationship_id} className="border-b">
                    <td className="px-3 py-2">
                      <div className="font-medium">{mentee.mentee_character_name}</div>
                      <div className="text-xs text-muted-foreground">{mentee.mentee_nickname || '-'}</div>
                    </td>
                    <td className="px-3 py-2">{t(`newbroMentor.status.${mentee.status}`)}</td>
                    <td className="px-3 py-2">{formatNumber(mentee.mentee_total_sp)}</td>
                  </tr>
                ))}
                {!loading && mentees.length === 0 ? (
                  <tr>
                    <td className="px-3 py-6 text-center text-muted-foreground" colSpan={3}>
                      {t('newbroMentor.noMentees')}
                    </td>
                  </tr>
                ) : null}
              </tbody>
            </table>
          </div>
        </div>

        <div className="rounded-lg border bg-card p-5">
          <div className="flex items-center justify-between">
            <h2 className="text-base font-semibold">{t('newbroMentor.rewardStagesSection')}</h2>
            <span className="text-sm text-muted-foreground">{formatNumber(rewardStages.length)}</span>
          </div>
          <div className="mt-4 space-y-3">
            {rewardStages.map((stage) => (
              <div key={stage.id} className="rounded-md border p-4">
                <div className="flex items-center justify-between gap-3">
                  <div>
                    <div className="font-medium">
                      {t('newbroMentor.rewardStageLabel', { order: stage.stage_order })}
                    </div>
                    <div className="text-sm text-muted-foreground">{stage.name}</div>
                  </div>
                  <div className="text-sm text-muted-foreground">
                    {t('newbroMentor.rewardAmount')}: {formatNumber(stage.reward_amount)}
                  </div>
                </div>
              </div>
            ))}
            {!loading && rewardStages.length === 0 ? (
              <p className="rounded-md border border-dashed p-6 text-center text-sm text-muted-foreground">
                {t('newbroMentor.noRewardStages')}
              </p>
            ) : null}
          </div>
        </div>
      </div>
    </section>
  )
}
