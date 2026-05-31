<template>
  <div class="wallet-analysis">
    <ElCard shadow="never">
      <div class="filter-row">
        <ElDatePicker
          v-model="dateRange"
          class="filter-date-range"
          type="daterange"
          clearable
          value-format="YYYY-MM-DD"
          format="YYYY-MM-DD"
          :start-placeholder="$t('walletAdmin.analysis.startDateAllHint')"
          :end-placeholder="$t('walletAdmin.analysis.endDateAllHint')"
          :teleported="true"
        />
        <ElSelect
          v-model="refTypes"
          multiple
          collapse-tags
          collapse-tags-tooltip
          :placeholder="$t('walletAdmin.analysis.refTypes')"
          style="width: 320px"
        >
          <ElOption
            v-for="item in refTypeOptions"
            :key="item.value"
            :label="item.label"
            :value="item.value"
          />
        </ElSelect>
        <ElInput
          v-model="userKeyword"
          :placeholder="$t('walletAdmin.analysis.userKeyword')"
          clearable
          style="width: 240px"
          @keyup.enter="loadData"
        />
        <ElButton type="primary" :loading="loading" @click="loadData">
          {{ $t('common.search') }}
        </ElButton>
      </div>
    </ElCard>

    <div class="summary-grid">
      <ElCard v-for="card in summaryCards" :key="card.key" shadow="never" class="summary-card">
        <div class="summary-label">{{ card.label }}</div>
        <div class="summary-value">{{ card.value }}</div>
      </ElCard>
    </div>

    <ElEmpty
      v-if="!loading && isEmptyData"
      :description="$t('walletAdmin.analysis.empty')"
      class="my-4"
    />

    <template v-else>
      <ElCard shadow="never" class="mb-4">
        <template #header>
          <span class="font-medium">{{ $t('walletAdmin.analysis.dailyTrend') }}</span>
        </template>
        <ArtLineChart
          height="320px"
          :data="trendSeries"
          :x-axis-data="trendDates"
          :show-legend="true"
          :show-area-color="false"
        />
      </ElCard>

      <div class="grid-two">
        <ElCard shadow="never">
          <template #header>
            <span class="font-medium">{{ $t('walletAdmin.analysis.incomeByRefType') }}</span>
          </template>
          <ArtRingChart height="320px" :data="incomeRefTypeData" :show-legend="true" />
        </ElCard>

        <ElCard shadow="never">
          <template #header>
            <span class="font-medium">{{ $t('walletAdmin.analysis.expenseByRefType') }}</span>
          </template>
          <ArtRingChart height="320px" :data="expenseRefTypeData" :show-legend="true" />
        </ElCard>
      </div>

      <div class="grid-two mt-4">
        <ElCard shadow="never">
          <template #header>
            <span class="font-medium">{{ $t('walletAdmin.analysis.topInflowUsers') }}</span>
          </template>
          <ArtTable
            :data="sortedTopInflowUsers"
            :columns="topUserColumns"
            visual-variant="ledger"
            :height="320"
            :empty-text="$t('walletAdmin.analysis.empty')"
            @sort-change="handleTopInflowSortChange"
          />
        </ElCard>

        <ElCard shadow="never">
          <template #header>
            <span class="font-medium">{{ $t('walletAdmin.analysis.topOutflowUsers') }}</span>
          </template>
          <ArtTable
            :data="sortedTopOutflowUsers"
            :columns="topUserColumns"
            visual-variant="ledger"
            :height="320"
            :empty-text="$t('walletAdmin.analysis.empty')"
            @sort-change="handleTopOutflowSortChange"
          />
        </ElCard>
      </div>

      <ElCard shadow="never" class="mt-4">
        <template #header>
          <span class="font-medium">{{ $t('walletAdmin.analysis.anomalies') }}</span>
        </template>
        <ElCollapse>
          <ElCollapseItem :title="$t('walletAdmin.analysis.largeTransactions')" name="large">
            <ArtTable
              :data="sortedLargeTransactions"
              :columns="largeTransactionColumns"
              visual-variant="ledger"
              :height="280"
              :empty-text="$t('walletAdmin.analysis.empty')"
              @sort-change="handleLargeTransactionsSortChange"
            />
          </ElCollapseItem>

          <ElCollapseItem :title="$t('walletAdmin.analysis.frequentAdjustments')" name="frequent">
            <ArtTable
              :data="sortedFrequentAdjustments"
              :columns="frequentAdjustmentColumns"
              visual-variant="ledger"
              :height="280"
              :empty-text="$t('walletAdmin.analysis.empty')"
              @sort-change="handleFrequentAdjustmentsSortChange"
            />
          </ElCollapseItem>

          <ElCollapseItem
            :title="$t('walletAdmin.analysis.operatorConcentration')"
            name="concentration"
          >
            <ArtTable
              :data="sortedOperatorConcentration"
              :columns="operatorConcentrationColumns"
              visual-variant="ledger"
              :height="280"
              :empty-text="$t('walletAdmin.analysis.empty')"
              @sort-change="handleOperatorConcentrationSortChange"
            />
          </ElCollapseItem>
        </ElCollapse>
      </ElCard>
    </template>
  </div>
</template>

<script setup lang="ts">
  import { useI18n } from 'vue-i18n'
  import { ElMessage } from 'element-plus'
  import { ColumnOption } from '@/types'
  import { formatFuxiCoinAmount } from '@utils/common'
  import { adminGetWalletAnalytics } from '@/api/sys-wallet'
  import ArtLineChart from '@/components/core/charts/art-line-chart/index.vue'
  import ArtRingChart from '@/components/core/charts/art-ring-chart/index.vue'

  defineOptions({ name: 'WalletAnalysis' })
  const { t } = useI18n()

  const dateRange = ref<[string, string] | null>(null)
  const refTypes = ref<string[]>([])
  const userKeyword = ref('')
  const loading = ref(false)
  const analytics = ref<Api.SysWallet.WalletAnalytics | null>(null)
  type LocalSort = { prop: string; order: 'ascending' | 'descending' }
  const localSortState = reactive<{
    topInflow: LocalSort
    topOutflow: LocalSort
    largeTransactions: LocalSort
    frequentAdjustments: LocalSort
    operatorConcentration: LocalSort
  }>({
    topInflow: { prop: 'amount', order: 'descending' },
    topOutflow: { prop: 'amount', order: 'descending' },
    largeTransactions: { prop: 'amount', order: 'descending' },
    frequentAdjustments: { prop: 'adjust_count', order: 'descending' },
    operatorConcentration: { prop: 'amount_total', order: 'descending' }
  })

  const refTypeOptions = computed(() =>
    [
      'pap_reward',
      'pap_fc_salary',
      'admin_adjust',
      'admin_award',
      'manual',
      'srp_payout',
      'welfare_payout',
      'shop_purchase',
      'shop_refund',
      'newbro_captain_reward',
      'mentor_reward',
      'recruit_link_reward'
    ].map((value) => ({ value, label: t(`walletAdmin.refTypes.${value}`) }))
  )

  const summaryCards = computed(() => {
    const s = analytics.value?.summary
    return [
      {
        key: 'wallet_count',
        label: t('walletAdmin.analysis.walletCount'),
        value: s?.wallet_count ?? 0
      },
      {
        key: 'active_wallet_count',
        label: t('walletAdmin.analysis.activeWalletCount'),
        value: s?.active_wallet_count ?? 0
      },
      {
        key: 'total_balance',
        label: t('walletAdmin.analysis.totalBalance'),
        value: formatFuxiCoinAmount(s?.total_balance ?? 0)
      },
      {
        key: 'income_total',
        label: t('walletAdmin.analysis.incomeTotal'),
        value: formatFuxiCoinAmount(s?.income_total ?? 0)
      },
      {
        key: 'expense_total',
        label: t('walletAdmin.analysis.expenseTotal'),
        value: formatFuxiCoinAmount(s?.expense_total ?? 0)
      },
      {
        key: 'net_flow',
        label: t('walletAdmin.analysis.netFlow'),
        value: formatFuxiCoinAmount(s?.net_flow ?? 0)
      }
    ]
  })

  const trendDates = computed(() => (analytics.value?.daily_series ?? []).map((item) => item.date))
  const trendSeries = computed(() => [
    {
      name: t('walletAdmin.analysis.incomeTotal'),
      data: (analytics.value?.daily_series ?? []).map((item) => item.income)
    },
    {
      name: t('walletAdmin.analysis.expenseTotal'),
      data: (analytics.value?.daily_series ?? []).map((item) => item.expense)
    },
    {
      name: t('walletAdmin.analysis.netFlow'),
      data: (analytics.value?.daily_series ?? []).map((item) => item.net_flow)
    }
  ])

  const incomeRefTypeData = computed(() =>
    (analytics.value?.ref_type_breakdown ?? [])
      .filter((item) => item.income > 0)
      .map((item) => ({ name: t(`walletAdmin.refTypes.${item.ref_type}`), value: item.income }))
  )

  const expenseRefTypeData = computed(() =>
    (analytics.value?.ref_type_breakdown ?? [])
      .filter((item) => item.expense > 0)
      .map((item) => ({ name: t(`walletAdmin.refTypes.${item.ref_type}`), value: item.expense }))
  )

  const isEmptyData = computed(() => {
    const summary = analytics.value?.summary
    return (
      !summary ||
      (summary.wallet_count === 0 &&
        summary.active_wallet_count === 0 &&
        summary.income_total === 0 &&
        summary.expense_total === 0)
    )
  })

  const topUserColumns = computed<ColumnOption[]>(() => [
    {
      prop: 'user_id',
      label: t('walletAdmin.transactions.userId'),
      width: 100,
      sortable: true
    },
    { prop: 'character_name', label: t('walletAdmin.transactions.characterName'), minWidth: 160 },
    {
      prop: 'amount',
      label: t('walletAdmin.analysis.amount'),
      minWidth: 140,
      align: 'right',
      sortable: true,
      formatter: (row: { amount: number }) => h('span', {}, formatFuxiCoinAmount(row.amount))
    }
  ])

  const largeTransactionColumns = computed<ColumnOption[]>(() => [
    { prop: 'id', label: 'ID', width: 90, sortable: true },
    { prop: 'user_id', label: t('walletAdmin.transactions.userId'), width: 100, sortable: true },
    { prop: 'character_name', label: t('walletAdmin.transactions.characterName'), minWidth: 140 },
    { prop: 'ref_type', label: t('common.type'), minWidth: 120, sortable: true },
    {
      prop: 'amount',
      label: t('walletAdmin.analysis.amount'),
      minWidth: 140,
      align: 'right',
      sortable: true,
      formatter: (row: { amount: number }) => h('span', {}, formatFuxiCoinAmount(row.amount))
    },
    { prop: 'created_at', label: t('common.time'), minWidth: 180, sortable: true }
  ])

  const frequentAdjustmentColumns = computed<ColumnOption[]>(() => [
    { prop: 'target_uid', label: t('walletAdmin.logs.targetUser'), width: 120, sortable: true },
    { prop: 'character_name', label: t('walletAdmin.transactions.characterName'), minWidth: 140 },
    {
      prop: 'adjust_count',
      label: t('walletAdmin.analysis.adjustCount'),
      width: 120,
      sortable: true
    },
    {
      prop: 'amount_total',
      label: t('walletAdmin.analysis.amountTotal'),
      minWidth: 140,
      align: 'right',
      sortable: true,
      formatter: (row: { amount_total: number }) =>
        h('span', {}, formatFuxiCoinAmount(row.amount_total))
    },
    {
      prop: 'last_adjustment_time',
      label: t('walletAdmin.analysis.lastAdjustmentTime'),
      minWidth: 180,
      sortable: true
    }
  ])

  const operatorConcentrationColumns = computed<ColumnOption[]>(() => [
    { prop: 'operator_id', label: t('walletAdmin.logs.operator'), width: 120, sortable: true },
    { prop: 'operator_name', label: t('walletAdmin.analysis.operatorName'), minWidth: 140 },
    { prop: 'count', label: t('walletAdmin.analysis.adjustCount'), width: 120, sortable: true },
    {
      prop: 'amount_total',
      label: t('walletAdmin.analysis.amountTotal'),
      minWidth: 140,
      align: 'right',
      sortable: true,
      formatter: (row: { amount_total: number }) =>
        h('span', {}, formatFuxiCoinAmount(row.amount_total))
    },
    {
      prop: 'ratio',
      label: t('walletAdmin.analysis.ratio'),
      width: 120,
      sortable: true,
      formatter: (row: { ratio: number }) => h('span', {}, `${(row.ratio * 100).toFixed(2)}%`)
    }
  ])

  const sortList = <T extends Record<string, any>>(list: T[], sort: LocalSort): T[] => {
    const orderFactor = sort.order === 'ascending' ? 1 : -1
    return [...list].sort((a, b) => {
      const av = a[sort.prop]
      const bv = b[sort.prop]
      if (typeof av === 'number' && typeof bv === 'number') {
        return (av - bv) * orderFactor
      }
      const as = av == null ? '' : String(av)
      const bs = bv == null ? '' : String(bv)
      return as.localeCompare(bs) * orderFactor
    })
  }

  const sortedTopInflowUsers = computed(() =>
    sortList(analytics.value?.top_inflow_users ?? [], localSortState.topInflow)
  )
  const sortedTopOutflowUsers = computed(() =>
    sortList(analytics.value?.top_outflow_users ?? [], localSortState.topOutflow)
  )
  const sortedLargeTransactions = computed(() =>
    sortList(analytics.value?.anomalies.large_transactions ?? [], localSortState.largeTransactions)
  )
  const sortedFrequentAdjustments = computed(() =>
    sortList(
      analytics.value?.anomalies.frequent_adjustments ?? [],
      localSortState.frequentAdjustments
    )
  )
  const sortedOperatorConcentration = computed(() =>
    sortList(
      analytics.value?.anomalies.operator_concentration ?? [],
      localSortState.operatorConcentration
    )
  )

  const handleLocalSortChange = (
    key: keyof typeof localSortState,
    sort: { prop?: string; order?: 'ascending' | 'descending' | null }
  ) => {
    if (!sort.prop || !sort.order) {
      const defaults: Record<keyof typeof localSortState, LocalSort> = {
        topInflow: { prop: 'amount', order: 'descending' },
        topOutflow: { prop: 'amount', order: 'descending' },
        largeTransactions: { prop: 'amount', order: 'descending' },
        frequentAdjustments: { prop: 'adjust_count', order: 'descending' },
        operatorConcentration: { prop: 'amount_total', order: 'descending' }
      }
      localSortState[key] = defaults[key]
      return
    }
    localSortState[key] = { prop: sort.prop, order: sort.order }
  }
  const handleTopInflowSortChange = (sort: {
    prop?: string
    order?: 'ascending' | 'descending' | null
  }) => handleLocalSortChange('topInflow', sort)
  const handleTopOutflowSortChange = (sort: {
    prop?: string
    order?: 'ascending' | 'descending' | null
  }) => handleLocalSortChange('topOutflow', sort)
  const handleLargeTransactionsSortChange = (sort: {
    prop?: string
    order?: 'ascending' | 'descending' | null
  }) => handleLocalSortChange('largeTransactions', sort)
  const handleFrequentAdjustmentsSortChange = (sort: {
    prop?: string
    order?: 'ascending' | 'descending' | null
  }) => handleLocalSortChange('frequentAdjustments', sort)
  const handleOperatorConcentrationSortChange = (sort: {
    prop?: string
    order?: 'ascending' | 'descending' | null
  }) => handleLocalSortChange('operatorConcentration', sort)

  const loadData = async () => {
    loading.value = true
    try {
      const payload: Api.SysWallet.AnalyticsParams = {
        ref_types: refTypes.value.length ? refTypes.value : undefined,
        user_keyword: userKeyword.value.trim() || undefined,
        top_n: 10
      }
      if (dateRange.value?.[0] && dateRange.value?.[1]) {
        payload.start_date = dateRange.value[0]
        payload.end_date = dateRange.value[1]
      }
      analytics.value = await adminGetWalletAnalytics({
        ...payload
      })
    } catch (e: any) {
      ElMessage.error(e?.message ?? t('walletAdmin.messages.actionFailed'))
    } finally {
      loading.value = false
    }
  }

  onMounted(loadData)
</script>

<style scoped lang="scss">
  .wallet-analysis {
    display: flex;
    flex-direction: column;
    min-height: 0;
    gap: 12px;
  }

  .filter-row {
    display: flex;
    gap: 12px;
    flex-wrap: wrap;
    align-items: center;

    .filter-date-range {
      flex: 0 0 280px;
      min-width: 280px;
    }
  }

  .summary-grid {
    display: grid;
    grid-template-columns: repeat(6, minmax(0, 1fr));
    gap: 12px;
  }

  .summary-card {
    .summary-label {
      color: var(--el-text-color-secondary);
      font-size: 12px;
      margin-bottom: 8px;
    }

    .summary-value {
      font-size: 18px;
      font-weight: 600;
    }
  }

  .grid-two {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 12px;

    > * {
      min-width: 0;
    }
  }

  @media (max-width: 1280px) {
    .summary-grid {
      grid-template-columns: repeat(3, minmax(0, 1fr));
    }
  }

  @media (max-width: 900px) {
    .summary-grid,
    .grid-two {
      grid-template-columns: repeat(1, minmax(0, 1fr));
    }
  }
</style>
