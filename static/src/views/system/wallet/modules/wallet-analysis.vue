<template>
  <div class="wallet-analysis">
    <ElCard shadow="never">
      <div class="filter-row">
        <ElDatePicker
          v-model="dateRange"
          type="daterange"
          value-format="YYYY-MM-DD"
          format="YYYY-MM-DD"
          :start-placeholder="$t('walletAdmin.analysis.startDate')"
          :end-placeholder="$t('walletAdmin.analysis.endDate')"
          style="width: 280px"
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
          <ElTable :data="analytics?.top_inflow_users ?? []" stripe border max-height="320">
            <ElTableColumn
              prop="user_id"
              :label="$t('walletAdmin.transactions.userId')"
              width="100"
            />
            <ElTableColumn
              prop="character_name"
              :label="$t('walletAdmin.transactions.characterName')"
              min-width="160"
            />
            <ElTableColumn :label="$t('walletAdmin.analysis.amount')" min-width="140" align="right">
              <template #default="{ row }">{{ formatFuxiCoinAmount(row.amount) }}</template>
            </ElTableColumn>
          </ElTable>
        </ElCard>

        <ElCard shadow="never">
          <template #header>
            <span class="font-medium">{{ $t('walletAdmin.analysis.topOutflowUsers') }}</span>
          </template>
          <ElTable :data="analytics?.top_outflow_users ?? []" stripe border max-height="320">
            <ElTableColumn
              prop="user_id"
              :label="$t('walletAdmin.transactions.userId')"
              width="100"
            />
            <ElTableColumn
              prop="character_name"
              :label="$t('walletAdmin.transactions.characterName')"
              min-width="160"
            />
            <ElTableColumn :label="$t('walletAdmin.analysis.amount')" min-width="140" align="right">
              <template #default="{ row }">{{ formatFuxiCoinAmount(row.amount) }}</template>
            </ElTableColumn>
          </ElTable>
        </ElCard>
      </div>

      <ElCard shadow="never" class="mt-4">
        <template #header>
          <span class="font-medium">{{ $t('walletAdmin.analysis.anomalies') }}</span>
        </template>
        <ElCollapse>
          <ElCollapseItem :title="$t('walletAdmin.analysis.largeTransactions')" name="large">
            <ElTable
              :data="analytics?.anomalies.large_transactions ?? []"
              stripe
              border
              max-height="280"
            >
              <ElTableColumn prop="id" label="ID" width="90" />
              <ElTableColumn
                prop="user_id"
                :label="$t('walletAdmin.transactions.userId')"
                width="100"
              />
              <ElTableColumn
                prop="character_name"
                :label="$t('walletAdmin.transactions.characterName')"
                min-width="140"
              />
              <ElTableColumn prop="ref_type" :label="$t('common.type')" min-width="120" />
              <ElTableColumn
                :label="$t('walletAdmin.analysis.amount')"
                min-width="140"
                align="right"
              >
                <template #default="{ row }">{{ formatFuxiCoinAmount(row.amount) }}</template>
              </ElTableColumn>
              <ElTableColumn prop="created_at" :label="$t('common.time')" min-width="180" />
            </ElTable>
          </ElCollapseItem>

          <ElCollapseItem :title="$t('walletAdmin.analysis.frequentAdjustments')" name="frequent">
            <ElTable
              :data="analytics?.anomalies.frequent_adjustments ?? []"
              stripe
              border
              max-height="280"
            >
              <ElTableColumn
                prop="target_uid"
                :label="$t('walletAdmin.logs.targetUser')"
                width="120"
              />
              <ElTableColumn
                prop="character_name"
                :label="$t('walletAdmin.transactions.characterName')"
                min-width="140"
              />
              <ElTableColumn
                prop="adjust_count"
                :label="$t('walletAdmin.analysis.adjustCount')"
                width="120"
              />
              <ElTableColumn
                :label="$t('walletAdmin.analysis.amountTotal')"
                min-width="140"
                align="right"
              >
                <template #default="{ row }">{{ formatFuxiCoinAmount(row.amount_total) }}</template>
              </ElTableColumn>
              <ElTableColumn
                prop="last_adjustment_time"
                :label="$t('walletAdmin.analysis.lastAdjustmentTime')"
                min-width="180"
              />
            </ElTable>
          </ElCollapseItem>

          <ElCollapseItem
            :title="$t('walletAdmin.analysis.operatorConcentration')"
            name="concentration"
          >
            <ElTable
              :data="analytics?.anomalies.operator_concentration ?? []"
              stripe
              border
              max-height="280"
            >
              <ElTableColumn
                prop="operator_id"
                :label="$t('walletAdmin.logs.operator')"
                width="120"
              />
              <ElTableColumn
                prop="operator_name"
                :label="$t('walletAdmin.analysis.operatorName')"
                min-width="140"
              />
              <ElTableColumn
                prop="count"
                :label="$t('walletAdmin.analysis.adjustCount')"
                width="120"
              />
              <ElTableColumn
                :label="$t('walletAdmin.analysis.amountTotal')"
                min-width="140"
                align="right"
              >
                <template #default="{ row }">{{ formatFuxiCoinAmount(row.amount_total) }}</template>
              </ElTableColumn>
              <ElTableColumn :label="$t('walletAdmin.analysis.ratio')" width="120">
                <template #default="{ row }">{{ `${(row.ratio * 100).toFixed(2)}%` }}</template>
              </ElTableColumn>
            </ElTable>
          </ElCollapseItem>
        </ElCollapse>
      </ElCard>
    </template>
  </div>
</template>

<script setup lang="ts">
  import { useI18n } from 'vue-i18n'
  import { ElMessage } from 'element-plus'
  import { formatFuxiCoinAmount } from '@utils/common'
  import { adminGetWalletAnalytics } from '@/api/sys-wallet'
  import ArtLineChart from '@/components/core/charts/art-line-chart/index.vue'
  import ArtRingChart from '@/components/core/charts/art-ring-chart/index.vue'

  defineOptions({ name: 'WalletAnalysis' })
  const { t } = useI18n()

  const today = new Date()
  const end = today.toISOString().slice(0, 10)
  const startDate = new Date(today.getTime() - 29 * 24 * 3600 * 1000).toISOString().slice(0, 10)
  const dateRange = ref<[string, string]>([startDate, end])
  const refTypes = ref<string[]>([])
  const userKeyword = ref('')
  const loading = ref(false)
  const analytics = ref<Api.SysWallet.WalletAnalytics | null>(null)

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

  const loadData = async () => {
    if (!dateRange.value?.[0] || !dateRange.value?.[1]) return
    loading.value = true
    try {
      analytics.value = await adminGetWalletAnalytics({
        start_date: dateRange.value[0],
        end_date: dateRange.value[1],
        ref_types: refTypes.value.length ? refTypes.value : undefined,
        user_keyword: userKeyword.value.trim() || undefined,
        top_n: 10
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
    gap: 12px;
  }

  .filter-row {
    display: flex;
    gap: 12px;
    flex-wrap: wrap;
    align-items: center;
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
