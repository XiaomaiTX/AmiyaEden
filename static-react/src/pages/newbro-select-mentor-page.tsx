import { useCallback, useEffect, useState } from 'react'
import { applyForMentor, fetchMentorCandidates, fetchMyMentorStatus } from '@/api/mentor'
import { Button } from '@/components/ui/button'
import { buildEveCharacterPortraitUrl } from '@/lib/eve-image'
import { useI18n } from '@/i18n'
import { formatDateTime, formatNumber, getErrorMessage } from '@/pages/newbro-page-utils'
import type { MentorCandidate, MyStatusResponse } from '@/types/api/mentor'

export function NewbroSelectMentorPage() {
  const { t } = useI18n()
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [state, setState] = useState<MyStatusResponse | null>(null)
  const [mentors, setMentors] = useState<MentorCandidate[]>([])
  const [submittingMentorId, setSubmittingMentorId] = useState<number | null>(null)
  const [refreshSeed, setRefreshSeed] = useState(0)

  const loadData = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const [status, mentorList] = await Promise.all([fetchMyMentorStatus(), fetchMentorCandidates()])
      setState(status)
      setMentors(mentorList)
    } catch (caughtError) {
      setState(null)
      setMentors([])
      setError(getErrorMessage(caughtError, t('newbroSelectMentor.messages.loadFailed')))
    } finally {
      setLoading(false)
    }
  }, [t])

  useEffect(() => {
    void loadData()
  }, [loadData, refreshSeed])

  const handleApply = async (mentorUserId: number) => {
    setSubmittingMentorId(mentorUserId)
    try {
      await applyForMentor({ mentor_user_id: mentorUserId })
      setRefreshSeed((current) => current + 1)
    } catch (caughtError) {
      setError(getErrorMessage(caughtError, t('newbroSelectMentor.messages.applyFailed')))
    } finally {
      setSubmittingMentorId(null)
    }
  }

  return (
    <section className="space-y-4">
      <div className="rounded-lg border bg-card p-5">
        <div className="flex flex-col gap-3 lg:flex-row lg:items-end lg:justify-between">
          <div>
            <h1 className="text-xl font-semibold">{t('newbroSelectMentor.title')}</h1>
            <p className="mt-1 text-sm text-muted-foreground">{t('newbroSelectMentor.subtitle')}</p>
          </div>
          <Button type="button" variant="outline" onClick={() => setRefreshSeed((v) => v + 1)}>
            {t('common.refresh')}
          </Button>
        </div>
      </div>

      {error ? <p className="text-sm text-destructive">{error}</p> : null}
      {loading ? <p className="text-sm text-muted-foreground">{t('newbroSelectMentor.loading')}</p> : null}

      <div className="grid gap-4 xl:grid-cols-[0.9fr_1.1fr]">
        <div className="rounded-lg border bg-card p-5">
          <h2 className="text-base font-semibold">{t('newbroSelectMentor.statusSection')}</h2>
          <div className="mt-3 space-y-3 text-sm">
            <div className="rounded-md border p-4">
              <div className="text-muted-foreground">{t('newbroSelectMentor.eligibility')}</div>
              <div className="mt-1 font-medium">
                {state?.is_eligible ? t('newbroSelectMentor.eligible') : t('newbroSelectMentor.ineligible')}
              </div>
              {!state?.is_eligible ? (
                <div className="mt-1 text-muted-foreground">
                  {t('newbroSelectMentor.disqualifiedReason', {
                    reason: state?.disqualified_reason || '-',
                  })}
                </div>
              ) : null}
            </div>

            <div className="rounded-md border p-4">
              <div className="text-muted-foreground">{t('newbroSelectMentor.currentRelationship')}</div>
              {state?.current_relationship ? (
                <div className="mt-2 space-y-1">
                  <div className="font-medium">{state.current_relationship.mentor_character_name}</div>
                  <div className="text-muted-foreground">{state.current_relationship.mentor_nickname || '-'}</div>
                  <div className="text-muted-foreground">
                    {t('newbroSelectMentor.appliedAt')}: {formatDateTime(state.current_relationship.applied_at)}
                  </div>
                </div>
              ) : (
                <div className="mt-1 text-muted-foreground">{t('newbroSelectMentor.noCurrentRelationship')}</div>
              )}
            </div>
          </div>
        </div>

        <div className="rounded-lg border bg-card p-5">
          <div className="flex items-center justify-between gap-3">
            <h2 className="text-base font-semibold">{t('newbroSelectMentor.mentorList')}</h2>
            <span className="text-sm text-muted-foreground">{formatNumber(mentors.length)}</span>
          </div>

          <div className="mt-4 grid gap-3">
            {mentors.map((mentor) => {
              const disabled = !state?.is_eligible || !!state?.current_relationship
              const submitting = submittingMentorId === mentor.mentor_user_id
              return (
                <div key={mentor.mentor_user_id} className="rounded-md border p-4">
                  <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
                    <div className="flex items-center gap-3">
                      <img
                        alt={mentor.mentor_character_name}
                        src={buildEveCharacterPortraitUrl(mentor.mentor_character_id, 64)}
                        className="h-16 w-16 rounded-full border object-cover"
                      />
                      <div>
                        <div className="font-medium">{mentor.mentor_character_name}</div>
                        <div className="text-sm text-muted-foreground">{mentor.mentor_nickname || '-'}</div>
                        <div className="text-sm text-muted-foreground">
                          {t('newbroSelectMentor.activeMentees')}: {mentor.active_mentee_count}
                        </div>
                        <div className="text-sm text-muted-foreground">
                          {t('newbroSelectMentor.lastOnline')}: {formatDateTime(mentor.last_online_at)}
                        </div>
                      </div>
                    </div>
                    <Button type="button" disabled={disabled || submitting} onClick={() => void handleApply(mentor.mentor_user_id)}>
                      {submitting ? t('newbroSelectMentor.applying') : t('newbroSelectMentor.apply')}
                    </Button>
                  </div>
                </div>
              )
            })}

            {!loading && mentors.length === 0 ? (
              <p className="rounded-md border border-dashed p-6 text-center text-sm text-muted-foreground">
                {t('newbroSelectMentor.noMentors')}
              </p>
            ) : null}
          </div>
        </div>
      </div>
    </section>
  )
}
