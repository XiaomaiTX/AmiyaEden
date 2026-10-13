import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import {
  fetchSkillReadinessCharacterSkills,
  fetchSkillReadinessOptions,
  fetchSkillReadinessUserCharacters,
  querySkillReadiness,
} from '@/api/skill-readiness'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { DataTable, type ColumnDef } from '@/components/ui/data-table'
import { Dialog, DialogDescription, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { MultiSelect } from '@/components/ui/multi-select'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { useI18n } from '@/i18n'
import type {
  SkillReadinessCharacter,
  SkillReadinessCharacterDetail,
  SkillReadinessRequest,
  SkillReadinessSkill,
  SkillReadinessUser,
} from '@/types/api/skill-readiness'

type Filters = { corporationIds: number[]; planId: number; page: number; pageSize: number }
type DetailTab = 'comparison' | 'skills'

function readFilters(params: URLSearchParams): Filters {
  const corporationIds = (params.get('corporations') ?? '')
    .split(',')
    .map(Number)
    .filter((id) => Number.isSafeInteger(id) && id > 0)
  const planId = Number(params.get('plan'))
  const page = Number(params.get('page'))
  const pageSize = Number(params.get('page_size'))
  return {
    corporationIds: [...new Set(corporationIds)],
    planId: Number.isSafeInteger(planId) && planId > 0 ? planId : 0,
    page: Number.isSafeInteger(page) && page > 0 ? page : 1,
    pageSize: [20, 50, 100].includes(pageSize) ? pageSize : 20,
  }
}

function formatTimestamp(value: number | null, locale: string) {
  return value ? new Date(value * 1000).toLocaleString(locale) : '—'
}

function skillLevel(value: number | null) {
  return value === null ? '—' : String(value)
}

export function DashboardSkillReadinessPage() {
  const { t, locale } = useI18n()
  const [searchParams, setSearchParams] = useSearchParams()
  const [initial] = useState(() => readFilters(searchParams))
  const [filters, setFilters] = useState<Filters>(initial)
  const [options, setOptions] = useState<Awaited<
    ReturnType<typeof fetchSkillReadinessOptions>
  > | null>(null)
  const [result, setResult] = useState<Awaited<ReturnType<typeof querySkillReadiness>> | null>(null)
  const [loadingOptions, setLoadingOptions] = useState(true)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [userOpen, setUserOpen] = useState(false)
  const [selectedUser, setSelectedUser] = useState<SkillReadinessUser | null>(null)
  const [characters, setCharacters] = useState<SkillReadinessCharacter[]>([])
  const [charactersLoading, setCharactersLoading] = useState(false)
  const [charactersError, setCharactersError] = useState<string | null>(null)
  const [detailOpen, setDetailOpen] = useState(false)
  const [detail, setDetail] = useState<SkillReadinessCharacterDetail | null>(null)
  const [detailLoading, setDetailLoading] = useState(false)
  const [detailError, setDetailError] = useState<string | null>(null)
  const [detailTab, setDetailTab] = useState<DetailTab>('comparison')
  const [matchedOnly, setMatchedOnly] = useState(false)
  const [skillSearch, setSkillSearch] = useState('')
  const querySequence = useRef(0)
  const userSequence = useRef(0)
  const detailSequence = useRef(0)
  const mounted = useRef(false)
  const lastWrittenSearch = useRef<string | null>(searchParams.toString())

  const requestFor = useCallback(
    (value: Filters): SkillReadinessRequest => ({
      corporation_ids: value.corporationIds,
      plan_id: value.planId,
      page: value.page,
      page_size: value.pageSize,
      language: locale.startsWith('zh') ? 'zh' : 'en',
    }),
    [locale]
  )

  const loadQuery = useCallback(
    async (value: Filters) => {
      if (!value.planId) {
        setResult(null)
        setError(null)
        return
      }
      const sequence = ++querySequence.current
      setLoading(true)
      setError(null)
      try {
        const data = await querySkillReadiness(requestFor(value))
        if (mounted.current && sequence === querySequence.current) setResult(data)
      } catch (caughtError) {
        if (mounted.current && sequence === querySequence.current) {
          setResult(null)
          setError(
            caughtError instanceof Error
              ? caughtError.message
              : t('dashboardSkillReadiness.queryFailed')
          )
        }
      } finally {
        if (mounted.current && sequence === querySequence.current) setLoading(false)
      }
    },
    [requestFor, t]
  )

  const writeFilters = useCallback(
    (value: Filters) => {
      const next = new URLSearchParams()
      if (value.corporationIds.length) next.set('corporations', value.corporationIds.join(','))
      if (value.planId) next.set('plan', String(value.planId))
      if (value.page !== 1) next.set('page', String(value.page))
      if (value.pageSize !== 20) next.set('page_size', String(value.pageSize))
      lastWrittenSearch.current = next.toString()
      setSearchParams(next, { replace: true })
    },
    [setSearchParams]
  )

  const clearScopedState = useCallback(() => {
    querySequence.current++
    userSequence.current++
    detailSequence.current++
    setLoading(false)
    setError(null)
    setResult(null)
    setUserOpen(false)
    setSelectedUser(null)
    setCharacters([])
    setCharactersLoading(false)
    setCharactersError(null)
    setDetailOpen(false)
    setDetail(null)
    setDetailLoading(false)
    setDetailError(null)
  }, [])

  useEffect(() => {
    mounted.current = true
    return () => {
      mounted.current = false
    }
  }, [])

  useEffect(() => {
    let active = true
    const initialSearchTimer = window.setTimeout(() => {
      if (initial.planId) void loadQuery(initial)
    }, 0)
    void fetchSkillReadinessOptions()
      .then((data) => {
        if (active) setOptions(data)
      })
      .catch((caughtError: unknown) => {
        if (active)
          setError(
            caughtError instanceof Error
              ? caughtError.message
              : t('dashboardSkillReadiness.loadFailed')
          )
      })
      .finally(() => {
        if (active) setLoadingOptions(false)
      })
    return () => {
      active = false
      window.clearTimeout(initialSearchTimer)
    }
  }, [initial, loadQuery, t])

  useEffect(() => {
    const key = searchParams.toString()
    if (lastWrittenSearch.current === key) {
      lastWrittenSearch.current = null
      return
    }
    const restored = readFilters(searchParams)
    const restoreTimer = window.setTimeout(() => {
      clearScopedState()
      setFilters(restored)
      if (restored.planId) void loadQuery(restored)
    }, 0)
    return () => window.clearTimeout(restoreTimer)
  }, [clearScopedState, loadQuery, searchParams])

  const runSearch = (value: Filters) => {
    const next = { ...value, page: 1 }
    clearScopedState()
    setFilters(next)
    writeFilters(next)
    void loadQuery(next)
  }

  const openUser = useCallback(
    async (user: SkillReadinessUser) => {
      setSelectedUser(user)
      setCharacters([])
      setCharactersError(null)
      setUserOpen(true)
      setCharactersLoading(true)
      const sequence = ++userSequence.current
      try {
        const rows = await fetchSkillReadinessUserCharacters(user.user_id, requestFor(filters))
        if (mounted.current && sequence === userSequence.current) setCharacters(rows)
      } catch (caughtError) {
        if (mounted.current && sequence === userSequence.current) {
          setCharactersError(
            caughtError instanceof Error
              ? caughtError.message
              : t('dashboardSkillReadiness.loadFailed')
          )
        }
      } finally {
        if (mounted.current && sequence === userSequence.current) setCharactersLoading(false)
      }
    },
    [filters, requestFor, t]
  )

  const openCharacter = async (character: SkillReadinessCharacter) => {
    setDetail(null)
    setDetailError(null)
    setDetailTab('comparison')
    setMatchedOnly(false)
    setSkillSearch('')
    setDetailOpen(true)
    setDetailLoading(true)
    const sequence = ++detailSequence.current
    try {
      const data = await fetchSkillReadinessCharacterSkills(
        character.character_id,
        requestFor(filters)
      )
      if (mounted.current && sequence === detailSequence.current) setDetail(data)
    } catch (caughtError) {
      if (mounted.current && sequence === detailSequence.current) {
        setDetailError(
          caughtError instanceof Error
            ? caughtError.message
            : t('dashboardSkillReadiness.loadFailed')
        )
      }
    } finally {
      if (mounted.current && sequence === detailSequence.current) setDetailLoading(false)
    }
  }

  const columns = useMemo<ColumnDef<SkillReadinessUser, unknown>[]>(
    () => [
      {
        accessorKey: 'nickname',
        header: t('dashboardSkillReadiness.columns.user'),
        cell: ({ row }) =>
          row.original.nickname ||
          `${t('dashboardSkillReadiness.columns.userId')} ${row.original.user_id}`,
      },
      { accessorKey: 'user_id', header: t('dashboardSkillReadiness.columns.userId') },
      { accessorKey: 'character_count', header: t('dashboardSkillReadiness.columns.characters') },
      {
        accessorKey: 'satisfied_character_count',
        header: t('dashboardSkillReadiness.columns.satisfiedCharacters'),
      },
      {
        accessorKey: 'unknown_character_count',
        header: t('dashboardSkillReadiness.columns.unknownCharacters'),
      },
      {
        id: 'actions',
        header: t('dashboardSkillReadiness.columns.actions'),
        cell: ({ row }) => (
          <Button
            type="button"
            size="sm"
            variant="outline"
            onPress={() => void openUser(row.original)}
          >
            {t('dashboardSkillReadiness.viewCharacters')}
          </Button>
        ),
      },
    ],
    [openUser, t]
  )

  const filteredSkills = useMemo(() => {
    if (!detail) return []
    const search = skillSearch.trim().toLocaleLowerCase()
    if (detailTab === 'comparison') {
      return detail.requirements.filter((skill) => {
        const matchesSearch =
          !search || `${skill.skill_name} ${skill.group_name}`.toLocaleLowerCase().includes(search)
        return matchesSearch && (!matchedOnly || skill.satisfied === false)
      })
    }
    return detail.skills.filter(
      (skill) =>
        !search || `${skill.skill_name} ${skill.group_name}`.toLocaleLowerCase().includes(search)
    )
  }, [detail, detailTab, matchedOnly, skillSearch])

  const summary = result?.summary
  const plans = options?.plans ?? []
  const pages = Math.max(1, Math.ceil((result?.total ?? 0) / filters.pageSize))
  const statusKey = detail?.character.status ?? 'unknown'

  return (
    <section className="space-y-4">
      <div className="rounded-lg border bg-card p-5">
        <div className="flex flex-col gap-1">
          <h1 className="text-xl font-semibold">{t('dashboardSkillReadiness.title')}</h1>
          <p className="text-sm text-muted-foreground">{t('dashboardSkillReadiness.subtitle')}</p>
        </div>
        <div className="mt-5 grid grid-cols-1 items-end gap-3 md:grid-cols-[minmax(220px,1fr)_minmax(220px,1fr)_auto_auto]">
          <label className="space-y-1 text-sm">
            <span className="text-muted-foreground">
              {t('dashboardSkillReadiness.filters.corporations')}
            </span>
            <MultiSelect
              value={filters.corporationIds.map(String)}
              ariaLabel={t('dashboardSkillReadiness.filters.corporations')}
              onValueChange={(values) => {
                clearScopedState()
                setFilters((current) => ({
                  ...current,
                  corporationIds: values.map(Number),
                  page: 1,
                }))
              }}
              options={(options?.corporations ?? []).map((item) => ({
                value: String(item.corporation_id),
                label: `${item.corporation_name || item.corporation_id} (${item.corporation_id})`,
              }))}
              placeholder={t('dashboardSkillReadiness.allCorporations')}
              emptyText={t('dashboardSkillReadiness.empty')}
              removeLabel={(name) => t('dashboardSkillReadiness.removeCorporation', { name })}
              disabled={loadingOptions}
              className="min-h-8 w-full"
            />
          </label>
          <label className="space-y-1 text-sm">
            <span className="text-muted-foreground">
              {t('dashboardSkillReadiness.filters.plan')}
            </span>
            <Select
              aria-label={t('dashboardSkillReadiness.filters.plan')}
              selectedKey={filters.planId ? String(filters.planId) : null}
              onSelectionChange={(key) => {
                clearScopedState()
                setFilters((current) => ({ ...current, planId: Number(key) || 0, page: 1 }))
              }}
              isDisabled={loadingOptions || plans.length === 0}
            >
              <SelectTrigger aria-label={t('dashboardSkillReadiness.filters.plan')}>
                <SelectValue>{t('dashboardSkillReadiness.selectPlan')}</SelectValue>
              </SelectTrigger>
              <SelectContent>
                {plans.map((plan) => (
                  <SelectItem key={plan.id} id={String(plan.id)} textValue={plan.title}>
                    {plan.title}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </label>
          <Button
            type="button"
            isDisabled={!filters.planId || loading}
            onPress={() => runSearch(filters)}
          >
            {t('dashboardSkillReadiness.search')}
          </Button>
          <Button
            type="button"
            variant="outline"
            isDisabled={!filters.planId || loading}
            onPress={() => runSearch(filters)}
          >
            {t('dashboardSkillReadiness.refresh')}
          </Button>
        </div>
        {plans.length === 0 && !loadingOptions ? (
          <p className="mt-3 text-sm text-muted-foreground">
            {t('dashboardSkillReadiness.noPlans')}
          </p>
        ) : null}
      </div>

      {summary ? (
        <div className="grid grid-cols-2 gap-3 xl:grid-cols-5">
          {(
            [
              ['characters', summary.character_count],
              ['users', summary.user_count],
              ['satisfiedUsers', summary.satisfied_user_count],
              ['satisfiedCharacters', summary.satisfied_character_count],
              ['unknownCharacters', summary.unknown_character_count],
            ] as const
          ).map(([key, value]) => (
            <div key={key} className="rounded-lg border bg-card p-4">
              <div className="text-sm text-muted-foreground">
                {t(`dashboardSkillReadiness.stats.${key}`)}
              </div>
              <div className="mt-1 text-2xl font-semibold tabular-nums">{value}</div>
            </div>
          ))}
        </div>
      ) : null}

      <div className="rounded-lg border bg-card p-4">
        <DataTable
          columns={columns}
          data={result?.list ?? []}
          getRowId={(row) => String(row.user_id)}
          loading={loading || loadingOptions}
          error={error}
          loadingText={t('dashboardSkillReadiness.loading')}
          emptyText={t('dashboardSkillReadiness.empty')}
          pagination={
            result
              ? {
                  page: filters.page,
                  pageSize: filters.pageSize,
                  total: result.total,
                  pageSizeOptions: [20, 50, 100],
                  onPageChange: (page) => {
                    const next = { ...filters, page }
                    setFilters(next)
                    writeFilters(next)
                    void loadQuery(next)
                  },
                  onPageSizeChange: (pageSize) => {
                    const next = { ...filters, page: 1, pageSize }
                    setFilters(next)
                    writeFilters(next)
                    void loadQuery(next)
                  },
                  previousLabel: t('dashboardSkillReadiness.previous'),
                  nextLabel: t('dashboardSkillReadiness.next'),
                  pageSizeLabel: t('dashboardSkillReadiness.pageSize'),
                }
              : undefined
          }
        />
        {result ? (
          <p className="mt-2 text-right text-xs text-muted-foreground">
            {t('dashboardSkillReadiness.page', { page: filters.page, pages })}
          </p>
        ) : null}
      </div>

      <Dialog
        isOpen={userOpen}
        onOpenChange={setUserOpen}
        className="max-h-[85vh] overflow-y-auto sm:max-w-5xl"
      >
        <DialogHeader>
          <DialogTitle>
            {selectedUser?.nickname ||
              `${t('dashboardSkillReadiness.columns.userId')} ${selectedUser?.user_id ?? ''}`}
          </DialogTitle>
          <DialogDescription>
            {t('dashboardSkillReadiness.charactersTitle')} · {selectedUser?.character_count ?? 0}
          </DialogDescription>
        </DialogHeader>
        {charactersLoading ? (
          <p className="py-6 text-center text-muted-foreground">
            {t('dashboardSkillReadiness.loading')}
          </p>
        ) : null}
        {charactersError ? (
          <p role="alert" className="py-4 text-destructive">
            {charactersError}
          </p>
        ) : null}
        {!charactersLoading && !charactersError ? (
          <div className="overflow-x-auto rounded-md border">
            <Table aria-label={t('dashboardSkillReadiness.charactersTitle')}>
              <TableHeader>
                <TableRow>
                  <TableHead>{t('dashboardSkillReadiness.character')}</TableHead>
                  <TableHead>{t('dashboardSkillReadiness.corporation')}</TableHead>
                  <TableHead>{t('dashboardSkillReadiness.columns.satisfiedCharacters')}</TableHead>
                  <TableHead>{t('dashboardSkillReadiness.skills')}</TableHead>
                  <TableHead>{t('dashboardSkillReadiness.updatedAt')}</TableHead>
                  <TableHead>{t('dashboardSkillReadiness.columns.actions')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {characters.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={6} className="py-6 text-center text-muted-foreground">
                      {t('dashboardSkillReadiness.noCharacters')}
                    </TableCell>
                  </TableRow>
                ) : null}
                {characters.map((character) => (
                  <TableRow key={character.character_id}>
                    <TableCell className="font-medium">
                      {character.character_name}
                      <div className="text-xs text-muted-foreground">{character.character_id}</div>
                    </TableCell>
                    <TableCell>
                      {character.corporation_name ||
                        (character.corporation_id > 0
                          ? character.corporation_id
                          : t('dashboardSkillReadiness.unknownCorporation'))}
                    </TableCell>
                    <TableCell>{t(`dashboardSkillReadiness.status.${character.status}`)}</TableCell>
                    <TableCell>
                      {character.status === 'unknown'
                        ? '—'
                        : `${character.matched_skills}/${character.total_skills}`}
                    </TableCell>
                    <TableCell>{formatTimestamp(character.skill_updated_at, locale)}</TableCell>
                    <TableCell>
                      <Button
                        type="button"
                        size="sm"
                        variant="outline"
                        onPress={() => void openCharacter(character)}
                      >
                        {t('dashboardSkillReadiness.details')}
                      </Button>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        ) : null}
      </Dialog>

      <Dialog
        isOpen={detailOpen}
        onOpenChange={setDetailOpen}
        className="max-h-[88vh] overflow-y-auto sm:max-w-5xl"
      >
        <DialogHeader>
          <DialogTitle>
            {detail?.character.character_name ?? t('dashboardSkillReadiness.details')}
          </DialogTitle>
          <DialogDescription>
            {detail ? (
              <>
                {t(`dashboardSkillReadiness.status.${statusKey}`)} ·{' '}
                {t('dashboardSkillReadiness.corporation')}:{' '}
                {detail.character.corporation_name ||
                  (detail.character.corporation_id > 0
                    ? detail.character.corporation_id
                    : t('dashboardSkillReadiness.unknownCorporation'))}{' '}
                · {t('dashboardSkillReadiness.updatedAt')}:{' '}
                {formatTimestamp(detail.character.skill_updated_at, locale)}
              </>
            ) : null}
          </DialogDescription>
        </DialogHeader>
        {detail?.character.token_invalid ? (
          <p className="rounded-md border border-amber-500/30 bg-amber-500/10 p-3 text-sm text-amber-700 dark:text-amber-300">
            {t('dashboardSkillReadiness.tokenInvalid')}
          </p>
        ) : null}
        {detail?.character.status === 'unknown' ? (
          <p className="rounded-md border bg-muted/50 p-3 text-sm text-muted-foreground">
            {t('dashboardSkillReadiness.unknownHint')}
          </p>
        ) : null}
        <div className="flex flex-wrap items-center gap-2 border-b pb-3">
          <Button
            type="button"
            size="sm"
            variant={detailTab === 'comparison' ? 'default' : 'outline'}
            onPress={() => setDetailTab('comparison')}
          >
            {t('dashboardSkillReadiness.comparison')}
          </Button>
          <Button
            type="button"
            size="sm"
            variant={detailTab === 'skills' ? 'default' : 'outline'}
            onPress={() => setDetailTab('skills')}
          >
            {t('dashboardSkillReadiness.allSkills')}
          </Button>
          <Input
            className="ml-auto max-w-xs"
            aria-label={t('dashboardSkillReadiness.searchSkills')}
            value={skillSearch}
            onChange={(event) => setSkillSearch(event.target.value)}
            placeholder={t('dashboardSkillReadiness.searchSkills')}
          />
          {detailTab === 'comparison' ? (
            <label className="flex items-center gap-2 text-sm">
              <Checkbox
                isSelected={matchedOnly}
                onChange={(selected) => setMatchedOnly(selected === true)}
              />
              {t('dashboardSkillReadiness.matchedOnly')}
            </label>
          ) : null}
        </div>
        {detailLoading ? (
          <p className="py-6 text-center text-muted-foreground">
            {t('dashboardSkillReadiness.loading')}
          </p>
        ) : null}
        {detailError ? (
          <p role="alert" className="py-4 text-destructive">
            {detailError}
          </p>
        ) : null}
        {detail && !detailLoading && !detailError ? (
          <div className="max-h-[50vh] overflow-auto rounded-md border">
            <Table
              aria-label={t(
                detailTab === 'comparison'
                  ? 'dashboardSkillReadiness.comparison'
                  : 'dashboardSkillReadiness.allSkills'
              )}
            >
              <TableHeader>
                <TableRow>
                  <TableHead>{t('dashboardSkillReadiness.skillName')}</TableHead>
                  <TableHead>{t('dashboardSkillReadiness.group')}</TableHead>
                  {detailTab === 'comparison' ? (
                    <>
                      <TableHead>{t('dashboardSkillReadiness.requiredLevel')}</TableHead>
                      <TableHead>{t('dashboardSkillReadiness.currentLevel')}</TableHead>
                      <TableHead>{t('common.status')}</TableHead>
                    </>
                  ) : (
                    <>
                      <TableHead>{t('dashboardSkillReadiness.currentLevel')}</TableHead>
                      <TableHead>{t('dashboardSkillReadiness.activeLevel')}</TableHead>
                      <TableHead>{t('dashboardSkillReadiness.skillPoints')}</TableHead>
                    </>
                  )}
                </TableRow>
              </TableHeader>
              <TableBody>
                {filteredSkills.map((skill: SkillReadinessSkill) => (
                  <TableRow key={skill.skill_type_id}>
                    <TableCell className="font-medium">
                      {skill.skill_name}
                      <div className="text-xs text-muted-foreground">{skill.skill_type_id}</div>
                    </TableCell>
                    <TableCell>{skill.group_name}</TableCell>
                    {detailTab === 'comparison' ? (
                      <>
                        <TableCell>{skill.required_level ?? '—'}</TableCell>
                        <TableCell>{skillLevel(skill.current_level)}</TableCell>
                        <TableCell>
                          {skill.satisfied === undefined
                            ? '—'
                            : t(
                                skill.satisfied
                                  ? 'dashboardSkillReadiness.status.satisfied'
                                  : 'dashboardSkillReadiness.status.unsatisfied'
                              )}
                        </TableCell>
                      </>
                    ) : (
                      <>
                        <TableCell>{skillLevel(skill.trained_level)}</TableCell>
                        <TableCell>{skillLevel(skill.active_level)}</TableCell>
                        <TableCell>
                          {skill.skillpoints_in_skill?.toLocaleString(locale) ?? '—'}
                        </TableCell>
                      </>
                    )}
                  </TableRow>
                ))}
                {filteredSkills.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={5} className="py-6 text-center text-muted-foreground">
                      {detailTab === 'comparison' && !matchedOnly
                        ? t('dashboardSkillReadiness.noMissingSkills')
                        : t('dashboardSkillReadiness.noSkills')}
                    </TableCell>
                  </TableRow>
                ) : null}
              </TableBody>
            </Table>
          </div>
        ) : null}
      </Dialog>
    </section>
  )
}
