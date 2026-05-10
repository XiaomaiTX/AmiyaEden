import { useEffect, useState } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import {
  fetchCorporationStructureFilterOptions,
  fetchCorporationStructureList,
  fetchCorporationStructureSettings,
  runCorporationStructuresTask,
  updateCorporationStructureAuthorizations,
} from '@/api/corporation-structures'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { cn } from '@/lib/utils'
import { useI18n } from '@/i18n'
import type {
  CorporationStructureFilterOptionsResponse,
  CorporationStructureListRequest,
  CorporationStructureRow,
  CorporationStructureServiceInfo,
  CorporationStructuresSettings,
  CorporationStructureSystemOption,
} from '@/types/api/dashboard'

type ActiveTab = 'list' | 'settings'
type DateTimeRange = [string, string] | null

const DEFAULT_FILTERS = {
  corporation_id: 0,
  keyword: '',
  state_groups: [] as string[],
  fuel_bucket: 'all' as CorporationStructureListRequest['fuel_bucket'],
  fuel_min_hours: '',
  fuel_max_hours: '',
  system_ids: [] as number[],
  security_bands: [] as ('highsec' | 'lowsec' | 'nullsec')[],
  security_min: '',
  security_max: '',
  type_ids: [] as number[],
  service_names: [] as string[],
  service_match_mode: 'and' as const,
  timer_bucket: 'all' as CorporationStructureListRequest['timer_bucket'],
}

function getErrorMessage(error: unknown, fallback: string) {
  return error instanceof Error && error.message ? error.message : fallback
}

function normalizeTab(value: string | null): ActiveTab {
  return value === 'settings' ? 'settings' : 'list'
}

function parseNumberInput(value: string) {
  if (!value.trim()) return ''
  const numeric = Number(value)
  return Number.isFinite(numeric) ? value : ''
}

function formatSecurity(value: number) {
  if (Number.isNaN(value)) return '--'
  return value.toFixed(1)
}

function formatUpdatedAt(value: number) {
  if (!value) return '--'
  return new Date(value * 1000).toLocaleString()
}

function formatTimeText(value: string) {
  if (!value) return '--'
  const parsed = new Date(value)
  return Number.isNaN(parsed.getTime()) ? value : parsed.toLocaleString()
}

function stateLabel(t: ReturnType<typeof useI18n>['t'], state: string) {
  const key = `corporationStructures.states.${state}`
  const translated = t(key)
  return translated === key ? state || '--' : translated
}

function formatServices(t: ReturnType<typeof useI18n>['t'], services: CorporationStructureServiceInfo[]) {
  if (!services.length) return t('corporationStructures.noServices')
  return services.map((service) => `${service.name} (${service.state})`).join(' / ')
}

function formatSystemOption(item: CorporationStructureSystemOption) {
  const regionText = item.region_name ? ` / ${item.region_name}` : ''
  return `${item.system_name}${regionText} (${formatSecurity(item.security)})`
}

function TabButton({
  active,
  children,
  onClick,
}: {
  active: boolean
  children: string
  onClick: () => void
}) {
  return (
    <button
      type="button"
      className={cn(
        'rounded-lg border px-3 py-1.5 text-sm font-medium transition-colors',
        active ? 'border-primary bg-primary text-primary-foreground' : 'border-border bg-background hover:bg-muted'
      )}
      onClick={onClick}
    >
      {children}
    </button>
  )
}

export function DashboardCorporationStructuresPage() {
  const { t } = useI18n()
  const location = useLocation()
  const navigate = useNavigate()
  const [loading, setLoading] = useState(false)
  const [settingsLoading, setSettingsLoading] = useState(false)
  const [savingAuthorizations, setSavingAuthorizations] = useState(false)
  const [runningTaskCorpId, setRunningTaskCorpId] = useState(0)
  const [error, setError] = useState<string | null>(null)
  const [settingsError, setSettingsError] = useState<string | null>(null)
  const [settings, setSettings] = useState<CorporationStructuresSettings>({
    corporations: [],
    fuel_notice_threshold_days: 7,
    timer_notice_threshold_days: 7,
  })
  const [noticeThresholds, setNoticeThresholds] = useState({
    fuel_notice_threshold_days: 7,
    timer_notice_threshold_days: 7,
  })
  const [authorizationByCorp, setAuthorizationByCorp] = useState<Record<number, number>>({})
  const [filterOptions, setFilterOptions] = useState<CorporationStructureFilterOptionsResponse>({
    systems: [],
    types: [],
    services: [],
  })
  const [tableData, setTableData] = useState<CorporationStructureRow[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)
  const [sort, setSort] = useState<{ sort_by?: CorporationStructureListRequest['sort_by']; sort_order?: CorporationStructureListRequest['sort_order'] }>({
    sort_by: 'fuel_remaining_hours',
    sort_order: 'asc',
  })
  const [draftTimerRange, setDraftTimerRange] = useState<DateTimeRange>(null)
  const [appliedTimerRange, setAppliedTimerRange] = useState<DateTimeRange>(null)
  const [filters, setFilters] = useState({
    corporation_id: 0,
    keyword: '',
    state_groups: [] as string[],
    fuel_bucket: 'all' as CorporationStructureListRequest['fuel_bucket'],
    fuel_min_hours: '',
    fuel_max_hours: '',
    system_ids: [] as number[],
    security_bands: [] as ('highsec' | 'lowsec' | 'nullsec')[],
    security_min: '',
    security_max: '',
    type_ids: [] as number[],
    service_names: [] as string[],
    service_match_mode: 'and' as const,
    timer_bucket: 'all' as CorporationStructureListRequest['timer_bucket'],
  })
  const [appliedFilters, setAppliedFilters] = useState(() => ({
    corporation_id: 0,
    keyword: '',
    state_groups: [] as string[],
    fuel_bucket: 'all' as CorporationStructureListRequest['fuel_bucket'],
    fuel_min_hours: '',
    fuel_max_hours: '',
    system_ids: [] as number[],
    security_bands: [] as ('highsec' | 'lowsec' | 'nullsec')[],
    security_min: '',
    security_max: '',
    type_ids: [] as number[],
    service_names: [] as string[],
    service_match_mode: 'and' as const,
    timer_bucket: 'all' as CorporationStructureListRequest['timer_bucket'],
  }))

  const activeTab = normalizeTab(new URLSearchParams(location.search).get('tab'))

  const setTab = (tab: ActiveTab) => {
    const searchParams = new URLSearchParams(location.search)
    if (tab === 'settings') {
      searchParams.set('tab', 'settings')
    } else {
      searchParams.delete('tab')
    }
    navigate({ search: searchParams.toString() ? `?${searchParams.toString()}` : '' }, { replace: true })
  }

  const loadSettings = async () => {
    setSettingsLoading(true)
    setSettingsError(null)
    try {
      const data = await fetchCorporationStructureSettings()
      setSettings(data)
      setNoticeThresholds({
        fuel_notice_threshold_days: data.fuel_notice_threshold_days,
        timer_notice_threshold_days: data.timer_notice_threshold_days,
      })
      const nextAuth: Record<number, number> = {}
      data.corporations.forEach((corp) => {
        nextAuth[corp.corporation_id] = corp.authorized_character_id || 0
      })
      setAuthorizationByCorp(nextAuth)
    } catch (caughtError) {
      setSettingsError(getErrorMessage(caughtError, t('corporationStructures.messages.loadFailed')))
    } finally {
      setSettingsLoading(false)
    }
  }

  const loadFilterOptions = async (corporationId = filters.corporation_id) => {
    const data = await fetchCorporationStructureFilterOptions({
      corporation_id: corporationId > 0 ? corporationId : undefined,
    })
    setFilterOptions(data)
  }

  useEffect(() => {
    let cancelled = false

    const loadInitialData = async () => {
      setSettingsLoading(true)
      setSettingsError(null)

      try {
        const [settingsData, filterData] = await Promise.all([
          fetchCorporationStructureSettings(),
          fetchCorporationStructureFilterOptions(),
        ])

        if (cancelled) {
          return
        }

        setSettings(settingsData)
        setNoticeThresholds({
          fuel_notice_threshold_days: settingsData.fuel_notice_threshold_days,
          timer_notice_threshold_days: settingsData.timer_notice_threshold_days,
        })
        const nextAuth: Record<number, number> = {}
        settingsData.corporations.forEach((corp) => {
          nextAuth[corp.corporation_id] = corp.authorized_character_id || 0
        })
        setAuthorizationByCorp(nextAuth)
        setFilterOptions(filterData)
      } catch (caughtError) {
        if (!cancelled) {
          setSettingsError(getErrorMessage(caughtError, t('corporationStructures.messages.loadFailed')))
        }
      } finally {
        if (!cancelled) {
          setSettingsLoading(false)
        }
      }
    }

    void loadInitialData()

    return () => {
      cancelled = true
    }
  }, [t])

  useEffect(() => {
    let cancelled = false

    const loadData = async () => {
      if (activeTab !== 'list') {
        return
      }

      setLoading(true)
      setError(null)

      try {
        const data = await fetchCorporationStructureList({
          corporation_id: appliedFilters.corporation_id > 0 ? appliedFilters.corporation_id : undefined,
          keyword: appliedFilters.keyword || undefined,
          state_groups: appliedFilters.state_groups.length ? appliedFilters.state_groups : undefined,
          fuel_bucket: appliedFilters.fuel_bucket,
          fuel_min_hours:
            appliedFilters.fuel_bucket === 'custom' && appliedFilters.fuel_min_hours !== ''
              ? Number(appliedFilters.fuel_min_hours)
              : undefined,
          fuel_max_hours:
            appliedFilters.fuel_bucket === 'custom' && appliedFilters.fuel_max_hours !== ''
              ? Number(appliedFilters.fuel_max_hours)
              : undefined,
          system_ids: appliedFilters.system_ids.length ? appliedFilters.system_ids : undefined,
          security_bands: appliedFilters.security_bands.length ? appliedFilters.security_bands : undefined,
          security_min: appliedFilters.security_min !== '' ? Number(appliedFilters.security_min) : undefined,
          security_max: appliedFilters.security_max !== '' ? Number(appliedFilters.security_max) : undefined,
          type_ids: appliedFilters.type_ids.length ? appliedFilters.type_ids : undefined,
          service_names: appliedFilters.service_names.length ? appliedFilters.service_names : undefined,
          service_match_mode: appliedFilters.service_match_mode,
          timer_bucket: appliedFilters.timer_bucket,
          timer_start:
            appliedFilters.timer_bucket === 'custom' && appliedTimerRange ? appliedTimerRange[0] : undefined,
          timer_end:
            appliedFilters.timer_bucket === 'custom' && appliedTimerRange ? appliedTimerRange[1] : undefined,
          sort_by: sort.sort_by,
          sort_order: sort.sort_order,
          page,
          page_size: pageSize,
        })

        if (cancelled) {
          return
        }

        setTableData(data.items)
        setTotal(data.total)
        setPage(data.page)
        setPageSize(data.page_size)
      } catch (caughtError) {
        if (!cancelled) {
          setError(getErrorMessage(caughtError, t('corporationStructures.empty.list')))
          setTableData([])
          setTotal(0)
        }
      } finally {
        if (!cancelled) {
          setLoading(false)
        }
      }
    }

    void loadData()

    return () => {
      cancelled = true
    }
  }, [activeTab, appliedFilters, appliedTimerRange, page, pageSize, sort, t])

  const applySearch = () => {
    setAppliedFilters({
      ...filters,
    })
    setAppliedTimerRange(filters.timer_bucket === 'custom' ? draftTimerRange : null)
    setPage(1)
  }

  const resetSearch = () => {
    const resetFilters = {
      ...DEFAULT_FILTERS,
      fuel_min_hours: '',
      fuel_max_hours: '',
      security_min: '',
      security_max: '',
    }
    setFilters(resetFilters)
    setAppliedFilters(resetFilters)
    setDraftTimerRange(null)
    setAppliedTimerRange(null)
    setSort({
      sort_by: 'fuel_remaining_hours',
      sort_order: 'asc',
    })
    setPage(1)
    void loadFilterOptions(0)
  }

  const saveAuthorizations = async () => {
    setSavingAuthorizations(true)
    try {
      await updateCorporationStructureAuthorizations({
        authorizations: settings.corporations.map((corp) => ({
          corporation_id: corp.corporation_id,
          character_id: authorizationByCorp[corp.corporation_id] || 0,
        })),
        fuel_notice_threshold_days: noticeThresholds.fuel_notice_threshold_days,
        timer_notice_threshold_days: noticeThresholds.timer_notice_threshold_days,
      })
      await loadSettings()
    } finally {
      setSavingAuthorizations(false)
    }
  }

  const handleRunTask = async (corporationId: number) => {
    setRunningTaskCorpId(corporationId)
    try {
      await runCorporationStructuresTask({ corporation_id: corporationId })
    } finally {
      setRunningTaskCorpId(0)
    }
  }

  const updateArrayFilter = <T,>(key: keyof typeof filters, nextValue: T[]) => {
    setFilters((current) => ({ ...current, [key]: nextValue }))
  }

  const loadFilterOptionsForSelectedCorp = async (corpId: number) => {
    await loadFilterOptions(corpId)
  }

  return (
    <section className="space-y-4">
      <div className="rounded-lg border bg-card p-5">
        <div className="space-y-1">
          <h1 className="text-xl font-semibold">{t('corporationStructures.title')}</h1>
          <p className="text-sm text-muted-foreground">{t('corporationStructures.subtitle')}</p>
        </div>
      </div>

      <div className="flex flex-wrap gap-2">
        <TabButton active={activeTab === 'list'} onClick={() => setTab('list')}>
          {t('corporationStructures.tabs.list')}
        </TabButton>
        <TabButton active={activeTab === 'settings'} onClick={() => setTab('settings')}>
          {t('corporationStructures.tabs.settings')}
        </TabButton>
      </div>

      {activeTab === 'list' ? (
        <>
          <div className="rounded-lg border bg-card p-5">
            <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
              <label className="space-y-1">
                <span className="text-sm text-muted-foreground">{t('corporationStructures.filters.corporation')}</span>
                <select
                  className="h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
                  value={filters.corporation_id}
                  onChange={async (event) => {
                    const corpId = Number(event.target.value)
                    setFilters((current) => ({ ...current, corporation_id: corpId }))
                    await loadFilterOptionsForSelectedCorp(corpId)
                  }}
                >
                  <option value={0}>{t('corporationStructures.allCorporations')}</option>
                  {settings.corporations.map((corp) => (
                    <option key={corp.corporation_id} value={corp.corporation_id}>
                      {corp.corporation_name} ({corp.corporation_id})
                    </option>
                  ))}
                </select>
              </label>

              <label className="space-y-1">
                <span className="text-sm text-muted-foreground">{t('corporationStructures.filters.keyword')}</span>
                <Input
                  value={filters.keyword}
                  onChange={(event) => setFilters((current) => ({ ...current, keyword: event.target.value }))}
                  placeholder={t('corporationStructures.placeholders.keyword')}
                />
              </label>

              <label className="space-y-1">
                <span className="text-sm text-muted-foreground">{t('corporationStructures.filters.systems')}</span>
                <select
                  multiple
                  className="min-h-24 w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
                  value={filters.system_ids.map(String)}
                  onChange={(event) =>
                    updateArrayFilter(
                      'system_ids',
                      Array.from(event.target.selectedOptions, (option) => Number(option.value))
                    )
                  }
                >
                  {filterOptions.systems.map((item) => (
                    <option key={item.system_id} value={item.system_id}>
                      {formatSystemOption(item)}
                    </option>
                  ))}
                </select>
              </label>

              <label className="space-y-1 md:col-span-2">
                <span className="text-sm text-muted-foreground">{t('corporationStructures.filters.stateGroups')}</span>
                <div className="flex flex-wrap gap-2">
                  {[
                    ['online', t('corporationStructures.stateGroups.online')],
                    ['low_power', t('corporationStructures.stateGroups.lowPower')],
                    ['abandoned', t('corporationStructures.stateGroups.abandoned')],
                    ['reinforced', t('corporationStructures.stateGroups.reinforced')],
                  ].map(([value, label]) => (
                    <button
                      key={value}
                      type="button"
                      className={cn(
                        'rounded-lg border px-3 py-1.5 text-sm',
                        filters.state_groups.includes(value)
                          ? 'border-primary bg-primary text-primary-foreground'
                          : 'border-border bg-background'
                      )}
                      onClick={() => {
                        const next = filters.state_groups.includes(value)
                          ? filters.state_groups.filter((item) => item !== value)
                          : [...filters.state_groups, value]
                        setFilters((current) => ({ ...current, state_groups: next }))
                      }}
                    >
                      {label}
                    </button>
                  ))}
                </div>
              </label>

              <label className="space-y-1 md:col-span-2 xl:col-span-3">
                <span className="text-sm text-muted-foreground">{t('corporationStructures.filters.fuel')}</span>
                <div className="flex flex-wrap items-center gap-2">
                  {[
                    ['all', t('corporationStructures.fuelBuckets.all')],
                    ['lt_24h', t('corporationStructures.fuelBuckets.lt24h')],
                    ['lt_72h', t('corporationStructures.fuelBuckets.lt3d')],
                    ['lt_168h', t('corporationStructures.fuelBuckets.lt7d')],
                    ['custom', t('corporationStructures.fuelBuckets.custom')],
                  ].map(([value, label]) => (
                    <button
                      key={value}
                      type="button"
                      className={cn(
                        'rounded-lg border px-3 py-1.5 text-sm',
                        filters.fuel_bucket === value
                          ? 'border-primary bg-primary text-primary-foreground'
                          : 'border-border bg-background'
                      )}
                      onClick={() => setFilters((current) => ({ ...current, fuel_bucket: value as typeof filters.fuel_bucket }))}
                    >
                      {label}
                    </button>
                  ))}
                  {filters.fuel_bucket === 'custom' ? (
                    <>
                      <Input
                        className="w-28"
                        inputMode="numeric"
                        value={filters.fuel_min_hours}
                        onChange={(event) =>
                          setFilters((current) => ({
                            ...current,
                            fuel_min_hours: parseNumberInput(event.target.value),
                          }))
                        }
                        placeholder="min"
                      />
                      <span>~</span>
                      <Input
                        className="w-28"
                        inputMode="numeric"
                        value={filters.fuel_max_hours}
                        onChange={(event) =>
                          setFilters((current) => ({
                            ...current,
                            fuel_max_hours: parseNumberInput(event.target.value),
                          }))
                        }
                        placeholder="max"
                      />
                    </>
                  ) : null}
                </div>
              </label>

              <label className="space-y-1 md:col-span-2">
                <span className="text-sm text-muted-foreground">{t('corporationStructures.filters.security')}</span>
                <div className="flex flex-wrap items-center gap-2">
                  {[
                    ['highsec', t('corporationStructures.securityBands.highsec')],
                    ['lowsec', t('corporationStructures.securityBands.lowsec')],
                    ['nullsec', t('corporationStructures.securityBands.nullsec')],
                  ].map(([value, label]) => (
                    <button
                      key={value}
                      type="button"
                      className={cn(
                        'rounded-lg border px-3 py-1.5 text-sm',
                        filters.security_bands.includes(value as 'highsec' | 'lowsec' | 'nullsec')
                          ? 'border-primary bg-primary text-primary-foreground'
                          : 'border-border bg-background'
                      )}
                      onClick={() => {
                        const band = value as 'highsec' | 'lowsec' | 'nullsec'
                        const next = filters.security_bands.includes(band)
                          ? filters.security_bands.filter((item) => item !== band)
                          : [...filters.security_bands, band]
                        setFilters((current) => ({ ...current, security_bands: next }))
                      }}
                    >
                      {label}
                    </button>
                  ))}
                  <Input
                    className="w-24"
                    inputMode="decimal"
                    value={filters.security_min}
                    onChange={(event) =>
                      setFilters((current) => ({ ...current, security_min: parseNumberInput(event.target.value) }))
                    }
                    placeholder="-1.0"
                  />
                  <span>~</span>
                  <Input
                    className="w-24"
                    inputMode="decimal"
                    value={filters.security_max}
                    onChange={(event) =>
                      setFilters((current) => ({ ...current, security_max: parseNumberInput(event.target.value) }))
                    }
                    placeholder="1.0"
                  />
                </div>
              </label>

              <label className="space-y-1">
                <span className="text-sm text-muted-foreground">{t('corporationStructures.filters.types')}</span>
                <select
                  multiple
                  className="min-h-24 w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
                  value={filters.type_ids.map(String)}
                  onChange={(event) =>
                    updateArrayFilter(
                      'type_ids',
                      Array.from(event.target.selectedOptions, (option) => Number(option.value))
                    )
                  }
                >
                  {filterOptions.types.map((item) => (
                    <option key={item.type_id} value={item.type_id}>
                      {item.type_name}
                    </option>
                  ))}
                </select>
              </label>

              <label className="space-y-1 md:col-span-2">
                <span className="text-sm text-muted-foreground">{t('corporationStructures.filters.services')}</span>
                <div className="flex flex-wrap items-center gap-2">
                  <select
                    multiple
                    className="min-h-24 w-full rounded-md border border-input bg-background px-3 py-2 text-sm md:w-[360px]"
                    value={filters.service_names}
                    onChange={(event) => {
                      const next = Array.from(event.target.selectedOptions, (option) => option.value)
                      setFilters((current) => ({ ...current, service_names: next }))
                    }}
                  >
                    {filterOptions.services.map((item) => (
                      <option key={item.name} value={item.name}>
                        {item.name}
                      </option>
                    ))}
                  </select>
                  <div className="flex flex-wrap gap-2">
                    {[
                      ['and', t('corporationStructures.serviceMatch.and')],
                      ['or', t('corporationStructures.serviceMatch.or')],
                    ].map(([value, label]) => (
                      <button
                        key={value}
                        type="button"
                        className={cn(
                          'rounded-lg border px-3 py-1.5 text-sm',
                          filters.service_match_mode === value
                            ? 'border-primary bg-primary text-primary-foreground'
                            : 'border-border bg-background'
                        )}
                        onClick={() =>
                          setFilters((current) => ({
                            ...current,
                            service_match_mode: value as typeof filters.service_match_mode,
                          }))
                        }
                      >
                        {label}
                      </button>
                    ))}
                  </div>
                </div>
              </label>

              <label className="space-y-1 md:col-span-2 xl:col-span-3">
                <span className="text-sm text-muted-foreground">{t('corporationStructures.filters.timer')}</span>
                <div className="flex flex-wrap items-center gap-2">
                  {[
                    ['all', t('corporationStructures.timerBuckets.all')],
                    ['current_hour', t('corporationStructures.timerBuckets.currentHour')],
                    ['next_2_hours', t('corporationStructures.timerBuckets.next2Hours')],
                    ['custom', t('corporationStructures.timerBuckets.custom')],
                  ].map(([value, label]) => (
                    <button
                      key={value}
                      type="button"
                      className={cn(
                        'rounded-lg border px-3 py-1.5 text-sm',
                        filters.timer_bucket === value
                          ? 'border-primary bg-primary text-primary-foreground'
                          : 'border-border bg-background'
                      )}
                      onClick={() =>
                        setFilters((current) => ({ ...current, timer_bucket: value as typeof filters.timer_bucket }))
                      }
                    >
                      {label}
                    </button>
                  ))}
                  {filters.timer_bucket === 'custom' ? (
                    <>
                      <Input
                        type="datetime-local"
                        className="w-[220px]"
                        value={draftTimerRange?.[0] ?? ''}
                        onChange={(event) =>
                          setDraftTimerRange((current) => [event.target.value, current?.[1] ?? ''])
                        }
                      />
                      <Input
                        type="datetime-local"
                        className="w-[220px]"
                        value={draftTimerRange?.[1] ?? ''}
                        onChange={(event) =>
                          setDraftTimerRange((current) => [current?.[0] ?? '', event.target.value])
                        }
                      />
                    </>
                  ) : null}
                </div>
              </label>
            </div>

            <div className="mt-4 flex flex-wrap items-center gap-3">
              <Button type="button" onClick={applySearch} disabled={loading}>
                {t('corporationStructures.actions.search')}
              </Button>
              <Button type="button" variant="outline" onClick={resetSearch}>
                {t('corporationStructures.actions.reset')}
              </Button>
              <Button
                type="button"
                variant="outline"
                disabled={filters.corporation_id <= 0 || runningTaskCorpId === filters.corporation_id}
                onClick={() => void handleRunTask(filters.corporation_id)}
              >
                {t('corporationStructures.actions.refreshSelected')}
              </Button>
            </div>
          </div>

          {error ? <p className="text-sm text-destructive">{error}</p> : null}
          {loading ? <p className="text-sm text-muted-foreground">{t('common.refresh')}</p> : null}

          <div className="overflow-hidden rounded-lg border bg-card">
            <div className="border-b px-4 py-3 text-sm font-medium">
              {t('corporationStructures.tabs.list')} ({total})
            </div>
            <div className="overflow-x-auto">
              <table className="min-w-full text-sm">
                <thead>
                  <tr className="border-b bg-muted/40 text-left">
                    <th className="px-3 py-2">{t('corporationStructures.table.corporation')}</th>
                    <th className="px-3 py-2">{t('corporationStructures.table.state')}</th>
                    <th className="px-3 py-2">{t('corporationStructures.table.system')}</th>
                    <th className="px-3 py-2">{t('corporationStructures.table.name')}</th>
                    <th className="px-3 py-2">{t('corporationStructures.table.type')}</th>
                    <th className="px-3 py-2">{t('corporationStructures.table.services')}</th>
                    <th className="px-3 py-2">{t('corporationStructures.table.fuelRemaining')}</th>
                    <th className="px-3 py-2">{t('corporationStructures.table.reinforceHour')}</th>
                    <th className="px-3 py-2">{t('corporationStructures.table.timerEnd')}</th>
                    <th className="px-3 py-2">{t('corporationStructures.table.updatedAt')}</th>
                  </tr>
                </thead>
                <tbody>
                  {tableData.map((row) => (
                    <tr key={row.structure_id} className="border-b">
                      <td className="px-3 py-2">{row.corporation_name}</td>
                      <td className="px-3 py-2">{stateLabel(t, row.state)}</td>
                      <td className="px-3 py-2">
                        <div>{row.system_name || '--'}</div>
                        <div className="text-xs text-muted-foreground">
                          {row.region_name || '--'} / {formatSecurity(row.security)}
                        </div>
                      </td>
                      <td className="px-3 py-2">{row.name}</td>
                      <td className="px-3 py-2">{row.type_name}</td>
                      <td className="px-3 py-2">{formatServices(t, row.services)}</td>
                      <td className="px-3 py-2">{row.fuel_remaining || '--'}</td>
                      <td className="px-3 py-2">{row.reinforce_hour > 0 ? String(row.reinforce_hour).padStart(2, '0') : '--'}</td>
                      <td className="px-3 py-2">{formatTimeText(row.state_timer_end)}</td>
                      <td className="px-3 py-2">{formatUpdatedAt(row.updated_at)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>

          <div className="flex flex-wrap items-center gap-3 text-sm">
            <span>{page}</span>
            <Button type="button" variant="outline" size="sm" onClick={() => setPage((current) => Math.max(1, current - 1))} disabled={page <= 1}>
              -
            </Button>
            <Button type="button" variant="outline" size="sm" onClick={() => setPage((current) => current + 1)} disabled={tableData.length < pageSize || page * pageSize >= total}>
              +
            </Button>
            <label className="flex items-center gap-2">
              <span>{t('common.refresh')}</span>
              <select
                className="h-8 rounded-md border border-input bg-background px-2 text-sm"
                value={pageSize}
                onChange={(event) => setPageSize(Number(event.target.value))}
              >
                {[10, 20, 50].map((size) => (
                  <option key={size} value={size}>
                    {size}
                  </option>
                ))}
              </select>
            </label>
          </div>
        </>
      ) : null}

      {activeTab === 'settings' ? (
        <div className="space-y-4 rounded-lg border bg-card p-5">
          <div>
            <h2 className="text-lg font-semibold">{t('corporationStructures.settings.noticeThresholds')}</h2>
            <p className="text-sm text-muted-foreground">{t('corporationStructures.settings.noticeThresholdHint')}</p>
          </div>

          {settingsError ? <p className="text-sm text-destructive">{settingsError}</p> : null}
          {settingsLoading ? <p className="text-sm text-muted-foreground">{t('common.refresh')}</p> : null}

          <div className="space-y-4">
            <div className="flex flex-wrap items-center gap-3">
              <span className="text-sm text-muted-foreground">{t('corporationStructures.settings.fuelNoticeThreshold')}</span>
              <Input
                className="w-24"
                inputMode="numeric"
                value={noticeThresholds.fuel_notice_threshold_days}
                onChange={(event) =>
                  setNoticeThresholds((current) => ({
                    ...current,
                    fuel_notice_threshold_days: Number(event.target.value || 0),
                  }))
                }
              />
              <span className="text-sm text-muted-foreground">{t('corporationStructures.settings.daysUnit')}</span>
            </div>

            <div className="flex flex-wrap items-center gap-3">
              <span className="text-sm text-muted-foreground">{t('corporationStructures.settings.timerNoticeThreshold')}</span>
              <Input
                className="w-24"
                inputMode="numeric"
                value={noticeThresholds.timer_notice_threshold_days}
                onChange={(event) =>
                  setNoticeThresholds((current) => ({
                    ...current,
                    timer_notice_threshold_days: Number(event.target.value || 0),
                  }))
                }
              />
              <span className="text-sm text-muted-foreground">{t('corporationStructures.settings.daysUnit')}</span>
            </div>
          </div>

          <div className="flex flex-wrap items-center gap-3">
            <Button type="button" variant="outline" onClick={() => void loadSettings()} disabled={settingsLoading}>
              {t('corporationStructures.actions.refreshSettings')}
            </Button>
            <Button type="button" onClick={() => void saveAuthorizations()} disabled={savingAuthorizations}>
              {t('corporationStructures.actions.save')}
            </Button>
          </div>

          <div className="overflow-hidden rounded-lg border">
            <table className="min-w-full text-sm">
              <thead>
                <tr className="border-b bg-muted/40 text-left">
                  <th className="px-3 py-2">{t('corporationStructures.table.corporation')}</th>
                  <th className="px-3 py-2">{t('corporationStructures.table.directorCharacter')}</th>
                </tr>
              </thead>
              <tbody>
                {settings.corporations.map((corp) => (
                  <tr key={corp.corporation_id} className="border-b">
                    <td className="px-3 py-2">
                      <div className="font-medium">{corp.corporation_name}</div>
                      <div className="text-xs text-muted-foreground">{corp.corporation_id}</div>
                    </td>
                    <td className="px-3 py-2">
                      <select
                        className="h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
                        value={authorizationByCorp[corp.corporation_id] || 0}
                        onChange={(event) =>
                          setAuthorizationByCorp((current) => ({
                            ...current,
                            [corp.corporation_id]: Number(event.target.value),
                          }))
                        }
                      >
                        <option value={0}>{t('corporationStructures.options.disabled')}</option>
                        {corp.director_characters.map((option) => (
                          <option key={option.character_id} value={option.character_id}>
                            {option.character_name} ({option.character_id})
                          </option>
                        ))}
                      </select>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          {!settingsLoading && settings.corporations.length === 0 ? (
            <p className="text-sm text-muted-foreground">{t('corporationStructures.empty.settings')}</p>
          ) : null}
        </div>
      ) : null}
    </section>
  )
}
