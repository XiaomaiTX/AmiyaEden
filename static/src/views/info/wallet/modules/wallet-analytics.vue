<template>
  <ElCard class="art-card mt-3" shadow="never">
    <template #header>
      <div class="flex items-center justify-between flex-wrap gap-3">
        <div class="flex items-center gap-3 flex-wrap">
          <span class="font-medium">{{ $t('info.walletAnalytics.title') }}</span>
          <ElButton
            v-for="preset in WALLET_ANALYTICS_PRESETS"
            :key="preset"
            size="small"
            :type="activePreset === preset ? 'primary' : 'default'"
            :disabled="!hasAvailableRange"
            @click="onPresetChange(preset)"
          >
            {{ $t(PRESET_LABEL_KEYS[preset]) }}
          </ElButton>
          <ElDatePicker
            v-model="dateRange"
            type="daterange"
            size="small"
            :start-placeholder="$t('info.walletAnalytics.rangeStart')"
            :end-placeholder="$t('info.walletAnalytics.rangeEnd')"
            format="YYYY-MM-DD"
            value-format="YYYY-MM-DD"
            :clearable="false"
            :disabled-date="isDisabledDate"
            class="!w-60"
            @change="onDateRangeChange"
          />
        </div>
        <div class="text-xs text-g-500 text-right leading-5">
          <p v-if="dateRange">
            {{
              $t('info.walletAnalytics.rangeSummary', {
                from: dateRange[0],
                to: dateRange[1],
                count: summary?.entry_count ?? 0
              })
            }}
          </p>
          <p>{{ $t('info.walletAnalytics.utcNote') }}</p>
        </div>
      </div>
    </template>

    <div v-if="!characterId" class="flex-cc py-12 text-sm text-g-500">
      {{ $t('info.walletAnalytics.selectCharacter') }}
    </div>
    <div v-else-if="loadError" class="flex-cc py-12 text-sm text-red-500">{{ loadError }}</div>
    <div v-else-if="!loading && !hasAvailableRange" class="flex-cc py-12 text-sm text-g-500">
      {{ $t('info.walletAnalytics.noData') }}
    </div>
    <div v-else v-loading="loading">
      <!-- 区间总览 -->
      <div class="grid grid-cols-2 lg:grid-cols-3 xl:grid-cols-6 gap-3">
        <div
          v-for="stat in summaryStats"
          :key="stat.key"
          class="rounded-lg border border-g-300 px-3 py-2"
        >
          <p class="text-xs text-g-500">{{ stat.label }}</p>
          <p class="text-base font-medium mt-1" :class="stat.className">{{ stat.value }}</p>
          <p v-if="stat.detail" class="text-xs text-g-500 mt-0.5 break-all">{{ stat.detail }}</p>
        </div>
      </div>

      <!-- 每日余额 / 每日收支：按 EVE 时间（UTC）自然日 -->
      <div class="grid grid-cols-1 xl:grid-cols-2 gap-x-4 gap-y-2 mt-4">
        <div>
          <p class="text-sm text-g-600 mb-1">{{ $t('info.walletAnalytics.balanceTrend') }}</p>
          <ArtLineChart
            height="15rem"
            :data="balanceSeries"
            :xAxisData="dayLabels"
            :showAreaColor="true"
            :valueFormatter="formatIskSmart"
            :loading="loading"
          />
        </div>
        <div>
          <p class="text-sm text-g-600 mb-1">{{ $t('info.walletAnalytics.dailyFlow') }}</p>
          <ArtDualBarCompareChart
            height="15rem"
            :positiveData="flowSeries.income"
            :negativeData="flowSeries.expense"
            :xAxisData="dayLabels"
            :positiveName="$t('info.walletAnalytics.income')"
            :negativeName="$t('info.walletAnalytics.expense')"
            :yAxisMin="-flowAxisBound"
            :yAxisMax="flowAxisBound"
            :showLegend="true"
            :valueFormatter="formatIskSmart"
            :colors="DAILY_FLOW_COLORS"
            :loading="loading"
          />
        </div>
        <div>
          <p class="text-sm text-g-600 mb-1">{{ $t('info.walletAnalytics.incomeByType') }}</p>
          <ArtHBarChart
            height="15rem"
            :data="incomeTypeSeries.values"
            :xAxisData="incomeTypeLabels"
            :valueFormatter="formatIskSmart"
            :reverseCategoryAxis="true"
            :loading="loading"
          />
        </div>
        <div>
          <p class="text-sm text-g-600 mb-1">{{ $t('info.walletAnalytics.expenseByType') }}</p>
          <ArtHBarChart
            height="15rem"
            :data="expenseTypeSeries.values"
            :xAxisData="expenseTypeLabels"
            :valueFormatter="formatIskSmart"
            :reverseCategoryAxis="true"
            :loading="loading"
          />
        </div>
      </div>
    </div>
  </ElCard>
</template>

<script setup lang="ts">
  import { useI18n } from 'vue-i18n'
  import { ElButton, ElCard, ElDatePicker } from 'element-plus'

  import ArtDualBarCompareChart from '@/components/core/charts/art-dual-bar-compare-chart/index.vue'
  import ArtHBarChart from '@/components/core/charts/art-h-bar-chart/index.vue'
  import ArtLineChart from '@/components/core/charts/art-line-chart/index.vue'
  import { fetchInfoWalletAnalytics } from '@/api/eve-info'
  import { formatIskPlain, formatIskSmart } from '@/utils/common'

  import { formatJournalTypeLabel } from './wallet-journal-type'
  import {
    WALLET_ANALYTICS_PRESETS,
    buildBalanceSeries,
    buildDailyFlowSeries,
    buildRefTypeSeries,
    resolveAnalyticsRange,
    resolveFlowAxisBound,
    type WalletAnalyticsPreset,
    type WalletAnalyticsRange
  } from './wallet-analytics.utils'

  defineOptions({ name: 'WalletAnalytics' })

  const props = defineProps<{
    /** 当前人物 ID，由页面的人物切换器提供 */
    characterId?: number
  }>()

  const { t } = useI18n()

  /** 预设按钮文案 key（显式映射，避免把 '7d' 这类以数字开头的值拼进 i18n key） */
  const PRESET_LABEL_KEYS: Record<WalletAnalyticsPreset, string> = {
    '1d': 'info.walletAnalytics.preset1d',
    '7d': 'info.walletAnalytics.preset7d',
    '30d': 'info.walletAnalytics.preset30d',
    '90d': 'info.walletAnalytics.preset90d',
    all: 'info.walletAnalytics.presetAll'
  }

  /**
   * 每日收支配色：收入绿、支出红（顺序与图表 series 一致：0=收入，1=支出）
   *
   * ECharts 底层 zrender 的颜色解析器不支持 oklch()（主题变量 `--art-*` 均为 oklch），
   * 故这里用十六进制常量，取值与页面汇总卡的 Tailwind green-600 / red-500 保持一致，
   * 明暗两种主题下均可读。
   */
  const DAILY_FLOW_COLORS = ['#16a34a', '#ef4444']

  const loading = ref(false)
  const loadError = ref('')
  const analytics = ref<Api.EveInfo.WalletAnalyticsResponse | null>(null)
  /** null 表示用户手动选定的自定义区间，此时不高亮任何预设 */
  const activePreset = ref<WalletAnalyticsPreset | null>('all')
  const dateRange = ref<[string, string] | null>(null)

  const summary = computed(() => analytics.value?.summary ?? null)
  const hasAvailableRange = computed(
    () => Boolean(summary.value?.available_from) && Boolean(summary.value?.available_to)
  )

  const balanceSeries = computed(() => buildBalanceSeries(analytics.value?.daily_series ?? []))
  const flowSeries = computed(() => buildDailyFlowSeries(analytics.value?.daily_series ?? []))
  const dayLabels = computed(() => flowSeries.value.labels)
  const flowAxisBound = computed(() =>
    resolveFlowAxisBound(flowSeries.value.income, flowSeries.value.expense)
  )

  const incomeTypeSeries = computed(() =>
    buildRefTypeSeries(analytics.value?.ref_type_breakdown ?? [], 'income')
  )
  const expenseTypeSeries = computed(() =>
    buildRefTypeSeries(analytics.value?.ref_type_breakdown ?? [], 'expense')
  )
  const incomeTypeLabels = computed(() =>
    incomeTypeSeries.value.refTypes.map((refType) => formatJournalTypeLabel(refType, t))
  )
  const expenseTypeLabels = computed(() =>
    expenseTypeSeries.value.refTypes.map((refType) => formatJournalTypeLabel(refType, t))
  )

  const summaryStats = computed(() => {
    const current = summary.value
    if (!current) {
      return []
    }
    return [
      {
        key: 'income',
        label: t('info.walletAnalytics.totalIncome'),
        value: formatIskSmart(current.total_income),
        detail: formatIskPlain(current.total_income),
        className: 'text-green-600'
      },
      {
        key: 'expense',
        label: t('info.walletAnalytics.totalExpense'),
        value: formatIskSmart(current.total_expense),
        detail: formatIskPlain(current.total_expense),
        className: 'text-red-500'
      },
      {
        key: 'net',
        label: t('info.walletAnalytics.totalNet'),
        value: formatIskSmart(current.total_net),
        detail: formatIskPlain(current.total_net),
        className: current.total_net >= 0 ? 'text-green-600' : 'text-red-500'
      },
      {
        key: 'tax',
        label: t('info.walletAnalytics.totalTax'),
        value: formatIskSmart(current.total_tax),
        detail: formatIskPlain(current.total_tax),
        className: 'text-g-700'
      },
      {
        key: 'opening',
        label: t('info.walletAnalytics.openingBalance'),
        value: formatIskSmart(current.opening_balance),
        detail: formatIskPlain(current.opening_balance),
        className: 'text-g-700'
      },
      {
        key: 'closing',
        label: t('info.walletAnalytics.closingBalance'),
        value: formatIskSmart(current.closing_balance),
        detail: formatIskPlain(current.closing_balance),
        className: 'text-g-700'
      }
    ]
  })

  /** 日期选择器按本地日历给出单元格日期；与可得范围比较时取本地日历日，实际查询区间以后端钳制为准 */
  const toLocalIsoDay = (date: Date): string =>
    `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`

  const isDisabledDate = (date: Date): boolean => {
    const day = toLocalIsoDay(date)
    const availableFrom = summary.value?.available_from
    const availableTo = summary.value?.available_to
    if (availableFrom && day < availableFrom) {
      return true
    }
    if (availableTo && day > availableTo) {
      return true
    }
    return false
  }

  const clampToAvailable = (from: string, to: string): WalletAnalyticsRange | null => {
    const availableFrom = summary.value?.available_from
    const availableTo = summary.value?.available_to
    if (!availableFrom || !availableTo) {
      return null
    }
    const start = from < availableFrom ? availableFrom : from
    const end = to > availableTo ? availableTo : to
    return end < start ? null : { from: start, to: end }
  }

  const loadAnalytics = async (range: WalletAnalyticsRange | null = null) => {
    const characterID = props.characterId
    if (!characterID) {
      analytics.value = null
      dateRange.value = null
      loadError.value = ''
      return
    }

    loading.value = true
    loadError.value = ''
    try {
      const res = await fetchInfoWalletAnalytics({
        character_id: characterID,
        from: range?.from,
        to: range?.to
      })
      analytics.value = res ?? null

      const availableFrom = res?.summary?.available_from ?? ''
      const availableTo = res?.summary?.available_to ?? ''
      const effective =
        range ?? resolveAnalyticsRange(activePreset.value ?? 'all', availableFrom, availableTo)
      dateRange.value = effective ? [effective.from, effective.to] : null
    } catch (error: unknown) {
      analytics.value = null
      dateRange.value = null
      loadError.value =
        (error as { message?: string }).message || t('info.walletAnalytics.loadFailed')
    } finally {
      loading.value = false
    }
  }

  const onPresetChange = (preset: WalletAnalyticsPreset) => {
    const range = resolveAnalyticsRange(
      preset,
      summary.value?.available_from ?? '',
      summary.value?.available_to ?? ''
    )
    if (!range) {
      return
    }
    activePreset.value = preset
    void loadAnalytics(range)
  }

  const onDateRangeChange = (value: [string, string] | null) => {
    if (!value || value.length !== 2) {
      return
    }
    const range = clampToAvailable(value[0], value[1])
    if (!range) {
      return
    }
    activePreset.value = null
    void loadAnalytics(range)
  }

  watch(
    () => props.characterId,
    (characterID) => {
      activePreset.value = 'all'
      analytics.value = null
      dateRange.value = null
      if (characterID) {
        void loadAnalytics()
      }
    },
    { immediate: true }
  )
</script>
