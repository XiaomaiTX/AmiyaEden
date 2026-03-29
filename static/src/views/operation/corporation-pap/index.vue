<!-- 分析页例外：该页仍是统计卡片 + 自定义筛选混排，但表格本身复用共享 ArtTable ledger 呈现。 -->
<template>
  <div class="corporation-pap-page art-full-height">
    <ElCard class="art-search-card" shadow="never">
      <div class="filter-toolbar">
        <div class="filter-toolbar__main">
          <ElSelect
            v-model="filters.period"
            :placeholder="t('fleet.corporationPap.filters.period')"
            class="period-filter"
          >
            <ElOption
              v-for="option in periodOptions"
              :key="option.value"
              :label="option.label"
              :value="option.value"
            />
          </ElSelect>
          <ElInputNumber
            v-if="filters.period === 'at_year'"
            v-model="filters.year"
            :min="2003"
            :max="2100"
            :step="1"
            class="year-filter"
          />
          <ElSelect
            v-model="filters.corpTickers"
            class="ticker-filter"
            multiple
            filterable
            allow-create
            default-first-option
            collapse-tags
            collapse-tags-tooltip
            :reserve-keyword="false"
            :placeholder="t('fleet.corporationPap.filters.corpTickers')"
          >
            <ElOption
              v-for="ticker in corpTickerOptions"
              :key="ticker"
              :label="ticker"
              :value="ticker"
            />
          </ElSelect>
        </div>
        <div class="filter-toolbar__actions">
          <ElButton type="primary" :loading="loading" @click="handleSearch">
            {{ t('fleet.corporationPap.search') }}
          </ElButton>
          <ElButton @click="handleReset">{{ t('fleet.corporationPap.reset') }}</ElButton>
        </div>
      </div>
    </ElCard>

    <div class="stats-grid">
      <ElCard v-for="card in statsCards" :key="card.label" shadow="never" class="stat-card">
        <p class="stat-label">{{ card.label }}</p>
        <p class="stat-value">{{ card.value }}</p>
      </ElCard>
    </div>

    <ElCard class="art-table-card table-card" shadow="never">
      <template #header>
        <div class="flex items-center justify-between gap-3">
          <h2 class="text-lg font-medium">{{ t('fleet.corporationPap.summaryTitle') }}</h2>
          <ElButton :loading="loading" @click="loadData">
            <el-icon class="mr-1"><Refresh /></el-icon>
            {{ t('common.refresh') }}
          </ElButton>
        </div>
      </template>

      <div class="table-wrap">
        <ArtTable
          :loading="loading"
          :data="records"
          :columns="columns"
          :pagination="pagination"
          visual-variant="ledger"
          :show-table-header="false"
          :empty-text="t('fleet.corporationPap.empty')"
          @pagination:current-change="handleCurrentChange"
          @pagination:size-change="handleSizeChange"
        />
      </div>
    </ElCard>
  </div>
</template>

<script setup lang="ts">
  import { Refresh } from '@element-plus/icons-vue'
  import { ElButton, ElCard, ElInputNumber, ElOption, ElSelect, ElTag } from 'element-plus'
  import { useI18n } from 'vue-i18n'
  import { fetchCorporationPapSummary } from '@/api/fleet'
  import type { ColumnOption } from '@/types/component'

  defineOptions({ name: 'CorporationPap' })

  const { t } = useI18n()

  const defaultTickers = ['FUXI', 'FMA.1']
  const currentYear = new Date().getFullYear()

  const loading = ref(false)
  const records = ref<Api.Fleet.CorporationPapSummaryItem[]>([])
  const filters = reactive<
    Required<Pick<Api.Fleet.CorporationPapSummaryParams, 'period' | 'year'>> & {
      corpTickers: string[]
    }
  >({
    period: 'last_month',
    year: currentYear,
    corpTickers: [...defaultTickers]
  })
  const pagination = reactive({
    current: 1,
    size: 200,
    total: 0
  })
  const overview = ref<Api.Fleet.CorporationPapOverview>({
    filtered_pap_total: 0,
    filtered_strat_op_total: 0,
    all_pap_total: 0,
    filtered_user_count: 0,
    period: 'last_month'
  })

  const periodOptions = computed(() => [
    { label: t('fleet.corporationPap.periods.currentMonth'), value: 'current_month' as const },
    { label: t('fleet.corporationPap.periods.lastMonth'), value: 'last_month' as const },
    { label: t('fleet.corporationPap.periods.atYear'), value: 'at_year' as const },
    { label: t('fleet.corporationPap.periods.all'), value: 'all' as const }
  ])

  const corpTickerOptions = computed(() =>
    Array.from(
      new Set([
        ...defaultTickers,
        ...filters.corpTickers.map((ticker) => ticker.trim()).filter(Boolean)
      ])
    )
  )

  const statsCards = computed(() => [
    {
      label: t('fleet.corporationPap.stats.filteredPap'),
      value: formatPap(overview.value.filtered_pap_total)
    },
    {
      label: t('fleet.corporationPap.stats.filteredStratOp'),
      value: formatPap(overview.value.filtered_strat_op_total)
    },
    {
      label: t('fleet.corporationPap.stats.allPap'),
      value: formatPap(overview.value.all_pap_total)
    },
    {
      label: t('fleet.corporationPap.stats.users'),
      value: String(overview.value.filtered_user_count)
    }
  ])

  const formatPap = (value: number) =>
    new Intl.NumberFormat(undefined, {
      minimumFractionDigits: Number.isInteger(value) ? 0 : 1,
      maximumFractionDigits: 1
    }).format(value ?? 0)

  const columns = computed<ColumnOption<Api.Fleet.CorporationPapSummaryItem>[]>(() => [
    { type: 'globalIndex', width: 60, label: '#' },
    {
      prop: 'corp_ticker',
      label: t('fleet.corporationPap.columns.corpTicker'),
      minWidth: 140,
      align: 'center'
    },
    {
      prop: 'nickname',
      label: t('fleet.corporationPap.columns.nickname'),
      minWidth: 180,
      showOverflowTooltip: true,
      formatter: (row) =>
        h('span', { class: row.nickname ? '' : 'text-gray-400' }, row.nickname || '-')
    },
    {
      prop: 'main_character_name',
      label: t('fleet.corporationPap.columns.mainCharacter'),
      minWidth: 220,
      showOverflowTooltip: true
    },
    {
      prop: 'character_count',
      label: t('fleet.corporationPap.columns.characterCount'),
      minWidth: 140,
      align: 'center'
    },
    {
      prop: 'strat_op_paps',
      label: t('fleet.corporationPap.columns.stratOpPaps'),
      minWidth: 160,
      align: 'center',
      formatter: (row) =>
        h(ElTag, { type: 'warning', size: 'small' }, () => formatPap(row.strat_op_paps))
    },
    {
      prop: 'skirmish_paps',
      label: t('fleet.corporationPap.columns.skirmishPaps'),
      minWidth: 160,
      align: 'center',
      formatter: (row) =>
        h(ElTag, { type: 'success', size: 'small' }, () => formatPap(row.skirmish_paps))
    },
    {
      prop: 'alliance_strat_paps',
      label: t('fleet.corporationPap.columns.allianceStratPaps'),
      minWidth: 180,
      align: 'center',
      formatter: (row) =>
        h(ElTag, { type: 'info', size: 'small' }, () => formatPap(row.alliance_strat_paps))
    }
  ])

  const emptyOverview = (): Api.Fleet.CorporationPapOverview => ({
    filtered_pap_total: 0,
    filtered_strat_op_total: 0,
    all_pap_total: 0,
    filtered_user_count: 0,
    period: filters.period
  })

  async function loadData() {
    loading.value = true
    try {
      const params: Api.Fleet.CorporationPapSummaryParams = {
        current: pagination.current,
        size: pagination.size,
        period: filters.period,
        corp_tickers: filters.corpTickers
          .map((ticker) => ticker.trim())
          .filter(Boolean)
          .join(',')
      }

      if (filters.period === 'at_year') {
        params.year = filters.year
      }

      const result = await fetchCorporationPapSummary(params)
      records.value = result?.list ?? []
      pagination.total = result?.total ?? 0
      pagination.current = result?.page ?? pagination.current
      pagination.size = result?.pageSize ?? pagination.size
      overview.value = result?.overview ?? emptyOverview()
    } catch {
      records.value = []
      pagination.total = 0
      overview.value = emptyOverview()
    } finally {
      loading.value = false
    }
  }

  function handleSearch() {
    pagination.current = 1
    loadData()
  }

  function handleReset() {
    filters.period = 'last_month'
    filters.year = currentYear
    filters.corpTickers = [...defaultTickers]
    pagination.current = 1
    pagination.size = 200
    loadData()
  }

  function handleCurrentChange(current: number) {
    pagination.current = current
    loadData()
  }

  function handleSizeChange(size: number) {
    pagination.current = 1
    pagination.size = size
    loadData()
  }

  watch(
    () => filters.period,
    (period) => {
      if (period === 'at_year' && (!filters.year || filters.year < 2003)) {
        filters.year = currentYear
      }
    }
  )

  onMounted(() => {
    loadData()
  })
</script>

<style scoped lang="scss">
  .corporation-pap-page {
    gap: 12px;
  }

  .filter-toolbar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    flex-wrap: wrap;
  }

  .filter-toolbar__main {
    display: flex;
    align-items: center;
    gap: 12px;
    flex: 1 1 560px;
    flex-wrap: wrap;
  }

  .filter-toolbar__actions {
    display: flex;
    align-items: center;
    gap: 12px;
    flex-wrap: wrap;
  }

  .period-filter,
  .year-filter {
    width: 180px;
  }

  .ticker-filter {
    width: 300px;
    max-width: 100%;
  }

  .stats-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
    gap: 12px;
  }

  .stat-card {
    border-radius: calc(var(--custom-radius) / 2 + 2px) !important;
  }

  .stat-label {
    margin-bottom: 10px;
    color: var(--el-text-color-secondary);
    font-size: 14px;
  }

  .stat-value {
    font-size: 42px;
    font-weight: 700;
    line-height: 1;
    color: var(--el-text-color-primary);
  }

  .table-card {
    flex: 1;
    min-height: 0;
    display: flex;
    flex-direction: column;
    border-radius: calc(var(--custom-radius) / 2 + 2px) !important;

    :deep(.el-card__body) {
      flex: 1;
      min-height: 0;
      display: flex;
      flex-direction: column;
      overflow: hidden;
    }
  }

  .table-wrap {
    flex: 1;
    min-height: 0;
    overflow: auto;
  }
  @media (max-width: 768px) {
    .ticker-filter,
    .period-filter,
    .year-filter {
      width: 100%;
    }
  }
</style>
