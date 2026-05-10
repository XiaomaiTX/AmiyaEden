import { useCallback, useEffect, useMemo, useState } from 'react'
import { fetchMyCharacters } from '@/api/auth'
import {
  fetchPersonalSkillPlanList,
  fetchSkillPlanCheckPlanSelection,
  fetchSkillPlanCheckSelection,
  fetchSkillPlanList,
  runSkillPlanCompletionCheck,
  saveSkillPlanCheckPlanSelection,
  saveSkillPlanCheckSelection,
} from '@/api/skill-plan'
import { Button } from '@/components/ui/button'
import { useI18n } from '@/i18n'
import { usePreferenceStore } from '@/stores'
import type { EveCharacter } from '@/types/api/auth'
import type {
  CompletionCheckResult,
  CompletionCharacter,
  SkillPlanListItem,
} from '@/types/api/skill-plan'
import { getErrorMessage, ShopBadge, ShopDialog } from './shop-page-utils'

function scopeLabel(t: ReturnType<typeof useI18n>['t'], scope: 'corp' | 'personal') {
  return scope === 'corp' ? t('skillPlan.scope.corp') : t('skillPlan.scope.personal')
}

function planLabel(t: ReturnType<typeof useI18n>['t'], plan: SkillPlanListItem) {
  return t('skillPlanCheck.planOptionLabel', {
    scope: scopeLabel(t, plan.plan_scope),
    title: plan.title,
  })
}

function copyToClipboard(text: string) {
  if (navigator.clipboard?.writeText) {
    return navigator.clipboard.writeText(text)
  }

  return Promise.reject(new Error('clipboard unavailable'))
}

export function SkillPlanCompletionCheckPage() {
  const { t } = useI18n()
  const locale = usePreferenceStore((state) => state.locale)
  const language = locale.startsWith('zh') ? 'zh' : 'en'

  const [loading, setLoading] = useState(true)
  const [running, setRunning] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [characters, setCharacters] = useState<EveCharacter[]>([])
  const [selectedCharacterIds, setSelectedCharacterIds] = useState<number[]>([])
  const [draftCharacterIds, setDraftCharacterIds] = useState<number[]>([])
  const [allPlans, setAllPlans] = useState<SkillPlanListItem[]>([])
  const [selectedPlanIds, setSelectedPlanIds] = useState<number[]>([])
  const [draftPlanIds, setDraftPlanIds] = useState<number[]>([])
  const [result, setResult] = useState<CompletionCheckResult | null>(null)
  const [characterDialogOpen, setCharacterDialogOpen] = useState(false)
  const [planDialogOpen, setPlanDialogOpen] = useState(false)

  const selectedCharacters = useMemo(
    () => characters.filter((character) => selectedCharacterIds.includes(character.character_id)),
    [characters, selectedCharacterIds]
  )
  const selectedPlans = useMemo(
    () => allPlans.filter((plan) => selectedPlanIds.includes(plan.id)),
    [allPlans, selectedPlanIds]
  )

  const loadAll = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const [characterList, corpPlans, personalPlans, savedSelection, savedPlanSelection] =
        await Promise.all([
          fetchMyCharacters(),
          fetchSkillPlanList({ current: 1, size: 200 }),
          fetchPersonalSkillPlanList({ current: 1, size: 200 }),
          fetchSkillPlanCheckSelection(),
          fetchSkillPlanCheckPlanSelection(),
        ])

      const nextPlans = [...(corpPlans.list ?? []), ...(personalPlans.list ?? [])]
      setCharacters(characterList ?? [])
      setAllPlans(nextPlans)
      setSelectedCharacterIds(savedSelection.character_ids ?? [])
      setDraftCharacterIds(savedSelection.character_ids ?? [])
      setSelectedPlanIds(savedPlanSelection.plan_ids ?? [])
      setDraftPlanIds(savedPlanSelection.plan_ids ?? [])
      setResult(null)
    } catch (caughtError) {
      setError(getErrorMessage(caughtError, t('httpMsg.requestFailed')))
    } finally {
      setLoading(false)
    }
  }, [t])

  useEffect(() => {
    const timer = window.setTimeout(() => {
      void loadAll()
    }, 0)
    return () => window.clearTimeout(timer)
  }, [loadAll])

  const openCharacterDialog = () => {
    setDraftCharacterIds(selectedCharacterIds)
    setCharacterDialogOpen(true)
  }

  const openPlanDialog = () => {
    setDraftPlanIds(selectedPlanIds)
    setPlanDialogOpen(true)
  }

  const saveCharacters = async () => {
    setError(null)
    try {
      const saved = await saveSkillPlanCheckSelection({ character_ids: draftCharacterIds })
      setSelectedCharacterIds(saved.character_ids ?? [])
      setCharacterDialogOpen(false)
    } catch (caughtError) {
      setError(getErrorMessage(caughtError, t('httpMsg.requestFailed')))
    }
  }

  const savePlans = async () => {
    setError(null)
    try {
      const saved = await saveSkillPlanCheckPlanSelection({ plan_ids: draftPlanIds })
      setSelectedPlanIds(saved.plan_ids ?? [])
      setPlanDialogOpen(false)
    } catch (caughtError) {
      setError(getErrorMessage(caughtError, t('httpMsg.requestFailed')))
    }
  }

  const runCheck = async () => {
    if (!selectedCharacterIds.length) {
      setError(t('skillPlanCheck.selectCharactersFirst'))
      return
    }

    setRunning(true)
    setError(null)
    try {
      const data = await runSkillPlanCompletionCheck({
        character_ids: selectedCharacterIds,
        language,
      })
      setResult(data)
    } catch (caughtError) {
      setError(getErrorMessage(caughtError, t('httpMsg.requestFailed')))
    } finally {
      setRunning(false)
    }
  }

  const copyMissingSkills = async (plan: CompletionCharacter['plans'][number]) => {
    const text = plan.missing_skills.map((skill) => `${skill.skill_name} ${skill.required_level}`).join('\n')
    if (!text) return

    try {
      await copyToClipboard(text)
    } catch (caughtError) {
      setError(getErrorMessage(caughtError, t('common.copyFailed')))
    }
  }

  return (
    <section className="space-y-4">
      <div className="rounded-lg border bg-card p-4">
        <div className="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
          <div>
            <h1 className="text-xl font-semibold">{t('skillPlanCheck.title')}</h1>
            <p className="mt-1 text-sm text-muted-foreground">{t('skillPlanCheck.subtitle')}</p>
          </div>
          <div className="flex flex-wrap gap-2">
            <Button type="button" variant="outline" onClick={openPlanDialog}>
              {t('skillPlanCheck.selectPlans')}
            </Button>
            <Button type="button" variant="outline" onClick={openCharacterDialog}>
              {t('skillPlanCheck.selectCharacters')}
            </Button>
            <Button type="button" onClick={() => void runCheck()} disabled={running || !selectedCharacterIds.length}>
              {t('skillPlanCheck.startCheck')}
            </Button>
          </div>
        </div>

        <div className="mt-4 flex flex-wrap gap-2">
          {selectedCharacters.length ? (
            selectedCharacters.map((character) => (
              <ShopBadge key={character.character_id} className="bg-slate-100 text-slate-700 dark:bg-slate-500/10 dark:text-slate-300">
                {character.character_name}
              </ShopBadge>
            ))
          ) : (
            <p className="text-sm text-muted-foreground">{t('skillPlanCheck.noCharactersSelected')}</p>
          )}
        </div>

        <div className="mt-4 flex flex-wrap gap-2">
          {selectedPlans.length ? (
            selectedPlans.map((plan) => (
              <ShopBadge key={plan.id} className="bg-slate-100 text-slate-700 dark:bg-slate-500/10 dark:text-slate-300">
                {planLabel(t, plan)}
              </ShopBadge>
            ))
          ) : (
            <p className="text-sm text-muted-foreground">{t('skillPlanCheck.noPlansSelected')}</p>
          )}
        </div>
      </div>

      {error ? <p className="text-sm text-destructive">{error}</p> : null}
      {loading ? <p className="text-sm text-muted-foreground">{t('skillPlanCheck.loading')}</p> : null}

      <div className="rounded-lg border bg-card p-4">
        <div className="flex items-center justify-between gap-3">
          <div>
            <h2 className="text-lg font-semibold">{t('skillPlanCheck.resultTitle')}</h2>
            <p className="mt-1 text-sm text-muted-foreground">{t('skillPlanCheck.resultSubtitle')}</p>
          </div>
          <ShopBadge className="bg-slate-100 text-slate-700 dark:bg-slate-500/10 dark:text-slate-300">
            {t('skillPlanCheck.planCount', { count: result?.plan_count ?? 0 })}
          </ShopBadge>
        </div>

        <div className="mt-4 space-y-4">
          {result?.characters?.length ? (
            result.characters.map((character) => (
              <details key={character.character_id} className="rounded-lg border bg-background p-3" open>
                <summary className="flex cursor-pointer list-none items-center justify-between gap-3">
                  <div>
                    <div className="font-medium">{character.character_name}</div>
                    <div className="text-xs text-muted-foreground">
                      {t('skillPlanCheck.characterSummary', {
                        completed: character.completed_plans,
                        total: character.total_plans,
                      })}
                    </div>
                  </div>
                  <ShopBadge className="bg-slate-100 text-slate-700 dark:bg-slate-500/10 dark:text-slate-300">
                    {character.completed_plans}/{character.total_plans}
                  </ShopBadge>
                </summary>

                <div className="mt-3 space-y-3">
                  {character.plans.map((plan) => (
                    <details key={plan.plan_id} className="rounded-lg border bg-card p-3">
                      <summary className="flex cursor-pointer list-none items-center justify-between gap-3">
                        <div className="min-w-0">
                          <div className="font-medium">{plan.plan_title}</div>
                          {plan.plan_description ? (
                            <div className="mt-1 text-xs text-muted-foreground">{plan.plan_description}</div>
                          ) : null}
                        </div>
                        <ShopBadge
                          className={
                            plan.fully_satisfied
                              ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-300'
                              : 'bg-rose-100 text-rose-700 dark:bg-rose-500/10 dark:text-rose-300'
                          }
                        >
                          {plan.fully_satisfied ? t('skillPlanCheck.planCompleted') : t('skillPlanCheck.planIncomplete')}
                        </ShopBadge>
                      </summary>

                      <div className="mt-3 space-y-3">
                        <div className="text-sm text-muted-foreground">
                          {plan.fully_satisfied
                            ? t('skillPlanCheck.planCompleteSummary')
                            : t('skillPlanCheck.planIncompleteSummary', {
                                matched: plan.matched_skills,
                                total: plan.total_skills,
                              })}
                        </div>
                        <div className="text-sm">
                          {t('skillPlanCheck.requiredVsCurrent', {
                            required: plan.total_skills,
                            current: plan.matched_skills,
                          })}
                        </div>
                        {plan.missing_skills.length ? (
                          <div className="rounded-lg border p-3">
                            <div className="flex items-center justify-between gap-3">
                              <div className="font-medium">{t('skillPlanCheck.missingSkillsTitle')}</div>
                              <Button type="button" variant="outline" size="sm" onClick={() => void copyMissingSkills(plan)}>
                                {t('skillPlanCheck.copyMissingSkills')}
                              </Button>
                            </div>
                            <ul className="mt-3 space-y-2 text-sm">
                              {plan.missing_skills.map((skill) => (
                                <li key={`${plan.plan_id}-${skill.skill_type_id}`} className="flex items-center justify-between gap-3">
                                  <span>{skill.skill_name}</span>
                                  <span className="text-muted-foreground">
                                    {t('skillPlanCheck.requiredVsCurrent', {
                                      required: skill.required_level,
                                      current: skill.current_level,
                                    })}
                                  </span>
                                </li>
                              ))}
                            </ul>
                          </div>
                        ) : null}
                      </div>
                    </details>
                  ))}
                </div>
              </details>
            ))
          ) : (
            <p className="text-sm text-muted-foreground">{t('skillPlanCheck.emptyResult')}</p>
          )}
        </div>
      </div>

      <ShopDialog
        open={characterDialogOpen}
        title={t('skillPlanCheck.selectCharacters')}
        closeLabel={t('common.close')}
        onClose={() => setCharacterDialogOpen(false)}
        footer={
          <>
            <Button type="button" variant="outline" onClick={() => setCharacterDialogOpen(false)}>
              {t('common.cancel')}
            </Button>
            <Button type="button" onClick={() => void saveCharacters()}>
              {t('common.confirm')}
            </Button>
          </>
        }
      >
        <div className="space-y-2">
          {characters.map((character) => (
            <label key={character.character_id} className="flex cursor-pointer items-center gap-3 rounded-lg border p-3">
              <input
                type="checkbox"
                checked={draftCharacterIds.includes(character.character_id)}
                onChange={(event) => {
                  setDraftCharacterIds((current) =>
                    event.target.checked
                      ? [...current, character.character_id]
                      : current.filter((id) => id !== character.character_id)
                  )
                }}
              />
              <span>{character.character_name}</span>
            </label>
          ))}
          {!characters.length ? <p className="text-sm text-muted-foreground">{t('skillPlanCheck.noAvailableCharacters')}</p> : null}
        </div>
      </ShopDialog>

      <ShopDialog
        open={planDialogOpen}
        title={t('skillPlanCheck.selectPlans')}
        closeLabel={t('common.close')}
        onClose={() => setPlanDialogOpen(false)}
        footer={
          <>
            <Button type="button" variant="outline" onClick={() => setPlanDialogOpen(false)}>
              {t('common.cancel')}
            </Button>
            <Button type="button" onClick={() => void savePlans()}>
              {t('common.confirm')}
            </Button>
          </>
        }
      >
        <div className="space-y-2">
          {allPlans.map((plan) => (
            <label key={plan.id} className="flex cursor-pointer items-center gap-3 rounded-lg border p-3">
              <input
                type="checkbox"
                checked={draftPlanIds.includes(plan.id)}
                onChange={(event) => {
                  setDraftPlanIds((current) =>
                    event.target.checked ? [...current, plan.id] : current.filter((id) => id !== plan.id)
                  )
                }}
              />
              <span>{planLabel(t, plan)}</span>
            </label>
          ))}
          {!allPlans.length ? <p className="text-sm text-muted-foreground">{t('skillPlanCheck.noAvailablePlans')}</p> : null}
        </div>
      </ShopDialog>
    </section>
  )
}
