<template>
  <div class="corp-npc-kills-page">
    <!-- 日期范围筛选 -->
    <ElCard class="art-card" shadow="never">
      <div class="flex items-center gap-4 flex-wrap">
        <ElDatePicker
          v-model="dateRange"
          type="daterange"
          :start-placeholder="$t('npcKill.startDate')"
          :end-placeholder="$t('npcKill.endDate')"
          format="YYYY-MM-DD"
          value-format="YYYY-MM-DD"
          style="width: 280px"
        />
        <ElSelect
          v-model="corpTickers"
          style="width: 300px"
          multiple
          filterable
          allow-create
          default-first-option
          collapse-tags
          collapse-tags-tooltip
          :reserve-keyword="false"
          :placeholder="$t('fleet.corporationPap.filters.corpTickers')"
        >
          <ElOption
            v-for="ticker in corpTickerOptions"
            :key="ticker"
            :label="ticker"
            :value="ticker"
          />
        </ElSelect>

        <ElButton type="primary" :loading="loading" @click="handleSearch">
          {{ $t('npcKill.search') }}
        </ElButton>
        <ElButton @click="handleReset">{{ $t('npcKill.reset') }}</ElButton>
        <ElSelect
          v-model="selectedRefTypes"
          multiple
          collapse-tags
          collapse-tags-tooltip
          :placeholder="$t('npcKill.filters.refTypes')"
          style="width: 260px"
        >
          <ElOption
            v-for="option in refTypeOptions"
            :key="option.value"
            :label="option.label"
            :value="option.value"
          />
        </ElSelect>
        <ElSelect
          v-model="solarSystemIdInputs"
          multiple
          filterable
          allow-create
          default-first-option
          collapse-tags
          collapse-tags-tooltip
          :reserve-keyword="false"
          :placeholder="$t('npcKill.filters.solarSystemIds')"
          style="width: 260px"
        />
        <ElSelect
          v-model="userIdInputs"
          multiple
          filterable
          allow-create
          default-first-option
          collapse-tags
          collapse-tags-tooltip
          :reserve-keyword="false"
          :placeholder="$t('npcKill.filters.userIds')"
          style="width: 220px"
        />
        <ElSelect
          v-model="characterIdInputs"
          multiple
          filterable
          allow-create
          default-first-option
          collapse-tags
          collapse-tags-tooltip
          :reserve-keyword="false"
          :placeholder="$t('npcKill.filters.characterIds')"
          style="width: 220px"
        />
        <ElInputNumber
          v-model="minAmount"
          :placeholder="$t('npcKill.filters.minAmount')"
          :min="0"
          :controls="false"
          style="width: 180px"
        />
        <ElInputNumber
          v-model="maxAmount"
          :placeholder="$t('npcKill.filters.maxAmount')"
          :min="0"
          :controls="false"
          style="width: 180px"
        />
      </div>
    </ElCard>

    <!-- 总览卡片 -->
    <div v-if="reportData" class="grid grid-cols-2 md:grid-cols-2 lg:grid-cols-4 gap-4 my-4">
      <ElCard shadow="never" class="text-center">
        <p class="text-sm text-gray-500">{{ $t('npcKill.totalBounty') }}</p>
        <p class="text-xl font-bold text-green-600 mt-1">{{
          formatIskPlain(reportData.summary.total_bounty)
        }}</p>
      </ElCard>
      <ElCard shadow="never" class="text-center">
        <p class="text-sm text-gray-500">{{ $t('npcKill.totalTax') }}</p>
        <p class="text-xl font-bold text-red-500 mt-1">{{
          formatIskPlain(reportData.summary.total_tax)
        }}</p>
      </ElCard>
      <ElCard shadow="never" class="text-center">
        <p class="text-sm text-gray-500">{{ $t('npcKill.actualIncome') }}</p>
        <p class="text-xl font-bold text-green-600 mt-1">{{
          formatIskPlain(reportData.summary.actual_income)
        }}</p>
      </ElCard>
      <ElCard shadow="never" class="text-center">
        <p class="text-sm text-gray-500">{{ $t('npcKill.totalRecords') }}</p>
        <p class="text-xl font-bold mt-1">{{ reportData.summary.total_records }}</p>
      </ElCard>
    </div>

    <!-- 成员统计表格 -->
    <ElCard v-if="reportData" shadow="never" class="art-table-card mb-4">
      <template #header>
        <span class="font-medium">{{ $t('npcKill.members') }}</span>
      </template>
      <ElTable
        :data="reportData.members"
        stripe
        border
        max-height="500"
        :default-sort="{ prop: 'actual_income', order: 'descending' }"
      >
        <ElTableColumn type="index" width="55" label="#" align="center" />
        <ElTableColumn
          prop="display_name"
          :label="$t('npcKill.userDisplayName')"
          min-width="140"
          show-overflow-tooltip
        />
        <ElTableColumn
          prop="character_count"
          :label="$t('npcKill.characterCount')"
          width="100"
          align="right"
          sortable
        />
        <ElTableColumn
          prop="total_bounty"
          :label="$t('npcKill.totalBounty')"
          width="160"
          align="right"
          sortable
        >
          <template #default="{ row }">
            <span class="text-green-600 font-medium">{{ formatIskPlain(row.total_bounty) }}</span>
          </template>
        </ElTableColumn>
        <ElTableColumn
          prop="total_tax"
          :label="$t('npcKill.totalTax')"
          width="140"
          align="right"
          sortable
        >
          <template #default="{ row }">
            <span class="text-red-500 font-medium">{{ formatIskPlain(row.total_tax) }}</span>
          </template>
        </ElTableColumn>
        <ElTableColumn
          prop="actual_income"
          :label="$t('npcKill.actualIncome')"
          width="160"
          align="right"
          sortable
        >
          <template #default="{ row }">
            <span class="text-green-600 font-bold">{{ formatIskPlain(row.actual_income) }}</span>
          </template>
        </ElTableColumn>
        <ElTableColumn
          prop="record_count"
          :label="$t('npcKill.recordCount')"
          width="100"
          align="right"
          sortable
        />
      </ElTable>
    </ElCard>

    <div v-if="reportData" class="grid grid-cols-1 lg:grid-cols-2 gap-4 mb-4">
      <!-- 按地点分类 -->
      <ElCard shadow="never" class="art-table-card">
        <template #header>
          <span class="font-medium">{{ $t('npcKill.bySystem') }}</span>
        </template>
        <ElTable :data="reportData.by_system" stripe border max-height="400">
          <ElTableColumn type="index" width="55" label="#" align="center" />
          <ElTableColumn
            prop="solar_system_name"
            :label="$t('npcKill.solarSystem')"
            min-width="160"
            show-overflow-tooltip
          />
          <ElTableColumn
            prop="count"
            :label="$t('npcKill.systemCount')"
            width="100"
            align="right"
            sortable
          />
          <ElTableColumn
            prop="amount"
            :label="$t('npcKill.systemAmount')"
            width="160"
            align="right"
            sortable
          >
            <template #default="{ row }">
              <span class="text-green-600 font-medium">{{ formatIskPlain(row.amount) }}</span>
            </template>
          </ElTableColumn>
        </ElTable>
      </ElCard>

      <!-- 时间趋势 -->
      <ElCard shadow="never" class="art-table-card">
        <template #header>
          <span class="font-medium">{{ $t('npcKill.trend') }}</span>
        </template>
        <ElTable :data="reportData.trend" stripe border max-height="400">
          <ElTableColumn prop="date" :label="$t('npcKill.trendDate')" width="140" />
          <ElTableColumn
            prop="amount"
            :label="$t('npcKill.trendAmount')"
            min-width="160"
            align="right"
            sortable
          >
            <template #default="{ row }">
              <span class="text-green-600 font-medium">{{ formatIskPlain(row.amount) }}</span>
            </template>
          </ElTableColumn>
          <ElTableColumn
            prop="count"
            :label="$t('npcKill.trendCount')"
            width="100"
            align="right"
            sortable
          />
        </ElTable>
      </ElCard>
    </div>
  </div>
</template>

<script setup lang="ts">
  import { ElDatePicker, ElOption, ElSelect, ElInputNumber } from 'element-plus'
  import { fetchCorpNpcKills } from '@/api/npc-kill'
  import { formatIskPlain } from '@/utils/common'
  import { useI18n } from 'vue-i18n'

  defineOptions({ name: 'CorpNpcKillReport' })

  const { t } = useI18n()
  const defaultTickers = ['FUXI', 'FMA.1']

  // ─── 状态 ───
  const dateRange = ref<[string, string] | null>(null)
  const corpTickers = ref<string[]>([...defaultTickers])
  const selectedRefTypes = ref<string[]>([])
  const solarSystemIdInputs = ref<string[]>([])
  const userIdInputs = ref<string[]>([])
  const characterIdInputs = ref<string[]>([])
  const minAmount = ref<number | undefined>()
  const maxAmount = ref<number | undefined>()
  const reportData = ref<Api.NpcKill.NpcKillCorpResponse | null>(null)
  const loading = ref(false)
  const corpTickerOptions = computed(() =>
    Array.from(
      new Set([
        ...defaultTickers,
        ...corpTickers.value.map((ticker) => ticker.trim()).filter(Boolean)
      ])
    )
  )
  const REF_TYPE_CONFIG: Record<string, string> = {
    bounty_prizes: 'npcKill.refTypes.bounty_prizes',
    ess_escrow_transfer: 'npcKill.refTypes.ess_escrow_transfer',
    corporate_reward_payout: 'npcKill.refTypes.corporate_reward_payout',
    agent_mission_reward: 'npcKill.refTypes.agent_mission_reward'
  }
  const refTypeOptions = computed(() =>
    Object.keys(REF_TYPE_CONFIG).map((value) => ({ value, label: t(REF_TYPE_CONFIG[value]) }))
  )
  const parseNumericInputs = (values: string[]) =>
    values.map((value) => Number(value)).filter((value) => Number.isInteger(value) && value > 0)

  // ─── 加载数据 ───
  const loadData = async () => {
    loading.value = true
    try {
      const params: Api.NpcKill.NpcKillCorpRequest = {}
      if (dateRange.value) {
        params.start_date = dateRange.value[0]
        params.end_date = dateRange.value[1]
      }
      const corpTickerParam = corpTickers.value
        .map((ticker) => ticker.trim())
        .filter(Boolean)
        .join(',')
      if (corpTickerParam) {
        params.corp_tickers = corpTickerParam
      }
      const solarSystemIds = parseNumericInputs(solarSystemIdInputs.value)
      const userIds = parseNumericInputs(userIdInputs.value)
      const characterIds = parseNumericInputs(characterIdInputs.value)
      if (selectedRefTypes.value.length > 0) {
        params.ref_types = [...selectedRefTypes.value]
      }
      if (solarSystemIds.length > 0) {
        params.solar_system_ids = solarSystemIds
      }
      if (userIds.length > 0) {
        params.user_ids = userIds
      }
      if (characterIds.length > 0) {
        params.character_ids = characterIds
      }
      params.min_amount = minAmount.value
      params.max_amount = maxAmount.value
      reportData.value = (await fetchCorpNpcKills(params)) ?? null
    } catch {
      reportData.value = null
    } finally {
      loading.value = false
    }
  }

  const handleSearch = () => {
    loadData()
  }

  const handleReset = () => {
    dateRange.value = null
    corpTickers.value = [...defaultTickers]
    selectedRefTypes.value = []
    solarSystemIdInputs.value = []
    userIdInputs.value = []
    characterIdInputs.value = []
    minAmount.value = undefined
    maxAmount.value = undefined
    loadData()
  }

  // ─── 初始化 ───
  onMounted(() => {
    loadData()
  })
</script>
