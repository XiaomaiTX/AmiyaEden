<template>
  <div class="mentor-manage-page">
    <div class="mb-4 flex justify-end">
      <ElButton class="min-w-[120px]" :disabled="refreshDisabled" @click="handleRefresh">{{
        $t('common.refresh')
      }}</ElButton>
    </div>

    <ElTabs v-model="activeTab" @tab-change="handleTabChange">
      <ElTabPane :label="t('newbro.mentorManage.relationshipsTab')" name="relationships">
        <ElCard shadow="never" class="mb-4">
          <div class="flex items-center gap-3 flex-wrap">
            <ElInput
              v-model="filters.keyword"
              clearable
              style="width: 280px"
              :placeholder="t('newbro.mentorManage.keyword')"
              @keyup="handleRelationshipSearchKeyup"
            />
            <ElSelect v-model="filters.status" style="width: 180px">
              <ElOption
                v-for="option in statusOptions"
                :key="option.value"
                :label="option.label"
                :value="option.value"
              />
            </ElSelect>
            <ElButton type="primary" @click="handleSearch">{{ $t('common.search') }}</ElButton>
            <ElButton @click="handleReset">{{ $t('common.reset') }}</ElButton>
          </div>
        </ElCard>

        <ElCard shadow="never">
          <template #header>
            <span>{{ t('newbro.mentorManage.relationshipList') }}</span>
          </template>

          <ElTable :data="rows" v-loading="loading" stripe border>
            <ElTableColumn :label="t('newbro.mentorManage.mentorColumn')" min-width="240">
              <template #default="{ row }">
                <div class="flex items-center gap-3">
                  <ElAvatar
                    :src="buildEveCharacterPortraitUrl(row.mentor_character_id, 40)"
                    :size="40"
                  />
                  <div>
                    <div class="font-medium">{{ row.mentor_character_name }}</div>
                    <div class="text-sm text-gray-500">
                      {{ t('newbro.mentorManage.mentorNickname') }}:
                      {{ row.mentor_nickname || '-' }}
                    </div>
                  </div>
                </div>
              </template>
            </ElTableColumn>
            <ElTableColumn :label="t('newbro.mentorManage.menteeColumn')" min-width="240">
              <template #default="{ row }">
                <div class="flex items-center gap-3">
                  <ElAvatar
                    :src="buildEveCharacterPortraitUrl(row.mentee_character_id, 40)"
                    :size="40"
                  />
                  <div>
                    <div class="font-medium">{{ row.mentee_character_name }}</div>
                    <div class="text-sm text-gray-500">
                      {{ t('newbro.mentorManage.menteeNickname') }}:
                      {{ row.mentee_nickname || '-' }}
                    </div>
                  </div>
                </div>
              </template>
            </ElTableColumn>
            <ElTableColumn :label="t('newbro.mentorManage.status')" width="140">
              <template #default="{ row }">
                <ElTag :type="statusTagType(row.status)" effect="light">
                  {{ formatStatus(row.status) }}
                </ElTag>
              </template>
            </ElTableColumn>
            <ElTableColumn
              prop="applied_at"
              :label="t('newbro.mentorManage.appliedAt')"
              width="180"
            >
              <template #default="{ row }">{{ formatDateTime(row.applied_at) }}</template>
            </ElTableColumn>
            <ElTableColumn
              prop="responded_at"
              :label="t('newbro.mentorManage.respondedAt')"
              width="180"
            >
              <template #default="{ row }">{{ formatDateTime(row.responded_at) }}</template>
            </ElTableColumn>
            <ElTableColumn
              prop="revoked_at"
              :label="t('newbro.mentorManage.revokedAt')"
              width="180"
            >
              <template #default="{ row }">{{ formatDateTime(row.revoked_at) }}</template>
            </ElTableColumn>
            <ElTableColumn
              prop="graduated_at"
              :label="t('newbro.mentorManage.graduatedAt')"
              width="180"
            >
              <template #default="{ row }">{{ formatDateTime(row.graduated_at) }}</template>
            </ElTableColumn>
            <ElTableColumn :label="$t('common.operation')" width="140" fixed="right">
              <template #default="{ row }">
                <ElButton
                  v-if="canRevoke(row.status)"
                  type="danger"
                  size="small"
                  :disabled="revokingId === row.id"
                  @click="handleRevoke(row)"
                >
                  {{ revokeActionLabel(row.status) }}
                </ElButton>
                <span v-else class="text-gray-400">-</span>
              </template>
            </ElTableColumn>
          </ElTable>

          <div class="flex justify-end mt-4">
            <ElPagination
              background
              layout="total, sizes, prev, pager, next"
              :current-page="page.current"
              :page-size="page.size"
              :page-sizes="[20, 50, 100]"
              :total="page.total"
              @current-change="handleCurrentChange"
              @size-change="handleSizeChange"
            />
          </div>
        </ElCard>
      </ElTabPane>

      <ElTabPane :label="t('newbro.mentorManage.rewardRecordsTab')" name="reward-records">
        <ElCard shadow="never">
          <template #header>
            <div class="flex items-center justify-between gap-4 flex-wrap">
              <span>{{ t('newbro.mentorManage.rewardRecordsList') }}</span>
              <div class="flex items-center gap-3 flex-wrap">
                <ElInput
                  v-model="rewardHistoryKeyword"
                  clearable
                  style="width: 260px"
                  :placeholder="t('newbro.mentorManage.rewardKeyword')"
                  @clear="handleRewardHistorySearch"
                  @keyup="handleRewardHistorySearchKeyup"
                />
                <ElButton type="primary" @click="handleRewardHistorySearch">
                  {{ $t('common.search') }}
                </ElButton>
                <ElButton @click="handleRewardHistoryReset">{{ $t('common.reset') }}</ElButton>
              </div>
            </div>
          </template>

          <ArtTable
            :loading="rewardHistoryLoading"
            :data="rewardHistoryRows"
            :columns="rewardHistoryColumns"
            :pagination="rewardHistoryPage"
            :pagination-options="rewardHistoryPaginationOptions"
            visual-variant="ledger"
            @pagination:size-change="handleRewardHistorySizeChange"
            @pagination:current-change="handleRewardHistoryCurrentChange"
          />
          <ElEmpty
            v-if="!rewardHistoryLoading && rewardHistoryRows.length === 0"
            :description="t('newbro.mentorManage.noRewardRecords')"
          />
        </ElCard>
      </ElTabPane>

      <ElTabPane :label="t('newbro.mentorManage.rewardStagesTab')" name="reward-stages">
        <ElCard shadow="never" class="mb-4">
          <div class="flex items-center justify-between gap-4 flex-wrap">
            <div>
              <div class="text-base font-semibold">{{ t('system.mentorRewardStages.title') }}</div>
              <div class="text-sm text-gray-500 mt-1">{{
                t('system.mentorRewardStages.subtitle')
              }}</div>
            </div>
            <div class="flex items-center gap-3 flex-wrap">
              <ElButton :disabled="rewardStagesActionsDisabled" @click="addStage">{{
                t('system.mentorRewardStages.addStage')
              }}</ElButton>
              <ElButton
                type="primary"
                :loading="saving"
                :disabled="rewardStagesActionsDisabled"
                @click="handleSaveRewardStages"
              >
                {{ t('system.mentorRewardStages.save') }}
              </ElButton>
              <ElButton
                type="success"
                :loading="processing"
                :disabled="rewardStagesActionsDisabled"
                @click="handleRunProcess"
              >
                {{ t('system.mentorRewardStages.runProcess') }}
              </ElButton>
            </div>
          </div>
        </ElCard>

        <ElAlert
          :title="t('system.mentorRewardStages.description')"
          type="info"
          :closable="false"
          show-icon
          class="mb-4"
        />

        <ElAlert
          v-if="rewardStagesLoadFailed"
          :title="t('newbro.mentorManage.rewardStagesLoadFailed')"
          type="warning"
          :closable="false"
          show-icon
          class="mb-4"
        >
          <ElButton size="small" @click="handleReloadRewardStageSettings">
            {{ $t('common.refresh') }}
          </ElButton>
        </ElAlert>

        <ElCard shadow="never" class="mb-4" v-loading="rewardStagesLoading">
          <ElEmpty
            v-if="!stages.length && !rewardStagesLoading"
            :description="t('system.mentorRewardStages.empty')"
            :image-size="72"
          />

          <ElTable v-else :data="stages" stripe border row-key="local_id">
            <ElTableColumn :label="t('system.mentorRewardStages.stageOrder')" width="140">
              <template #default="{ row }">
                <ElInputNumber
                  v-model="row.stage_order"
                  :min="1"
                  :step="1"
                  :controls="false"
                  step-strictly
                  style="width: 100%"
                />
              </template>
            </ElTableColumn>
            <ElTableColumn :label="t('system.mentorRewardStages.stageName')" min-width="220">
              <template #default="{ row }">
                <ElInput v-model="row.name" />
              </template>
            </ElTableColumn>
            <ElTableColumn :label="t('system.mentorRewardStages.conditionType')" width="220">
              <template #default="{ row }">
                <ElSelect v-model="row.condition_type" style="width: 100%">
                  <ElOption
                    v-for="option in conditionOptions"
                    :key="option.value"
                    :label="option.label"
                    :value="option.value"
                  />
                </ElSelect>
              </template>
            </ElTableColumn>
            <ElTableColumn :label="t('system.mentorRewardStages.threshold')" width="180">
              <template #default="{ row }">
                <ElInputNumber
                  v-model="row.threshold"
                  :min="1"
                  :step="1"
                  :controls="false"
                  step-strictly
                  style="width: 100%"
                />
              </template>
            </ElTableColumn>
            <ElTableColumn :label="t('system.mentorRewardStages.rewardAmount')" width="180">
              <template #default="{ row }">
                <ElInputNumber
                  v-model="row.reward_amount"
                  :min="1"
                  :step="1"
                  :controls="false"
                  step-strictly
                  style="width: 100%"
                />
              </template>
            </ElTableColumn>
            <ElTableColumn
              :label="t('system.mentorRewardStages.operation')"
              width="120"
              fixed="right"
            >
              <template #default="{ $index }">
                <ElButton
                  link
                  type="danger"
                  :disabled="rewardStagesActionsDisabled"
                  @click="removeStage($index)"
                >
                  {{ t('system.mentorRewardStages.remove') }}
                </ElButton>
              </template>
            </ElTableColumn>
          </ElTable>
        </ElCard>

        <ElCard shadow="never" v-loading="settingsLoading">
          <template #header>
            <div class="flex items-center justify-between gap-4 flex-wrap">
              <div>
                <div class="text-base font-semibold">{{
                  t('system.mentorRewardStages.eligibilityTitle')
                }}</div>
                <div class="text-sm text-gray-500 mt-1">{{
                  t('system.mentorRewardStages.eligibilityDescription')
                }}</div>
              </div>
              <ElButton
                type="primary"
                :loading="settingsSaving"
                :disabled="rewardStagesActionsDisabled"
                @click="handleSaveEligibility"
              >
                {{ t('system.mentorRewardStages.saveEligibility') }}
              </ElButton>
            </div>
          </template>

          <ElForm label-width="220px" label-position="left" class="max-w-2xl">
            <ElFormItem :label="t('system.mentorRewardStages.maxCharacterSP')">
              <ElInputNumber
                v-model="mentorSettings.max_character_sp"
                :min="1"
                :step="1000000"
                :controls="false"
                step-strictly
                style="width: 100%"
              />
            </ElFormItem>
            <ElFormItem :label="t('system.mentorRewardStages.maxAccountAgeDays')">
              <ElInputNumber
                v-model="mentorSettings.max_account_age_days"
                :min="1"
                :step="1"
                :controls="false"
                step-strictly
                style="width: 100%"
              />
            </ElFormItem>
          </ElForm>
        </ElCard>
      </ElTabPane>
    </ElTabs>
  </div>
</template>

<script setup lang="ts">
  import type { ColumnOption } from '@/types/component'
  import { ElButton, ElMessage, ElMessageBox, ElTag } from 'element-plus'
  import { useI18n } from 'vue-i18n'
  import {
    fetchAdminMentorRelationships,
    fetchAdminMentorRewardDistributions,
    fetchMentorSettings,
    fetchMentorRewardStages,
    revokeMentorRelationship,
    runMentorRewardProcessing,
    updateMentorSettings,
    updateMentorRewardStages
  } from '@/api/mentor'
  import { buildEveCharacterPortraitUrl } from '@/utils/eve-image'
  import { useEnterSearch } from '@/hooks/core/useEnterSearch'
  import { useNewbroFormatters } from '@/hooks/newbro/useNewbroFormatters'

  defineOptions({ name: 'MentorManage' })

  type StageRow = Api.Mentor.RewardStageInput & {
    local_id: number
  }

  type MentorManageTab = 'relationships' | 'reward-records' | 'reward-stages'

  const { t } = useI18n()
  const { createEnterSearchHandler } = useEnterSearch()
  const { formatDateTime, formatCredit } = useNewbroFormatters()
  const numberFormatter = new Intl.NumberFormat('en-US', { maximumFractionDigits: 2 })

  const activeTab = ref<MentorManageTab>('relationships')
  const loading = ref(false)
  const rewardHistoryLoading = ref(false)
  const rewardHistoryLoaded = ref(false)
  const rewardStagesLoading = ref(false)
  const rewardStagesLoaded = ref(false)
  const rewardStagesLoadAttempted = ref(false)
  const saving = ref(false)
  const processing = ref(false)
  const settingsLoading = ref(false)
  const settingsSaving = ref(false)
  const revokingId = ref<number | null>(null)
  const rows = ref<Api.Mentor.RelationshipView[]>([])
  const rewardHistoryRows = ref<Api.Mentor.RewardDistributionView[]>([])
  const stages = ref<StageRow[]>([])
  const rewardHistoryKeyword = ref('')
  const rewardHistoryPaginationOptions = {
    pageSizes: [50, 100, 200, 500, 1000]
  }
  const mentorSettings = reactive<Api.Mentor.Settings>({
    max_character_sp: 4_000_000,
    max_account_age_days: 7
  })
  const filters = reactive({
    keyword: '',
    status: 'all' as Api.Mentor.MenteeStatusFilter
  })
  const page = reactive({ current: 1, size: 20, total: 0 })
  const rewardHistoryPage = reactive({ current: 1, size: 50, total: 0 })
  let nextLocalId = 1

  const statusOptions = computed(() => [
    { label: t('newbro.mentorManage.allStatuses'), value: 'all' },
    { label: formatStatus('pending'), value: 'pending' },
    { label: formatStatus('active'), value: 'active' },
    { label: formatStatus('rejected'), value: 'rejected' },
    { label: formatStatus('revoked'), value: 'revoked' },
    { label: formatStatus('graduated'), value: 'graduated' }
  ])

  const conditionOptions = computed(() => [
    {
      label: t('newbro.mentorConditionTypes.skill_points'),
      value: 'skill_points'
    },
    {
      label: t('newbro.mentorConditionTypes.pap_count'),
      value: 'pap_count'
    },
    {
      label: t('newbro.mentorConditionTypes.days_active'),
      value: 'days_active'
    }
  ])

  const refreshDisabled = computed(() => {
    if (revokingId.value !== null || saving.value || processing.value || settingsSaving.value) {
      return true
    }

    if (activeTab.value === 'relationships') {
      return loading.value
    }

    if (activeTab.value === 'reward-records') {
      return rewardHistoryLoading.value
    }

    return rewardStagesLoading.value || settingsLoading.value
  })

  const rewardStagesActionsDisabled = computed(
    () =>
      !rewardStagesLoaded.value ||
      rewardStagesLoading.value ||
      settingsLoading.value ||
      saving.value ||
      processing.value ||
      settingsSaving.value
  )

  const rewardStagesLoadFailed = computed(
    () =>
      rewardStagesLoadAttempted.value &&
      !rewardStagesLoaded.value &&
      !rewardStagesLoading.value &&
      !settingsLoading.value
  )

  const rewardHistoryColumns = computed<ColumnOption<Api.Mentor.RewardDistributionView>[]>(() => [
    {
      prop: 'mentor_character_name',
      label: t('newbro.mentorManage.mentorColumn'),
      minWidth: 220,
      formatter: (row) =>
        h('div', {}, [
          h('div', { class: 'font-medium' }, row.mentor_character_name),
          h(
            'div',
            { class: `text-sm ${row.mentor_nickname ? 'text-gray-500' : 'text-gray-400'}` },
            `${t('newbro.mentorManage.mentorNickname')}: ${row.mentor_nickname || '-'}`
          )
        ])
    },
    {
      prop: 'mentee_character_name',
      label: t('newbro.mentorManage.menteeColumn'),
      minWidth: 220,
      formatter: (row) =>
        h('div', {}, [
          h('div', { class: 'font-medium' }, row.mentee_character_name),
          h(
            'div',
            { class: `text-sm ${row.mentee_nickname ? 'text-gray-500' : 'text-gray-400'}` },
            `${t('newbro.mentorManage.menteeNickname')}: ${row.mentee_nickname || '-'}`
          )
        ])
    },
    {
      prop: 'stage_order',
      label: t('newbro.mentorManage.stageOrder'),
      width: 140,
      formatter: (row) => `#${row.stage_order}`
    },
    {
      prop: 'reward_amount',
      label: t('newbro.mentorManage.rewardAmount'),
      width: 160,
      formatter: (row) => formatCredit(row.reward_amount)
    },
    {
      prop: 'distributed_at',
      label: t('newbro.mentorManage.distributedAt'),
      width: 180,
      formatter: (row) => formatDateTime(row.distributed_at)
    }
  ])

  function formatStatus(status: Api.Mentor.MentorRelationshipStatus) {
    return t(`newbro.mentorStatus.${status}`)
  }

  function toStageRow(stage?: Api.Mentor.RewardStage | Api.Mentor.RewardStageInput): StageRow {
    return {
      local_id: nextLocalId++,
      stage_order: stage?.stage_order ?? stages.value.length + 1,
      name: stage?.name ?? '',
      condition_type: stage?.condition_type ?? 'skill_points',
      threshold: stage?.threshold ?? 1,
      reward_amount: stage?.reward_amount ?? 1
    }
  }

  function statusTagType(status: Api.Mentor.MentorRelationshipStatus) {
    switch (status) {
      case 'active':
        return 'success'
      case 'pending':
        return 'warning'
      case 'graduated':
        return 'primary'
      case 'rejected':
      case 'revoked':
        return 'info'
      default:
        return 'info'
    }
  }

  function canRevoke(status: Api.Mentor.MentorRelationshipStatus) {
    return status === 'pending' || status === 'active'
  }

  function revokeActionLabel(status: Api.Mentor.MentorRelationshipStatus) {
    return status === 'pending'
      ? t('newbro.mentorManage.cancelPending')
      : t('newbro.mentorManage.revoke')
  }

  function revokeActionSuccessMessage(status: Api.Mentor.MentorRelationshipStatus) {
    return status === 'pending'
      ? t('newbro.mentorManage.cancelPendingSuccess')
      : t('newbro.mentorManage.revokeSuccess')
  }

  async function loadData() {
    loading.value = true
    try {
      const data = await fetchAdminMentorRelationships({
        current: page.current,
        size: page.size,
        keyword: filters.keyword.trim() || undefined,
        status: filters.status === 'all' ? undefined : filters.status
      })
      rows.value = data.list
      page.total = data.total
    } catch (error) {
      console.error('Failed to load mentor relationships', error)
      rows.value = []
      page.total = 0
      ElMessage.error((error as Error)?.message || t('httpMsg.requestFailed'))
    } finally {
      loading.value = false
    }
  }

  async function loadRewardHistory() {
    rewardHistoryLoading.value = true
    try {
      const data = await fetchAdminMentorRewardDistributions({
        current: rewardHistoryPage.current,
        size: rewardHistoryPage.size,
        keyword: rewardHistoryKeyword.value.trim() || undefined
      })
      rewardHistoryRows.value = data.list
      rewardHistoryPage.total = data.total
      rewardHistoryLoaded.value = true
    } catch (error) {
      console.error('Failed to load mentor reward distributions', error)
      rewardHistoryRows.value = []
      rewardHistoryPage.total = 0
      ElMessage.error((error as Error)?.message || t('httpMsg.requestFailed'))
    } finally {
      rewardHistoryLoading.value = false
    }
  }

  async function loadRewardStages() {
    rewardStagesLoading.value = true
    try {
      const data = await fetchMentorRewardStages()
      stages.value = data.map((stage) => toStageRow(stage))
      return true
    } catch (error) {
      console.error('Failed to load mentor reward stages', error)
      stages.value = []
      ElMessage.error((error as Error)?.message || t('httpMsg.requestFailed'))
      return false
    } finally {
      rewardStagesLoading.value = false
    }
  }

  async function loadMentorSettings() {
    settingsLoading.value = true
    try {
      const data = await fetchMentorSettings()
      mentorSettings.max_character_sp = data.max_character_sp
      mentorSettings.max_account_age_days = data.max_account_age_days
      return true
    } catch (error) {
      console.error('Failed to load mentor settings', error)
      ElMessage.error((error as Error)?.message || t('httpMsg.requestFailed'))
      return false
    } finally {
      settingsLoading.value = false
    }
  }

  async function loadRewardStageSettings() {
    rewardStagesLoadAttempted.value = true
    const [stagesLoaded, settingsLoaded] = await Promise.all([
      loadRewardStages(),
      loadMentorSettings()
    ])
    rewardStagesLoaded.value = stagesLoaded && settingsLoaded
  }

  async function ensureRewardHistoryLoaded() {
    if (!rewardHistoryLoaded.value) {
      await loadRewardHistory()
    }
  }

  async function ensureRewardStageSettingsLoaded() {
    if (!rewardStagesLoaded.value) {
      await loadRewardStageSettings()
    }
  }

  async function handleTabChange(name: string | number) {
    if (name === 'reward-records') {
      await ensureRewardHistoryLoaded()
      return
    }

    if (name === 'reward-stages') {
      await ensureRewardStageSettingsLoaded()
    }
  }

  async function handleRefresh() {
    if (activeTab.value === 'reward-records') {
      await loadRewardHistory()
      return
    }

    if (activeTab.value === 'reward-stages') {
      await loadRewardStageSettings()
      return
    }

    await loadData()
  }

  async function handleReloadRewardStageSettings() {
    await loadRewardStageSettings()
  }

  async function handleSearch() {
    page.current = 1
    await loadData()
  }

  async function handleReset() {
    filters.keyword = ''
    filters.status = 'all'
    page.current = 1
    await loadData()
  }

  async function handleCurrentChange(value: number) {
    page.current = value
    await loadData()
  }

  async function handleSizeChange(value: number) {
    page.size = value
    page.current = 1
    await loadData()
  }

  async function handleRewardHistorySearch() {
    rewardHistoryPage.current = 1
    await loadRewardHistory()
  }

  async function handleRewardHistoryReset() {
    rewardHistoryKeyword.value = ''
    rewardHistoryPage.current = 1
    await loadRewardHistory()
  }

  async function handleRewardHistoryCurrentChange(value: number) {
    rewardHistoryPage.current = value
    await loadRewardHistory()
  }

  async function handleRewardHistorySizeChange(value: number) {
    rewardHistoryPage.size = value
    rewardHistoryPage.current = 1
    await loadRewardHistory()
  }

  function ensureRewardStageSettingsReady() {
    if (rewardStagesLoaded.value) {
      return true
    }

    ElMessage.warning(t('newbro.mentorManage.rewardStagesNotReady'))
    return false
  }

  function addStage() {
    if (!ensureRewardStageSettingsReady()) {
      return
    }

    const nextOrder =
      stages.value.reduce((maxOrder, stage) => Math.max(maxOrder, stage.stage_order), 0) + 1

    stages.value.push(
      toStageRow({
        stage_order: nextOrder,
        name: '',
        condition_type: 'skill_points',
        threshold: 1,
        reward_amount: 1
      })
    )
  }

  function removeStage(index: number) {
    if (!ensureRewardStageSettingsReady()) {
      return
    }

    stages.value.splice(index, 1)
  }

  function validateStages(rows: StageRow[]) {
    const stageOrders = new Set<number>()
    for (const row of rows) {
      if (!Number.isInteger(row.stage_order) || row.stage_order <= 0) {
        return t('system.mentorRewardStages.validation.stageOrder')
      }
      if (stageOrders.has(row.stage_order)) {
        return t('system.mentorRewardStages.validation.stageOrder')
      }
      stageOrders.add(row.stage_order)
      if (!row.name.trim()) {
        return t('system.mentorRewardStages.validation.name')
      }
      if (!row.condition_type) {
        return t('system.mentorRewardStages.validation.conditionType')
      }
      if (!Number.isInteger(row.threshold) || row.threshold <= 0) {
        return t('system.mentorRewardStages.validation.threshold')
      }
      if (!Number.isInteger(row.reward_amount) || row.reward_amount <= 0) {
        return t('system.mentorRewardStages.validation.rewardAmount')
      }
    }
    return ''
  }

  function validateMentorSettings(settings: Api.Mentor.Settings) {
    if (!Number.isInteger(settings.max_character_sp) || settings.max_character_sp <= 0) {
      return t('system.mentorRewardStages.validation.maxCharacterSP')
    }
    if (!Number.isInteger(settings.max_account_age_days) || settings.max_account_age_days <= 0) {
      return t('system.mentorRewardStages.validation.maxAccountAgeDays')
    }
    return ''
  }

  async function handleSaveEligibility() {
    if (!ensureRewardStageSettingsReady()) {
      return
    }

    const validationMessage = validateMentorSettings(mentorSettings)
    if (validationMessage) {
      ElMessage.warning(validationMessage)
      return
    }

    settingsSaving.value = true
    try {
      const data = await updateMentorSettings({
        max_character_sp: mentorSettings.max_character_sp,
        max_account_age_days: mentorSettings.max_account_age_days
      })
      mentorSettings.max_character_sp = data.max_character_sp
      mentorSettings.max_account_age_days = data.max_account_age_days
      ElMessage.success(t('system.mentorRewardStages.saveEligibilitySuccess'))
    } catch (error) {
      console.error('Failed to save mentor settings', error)
      ElMessage.error((error as Error)?.message || t('httpMsg.requestFailed'))
    } finally {
      settingsSaving.value = false
    }
  }

  async function handleSaveRewardStages() {
    if (!ensureRewardStageSettingsReady()) {
      return
    }

    const validationMessage = validateStages(stages.value)
    if (validationMessage) {
      ElMessage.warning(validationMessage)
      return
    }

    saving.value = true
    try {
      const payload = {
        stages: [...stages.value]
          .sort((a, b) => a.stage_order - b.stage_order)
          .map(({ stage_order, name, condition_type, threshold, reward_amount }) => ({
            stage_order,
            name: name.trim(),
            condition_type,
            threshold,
            reward_amount
          }))
      }
      const data = await updateMentorRewardStages(payload)
      stages.value = data.map((stage) => toStageRow(stage))
      ElMessage.success(t('system.mentorRewardStages.saveSuccess'))
    } catch (error) {
      console.error('Failed to save mentor reward stages', error)
      ElMessage.error((error as Error)?.message || t('httpMsg.requestFailed'))
    } finally {
      saving.value = false
    }
  }

  async function handleRunProcess() {
    if (!ensureRewardStageSettingsReady()) {
      return
    }

    processing.value = true
    try {
      const data = await runMentorRewardProcessing()
      ElMessage.success(
        t('system.mentorRewardStages.runProcessSuccess', {
          relationships: data.processed_relationships,
          rewards: data.rewards_distributed,
          total: numberFormatter.format(data.total_coin_awarded),
          graduated: data.graduated_count
        })
      )
    } catch (error) {
      console.error('Failed to process mentor rewards', error)
      ElMessage.error((error as Error)?.message || t('httpMsg.requestFailed'))
    } finally {
      processing.value = false
    }
  }

  const handleRelationshipSearchKeyup = createEnterSearchHandler(handleSearch)
  const handleRewardHistorySearchKeyup = createEnterSearchHandler(handleRewardHistorySearch)

  onMounted(() => {
    loadData()
  })

  async function handleRevoke(row: Api.Mentor.RelationshipView) {
    try {
      await ElMessageBox.confirm(
        `${row.mentor_character_name} -> ${row.mentee_character_name}`,
        revokeActionLabel(row.status),
        { type: 'warning' }
      )
    } catch {
      return
    }

    revokingId.value = row.id
    try {
      await revokeMentorRelationship({ relationship_id: row.id })
      ElMessage.success(revokeActionSuccessMessage(row.status))
      await loadData()
    } catch (error) {
      console.error('Failed to revoke mentor relationship', error)
      ElMessage.error((error as Error)?.message || t('httpMsg.requestFailed'))
    } finally {
      revokingId.value = null
    }
  }
</script>
