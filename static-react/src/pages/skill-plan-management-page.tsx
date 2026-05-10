import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { useI18n } from '@/i18n'
import { usePreferenceStore } from '@/stores'
import type {
  CreateSkillPlanParams,
  SkillPlanDetail,
  SkillPlanListItem,
  SkillPlanScope,
  UpdateSkillPlanParams,
} from '@/types/api/skill-plan'
import { formatDateTime, getErrorMessage, ShopBadge, ShopDialog } from './shop-page-utils'

interface SkillPlanManagementPageProps {
  titleKey: string
  subtitleKey: string
  emptyKey: string
  canManage: boolean
  fetchList: (params?: { current?: number; size?: number; keyword?: string }) => Promise<{
    list: SkillPlanListItem[]
    total: number
    page: number
    pageSize: number
  }>
  fetchDetail: (id: number, lang?: string) => Promise<SkillPlanDetail>
  createPlan: (data: CreateSkillPlanParams, lang?: string) => Promise<SkillPlanDetail>
  updatePlan: (id: number, data: UpdateSkillPlanParams, lang?: string) => Promise<SkillPlanDetail>
  deletePlan: (id: number) => Promise<null>
  reorderPlans: (ids: number[]) => Promise<null>
}

type SkillPlanFormState = {
  id: number
  title: string
  description: string
  ship_type_id: string
  sort_order: number
  skills_text: string
}

const emptyForm: SkillPlanFormState = {
  id: 0,
  title: '',
  description: '',
  ship_type_id: '',
  sort_order: 0,
  skills_text: '',
}

function scopeLabel(t: ReturnType<typeof useI18n>['t'], scope: SkillPlanScope) {
  return scope === 'corp' ? t('skillPlan.scope.corp') : t('skillPlan.scope.personal')
}

function buildSkillsText(plan?: SkillPlanDetail | null) {
  if (!plan?.skills?.length) return ''
  return plan.skills
    .slice()
    .sort((left, right) => left.sort - right.sort)
    .map((skill) => `${skill.skill_name} ${skill.required_level}`)
    .join('\n')
}

export function SkillPlanManagementPage({
  titleKey,
  subtitleKey,
  emptyKey,
  canManage,
  fetchList,
  fetchDetail,
  createPlan,
  updatePlan,
  deletePlan,
  reorderPlans,
}: SkillPlanManagementPageProps) {
  const { t } = useI18n()
  const locale = usePreferenceStore((state) => state.locale)
  const language = locale.startsWith('zh') ? 'zh' : 'en'
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [keyword, setKeyword] = useState('')
  const [plans, setPlans] = useState<SkillPlanListItem[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [selectedPlanId, setSelectedPlanId] = useState<number | null>(null)
  const [selectedPlan, setSelectedPlan] = useState<SkillPlanDetail | null>(null)
  const [detailLoading, setDetailLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [dialogOpen, setDialogOpen] = useState(false)
  const [reorderSaving, setReorderSaving] = useState(false)
  const [form, setForm] = useState<SkillPlanFormState>(emptyForm)
  const selectedPlanIdRef = useRef<number | null>(null)

  const pageCount = useMemo(() => Math.max(1, Math.ceil(total / pageSize) || 1), [pageSize, total])

  const loadDetail = useCallback(
    async (id: number) => {
      if (!id) return

      setDetailLoading(true)
      setError(null)
      try {
        const detail = await fetchDetail(id, language)
        setSelectedPlan(detail)
        setSelectedPlanId(id)
        selectedPlanIdRef.current = id
      } catch (caughtError) {
        setError(getErrorMessage(caughtError, t('httpMsg.requestFailed')))
        setSelectedPlan(null)
        selectedPlanIdRef.current = null
      } finally {
        setDetailLoading(false)
      }
    },
    [fetchDetail, language, t]
  )

  const loadPlans = useCallback(
    async (nextPage = page, nextSize = pageSize) => {
      setLoading(true)
      setError(null)
      try {
        const response = await fetchList({
          current: nextPage,
          size: nextSize,
          keyword: keyword.trim() || undefined,
        })
        setPlans(response.list ?? [])
        setTotal(response.total ?? 0)
        setPage(response.page ?? nextPage)
        setPageSize(response.pageSize ?? nextSize)

        const preferredId = selectedPlanIdRef.current ?? response.list?.[0]?.id ?? null
        if (preferredId) {
          await loadDetail(preferredId)
        } else {
          setSelectedPlan(null)
          setSelectedPlanId(null)
          selectedPlanIdRef.current = null
        }
      } catch (caughtError) {
        setError(getErrorMessage(caughtError, t('httpMsg.requestFailed')))
        setPlans([])
        setTotal(0)
        setSelectedPlan(null)
        setSelectedPlanId(null)
        selectedPlanIdRef.current = null
      } finally {
        setLoading(false)
      }
    },
    [fetchList, keyword, loadDetail, page, pageSize, t]
  )

  useEffect(() => {
    const timer = window.setTimeout(() => {
      void loadPlans()
    }, 0)
    return () => window.clearTimeout(timer)
  }, [loadPlans])

  const openCreate = () => {
    setForm(emptyForm)
    setDialogOpen(true)
  }

  const openEdit = () => {
    if (!selectedPlan) return

    setForm({
      id: selectedPlan.id,
      title: selectedPlan.title,
      description: selectedPlan.description,
      ship_type_id: selectedPlan.ship_type_id ? String(selectedPlan.ship_type_id) : '',
      sort_order: selectedPlan.sort_order,
      skills_text: buildSkillsText(selectedPlan),
    })
    setDialogOpen(true)
  }

  const submit = async () => {
    const title = form.title.trim()
    if (!title) {
      setError(t('skillPlan.fields.titleRequired'))
      return
    }

    setSaving(true)
    setError(null)
    try {
      const payload: CreateSkillPlanParams = {
        title,
        description: form.description.trim() || undefined,
        ship_type_id: form.ship_type_id ? Number(form.ship_type_id) : undefined,
        sort_order: form.sort_order,
        skills_text: form.skills_text.trim() || undefined,
      }

      if (form.id > 0) {
        await updatePlan(form.id, payload, language)
      } else {
        await createPlan(payload, language)
      }

      setDialogOpen(false)
      await loadPlans()
    } catch (caughtError) {
      setError(getErrorMessage(caughtError, t('httpMsg.requestFailed')))
    } finally {
      setSaving(false)
    }
  }

  const remove = async () => {
    if (!selectedPlan) return
    if (!window.confirm(t('skillPlan.deleteConfirm', { title: selectedPlan.title }))) return

    setSaving(true)
    setError(null)
    try {
      await deletePlan(selectedPlan.id)
      setSelectedPlan(null)
      setSelectedPlanId(null)
      selectedPlanIdRef.current = null
      await loadPlans()
    } catch (caughtError) {
      setError(getErrorMessage(caughtError, t('httpMsg.requestFailed')))
    } finally {
      setSaving(false)
    }
  }

  const reorder = async (fromIndex: number, delta: number) => {
    const toIndex = fromIndex + delta
    if (toIndex < 0 || toIndex >= plans.length) return

    const next = [...plans]
    const [moved] = next.splice(fromIndex, 1)
    next.splice(toIndex, 0, moved)
    setPlans(next)

    setReorderSaving(true)
    try {
      await reorderPlans(next.map((item) => item.id))
    } catch (caughtError) {
      setError(getErrorMessage(caughtError, t('skillPlan.reorderFailed')))
      await loadPlans()
    } finally {
      setReorderSaving(false)
    }
  }

  return (
    <section className="grid gap-4 xl:grid-cols-[320px_minmax(0,1fr)]">
      <div className="rounded-lg border bg-card p-4">
        <div className="flex items-start justify-between gap-3">
          <div>
            <h1 className="text-xl font-semibold">{t(titleKey)}</h1>
            <p className="mt-1 text-sm text-muted-foreground">{t(subtitleKey)}</p>
          </div>
          {canManage ? (
            <Button type="button" onClick={openCreate}>
              {t('skillPlan.create')}
            </Button>
          ) : null}
        </div>

        <div className="mt-4 flex gap-2">
          <Input
            value={keyword}
            placeholder={t('skillPlan.searchPlaceholder')}
            onChange={(event) => setKeyword(event.target.value)}
            onKeyDown={(event) => {
              if (event.key === 'Enter') {
                setPage(1)
              }
            }}
          />
          <Button
            type="button"
            variant="outline"
            onClick={() => {
              setPage(1)
            }}
          >
            {t('common.search')}
          </Button>
        </div>

        <div className="mt-4 text-sm text-muted-foreground">
          {t('skillPlan.listTitle')} ({total})
        </div>

        <div className="mt-3 space-y-2">
          {loading ? (
            <p className="text-sm text-muted-foreground">{t('skillPlan.loading')}</p>
          ) : null}
          {plans.map((plan, index) => (
            <button
              key={plan.id}
              type="button"
              className={`w-full rounded-lg border p-3 text-left transition ${
                selectedPlanId === plan.id ? 'border-primary bg-primary/5' : 'bg-background'
              }`}
              onClick={() => void loadDetail(plan.id)}
            >
              <div className="flex items-start justify-between gap-2">
                <div className="min-w-0">
                  <div className="font-medium">{plan.title}</div>
                  <div className="mt-1 line-clamp-2 text-xs text-muted-foreground">
                    {plan.description || t('skillPlan.descriptionEmpty')}
                  </div>
                </div>
                <ShopBadge className="bg-slate-100 text-slate-700 dark:bg-slate-500/10 dark:text-slate-300">
                  {scopeLabel(t, plan.plan_scope)}
                </ShopBadge>
              </div>
              {canManage ? (
                <div className="mt-3 flex gap-2">
                  <Button
                    type="button"
                    size="sm"
                    variant="outline"
                    disabled={reorderSaving || index === 0}
                    onClick={(event) => {
                      event.stopPropagation()
                      void reorder(index, -1)
                    }}
                  >
                    ↑
                  </Button>
                  <Button
                    type="button"
                    size="sm"
                    variant="outline"
                    disabled={reorderSaving || index === plans.length - 1}
                    onClick={(event) => {
                      event.stopPropagation()
                      void reorder(index, 1)
                    }}
                  >
                    ↓
                  </Button>
                </div>
              ) : null}
            </button>
          ))}
          {!loading && plans.length === 0 ? (
            <p className="rounded-lg border bg-card p-4 text-sm text-muted-foreground">
              {t(emptyKey)}
            </p>
          ) : null}
        </div>

        <div className="mt-4 flex items-center gap-2 text-sm">
          <span>
            {page}/{pageCount}
          </span>
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={() => {
              setPage((current) => Math.max(1, current - 1))
            }}
            disabled={page <= 1}
          >
            {t('welfareMy.pagination.prev')}
          </Button>
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={() => {
              setPage((current) => current + 1)
            }}
            disabled={plans.length < pageSize || page * pageSize >= total}
          >
            {t('welfareMy.pagination.next')}
          </Button>
        </div>
      </div>

      <div className="rounded-lg border bg-card p-4">
        {error ? <p className="mb-4 text-sm text-destructive">{error}</p> : null}
        {detailLoading ? <p className="mb-4 text-sm text-muted-foreground">{t('skillPlan.loading')}</p> : null}
        {selectedPlan ? (
          <div className="space-y-4">
            <div className="flex flex-wrap items-start justify-between gap-3">
              <div>
                <h2 className="text-xl font-semibold">{selectedPlan.title}</h2>
                <p className="mt-1 text-sm text-muted-foreground">{selectedPlan.description || t('skillPlan.descriptionEmpty')}</p>
              </div>
              {canManage ? (
                <div className="flex gap-2">
                  <Button type="button" variant="outline" onClick={openEdit}>
                    {t('common.edit')}
                  </Button>
                  <Button type="button" variant="outline" onClick={() => void remove()} disabled={saving}>
                    {t('common.delete')}
                  </Button>
                </div>
              ) : null}
            </div>

            <div className="grid gap-3 sm:grid-cols-3">
              <div className="rounded-lg border p-3">
                <div className="text-xs text-muted-foreground">{t('skillPlan.skillCount')}</div>
                <div className="mt-1 text-lg font-semibold">{selectedPlan.skill_count}</div>
              </div>
              <div className="rounded-lg border p-3">
                <div className="text-xs text-muted-foreground">{t('common.updatedAt')}</div>
                <div className="mt-1 text-sm font-medium">{formatDateTime(selectedPlan.updated_at)}</div>
              </div>
              <div className="rounded-lg border p-3">
                <div className="text-xs text-muted-foreground">{t('common.createdAt')}</div>
                <div className="mt-1 text-sm font-medium">{formatDateTime(selectedPlan.created_at)}</div>
              </div>
            </div>

            <div className="rounded-lg border p-3">
              <div className="text-sm font-medium text-muted-foreground">{t('skillPlan.skillListTitle')}</div>
              <div className="mt-3 overflow-x-auto">
                <table className="min-w-full text-sm">
                  <thead>
                    <tr className="border-b bg-muted/40 text-left">
                      <th className="px-3 py-2">{t('skillPlan.table.skill')}</th>
                      <th className="px-3 py-2">{t('skillPlan.table.group')}</th>
                      <th className="px-3 py-2">{t('skillPlan.table.requiredLevel')}</th>
                    </tr>
                  </thead>
                  <tbody>
                    {selectedPlan.skills.map((skill) => (
                      <tr key={skill.id} className="border-b">
                        <td className="px-3 py-2">{skill.skill_name}</td>
                        <td className="px-3 py-2">{skill.group_name}</td>
                        <td className="px-3 py-2">{t('skillPlan.level', { level: skill.required_level })}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          </div>
        ) : (
          <p className="text-sm text-muted-foreground">{t('skillPlan.emptyDetail')}</p>
        )}
      </div>

      <ShopDialog
        open={dialogOpen}
        title={form.id > 0 ? t('skillPlan.edit') : t('skillPlan.create')}
        widthClass="max-w-2xl"
        closeLabel={t('common.close')}
        onClose={() => setDialogOpen(false)}
        footer={
          <>
            <Button type="button" variant="outline" onClick={() => setDialogOpen(false)} disabled={saving}>
              {t('common.cancel')}
            </Button>
            <Button type="button" onClick={() => void submit()} disabled={saving || !form.title.trim()}>
              {saving ? t('shopManage.saving') : t('common.confirm')}
            </Button>
          </>
        }
      >
        <div className="grid gap-4 md:grid-cols-2">
          <label className="space-y-2 md:col-span-2">
            <span className="text-sm text-muted-foreground">{t('skillPlan.fields.title')}</span>
            <Input
              value={form.title}
              placeholder={t('skillPlan.fields.titlePlaceholder')}
              required
              onChange={(event) => setForm((current) => ({ ...current, title: event.target.value }))}
            />
          </label>
          <label className="space-y-2 md:col-span-2">
            <span className="text-sm text-muted-foreground">{t('skillPlan.fields.description')}</span>
            <textarea
              className="min-h-24 w-full rounded-lg border border-input bg-background px-3 py-2 text-sm outline-none"
              value={form.description}
              placeholder={t('skillPlan.fields.descriptionPlaceholder')}
              onChange={(event) => setForm((current) => ({ ...current, description: event.target.value }))}
            />
          </label>
          <label className="space-y-2">
            <span className="text-sm text-muted-foreground">{t('skillPlan.fields.ship')}</span>
            <Input
              type="number"
              value={form.ship_type_id}
              placeholder={t('skillPlan.fields.shipPlaceholder')}
              onChange={(event) => setForm((current) => ({ ...current, ship_type_id: event.target.value }))}
            />
          </label>
          <label className="space-y-2">
            <span className="text-sm text-muted-foreground">{t('skillPlan.sortOrder')}</span>
            <Input
              type="number"
              value={String(form.sort_order)}
              onChange={(event) => setForm((current) => ({ ...current, sort_order: Number(event.target.value) }))}
            />
          </label>
          <label className="space-y-2 md:col-span-2">
            <span className="text-sm text-muted-foreground">{t('skillPlan.fields.skillsText')}</span>
            <textarea
              className="min-h-40 w-full rounded-lg border border-input bg-background px-3 py-2 text-sm outline-none"
              value={form.skills_text}
              placeholder={t('skillPlan.fields.skillsTextPlaceholder')}
              onChange={(event) => setForm((current) => ({ ...current, skills_text: event.target.value }))}
            />
          </label>
        </div>
      </ShopDialog>
    </section>
  )
}
