import { Textarea } from '@/components/ui/textarea'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { MultiSelect } from '@/components/ui/multi-select'
import {
  Table,
  TableHeader,
  TableBody,
  TableRow,
  TableHead,
  TableCell,
} from '@/components/ui/table'
import { useEffect, useState } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import {
  fetchCorporationStructureFilterOptions,
  fetchCorporationStructureList,
  fetchCorporationStructureSettings,
  fetchStructureServiceCatalog,
  runCorporationStructuresTask,
  updateCorporationStructureAuthorizations,
  updateStructureServiceCatalog,
} from '@/api/corporation-structures'
import { runTask } from '@/api/task-manager'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { useCorpCapability } from '@/hooks/use-corp-capability'
import { useI18n } from '@/i18n'
import type {
  CorporationStructureFilterOptionsResponse,
  CorporationStructureListRequest,
  CorporationStructureRow,
  CorporationStructureServiceInfo,
  CorporationStructuresSettings,
  CorporationStructureSystemOption,
  StructureServiceCatalog,
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
  region_ids: [] as number[],
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

function formatServices(
  t: ReturnType<typeof useI18n>['t'],
  services: CorporationStructureServiceInfo[]
) {
  if (!services.length) return t('corporationStructures.noServices')
  return services.map((service) => `${service.name} (${service.state})`).join(' / ')
}

function formatFuelEstimate(
  t: ReturnType<typeof useI18n>['t'],
  row: CorporationStructureRow,
  field: 'fuel_per_hour' | 'fuel_to_month_end'
) {
  if (row.fuel_estimate_incomplete) {
    const keyByStatus: Record<string, string> = {
      authorization_required: 'fuelEstimateAuthorizationRequired',
      activity_mapping_required: 'fuelEstimateActivityMappingRequired',
      module_mismatch: 'fuelEstimateModuleMismatch',
      rate_unavailable: 'fuelEstimateRateUnavailable',
      ambiguous_module: 'fuelEstimateAmbiguousModule',
    }
    return t(
      `corporationStructures.table.${keyByStatus[row.fuel_estimate_status || ''] || 'fuelEstimateIncomplete'}`
    )
  }
  return row[field] ?? '--'
}

function formatSystemOption(item: CorporationStructureSystemOption) {
  const regionText = item.region_name ? ` / ${item.region_name}` : ''
  return `${item.system_name}${regionText} (${formatSecurity(item.security)})`
}

function normalizeServiceCatalog(
  catalog: Partial<StructureServiceCatalog> | null | undefined
): StructureServiceCatalog {
  return {
    modules: Array.isArray(catalog?.modules) ? catalog.modules : [],
    activities: Array.isArray(catalog?.activities) ? catalog.activities : [],
    unmapped_activities: Array.isArray(catalog?.unmapped_activities)
      ? catalog.unmapped_activities
      : [],
  }
}

export function DashboardCorporationStructuresPage() {
  const { t } = useI18n()
  const { hasCapability } = useCorpCapability()
  const location = useLocation()
  const navigate = useNavigate()
  const [loading, setLoading] = useState(false)
  const [settingsLoading, setSettingsLoading] = useState(false)
  const [savingAuthorizations, setSavingAuthorizations] = useState(false)
  const [savingServiceCatalog, setSavingServiceCatalog] = useState(false)
  const [alertScanRunning, setAlertScanRunning] = useState(false)
  const [runningTaskCorpId, setRunningTaskCorpId] = useState(0)
  const [error, setError] = useState<string | null>(null)
  const [settingsError, setSettingsError] = useState<string | null>(null)
  const [alertScanMessage, setAlertScanMessage] = useState<string | null>(null)
  const [settings, setSettings] = useState<CorporationStructuresSettings>({
    corporations: [],
    fuel_notice_threshold_days: 7,
    timer_notice_threshold_days: 7,
    alert_enabled: false,
    alert_group_ids: [],
  })
  const [noticeThresholds, setNoticeThresholds] = useState({
    fuel_notice_threshold_days: 7,
    timer_notice_threshold_days: 7,
  })
  const [alertEnabled, setAlertEnabled] = useState(false)
  const [alertGroupIDsText, setAlertGroupIDsText] = useState('')
  const [authorizationByCorp, setAuthorizationByCorp] = useState<Record<number, number>>({})
  const [serviceCatalog, setServiceCatalog] = useState<StructureServiceCatalog>({
    modules: [],
    activities: [],
    unmapped_activities: [],
  })
  const [activityModules, setActivityModules] = useState<Record<string, number[]>>({})
  const [filterOptions, setFilterOptions] = useState<CorporationStructureFilterOptionsResponse>({
    systems: [],
    regions: [],
    types: [],
    services: [],
  })
  const [tableData, setTableData] = useState<CorporationStructureRow[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)
  const [sort, setSort] = useState<{
    sort_by?: CorporationStructureListRequest['sort_by']
    sort_order?: CorporationStructureListRequest['sort_order']
  }>({
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
    region_ids: [] as number[],
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
    region_ids: [] as number[],
    security_bands: [] as ('highsec' | 'lowsec' | 'nullsec')[],
    security_min: '',
    security_max: '',
    type_ids: [] as number[],
    service_names: [] as string[],
    service_match_mode: 'and' as const,
    timer_bucket: 'all' as CorporationStructureListRequest['timer_bucket'],
  }))

  const activeTab = normalizeTab(new URLSearchParams(location.search).get('tab'))
  const canRunAlertScan = hasCapability('system.task.run')

  const setTab = (tab: ActiveTab) => {
    const searchParams = new URLSearchParams(location.search)
    if (tab === 'settings') {
      searchParams.set('tab', 'settings')
    } else {
      searchParams.delete('tab')
    }
    navigate(
      { search: searchParams.toString() ? `?${searchParams.toString()}` : '' },
      { replace: true }
    )
  }

  const loadSettings = async () => {
    setSettingsLoading(true)
    setSettingsError(null)
    try {
      const [data, catalog] = await Promise.all([
        fetchCorporationStructureSettings(),
        fetchStructureServiceCatalog(),
      ])
      setSettings(data)
      setNoticeThresholds({
        fuel_notice_threshold_days: data.fuel_notice_threshold_days,
        timer_notice_threshold_days: data.timer_notice_threshold_days,
      })
      setAlertEnabled(data.alert_enabled)
      setAlertGroupIDsText(data.alert_group_ids.join('\n'))
      const nextAuth: Record<number, number> = {}
      data.corporations.forEach((corp) => {
        nextAuth[corp.corporation_id] = corp.authorized_character_id || 0
      })
      setAuthorizationByCorp(nextAuth)
      const normalizedCatalog = normalizeServiceCatalog(catalog)
      setServiceCatalog(normalizedCatalog)
      setActivityModules((current) => {
        const next = { ...current }
        normalizedCatalog.unmapped_activities.forEach((item) => {
          next[item.activity_name] ||= []
        })
        return next
      })
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
        const [settingsData, filterData, catalogData] = await Promise.all([
          fetchCorporationStructureSettings(),
          fetchCorporationStructureFilterOptions(),
          fetchStructureServiceCatalog(),
        ])

        if (cancelled) {
          return
        }

        setSettings(settingsData)
        setNoticeThresholds({
          fuel_notice_threshold_days: settingsData.fuel_notice_threshold_days,
          timer_notice_threshold_days: settingsData.timer_notice_threshold_days,
        })
        setAlertEnabled(settingsData.alert_enabled)
        setAlertGroupIDsText(settingsData.alert_group_ids.join('\n'))
        const nextAuth: Record<number, number> = {}
        settingsData.corporations.forEach((corp) => {
          nextAuth[corp.corporation_id] = corp.authorized_character_id || 0
        })
        setAuthorizationByCorp(nextAuth)
        setFilterOptions(filterData)
        const normalizedCatalog = normalizeServiceCatalog(catalogData)
        setServiceCatalog(normalizedCatalog)
        setActivityModules((current) => {
          const next = { ...current }
          normalizedCatalog.unmapped_activities.forEach((item) => {
            next[item.activity_name] ||= []
          })
          return next
        })
      } catch (caughtError) {
        if (!cancelled) {
          setSettingsError(
            getErrorMessage(caughtError, t('corporationStructures.messages.loadFailed'))
          )
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
          corporation_id:
            appliedFilters.corporation_id > 0 ? appliedFilters.corporation_id : undefined,
          keyword: appliedFilters.keyword || undefined,
          state_groups: appliedFilters.state_groups.length
            ? appliedFilters.state_groups
            : undefined,
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
          region_ids: appliedFilters.region_ids.length ? appliedFilters.region_ids : undefined,
          security_bands: appliedFilters.security_bands.length
            ? appliedFilters.security_bands
            : undefined,
          security_min:
            appliedFilters.security_min !== '' ? Number(appliedFilters.security_min) : undefined,
          security_max:
            appliedFilters.security_max !== '' ? Number(appliedFilters.security_max) : undefined,
          type_ids: appliedFilters.type_ids.length ? appliedFilters.type_ids : undefined,
          service_names: appliedFilters.service_names.length
            ? appliedFilters.service_names
            : undefined,
          service_match_mode: appliedFilters.service_match_mode,
          timer_bucket: appliedFilters.timer_bucket,
          timer_start:
            appliedFilters.timer_bucket === 'custom' && appliedTimerRange
              ? appliedTimerRange[0]
              : undefined,
          timer_end:
            appliedFilters.timer_bucket === 'custom' && appliedTimerRange
              ? appliedTimerRange[1]
              : undefined,
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
        alert_enabled: alertEnabled,
        alert_group_ids: parseAlertGroupIDs(alertGroupIDsText),
      })
      await loadSettings()
    } finally {
      setSavingAuthorizations(false)
    }
  }

  const saveServiceActivityMappings = async () => {
    const activities = Array.from(
      new Set(serviceCatalog.unmapped_activities.map((item) => item.activity_name))
    )
      .filter((name) => (activityModules[name] || []).length > 0)
      .map((activity_name) => ({ activity_name, type_ids: activityModules[activity_name] }))
    if (!activities.length) return

    setSavingServiceCatalog(true)
    try {
      await updateStructureServiceCatalog({ modules: [], activities })
      await loadSettings()
    } finally {
      setSavingServiceCatalog(false)
    }
  }

  const handleRunAlertScan = async () => {
    if (!window.confirm(t('corporationStructures.confirm.runAlertScan'))) {
      return
    }

    setAlertScanRunning(true)
    setSettingsError(null)
    setAlertScanMessage(null)
    try {
      await runTask('corporation_structure_alert_scan')
      setAlertScanMessage(t('corporationStructures.messages.alertScanTriggered'))
    } catch (caughtError) {
      setSettingsError(
        getErrorMessage(caughtError, t('corporationStructures.messages.alertScanTriggerFailed'))
      )
    } finally {
      setAlertScanRunning(false)
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

      <Tabs selectedKey={activeTab} onSelectionChange={(key) => ((value) => setTab(value as ActiveTab))(String(key))}>
        <TabsList>
          <TabsTrigger id="list">{t('corporationStructures.tabs.list')}</TabsTrigger>
          <TabsTrigger id="settings">{t('corporationStructures.tabs.settings')}</TabsTrigger>
        </TabsList>
      </Tabs>

      {activeTab === 'list' ? (
        <>
          <div className="rounded-lg border bg-card p-5">
            <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
              <label className="space-y-1">
                <span className="text-sm text-muted-foreground">
                  {t('corporationStructures.filters.corporation')}
                </span>
                <Select
                  selectedKey={String(filters.corporation_id ?? '')}
                  onSelectionChange={(key) => (async (value) => {
                    const corpId = Number(value)
                    setFilters((current) => ({ ...current, corporation_id: corpId }))
                    await loadFilterOptionsForSelectedCorp(corpId)
                  })(String(key))}
                >
                  <SelectTrigger className="h-10 w-full rounded-md border border-input bg-background px-3 text-sm">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem id={String(0)}>
                      {t('corporationStructures.allCorporations')}
                    </SelectItem>
                    {settings.corporations.map((corp) => (
                      <SelectItem
                        key={corp.corporation_id}
                        id={String(corp.corporation_id ?? '')}
                      >
                        {corp.corporation_name} ({corp.corporation_id})
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </label>

              <label className="space-y-1">
                <span className="text-sm text-muted-foreground">
                  {t('corporationStructures.filters.keyword')}
                </span>
                <Input
                  value={filters.keyword}
                  onChange={(event) =>
                    setFilters((current) => ({ ...current, keyword: event.target.value }))
                  }
                  placeholder={t('corporationStructures.placeholders.keyword')}
                />
              </label>

              <label className="space-y-1">
                <span className="text-sm text-muted-foreground">
                  {t('corporationStructures.filters.regions')}
                </span>
                <MultiSelect
                  className="w-full"
                  placeholder={t('corporationStructures.filters.regions')}
                  value={filters.region_ids.map(String)}
                  onValueChange={(value) => updateArrayFilter('region_ids', value.map(Number))}
                  options={filterOptions.regions.map((item) => ({
                    value: String(item.region_id),
                    label: item.region_name,
                  }))}
                />
              </label>

              <label className="space-y-1">
                <span className="text-sm text-muted-foreground">
                  {t('corporationStructures.filters.systems')}
                </span>
                <MultiSelect
                  className="w-full"
                  placeholder={t('corporationStructures.filters.systems')}
                  value={filters.system_ids.map(String)}
                  onValueChange={(value) => updateArrayFilter('system_ids', value.map(Number))}
                  options={filterOptions.systems.map((item) => ({
                    value: String(item.system_id),
                    label: formatSystemOption(item),
                  }))}
                />
              </label>

              <label className="space-y-1 md:col-span-2">
                <span className="text-sm text-muted-foreground">
                  {t('corporationStructures.filters.stateGroups')}
                </span>
                <div className="flex flex-wrap gap-2">
                  {[
                    ['online', t('corporationStructures.stateGroups.online')],
                    ['low_power', t('corporationStructures.stateGroups.lowPower')],
                    ['abandoned', t('corporationStructures.stateGroups.abandoned')],
                    ['reinforced', t('corporationStructures.stateGroups.reinforced')],
                  ].map(([value, label]) => (
                    <Button
                      key={value}
                      type="button"
                      variant={filters.state_groups.includes(value) ? 'default' : 'outline'}
                      className="px-3 py-1.5"
                      aria-pressed={filters.state_groups.includes(value)}
                      onClick={() => {
                        const next = filters.state_groups.includes(value)
                          ? filters.state_groups.filter((item) => item !== value)
                          : [...filters.state_groups, value]
                        setFilters((current) => ({ ...current, state_groups: next }))
                      }}
                    >
                      {label}
                    </Button>
                  ))}
                </div>
              </label>

              <label className="space-y-1 md:col-span-2 xl:col-span-3">
                <span className="text-sm text-muted-foreground">
                  {t('corporationStructures.filters.fuel')}
                </span>
                <div className="flex flex-wrap items-center gap-2">
                  {[
                    ['all', t('corporationStructures.fuelBuckets.all')],
                    ['lt_24h', t('corporationStructures.fuelBuckets.lt24h')],
                    ['lt_72h', t('corporationStructures.fuelBuckets.lt3d')],
                    ['lt_168h', t('corporationStructures.fuelBuckets.lt7d')],
                    ['custom', t('corporationStructures.fuelBuckets.custom')],
                  ].map(([value, label]) => (
                    <Button
                      key={value}
                      type="button"
                      variant={filters.fuel_bucket === value ? 'default' : 'outline'}
                      className="px-3 py-1.5"
                      aria-pressed={filters.fuel_bucket === value}
                      onClick={() =>
                        setFilters((current) => ({
                          ...current,
                          fuel_bucket: value as typeof filters.fuel_bucket,
                        }))
                      }
                    >
                      {label}
                    </Button>
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
                        placeholder={t('common.min')}
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
                        placeholder={t('common.max')}
                      />
                    </>
                  ) : null}
                </div>
              </label>

              <label className="space-y-1 md:col-span-2">
                <span className="text-sm text-muted-foreground">
                  {t('corporationStructures.filters.security')}
                </span>
                <div className="flex flex-wrap items-center gap-2">
                  {[
                    ['highsec', t('corporationStructures.securityBands.highsec')],
                    ['lowsec', t('corporationStructures.securityBands.lowsec')],
                    ['nullsec', t('corporationStructures.securityBands.nullsec')],
                  ].map(([value, label]) => (
                    <Button
                      key={value}
                      type="button"
                      variant={
                        filters.security_bands.includes(value as 'highsec' | 'lowsec' | 'nullsec')
                          ? 'default'
                          : 'outline'
                      }
                      className="px-3 py-1.5"
                      aria-pressed={filters.security_bands.includes(
                        value as 'highsec' | 'lowsec' | 'nullsec'
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
                    </Button>
                  ))}
                  <Input
                    className="w-24"
                    inputMode="decimal"
                    value={filters.security_min}
                    onChange={(event) =>
                      setFilters((current) => ({
                        ...current,
                        security_min: parseNumberInput(event.target.value),
                      }))
                    }
                    placeholder="-1.0"
                  />
                  <span>~</span>
                  <Input
                    className="w-24"
                    inputMode="decimal"
                    value={filters.security_max}
                    onChange={(event) =>
                      setFilters((current) => ({
                        ...current,
                        security_max: parseNumberInput(event.target.value),
                      }))
                    }
                    placeholder="1.0"
                  />
                </div>
              </label>

              <label className="space-y-1">
                <span className="text-sm text-muted-foreground">
                  {t('corporationStructures.filters.types')}
                </span>
                <MultiSelect
                  className="w-full"
                  placeholder={t('corporationStructures.filters.types')}
                  value={filters.type_ids.map(String)}
                  onValueChange={(value) => updateArrayFilter('type_ids', value.map(Number))}
                  options={filterOptions.types.map((item) => ({
                    value: String(item.type_id),
                    label: item.type_name,
                  }))}
                />
              </label>

              <label className="space-y-1 md:col-span-2">
                <span className="text-sm text-muted-foreground">
                  {t('corporationStructures.filters.services')}
                </span>
                <div className="flex flex-wrap items-center gap-2">
                  <MultiSelect
                    className="w-full md:w-[360px]"
                    placeholder={t('corporationStructures.filters.services')}
                    value={filters.service_names}
                    onValueChange={(value) =>
                      setFilters((current) => ({ ...current, service_names: value }))
                    }
                    options={filterOptions.services.map((item) => ({
                      value: item.name,
                      label: item.name,
                    }))}
                  />
                  <div className="flex flex-wrap gap-2">
                    {[
                      ['and', t('corporationStructures.serviceMatch.and')],
                      ['or', t('corporationStructures.serviceMatch.or')],
                    ].map(([value, label]) => (
                      <Button
                        key={value}
                        type="button"
                        variant={filters.service_match_mode === value ? 'default' : 'outline'}
                        className="px-3 py-1.5"
                        aria-pressed={filters.service_match_mode === value}
                        onClick={() =>
                          setFilters((current) => ({
                            ...current,
                            service_match_mode: value as typeof filters.service_match_mode,
                          }))
                        }
                      >
                        {label}
                      </Button>
                    ))}
                  </div>
                </div>
              </label>

              <label className="space-y-1 md:col-span-2 xl:col-span-3">
                <span className="text-sm text-muted-foreground">
                  {t('corporationStructures.filters.timer')}
                </span>
                <div className="flex flex-wrap items-center gap-2">
                  {[
                    ['all', t('corporationStructures.timerBuckets.all')],
                    ['current_hour', t('corporationStructures.timerBuckets.currentHour')],
                    ['next_2_hours', t('corporationStructures.timerBuckets.next2Hours')],
                    ['custom', t('corporationStructures.timerBuckets.custom')],
                  ].map(([value, label]) => (
                    <Button
                      key={value}
                      type="button"
                      variant={filters.timer_bucket === value ? 'default' : 'outline'}
                      className="px-3 py-1.5"
                      aria-pressed={filters.timer_bucket === value}
                      onClick={() =>
                        setFilters((current) => ({
                          ...current,
                          timer_bucket: value as typeof filters.timer_bucket,
                        }))
                      }
                    >
                      {label}
                    </Button>
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
              <Button type="button" onClick={applySearch} isDisabled={loading}>
                {t('corporationStructures.actions.search')}
              </Button>
              <Button type="button" variant="outline" onClick={resetSearch}>
                {t('corporationStructures.actions.reset')}
              </Button>
              <Button
                type="button"
                variant="outline"
                isDisabled={
                  filters.corporation_id <= 0 || runningTaskCorpId === filters.corporation_id
                }
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
              <Table className="min-w-full text-sm">
                <TableHeader>
                  <TableRow className="border-b bg-muted/40 text-left">
                    <TableHead className="px-3 py-2">
                      {t('corporationStructures.table.corporation')}
                    </TableHead>
                    <TableHead className="px-3 py-2">
                      {t('corporationStructures.salary.assignedFuelOfficer')}
                    </TableHead>
                    <TableHead className="px-3 py-2">
                      {t('corporationStructures.table.state')}
                    </TableHead>
                    <TableHead className="px-3 py-2">
                      {t('corporationStructures.table.system')}
                    </TableHead>
                    <TableHead className="px-3 py-2">
                      {t('corporationStructures.table.name')}
                    </TableHead>
                    <TableHead className="px-3 py-2">
                      {t('corporationStructures.table.type')}
                    </TableHead>
                    <TableHead className="px-3 py-2">
                      {t('corporationStructures.table.services')}
                    </TableHead>
                    <TableHead className="px-3 py-2">
                      {t('corporationStructures.table.fuelRemaining')}
                    </TableHead>
                    <TableHead className="px-3 py-2">
                      {t('corporationStructures.table.fuelPerHour')}
                    </TableHead>
                    <TableHead className="px-3 py-2">
                      {t('corporationStructures.table.fuelToMonthEnd')}
                    </TableHead>
                    <TableHead className="px-3 py-2">
                      {t('corporationStructures.table.reinforceHour')}
                    </TableHead>
                    <TableHead className="px-3 py-2">
                      {t('corporationStructures.table.timerEnd')}
                    </TableHead>
                    <TableHead className="px-3 py-2">
                      {t('corporationStructures.table.updatedAt')}
                    </TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {tableData.map((row) => (
                    <TableRow key={row.structure_id} className="border-b">
                      <TableCell className="px-3 py-2">{row.corporation_name}</TableCell>
                      <TableCell className="px-3 py-2">
                        {row.assigned_user_id > 0
                          ? row.assigned_character_name || '--'
                          : t('corporationStructures.salary.unassignedLabel')}
                      </TableCell>
                      <TableCell className="px-3 py-2">{stateLabel(t, row.state)}</TableCell>
                      <TableCell className="px-3 py-2">
                        <div>{row.system_name || '--'}</div>
                        <div className="text-xs text-muted-foreground">
                          {row.region_name || '--'} / {formatSecurity(row.security)}
                        </div>
                      </TableCell>
                      <TableCell className="px-3 py-2">{row.name}</TableCell>
                      <TableCell className="px-3 py-2">{row.type_name}</TableCell>
                      <TableCell className="px-3 py-2">{formatServices(t, row.services)}</TableCell>
                      <TableCell className="px-3 py-2">{row.fuel_remaining || '--'}</TableCell>
                      <TableCell className="px-3 py-2">
                        {formatFuelEstimate(t, row, 'fuel_per_hour')}
                      </TableCell>
                      <TableCell className="px-3 py-2">
                        {formatFuelEstimate(t, row, 'fuel_to_month_end')}
                      </TableCell>
                      <TableCell className="px-3 py-2">
                        {row.reinforce_hour > 0
                          ? String(row.reinforce_hour).padStart(2, '0')
                          : '--'}
                      </TableCell>
                      <TableCell className="px-3 py-2">
                        {formatTimeText(row.state_timer_end)}
                      </TableCell>
                      <TableCell className="px-3 py-2">{formatUpdatedAt(row.updated_at)}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          </div>

          <div className="flex flex-wrap items-center gap-3 text-sm">
            <span>{page}</span>
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => setPage((current) => Math.max(1, current - 1))}
              isDisabled={page <= 1}
            >
              -
            </Button>
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => setPage((current) => current + 1)}
              isDisabled={tableData.length < pageSize || page * pageSize >= total}
            >
              +
            </Button>
            <label className="flex items-center gap-2">
              <span>{t('common.refresh')}</span>
              <Select
                selectedKey={String(pageSize ?? '')}
                onSelectionChange={(key) => ((value) => setPageSize(Number(value)))(String(key))}
              >
                <SelectTrigger className="h-8 rounded-md border border-input bg-background px-2 text-sm">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {[10, 20, 50].map((size) => (
                    <SelectItem key={size} id={String(size ?? '')}>
                      {size}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </label>
          </div>
        </>
      ) : null}

      {activeTab === 'settings' ? (
        <div className="space-y-4 rounded-lg border bg-card p-5">
          <div>
            <h2 className="text-lg font-semibold">
              {t('corporationStructures.settings.noticeThresholds')}
            </h2>
            <p className="text-sm text-muted-foreground">
              {t('corporationStructures.settings.noticeThresholdHint')}
            </p>
          </div>

          {settingsError ? <p className="text-sm text-destructive">{settingsError}</p> : null}
          {alertScanMessage ? (
            <p className="text-sm text-muted-foreground">{alertScanMessage}</p>
          ) : null}
          {settingsLoading ? (
            <p className="text-sm text-muted-foreground">{t('common.refresh')}</p>
          ) : null}

          <div className="space-y-4">
            <div className="flex flex-wrap items-center gap-3">
              <span className="text-sm text-muted-foreground">
                {t('corporationStructures.settings.fuelNoticeThreshold')}
              </span>
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
              <span className="text-sm text-muted-foreground">
                {t('corporationStructures.settings.daysUnit')}
              </span>
            </div>

            <div className="flex flex-wrap items-center gap-3">
              <label
                className="flex items-center gap-2 text-sm text-muted-foreground"
                htmlFor="corporation-structure-alert-enabled"
              >
                <Input
                  id="corporation-structure-alert-enabled"
                  type="checkbox"
                  checked={alertEnabled}
                  onChange={(event) => setAlertEnabled(event.target.checked)}
                />
                {t('corporationStructures.settings.alertEnabled')}
              </label>
            </div>

            <div className="flex flex-wrap items-start gap-3">
              <label
                className="text-sm text-muted-foreground"
                htmlFor="corporation-structure-alert-groups"
              >
                {t('corporationStructures.settings.alertGroupIDs')}
              </label>
              <Textarea
                id="corporation-structure-alert-groups"
                className="min-h-20 w-80 rounded-md border border-input bg-background px-3 py-2 text-sm disabled:cursor-not-allowed disabled:opacity-50"
                value={alertGroupIDsText}
                disabled={!alertEnabled}
                placeholder={t('corporationStructures.settings.alertGroupIDsPlaceholder')}
                onChange={(event) => setAlertGroupIDsText(event.target.value)}
              />
              <span className="text-sm text-muted-foreground">
                {t('corporationStructures.settings.alertGroupIDsHint')}
              </span>
            </div>

            <div className="flex flex-wrap items-center gap-3">
              <span className="text-sm text-muted-foreground">
                {t('corporationStructures.settings.timerNoticeThreshold')}
              </span>
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
              <span className="text-sm text-muted-foreground">
                {t('corporationStructures.settings.daysUnit')}
              </span>
            </div>
          </div>

          <div className="flex flex-wrap items-center gap-3">
            <Button
              type="button"
              variant="outline"
              onClick={() => void loadSettings()}
              isDisabled={settingsLoading}
            >
              {t('corporationStructures.actions.refreshSettings')}
            </Button>
            <Button
              type="button"
              onClick={() => void saveAuthorizations()}
              isDisabled={savingAuthorizations}
            >
              {t('corporationStructures.actions.save')}
            </Button>
            {canRunAlertScan ? (
              <Button
                type="button"
                variant="outline"
                onClick={() => void handleRunAlertScan()}
                isDisabled={!settings.alert_enabled || alertScanRunning}
              >
                {t('corporationStructures.actions.runAlertScan')}
              </Button>
            ) : null}
          </div>

          <div className="overflow-hidden rounded-lg border">
            <Table className="min-w-full text-sm">
              <TableHeader>
                <TableRow className="border-b bg-muted/40 text-left">
                  <TableHead className="px-3 py-2">
                    {t('corporationStructures.table.corporation')}
                  </TableHead>
                  <TableHead className="px-3 py-2">
                    {t('corporationStructures.table.directorCharacter')}
                  </TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {settings.corporations.map((corp) => (
                  <TableRow key={corp.corporation_id} className="border-b">
                    <TableCell className="px-3 py-2">
                      <div className="font-medium">{corp.corporation_name}</div>
                      <div className="text-xs text-muted-foreground">{corp.corporation_id}</div>
                    </TableCell>
                    <TableCell className="px-3 py-2">
                      <Select
                        selectedKey={String(authorizationByCorp[corp.corporation_id] || 0)}
                        onSelectionChange={(key) => ((value) =>
                          setAuthorizationByCorp((current) => ({
                            ...current,
                            [corp.corporation_id]: Number(value),
                          })))(String(key))}
                      >
                        <SelectTrigger className="h-10 w-full rounded-md border border-input bg-background px-3 text-sm">
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem id={String(0)}>
                            {t('corporationStructures.options.disabled')}
                          </SelectItem>
                          {corp.director_characters.map((option) => (
                            <SelectItem
                              key={option.character_id}
                              id={String(option.character_id ?? '')}
                            >
                              {option.character_name} ({option.character_id})
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>

          <div className="space-y-3 border-t pt-5">
            <div>
              <h2 className="text-lg font-semibold">
                {t('corporationStructures.serviceCatalog.title')}
              </h2>
              <p className="text-sm text-muted-foreground">
                {t('corporationStructures.serviceCatalog.hint')}
              </p>
            </div>

            {serviceCatalog.unmapped_activities.length === 0 ? (
              <p className="text-sm text-muted-foreground">
                {t('corporationStructures.serviceCatalog.empty')}
              </p>
            ) : (
              <div className="overflow-x-auto rounded-lg border">
                <Table className="min-w-full text-sm">
                  <TableHeader>
                    <TableRow className="border-b bg-muted/40 text-left">
                      <TableHead className="px-3 py-2">
                        {t('corporationStructures.serviceCatalog.activity')}
                      </TableHead>
                      <TableHead className="px-3 py-2">
                        {t('corporationStructures.serviceCatalog.installedModules')}
                      </TableHead>
                      <TableHead className="px-3 py-2">
                        {t('corporationStructures.serviceCatalog.module')}
                      </TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {serviceCatalog.unmapped_activities.map((item) => (
                      <TableRow
                        key={`${item.structure_id}-${item.activity_name}`}
                        className="border-b"
                      >
                        <TableCell className="px-3 py-2">
                          <div>{item.activity_name}</div>
                          <div className="text-xs text-muted-foreground">
                            {item.structure_name} ({item.structure_id})
                          </div>
                        </TableCell>
                        <TableCell className="px-3 py-2">
                          {item.installed_module_type_ids.join(', ') || '--'}
                        </TableCell>
                        <TableCell className="px-3 py-2">
                          <MultiSelect
                            className="min-w-64"
                            placeholder={item.activity_name}
                            value={(activityModules[item.activity_name] || []).map(String)}
                            onValueChange={(value) =>
                              setActivityModules((current) => ({
                                ...current,
                                [item.activity_name]: value.map(Number),
                              }))
                            }
                            options={serviceCatalog.modules.map((module) => ({
                              value: String(module.type_id),
                              label: `${module.type_name} (${module.type_id})`,
                            }))}
                          />
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            )}
            {serviceCatalog.unmapped_activities.length > 0 ? (
              <Button
                type="button"
                onClick={() => void saveServiceActivityMappings()}
                isDisabled={savingServiceCatalog}
              >
                {t('corporationStructures.serviceCatalog.save')}
              </Button>
            ) : null}

            <div className="overflow-x-auto rounded-lg border">
              <Table className="min-w-full text-sm">
                <TableHeader>
                  <TableRow className="border-b bg-muted/40 text-left">
                    <TableHead className="px-3 py-2">
                      {t('corporationStructures.serviceCatalog.activity')}
                    </TableHead>
                    <TableHead className="px-3 py-2">
                      {t('corporationStructures.serviceCatalog.typeId')}
                    </TableHead>
                    <TableHead className="px-3 py-2">
                      {t('corporationStructures.serviceCatalog.management')}
                    </TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {serviceCatalog.activities.map((item) => (
                    <TableRow key={item.activity_name} className="border-b">
                      <TableCell className="px-3 py-2">{item.activity_name}</TableCell>
                      <TableCell className="px-3 py-2">{item.type_ids.join(', ')}</TableCell>
                      <TableCell className="px-3 py-2">
                        {item.system_managed
                          ? t('corporationStructures.serviceCatalog.systemManaged')
                          : t('corporationStructures.serviceCatalog.customManaged')}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>

            <div className="space-y-2">
              <h3 className="text-sm font-medium">
                {t('corporationStructures.serviceCatalog.modules')}
              </h3>
              <div className="overflow-x-auto rounded-lg border">
                <Table className="min-w-full text-sm">
                  <TableHeader>
                    <TableRow className="border-b bg-muted/40 text-left">
                      <TableHead className="px-3 py-2">
                        {t('corporationStructures.serviceCatalog.module')}
                      </TableHead>
                      <TableHead className="px-3 py-2">
                        {t('corporationStructures.serviceCatalog.typeId')}
                      </TableHead>
                      <TableHead className="px-3 py-2">
                        {t('corporationStructures.serviceCatalog.fuelRate')}
                      </TableHead>
                      <TableHead className="px-3 py-2">
                        {t('corporationStructures.serviceCatalog.category')}
                      </TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {serviceCatalog.modules.map((module) => (
                      <TableRow key={module.type_id} className="border-b">
                        <TableCell className="px-3 py-2">{module.type_name}</TableCell>
                        <TableCell className="px-3 py-2">{module.type_id}</TableCell>
                        <TableCell className="px-3 py-2">{module.fuel_per_hour}</TableCell>
                        <TableCell className="px-3 py-2">{module.fuel_category}</TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            </div>
          </div>

          {!settingsLoading && settings.corporations.length === 0 ? (
            <p className="text-sm text-muted-foreground">
              {t('corporationStructures.empty.settings')}
            </p>
          ) : null}
        </div>
      ) : null}
    </section>
  )
}

function parseAlertGroupIDs(value: string): number[] {
  return value
    .split(/\r?\n/)
    .map((item) => item.trim())
    .filter((item) => item.length > 0)
    .map((item) => Number(item))
}
