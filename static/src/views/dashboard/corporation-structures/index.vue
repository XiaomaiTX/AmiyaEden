<template>
  <div class="corporation-structures-page art-full-height">
    <ElCard shadow="never" class="art-card mb-4">
      <div class="flex flex-col gap-1">
        <h2 class="text-lg font-medium">{{ $t('corporationStructures.title') }}</h2>
        <p class="text-sm text-g-500">{{ $t('corporationStructures.subtitle') }}</p>
      </div>
    </ElCard>

    <ElTabs v-model="activeTab" @tab-change="handleTabChange">
      <ElTabPane :label="$t('corporationStructures.tabs.list')" name="list">
        <ElCard shadow="never" class="art-card mb-4">
          <div class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-3">
            <ElFormItem :label="$t('corporationStructures.filters.corporation')" class="mb-0">
              <ElSelect
                v-model="filters.corporation_id"
                filterable
                clearable
                class="w-full"
                @change="handleCorporationFilterChange"
                @clear="handleCorporationFilterChange"
              >
                <ElOption :label="$t('corporationStructures.allCorporations')" :value="0" />
                <ElOption
                  v-for="corp in settings.corporations"
                  :key="corp.corporation_id"
                  :label="`${corp.corporation_name} (${corp.corporation_id})`"
                  :value="corp.corporation_id"
                />
              </ElSelect>
            </ElFormItem>

            <ElFormItem :label="$t('corporationStructures.filters.keyword')" class="mb-0">
              <ElInput
                v-model="filters.keyword"
                clearable
                :placeholder="$t('corporationStructures.placeholders.keyword')"
              />
            </ElFormItem>

            <ElFormItem :label="$t('corporationStructures.filters.systems')" class="mb-0">
              <ElSelect
                v-model="filters.system_ids"
                multiple
                filterable
                clearable
                collapse-tags
                collapse-tags-tooltip
                class="w-full"
              >
                <ElOption
                  v-for="item in filterOptions.systems"
                  :key="item.system_id"
                  :label="formatSystemOption(item)"
                  :value="item.system_id"
                />
              </ElSelect>
            </ElFormItem>

            <ElFormItem
              :label="$t('corporationStructures.filters.stateGroups')"
              class="mb-0 md:col-span-2"
            >
              <ElCheckboxGroup v-model="filters.state_groups">
                <ElCheckboxButton value="online">{{
                  $t('corporationStructures.stateGroups.online')
                }}</ElCheckboxButton>
                <ElCheckboxButton value="low_power">{{
                  $t('corporationStructures.stateGroups.lowPower')
                }}</ElCheckboxButton>
                <ElCheckboxButton value="abandoned">{{
                  $t('corporationStructures.stateGroups.abandoned')
                }}</ElCheckboxButton>
                <ElCheckboxButton value="reinforced">{{
                  $t('corporationStructures.stateGroups.reinforced')
                }}</ElCheckboxButton>
              </ElCheckboxGroup>
            </ElFormItem>

            <ElFormItem
              :label="$t('corporationStructures.filters.fuel')"
              class="mb-0 md:col-span-2 xl:col-span-3"
            >
              <div class="flex flex-wrap items-center gap-2">
                <ElRadioGroup v-model="filters.fuel_bucket">
                  <ElRadioButton value="all">{{
                    $t('corporationStructures.fuelBuckets.all')
                  }}</ElRadioButton>
                  <ElRadioButton value="lt_24h">{{
                    $t('corporationStructures.fuelBuckets.lt24h')
                  }}</ElRadioButton>
                  <ElRadioButton value="lt_72h">{{
                    $t('corporationStructures.fuelBuckets.lt3d')
                  }}</ElRadioButton>
                  <ElRadioButton value="lt_168h">{{
                    $t('corporationStructures.fuelBuckets.lt7d')
                  }}</ElRadioButton>
                  <ElRadioButton value="custom">{{
                    $t('corporationStructures.fuelBuckets.custom')
                  }}</ElRadioButton>
                </ElRadioGroup>
                <template v-if="filters.fuel_bucket === 'custom'">
                  <ElInputNumber v-model="filters.fuel_min_hours" :min="0" :step="1" />
                  <span>~</span>
                  <ElInputNumber v-model="filters.fuel_max_hours" :min="0" :step="1" />
                </template>
              </div>
            </ElFormItem>

            <ElFormItem
              :label="$t('corporationStructures.filters.security')"
              class="mb-0 md:col-span-2"
            >
              <div class="flex flex-wrap items-center gap-2">
                <ElCheckboxGroup v-model="filters.security_bands">
                  <ElCheckboxButton value="highsec">{{
                    $t('corporationStructures.securityBands.highsec')
                  }}</ElCheckboxButton>
                  <ElCheckboxButton value="lowsec">{{
                    $t('corporationStructures.securityBands.lowsec')
                  }}</ElCheckboxButton>
                  <ElCheckboxButton value="nullsec">{{
                    $t('corporationStructures.securityBands.nullsec')
                  }}</ElCheckboxButton>
                </ElCheckboxGroup>
                <ElInputNumber
                  v-model="filters.security_min"
                  :min="-1"
                  :max="1"
                  :step="0.1"
                  :precision="1"
                />
                <span>~</span>
                <ElInputNumber
                  v-model="filters.security_max"
                  :min="-1"
                  :max="1"
                  :step="0.1"
                  :precision="1"
                />
              </div>
            </ElFormItem>

            <ElFormItem :label="$t('corporationStructures.filters.types')" class="mb-0">
              <ElSelect
                v-model="filters.type_ids"
                multiple
                filterable
                clearable
                collapse-tags
                collapse-tags-tooltip
                class="w-full"
              >
                <ElOption
                  v-for="item in filterOptions.types"
                  :key="item.type_id"
                  :label="item.type_name"
                  :value="item.type_id"
                />
              </ElSelect>
            </ElFormItem>

            <ElFormItem
              :label="$t('corporationStructures.filters.services')"
              class="mb-0 md:col-span-2"
            >
              <div class="flex flex-wrap items-center gap-2 w-full">
                <ElSelect
                  v-model="filters.service_names"
                  multiple
                  filterable
                  clearable
                  class="w-full md:w-[360px]"
                >
                  <ElOption
                    v-for="item in filterOptions.services"
                    :key="item.name"
                    :label="item.name"
                    :value="item.name"
                  />
                </ElSelect>
                <ElRadioGroup v-model="filters.service_match_mode">
                  <ElRadioButton value="and">{{
                    $t('corporationStructures.serviceMatch.and')
                  }}</ElRadioButton>
                  <ElRadioButton value="or">{{
                    $t('corporationStructures.serviceMatch.or')
                  }}</ElRadioButton>
                </ElRadioGroup>
              </div>
            </ElFormItem>

            <ElFormItem
              :label="$t('corporationStructures.filters.timer')"
              class="mb-0 md:col-span-2 xl:col-span-3"
            >
              <div class="flex flex-wrap items-center gap-2">
                <ElRadioGroup v-model="filters.timer_bucket">
                  <ElRadioButton value="all">{{
                    $t('corporationStructures.timerBuckets.all')
                  }}</ElRadioButton>
                  <ElRadioButton value="current_hour">{{
                    $t('corporationStructures.timerBuckets.currentHour')
                  }}</ElRadioButton>
                  <ElRadioButton value="next_2_hours">{{
                    $t('corporationStructures.timerBuckets.next2Hours')
                  }}</ElRadioButton>
                  <ElRadioButton value="custom">{{
                    $t('corporationStructures.timerBuckets.custom')
                  }}</ElRadioButton>
                </ElRadioGroup>
                <ElDatePicker
                  v-if="filters.timer_bucket === 'custom'"
                  v-model="timerRange"
                  type="datetimerange"
                  class="w-[380px]"
                  value-format="YYYY-MM-DDTHH:mm:ss"
                />
              </div>
            </ElFormItem>
          </div>

          <div class="flex flex-wrap items-center gap-3 mt-4">
            <ElButton type="primary" :loading="loading" @click="handleSearch">
              {{ $t('common.search') }}
            </ElButton>
            <ElButton @click="handleReset">{{ $t('common.reset') }}</ElButton>
            <ElButton
              type="primary"
              :loading="runningTaskCorpId === filters.corporation_id && filters.corporation_id > 0"
              :disabled="filters.corporation_id <= 0"
              @click="handleRunTaskForSelectedCorporation"
            >
              {{ $t('corporationStructures.actions.refreshSelected') }}
            </ElButton>
          </div>
        </ElCard>

        <ElCard shadow="never" class="art-table-card">
          <ArtTableHeader
            v-model:columns="columnChecks"
            :loading="loading"
            @refresh="refreshData"
          />
          <ArtTable
            :loading="loading"
            :data="data"
            :columns="columns"
            :pagination="pagination"
            :default-sort="{ prop: 'fuel_remaining_hours', order: 'ascending' }"
            :empty-text="$t('corporationStructures.empty.list')"
            @sort-change="handleSortChange"
            @pagination:size-change="handleSizeChange"
            @pagination:current-change="handleCurrentChange"
          />
        </ElCard>
      </ElTabPane>

      <ElTabPane :label="$t('corporationStructures.tabs.settings')" name="settings">
        <ElCard shadow="never" class="art-table-card">
          <ElFormItem :label="$t('corporationStructures.settings.noticeThresholds')" class="mb-4">
            <div class="flex flex-wrap items-center gap-4">
              <div class="flex items-center gap-2">
                <span class="text-sm text-g-500">
                  {{ $t('corporationStructures.settings.fuelNoticeThreshold') }}
                </span>
                <ElInputNumber
                  v-model="noticeThresholds.fuel_notice_threshold_days"
                  :min="0"
                  :step="1"
                  step-strictly
                />
                <span class="text-sm text-g-500">{{
                  $t('corporationStructures.settings.daysUnit')
                }}</span>
              </div>
              <div class="flex items-center gap-2">
                <span class="text-sm text-g-500">
                  {{ $t('corporationStructures.settings.timerNoticeThreshold') }}
                </span>
                <ElInputNumber
                  v-model="noticeThresholds.timer_notice_threshold_days"
                  :min="0"
                  :step="1"
                  step-strictly
                />
                <span class="text-sm text-g-500">{{
                  $t('corporationStructures.settings.daysUnit')
                }}</span>
              </div>
              <span class="text-xs text-g-500">
                {{ $t('corporationStructures.settings.noticeThresholdHint') }}
              </span>
            </div>
          </ElFormItem>

          <div class="flex flex-wrap items-center gap-3 mb-4">
            <ElButton :loading="settingsLoading" @click="loadSettings">
              {{ $t('common.refresh') }}
            </ElButton>
            <ElButton type="primary" :loading="savingAuthorizations" @click="saveAuthorizations">
              {{ $t('common.save') }}
            </ElButton>
          </div>

          <ElTable v-loading="settingsLoading" :data="settings.corporations" stripe border>
            <ElTableColumn :label="$t('corporationStructures.table.corporation')" min-width="260">
              <template #default="{ row }">
                <div class="font-medium">{{ row.corporation_name }}</div>
                <div class="text-xs text-g-500">{{ row.corporation_id }}</div>
              </template>
            </ElTableColumn>
            <ElTableColumn
              :label="$t('corporationStructures.table.directorCharacter')"
              min-width="320"
            >
              <template #default="{ row }">
                <ElSelect
                  v-model="authorizationByCorp[row.corporation_id]"
                  clearable
                  :placeholder="$t('corporationStructures.placeholders.selectDirector')"
                  class="w-full"
                  @clear="authorizationByCorp[row.corporation_id] = 0"
                >
                  <ElOption :label="$t('corporationStructures.options.disabled')" :value="0" />
                  <ElOption
                    v-for="option in row.director_characters"
                    :key="option.character_id"
                    :label="`${option.character_name} (${option.character_id})`"
                    :value="option.character_id"
                  />
                </ElSelect>
              </template>
            </ElTableColumn>
          </ElTable>

          <ElEmpty
            v-if="!settingsLoading && settings.corporations.length === 0"
            :description="$t('corporationStructures.empty.settings')"
            class="mt-4"
          />
        </ElCard>
      </ElTabPane>
    </ElTabs>
  </div>
</template>

<script setup lang="ts">
  import { ElMessage, ElTag } from 'element-plus'
  import { useI18n } from 'vue-i18n'
  import { useRoute, useRouter } from 'vue-router'
  import { useTable } from '@/hooks/core/useTable'
  import {
    fetchCorporationStructureFilterOptions,
    fetchCorporationStructureList,
    fetchCorporationStructureSettings,
    runCorporationStructuresTask,
    updateCorporationStructureAuthorizations
  } from '@/api/corporation-structures'

  defineOptions({ name: 'DashboardCorporationStructures' })

  type StructureTab = 'list' | 'settings'
  type StructureRow = Api.Dashboard.CorporationStructureRow
  type TableSort = { prop?: string; order?: 'ascending' | 'descending' | null }

  const { t } = useI18n()
  const route = useRoute()
  const router = useRouter()

  const settings = ref<Api.Dashboard.CorporationStructuresSettings>({
    corporations: [],
    fuel_notice_threshold_days: 7,
    timer_notice_threshold_days: 7
  })
  const noticeThresholds = reactive({
    fuel_notice_threshold_days: 7,
    timer_notice_threshold_days: 7
  })
  const settingsLoading = ref(false)
  const savingAuthorizations = ref(false)
  const runningTaskCorpId = ref<number>(0)
  const authorizationByCorp = reactive<Record<number, number>>({})
  const timerRange = ref<[string, string] | null>(null)

  const filterOptions = ref<Api.Dashboard.CorporationStructureFilterOptionsResponse>({
    systems: [],
    types: [],
    services: []
  })

  const buildDefaultFilters = () => ({
    corporation_id: 0,
    keyword: '',
    state_groups: [] as string[],
    fuel_bucket: 'all' as Api.Dashboard.CorporationStructureListRequest['fuel_bucket'],
    fuel_min_hours: undefined as number | undefined,
    fuel_max_hours: undefined as number | undefined,
    system_ids: [] as number[],
    security_bands: [] as ('highsec' | 'lowsec' | 'nullsec')[],
    security_min: undefined as number | undefined,
    security_max: undefined as number | undefined,
    type_ids: [] as number[],
    service_names: [] as string[],
    service_match_mode: 'and' as const,
    timer_bucket: 'all' as Api.Dashboard.CorporationStructureListRequest['timer_bucket']
  })

  const filters = reactive(buildDefaultFilters())

  const normalizeTab = (value: unknown): StructureTab => {
    const queryValue = Array.isArray(value) ? value[0] : value
    return queryValue === 'settings' ? 'settings' : 'list'
  }

  const activeTab = ref<StructureTab>(normalizeTab(route.query.tab))

  const normalizeFuelHours = (value: number | undefined) => {
    if (value == null || Number.isNaN(value)) {
      return undefined
    }
    return Math.max(0, Math.floor(value))
  }

  const normalizeThresholdDays = (value: number) => {
    if (Number.isNaN(value)) {
      return 0
    }
    return Math.max(0, Math.floor(value))
  }

  const fetchStructurePage = async (
    params: Api.Dashboard.CorporationStructureListRequest & { current: number; size: number }
  ): Promise<Api.Common.PaginatedResponse<StructureRow>> => {
    const corpId = params.corporation_id ?? 0
    const response = await fetchCorporationStructureList({
      corporation_id: corpId > 0 ? corpId : undefined,
      page: params.current,
      page_size: params.size,
      keyword: params.keyword || undefined,
      state_groups: params.state_groups?.length ? params.state_groups : undefined,
      fuel_bucket: params.fuel_bucket,
      fuel_min_hours: normalizeFuelHours(params.fuel_min_hours),
      fuel_max_hours: normalizeFuelHours(params.fuel_max_hours),
      system_ids: params.system_ids?.length ? params.system_ids : undefined,
      security_bands: params.security_bands?.length ? params.security_bands : undefined,
      security_min: params.security_min,
      security_max: params.security_max,
      type_ids: params.type_ids?.length ? params.type_ids : undefined,
      service_names: params.service_names?.length ? params.service_names : undefined,
      service_match_mode: params.service_match_mode,
      timer_bucket: params.timer_bucket,
      timer_start: params.timer_start,
      timer_end: params.timer_end,
      sort_by: params.sort_by,
      sort_order: params.sort_order
    })
    return {
      list: response?.items ?? [],
      total: response?.total ?? 0,
      page: response?.page ?? params.current,
      pageSize: response?.page_size ?? params.size
    }
  }

  const stateTagTypeMap: Record<string, any> = {
    shield_vulnerable: 'success',
    low_power: 'warning',
    abandoned: 'info',
    shield_reinforce: 'danger',
    armor_reinforce: 'danger',
    armor_vulnerable: 'danger',
    hull_reinforce: 'danger',
    hull_vulnerable: 'danger'
  }

  const {
    columns,
    columnChecks,
    data,
    loading,
    pagination,
    searchParams,
    handleSizeChange,
    handleCurrentChange,
    refreshData,
    getData
  } = useTable({
    core: {
      apiFn: fetchStructurePage,
      apiParams: {
        corporation_id: 0,
        keyword: '',
        state_groups: [],
        fuel_bucket: 'all',
        fuel_min_hours: undefined,
        fuel_max_hours: undefined,
        system_ids: [],
        security_bands: [],
        security_min: undefined,
        security_max: undefined,
        type_ids: [],
        service_names: [],
        service_match_mode: 'and',
        timer_bucket: 'all',
        timer_start: undefined,
        timer_end: undefined,
        sort_by: 'fuel_remaining_hours',
        sort_order: 'asc',
        current: 1,
        size: 20
      },
      immediate: false,
      columnsFactory: () => [
        {
          prop: 'corporation_name',
          label: t('corporationStructures.table.corporation'),
          minWidth: 180,
          showOverflowTooltip: true
        },
        {
          prop: 'state',
          label: t('corporationStructures.table.state'),
          width: 180,
          formatter: (row: StructureRow) =>
            h(
              ElTag,
              {
                type: stateTagTypeMap[row.state] || 'info',
                size: 'small',
                effect: 'plain'
              },
              () => mapStateLabel(row.state)
            )
        },
        {
          prop: 'system_name',
          label: t('corporationStructures.table.system'),
          minWidth: 220,
          sortable: 'custom' as const,
          formatter: (row: StructureRow) =>
            h('div', { class: 'leading-5' }, [
              h('div', {}, row.system_name || '--'),
              h(
                'div',
                { class: 'text-xs text-g-500' },
                `${row.region_name || '--'} / ${formatSecurity(row.security)}`
              )
            ])
        },
        {
          prop: 'name',
          label: t('corporationStructures.table.name'),
          minWidth: 200,
          sortable: 'custom' as const,
          showOverflowTooltip: true
        },
        {
          prop: 'type_name',
          label: t('corporationStructures.table.type'),
          minWidth: 180,
          sortable: 'custom' as const,
          showOverflowTooltip: true
        },
        {
          prop: 'services',
          label: t('corporationStructures.table.services'),
          minWidth: 260,
          formatter: (row: StructureRow) => formatServices(row.services)
        },
        {
          prop: 'fuel_remaining_hours',
          label: t('corporationStructures.table.fuelRemaining'),
          width: 170,
          sortable: 'custom' as const,
          formatter: (row: StructureRow) => row.fuel_remaining || '--'
        },
        {
          prop: 'reinforce_hour',
          label: t('corporationStructures.table.reinforceHour'),
          width: 150,
          sortable: 'custom' as const,
          formatter: (row: StructureRow) =>
            row.reinforce_hour > 0 ? String(row.reinforce_hour).padStart(2, '0') : '--'
        },
        {
          prop: 'state_timer_end',
          label: t('corporationStructures.table.timerEnd'),
          width: 190,
          sortable: 'custom' as const,
          formatter: (row: StructureRow) => formatTimeText(row.state_timer_end)
        },
        {
          prop: 'updated_at',
          label: t('corporationStructures.table.updatedAt'),
          width: 190,
          sortable: 'custom' as const,
          formatter: (row: StructureRow) => formatUpdatedAt(row.updated_at)
        }
      ]
    }
  })

  const syncAuthorizationsFromSettings = () => {
    Object.keys(authorizationByCorp).forEach((key) => {
      delete authorizationByCorp[Number(key)]
    })
    settings.value.corporations.forEach((item) => {
      authorizationByCorp[item.corporation_id] = item.authorized_character_id || 0
    })
    noticeThresholds.fuel_notice_threshold_days = normalizeThresholdDays(
      settings.value.fuel_notice_threshold_days
    )
    noticeThresholds.timer_notice_threshold_days = normalizeThresholdDays(
      settings.value.timer_notice_threshold_days
    )
  }

  const loadSettings = async () => {
    settingsLoading.value = true
    try {
      settings.value = await fetchCorporationStructureSettings()
      syncAuthorizationsFromSettings()

      const managedCorpSet = new Set(settings.value.corporations.map((item) => item.corporation_id))
      if (filters.corporation_id > 0 && !managedCorpSet.has(filters.corporation_id)) {
        filters.corporation_id = 0
      }
    } finally {
      settingsLoading.value = false
    }
  }

  const loadFilterOptions = async () => {
    filterOptions.value = (await fetchCorporationStructureFilterOptions({
      corporation_id: filters.corporation_id > 0 ? filters.corporation_id : undefined
    })) || {
      systems: [],
      types: [],
      services: []
    }
  }

  const copyFiltersToSearchParams = () => {
    searchParams.corporation_id = filters.corporation_id
    searchParams.keyword = filters.keyword
    searchParams.state_groups = [...filters.state_groups]
    searchParams.fuel_bucket = filters.fuel_bucket
    searchParams.fuel_min_hours =
      filters.fuel_bucket === 'custom' ? filters.fuel_min_hours : undefined
    searchParams.fuel_max_hours =
      filters.fuel_bucket === 'custom' ? filters.fuel_max_hours : undefined
    searchParams.system_ids = [...filters.system_ids]
    searchParams.security_bands = [...filters.security_bands]
    searchParams.security_min = filters.security_min
    searchParams.security_max = filters.security_max
    searchParams.type_ids = [...filters.type_ids]
    searchParams.service_names = [...filters.service_names]
    searchParams.service_match_mode = filters.service_match_mode
    searchParams.timer_bucket = filters.timer_bucket
    searchParams.timer_start =
      filters.timer_bucket === 'custom' && timerRange.value ? timerRange.value[0] : undefined
    searchParams.timer_end =
      filters.timer_bucket === 'custom' && timerRange.value ? timerRange.value[1] : undefined
  }

  const handleSearch = () => {
    copyFiltersToSearchParams()
    searchParams.current = 1
    getData()
  }

  const handleReset = () => {
    Object.assign(filters, buildDefaultFilters())
    timerRange.value = null
    copyFiltersToSearchParams()
    searchParams.sort_by = 'fuel_remaining_hours'
    searchParams.sort_order = 'asc'
    searchParams.current = 1
    getData()
    void loadFilterOptions()
  }

  const saveAuthorizations = async () => {
    const authorizations: Api.Dashboard.CorporationStructureAuthorizationBinding[] =
      settings.value.corporations.map((corp) => ({
        corporation_id: corp.corporation_id,
        character_id: authorizationByCorp[corp.corporation_id] || 0
      }))

    savingAuthorizations.value = true
    try {
      await updateCorporationStructureAuthorizations({
        authorizations,
        fuel_notice_threshold_days: normalizeThresholdDays(
          noticeThresholds.fuel_notice_threshold_days
        ),
        timer_notice_threshold_days: normalizeThresholdDays(
          noticeThresholds.timer_notice_threshold_days
        )
      })
      await loadSettings()
      ElMessage.success(t('corporationStructures.messages.authorizationSaved'))
    } finally {
      savingAuthorizations.value = false
    }
  }

  const handleRunTaskForCorporation = async (corporationId: number) => {
    runningTaskCorpId.value = corporationId
    try {
      const result = await runCorporationStructuresTask({ corporation_id: corporationId })
      if (result.running) {
        ElMessage.warning(
          result.message || t('corporationStructures.messages.refreshAlreadyRunning')
        )
        return
      }
      ElMessage.success(result.message || t('corporationStructures.messages.refreshQueued'))
    } finally {
      runningTaskCorpId.value = 0
    }
  }

  const handleRunTaskForSelectedCorporation = async () => {
    if (filters.corporation_id <= 0) {
      ElMessage.warning(t('corporationStructures.messages.selectCorporationFirst'))
      return
    }
    await handleRunTaskForCorporation(filters.corporation_id)
  }

  const handleCorporationFilterChange = async () => {
    await loadFilterOptions()
  }

  const handleSortChange = (sort: TableSort) => {
    if (!sort?.prop || !sort.order) {
      searchParams.sort_by = 'fuel_remaining_hours'
      searchParams.sort_order = 'asc'
    } else {
      searchParams.sort_by = sort.prop as Api.Dashboard.CorporationStructureListRequest['sort_by']
      searchParams.sort_order = sort.order === 'descending' ? 'desc' : 'asc'
    }
    searchParams.current = 1
    getData()
  }

  const formatServices = (services: Api.Dashboard.CorporationStructureServiceInfo[]) => {
    if (!services || services.length === 0) {
      return t('corporationStructures.noServices')
    }
    return services.map((service) => `${service.name} (${service.state})`).join(' / ')
  }

  const formatSecurity = (security: number) => {
    if (typeof security !== 'number' || Number.isNaN(security)) {
      return '--'
    }
    return security.toFixed(1)
  }

  const formatUpdatedAt = (updatedAt: number) => {
    if (!updatedAt) {
      return '--'
    }
    return new Date(updatedAt * 1000).toLocaleString()
  }

  const formatTimeText = (value: string) => {
    if (!value) {
      return '--'
    }
    const parsed = new Date(value)
    if (Number.isNaN(parsed.getTime())) {
      return value
    }
    return parsed.toLocaleString()
  }

  const mapStateLabel = (state: string) => {
    const key = `corporationStructures.states.${state}`
    const translated = t(key)
    if (translated === key) {
      return state || '--'
    }
    return translated
  }

  const formatSystemOption = (item: Api.Dashboard.CorporationStructureSystemOption) => {
    const regionText = item.region_name ? ` / ${item.region_name}` : ''
    return `${item.system_name}${regionText} (${formatSecurity(item.security)})`
  }

  const handleTabChange = (tab: string | number) => {
    activeTab.value = normalizeTab(tab)
  }

  watch(
    () => route.query.tab,
    (value) => {
      const nextTab = normalizeTab(value)
      if (nextTab !== activeTab.value) {
        activeTab.value = nextTab
      }
    }
  )

  watch(activeTab, (tab) => {
    const queryTab = normalizeTab(route.query.tab)
    if (queryTab === tab && route.query.tab) {
      return
    }
    void router.replace({
      query: {
        ...route.query,
        tab
      }
    })
  })

  onMounted(async () => {
    if (!route.query.tab || normalizeTab(route.query.tab) !== route.query.tab) {
      await router.replace({
        query: {
          ...route.query,
          tab: activeTab.value
        }
      })
    }

    await loadSettings()
    await loadFilterOptions()
    copyFiltersToSearchParams()
    await getData()
  })
</script>
