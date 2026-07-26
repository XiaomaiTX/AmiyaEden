<template>
  <div class="qq-governance-page">
    <ElCard shadow="never">
      <div class="page-title">
        <div
          ><div class="text-base font-semibold">{{ t('qqGovernance.title') }}</div
          ><div class="mt-1 text-sm text-gray-500">{{ t('qqGovernance.description') }}</div></div
        >
        <ElButton @click="refreshActive">{{ t('common.refresh') }}</ElButton>
      </div>
    </ElCard>

    <ElTabs v-model="activeTab" class="qq-governance-tabs">
      <ElTabPane :label="t('qqGovernance.v2.rules')" name="rules">
        <ElCard class="art-table-card" shadow="never">
          <ArtTableHeader
            v-model:columns="policyColumns"
            :loading="loading.policies"
            @refresh="loadRules"
          >
            <template #left>
              <ElButton type="primary" @click="openPolicyDialog()">
                {{ t('qqGovernance.actions.addPolicy') }}
              </ElButton>
            </template>
          </ArtTableHeader>
          <ArtTable :loading="loading.policies" :data="policies" :columns="visiblePolicyColumns">
            <template #group_name="{ row }">{{ groupNames.get(row.group_id) || '-' }}</template>
            <template #enabled="{ row }">
              <ElTag :type="row.enabled ? 'success' : 'info'">
                {{ row.enabled ? t('common.enable') : t('common.disable') }}
              </ElTag>
            </template>
            <template #member_count="{ row }">
              {{ groupStatuses.get(row.group_id)?.member_count ?? '-' }}
            </template>
            <template #operation="{ row }">
              <ElTooltip :content="t('qqGovernance.actions.edit')" placement="top">
                <ElButton
                  text
                  type="primary"
                  :icon="Edit"
                  :aria-label="t('qqGovernance.actions.edit')"
                  @click="openPolicyDialog(row)"
                />
              </ElTooltip>
              <ElTooltip :content="t('qqGovernance.actions.reconcile')" placement="top">
                <ElButton
                  text
                  type="primary"
                  :icon="Refresh"
                  :aria-label="t('qqGovernance.actions.reconcile')"
                  @click="reconcile(row.group_id)"
                />
              </ElTooltip>
              <ElPopconfirm
                :title="t('qqGovernance.messages.deleteConfirm')"
                @confirm="removePolicy(row.group_id)"
              >
                <template #reference>
                  <ElTooltip :content="t('qqGovernance.actions.delete')" placement="top">
                    <ElButton
                      text
                      type="danger"
                      :icon="Delete"
                      :aria-label="t('qqGovernance.actions.delete')"
                    />
                  </ElTooltip>
                </template>
              </ElPopconfirm>
            </template>
          </ArtTable>
        </ElCard>
      </ElTabPane>

      <ElTabPane :label="t('qqGovernance.v2.groups')" name="groups">
        <ElCard class="art-table-card" shadow="never">
          <ArtTableHeader
            v-model:columns="groupColumns"
            :loading="loading.groups"
            @refresh="loadGroups"
          />
          <ArtTable :loading="loading.groups" :data="groups" :columns="visibleGroupColumns">
            <template #group_name="{ row }">{{ row.group_name || '-' }}</template>
            <template #member_count="{ row }">
              {{ row.member_count
              }}<span v-if="row.max_member_count"> / {{ row.max_member_count }}</span>
            </template>
            <template #bot_is_admin="{ row }">
              <ElTag v-if="row.bot_is_admin === true" type="success">{{
                t('qqGovernance.values.yes')
              }}</ElTag>
              <ElTag v-else-if="row.bot_is_admin === false" type="danger">{{
                t('qqGovernance.values.no')
              }}</ElTag>
              <ElTag v-else type="info">{{ t('qqGovernance.values.unknown') }}</ElTag>
            </template>
            <template #progress="{ row }">
              <span v-if="row.reconcile_run_status">
                {{ row.reconcile_processed }} / {{ row.reconcile_expected }}
                <ElTag class="ml-2" size="small" :type="runTag(row.reconcile_run_status)">
                  {{ runLabel(row.reconcile_run_status) }}
                </ElTag>
              </span>
              <span v-else>-</span>
            </template>
            <template #invalid_count="{ row }">
              {{ row.invalid_candidate_count + row.invalid_confirmed_count }}
            </template>
            <template #snapshot_state="{ row }">
              <ElTag :type="snapshotTag(row.snapshot_state)">{{
                snapshotLabel(row.snapshot_state)
              }}</ElTag>
            </template>
            <template #last_synced_at="{ row }">{{
              formatTime(row.last_synced_at) || '-'
            }}</template>
            <template #operation="{ row }">
              <ElTooltip :content="t('qqGovernance.actions.reconcile')" placement="top">
                <ElButton
                  text
                  type="primary"
                  :icon="Refresh"
                  :aria-label="t('qqGovernance.actions.reconcile')"
                  @click="reconcile(row.group_id)"
                />
              </ElTooltip>
            </template>
          </ArtTable>
        </ElCard>
      </ElTabPane>

      <ElTabPane :label="t('qqGovernance.v2.operations')" name="operations">
        <ElForm inline class="operation-filters">
          <ElFormItem :label="t('qqGovernance.fields.groupId')">
            <ElInput v-model="operationFilters.groupId" inputmode="numeric" clearable />
          </ElFormItem>
          <ElFormItem :label="t('qqGovernance.fields.status')">
            <ElSelect v-model="operationFilters.status" clearable class="!w-36">
              <ElOption value="pending" :label="t('qqGovernance.v2.statusPending')" />
              <ElOption value="running" :label="t('qqGovernance.v2.statusRunning')" />
              <ElOption value="retry_wait" :label="t('qqGovernance.v2.statusRetryWait')" />
              <ElOption value="succeeded" :label="t('qqGovernance.v2.statusSucceeded')" />
              <ElOption value="dead" :label="t('qqGovernance.v2.statusDead')" />
              <ElOption value="open" :label="t('qqGovernance.v2.statusOpen')" />
              <ElOption value="acknowledged" :label="t('qqGovernance.v2.statusAcknowledged')" />
            </ElSelect>
          </ElFormItem>
          <ElFormItem :label="t('qqGovernance.fields.actionType')">
            <ElSelect v-model="operationFilters.actionType" clearable class="!w-36">
              <ElOption value="approve" :label="t('qqGovernance.actions.approve')" />
              <ElOption value="reject" :label="t('qqGovernance.actions.reject')" />
              <ElOption value="set_card" :label="t('qqGovernance.actions.card')" />
              <ElOption value="snapshot" :label="t('qqGovernance.v2.snapshotTask')" />
              <ElOption value="refresh_group_info" :label="t('qqGovernance.v2.groupInfoTask')" />
              <ElOption value="compute_batch" :label="t('qqGovernance.v2.computeTask')" />
              <ElOption value="kick" :label="t('qqGovernance.actions.kick')" />
            </ElSelect>
          </ElFormItem>
          <ElFormItem :label="t('qqGovernance.fields.decision')">
            <ElSelect v-model="operationFilters.decision" clearable class="!w-36">
              <ElOption value="matched" :label="t('qqGovernance.v2.decisionMatched')" />
              <ElOption value="unmatched" :label="t('qqGovernance.v2.decisionUnmatched')" />
              <ElOption value="review_wait" :label="t('qqGovernance.v2.decisionWait')" />
            </ElSelect>
          </ElFormItem>
          <ElButton type="primary" @click="loadOperations">{{ t('common.search') }}</ElButton>
          <ElButton @click="resetOperationFilters">{{ t('common.reset') }}</ElButton>
        </ElForm>
        <ElRow :gutter="12" class="mb-3"
          ><ElCol v-for="card in metricCards" :key="card.label" :xs="12" :sm="8" :lg="4"
            ><ElCard shadow="never"
              ><div class="text-sm text-gray-500">{{ card.label }}</div
              ><div class="mt-2 text-xl font-semibold">{{ card.value }}</div></ElCard
            ></ElCol
          ></ElRow
        >
        <ElTabs v-model="operationTab" class="operation-tabs">
          <ElTabPane :label="t('qqGovernance.v2.queue')" name="tasks">
            <ElCard class="art-table-card" shadow="never">
              <ArtTableHeader
                v-model:columns="taskColumnChecks"
                :loading="taskLoading"
                @refresh="taskTable.refreshData"
              />
              <ArtTable
                :loading="taskLoading"
                :data="taskData"
                :columns="taskColumns"
                :pagination="taskPagination"
                visual-variant="ledger"
                @pagination:size-change="taskHandleSizeChange"
                @pagination:current-change="taskHandleCurrentChange"
              >
                <template #action_type="{ row }">{{ actionLabel(row.action_type) }}</template>
                <template #operation="{ row }">
                  <ElTooltip
                    v-if="row.status === 'dead'"
                    :content="t('qqGovernance.actions.retry')"
                    placement="top"
                  >
                    <ElButton
                      text
                      type="primary"
                      :icon="RefreshRight"
                      :aria-label="t('qqGovernance.actions.retry')"
                      @click="retryTask(row.id)"
                    />
                  </ElTooltip>
                </template>
              </ArtTable>
            </ElCard>
          </ElTabPane>
          <ElTabPane :label="t('qqGovernance.v2.history')" name="reviews">
            <ElCard class="art-table-card" shadow="never">
              <ArtTableHeader
                v-model:columns="reviewColumnChecks"
                :loading="reviewLoading"
                @refresh="reviewTable.refreshData"
              />
              <ArtTable
                :loading="reviewLoading"
                :data="reviewData"
                :columns="reviewColumns"
                :pagination="reviewPagination"
                visual-variant="ledger"
                @pagination:size-change="reviewHandleSizeChange"
                @pagination:current-change="reviewHandleCurrentChange"
              >
                <template #decision="{ row }">{{ row.decision }}</template>
              </ArtTable>
            </ElCard>
          </ElTabPane>
          <ElTabPane :label="t('qqGovernance.tabs.alerts')" name="alerts">
            <ElCard class="art-table-card" shadow="never">
              <ArtTableHeader
                v-model:columns="alertColumnChecks"
                :loading="alertLoading"
                @refresh="alertTable.refreshData"
              />
              <ArtTable
                :loading="alertLoading"
                :data="alertData"
                :columns="alertColumns"
                :pagination="alertPagination"
                visual-variant="ledger"
                @pagination:size-change="alertHandleSizeChange"
                @pagination:current-change="alertHandleCurrentChange"
              >
                <template #status="{ row }">
                  <ElTag :type="row.status === 'open' ? 'danger' : 'info'">{{ row.status }}</ElTag>
                </template>
                <template #operation="{ row }">
                  <ElTooltip
                    v-if="row.status === 'open'"
                    :content="t('qqGovernance.actions.ack')"
                    placement="top"
                  >
                    <ElButton
                      text
                      type="primary"
                      :icon="CircleCheck"
                      :aria-label="t('qqGovernance.actions.ack')"
                      @click="acknowledge(row.id)"
                    />
                  </ElTooltip>
                </template>
              </ArtTable>
            </ElCard>
          </ElTabPane>
        </ElTabs>
      </ElTabPane>

      <ElTabPane :label="t('qqGovernance.v2.settings')" name="settings"
        ><ElRow :gutter="16"
          ><ElCol :lg="12"
            ><ElCard shadow="never"
              ><template #header>{{ t('qqGovernance.v2.globalSettings') }}</template
              ><ElForm :model="settings" label-width="160px"
                ><ElFormItem :label="t('qqGovernance.fields.scanInterval')"
                  ><ElInputNumber
                    v-model="settings.scan_interval_minutes"
                    :min="15"
                    :max="360"
                    :step="15" /></ElFormItem
                ><ElFormItem :label="t('qqGovernance.fields.confirmations')"
                  ><ElInputNumber
                    v-model="settings.mismatch_confirmations"
                    :min="2"
                    :max="3" /></ElFormItem
                ><ElFormItem :label="t('qqGovernance.fields.observation')"
                  ><ElInputNumber
                    v-model="settings.mismatch_observation_hours"
                    :min="1"
                    :max="6" /></ElFormItem
                ><ElButton type="primary" :loading="savingSettings" @click="saveSettings">{{
                  t('qqGovernance.v2.saveSettings')
                }}</ElButton></ElForm
              ></ElCard
            ></ElCol
          ><ElCol :lg="12"
            ><ElCard shadow="never"
              ><template #header>{{ t('qqGovernance.fields.connection') }}</template
              ><ElDescriptions :column="1" border
                ><ElDescriptionsItem :label="t('qqGovernance.fields.connection')"
                  ><ElTag :type="connection.connected ? 'success' : 'danger'">{{
                    connection.connected
                      ? t('qqGovernance.values.connected')
                      : t('qqGovernance.values.disconnected')
                  }}</ElTag></ElDescriptionsItem
                ><ElDescriptionsItem :label="t('qqGovernance.fields.risk')">{{
                  connection.risk_level
                }}</ElDescriptionsItem
                ><ElDescriptionsItem :label="t('qqGovernance.fields.rateLimit')"
                  ><ElTag :type="connection.rate_limit.available ? 'success' : 'warning'">{{
                    connection.rate_limit.available
                      ? t('qqGovernance.values.normal')
                      : t('qqGovernance.values.rateUnavailable')
                  }}</ElTag></ElDescriptionsItem
                ><template v-if="connection.rate_limit.available"
                  ><ElDescriptionsItem :label="t('qqGovernance.fields.globalLimit')">{{
                    rateLimitLabel(connection.rate_limit.global)
                  }}</ElDescriptionsItem
                  ><ElDescriptionsItem
                    v-for="group in connection.rate_limit.groups"
                    :key="group.group_id"
                    :label="`${t('qqGovernance.fields.groupLimit')} ${group.group_id}`"
                    >{{ rateLimitLabel(group.bucket) }}</ElDescriptionsItem
                  ></template
                ></ElDescriptions
              ><div class="mt-4"
                ><ElButton type="danger" plain @click="resetRisk">{{
                  t('qqGovernance.actions.resetRisk')
                }}</ElButton
                ><ElPopconfirm
                  :title="t('qqGovernance.messages.recoverDisconnectedConfirm')"
                  @confirm="recoverDisconnectedTasks"
                  ><template #reference
                    ><ElButton
                      class="ml-2"
                      type="primary"
                      plain
                      :disabled="!connection.connected"
                      :loading="recoveringDisconnectedTasks"
                      >{{ t('qqGovernance.actions.recoverDisconnected') }}</ElButton
                    ></template
                  ></ElPopconfirm
                ></div
              ></ElCard
            ></ElCol
          ></ElRow
        ></ElTabPane
      >
    </ElTabs>

    <ElDialog
      v-model="policyDialog"
      :title="t('qqGovernance.v2.rules')"
      width="620px"
      destroy-on-close
      ><ElForm :model="policyForm" label-width="160px"
        ><ElFormItem :label="t('qqGovernance.fields.groupId')"
          ><ElInput
            v-model="policyForm.group_id"
            inputmode="numeric"
            :disabled="editingGroupId > 0" /></ElFormItem
        ><ElFormItem :label="t('qqGovernance.fields.enabled')"
          ><ElSwitch v-model="policyForm.enabled" /></ElFormItem
        ><ElFormItem :label="t('qqGovernance.fields.corporations')"
          ><ElSelect
            v-model="policyForm.allowed_corporation_ids"
            multiple
            filterable
            remote
            reserve-keyword
            class="!w-full"
            :loading="searchingCorporations"
            :remote-method="searchCorporations"
            :placeholder="t('qqGovernance.v2.corpSearch')"
            ><ElOption
              v-for="corp in corporationOptions"
              :key="corp.corporation_id"
              :label="formatCorporationLabel(corp)"
              :value="corp.corporation_id" /></ElSelect></ElFormItem
        ><ElFormItem :label="t('qqGovernance.fields.roles')"
          ><ElSelect v-model="policyForm.allowed_role_codes" multiple class="!w-full"
            ><ElOption
              v-for="role in roleOptions"
              :key="role.code"
              :label="role.name"
              :value="role.code" /></ElSelect></ElFormItem
        ><ElFormItem :label="t('qqGovernance.fields.autoReject')"
          ><ElSwitch v-model="policyForm.auto_reject_unmatched" /></ElFormItem
        ><ElFormItem :label="t('qqGovernance.fields.violationPolicy')"
          ><ElSelect v-model="policyForm.member_violation_policy" class="!w-full"
            ><ElOption :label="t('qqGovernance.values.reviewOnly')" value="review_only" /><ElOption
              :label="t('qqGovernance.values.autoKick')"
              value="auto_kick_after_confirmed_mismatch" /></ElSelect
        ></ElFormItem>
        <ElFormItem :label="t('qqGovernance.fields.cardTemplate')"
          ><ElInput
            v-model="policyForm.card_template"
            type="textarea"
            maxlength="100"
            show-word-limit
          /><div class="form-hint">{{
            t('qqGovernance.v2.templateHint', {
              nickname: '{nickname}',
              primary_character_name: '{primary_character_name}',
              primary_corporation_name: '{primary_corporation_name}'
            })
          }}</div></ElFormItem
        ><ElFormItem :label="t('qqGovernance.fields.cardSync')"
          ><ElSwitch v-model="policyForm.card_sync_enabled" /><div class="form-hint">{{
            t('qqGovernance.v2.cardSyncHint')
          }}</div></ElFormItem
        ></ElForm
      ><template #footer
        ><ElButton @click="policyDialog = false">{{ t('common.cancel') }}</ElButton
        ><ElButton type="primary" :loading="savingPolicy" @click="savePolicy">{{
          t('qqGovernance.actions.save')
        }}</ElButton></template
      ></ElDialog
    >
  </div>
</template>

<script setup lang="ts">
  import { computed, reactive, ref, watch } from 'vue'
  import { useI18n } from 'vue-i18n'
  import { CircleCheck, Delete, Edit, Refresh, RefreshRight } from '@element-plus/icons-vue'
  import { useTable } from '@/hooks/core/useTable'
  import type { ColumnOption } from '@/types/component'
  import { formatTime } from '@/utils/common'
  import { fetchGetRoleDefinitions } from '@/api/system-manage'
  import {
    acknowledgeQQGovernanceAlert,
    deleteQQGovernancePolicy,
    fetchQQGovernanceAlerts,
    fetchQQGovernanceConnection,
    fetchQQGovernanceGroups,
    fetchQQGovernanceMetrics,
    fetchQQGovernancePolicies,
    fetchQQGovernanceReviews,
    fetchQQGovernanceSettings,
    fetchQQGovernanceTasks,
    recoverQQGovernanceDisconnectedTasks,
    retryQQGovernanceTask,
    resetQQGovernanceRisk,
    saveQQGovernancePolicy,
    searchQQGovernanceCorporations,
    triggerQQGovernanceReconcile,
    updateQQGovernancePolicy,
    updateQQGovernanceSettings
  } from '@/api/qq-governance'
  defineOptions({ name: 'QQGovernance' })
  const { t } = useI18n()
  const activeTab = ref('rules'),
    operationTab = ref('tasks'),
    policyDialog = ref(false),
    savingPolicy = ref(false),
    savingSettings = ref(false),
    recoveringDisconnectedTasks = ref(false),
    editingGroupId = ref(0)
  const policies = ref<Api.QQGovernance.Policy[]>([]),
    groups = ref<Api.QQGovernance.GroupStatus[]>([]),
    corporationOptions = ref<Api.QQGovernance.CorporationOption[]>([]),
    roleOptions = ref<Api.SystemManage.RoleDefinition[]>([]),
    searchingCorporations = ref(false)
  const metrics = ref<Api.QQGovernance.Metrics>({
      window_minutes: 60,
      created: 0,
      succeeded: 0,
      failed: 0,
      dead: 0,
      failure_rate: 0,
      connected: false,
      risk_level: 0
    }),
    connection = ref<Api.QQGovernance.Connection>({
      connected: false,
      risk_level: 0,
      rate_limit: {
        available: false,
        global: { capacity: 3, tokens: 0, wait_ms: 0 },
        groups: []
      }
    })
  const settings = reactive<Api.QQGovernance.Settings>({
    scan_interval_minutes: 15,
    mismatch_confirmations: 2,
    mismatch_observation_hours: 2
  })
  const loading = reactive({
    policies: false,
    groups: false
  })
  const operationFilters = reactive({ groupId: '', status: '', actionType: '', decision: '' })
  const policyForm = reactive({
    group_id: '',
    enabled: true,
    allowed_corporation_ids: [] as number[],
    allowed_role_codes: [] as string[],
    auto_reject_unmatched: false,
    member_violation_policy: 'review_only' as Api.QQGovernance.Policy['member_violation_policy'],
    card_template: '',
    card_sync_enabled: false
  })
  const groupNames = computed(
    () => new Map(groups.value.map((group) => [group.group_id, group.group_name]))
  )
  const groupStatuses = computed(
    () => new Map(groups.value.map((group) => [group.group_id, group]))
  )
  const metricCards = computed(() => [
    { label: t('qqGovernance.v2.overview'), value: metrics.value.created },
    { label: t('qqGovernance.values.connected'), value: metrics.value.succeeded },
    { label: t('qqGovernance.fields.error'), value: metrics.value.failed },
    { label: t('qqGovernance.fields.dead'), value: metrics.value.dead },
    {
      label: t('qqGovernance.fields.failureRate'),
      value: `${(metrics.value.failure_rate * 100).toFixed(1)}%`
    },
    { label: t('qqGovernance.fields.risk'), value: metrics.value.risk_level }
  ])
  const snapshotTag = (state: string) =>
    state === 'fresh' ? 'success' : state === 'stale' ? 'warning' : 'info'
  const snapshotLabel = (state: string) =>
    t(`qqGovernance.v2.${state === 'never_synced' ? 'notSynced' : state}`)
  const runTag = (state: string) =>
    state === 'completed'
      ? 'success'
      : state === 'failed'
        ? 'danger'
        : state === 'running'
          ? 'warning'
          : 'info'
  const runLabel = (state: string) =>
    t('qqGovernance.v2.run' + state[0].toUpperCase() + state.slice(1))
  const actionLabel = (action: string) => {
    const labels: Record<string, string> = {
      snapshot: 'qqGovernance.v2.snapshotTask',
      refresh_group_info: 'qqGovernance.v2.groupInfoTask',
      compute_batch: 'qqGovernance.v2.computeTask'
    }
    return labels[action] ? t(labels[action]) : action
  }

  type QQTableParams = Omit<Api.QQGovernance.PageParams, 'page_size'> & { pageSize?: number }
  const fetchTaskTableData = async (
    params: QQTableParams
  ): Promise<Api.Common.PaginatedResponse<Api.QQGovernance.ActionTask>> => {
    const { pageSize, ...query } = params
    const response = await fetchQQGovernanceTasks({ ...query, page_size: pageSize })
    return {
      list: response.list,
      total: response.total,
      page: response.page,
      pageSize: response.page_size
    }
  }
  const fetchReviewTableData = async (
    params: QQTableParams
  ): Promise<Api.Common.PaginatedResponse<Api.QQGovernance.Review>> => {
    const { pageSize, ...query } = params
    const response = await fetchQQGovernanceReviews({ ...query, page_size: pageSize })
    return {
      list: response.list,
      total: response.total,
      page: response.page,
      pageSize: response.page_size
    }
  }
  const fetchAlertTableData = async (
    params: QQTableParams
  ): Promise<Api.Common.PaginatedResponse<Api.QQGovernance.Alert>> => {
    const { pageSize, ...query } = params
    const response = await fetchQQGovernanceAlerts({ ...query, page_size: pageSize })
    return {
      list: response.list,
      total: response.total,
      page: response.page,
      pageSize: response.page_size
    }
  }

  const policyColumns = ref<ColumnOption<Api.QQGovernance.Policy>[]>([
    {
      prop: 'group_name',
      label: t('qqGovernance.v2.groupName'),
      minWidth: 180,
      useSlot: true
    },
    { prop: 'group_id', label: t('qqGovernance.fields.groupId'), minWidth: 120, sortable: true },
    { prop: 'enabled', label: t('qqGovernance.fields.enabled'), minWidth: 100, useSlot: true },
    { prop: 'member_count', label: t('qqGovernance.v2.memberCount'), minWidth: 120, useSlot: true },
    {
      prop: 'operation',
      label: t('common.operation'),
      minWidth: 140,
      fixed: 'right',
      useSlot: true
    }
  ])

  const groupColumns = ref<ColumnOption<Api.QQGovernance.GroupStatus>[]>([
    { prop: 'group_name', label: t('qqGovernance.v2.groupName'), minWidth: 170 },
    { prop: 'group_id', label: t('qqGovernance.fields.groupId'), minWidth: 120 },
    { prop: 'member_count', label: t('qqGovernance.v2.memberCount'), minWidth: 120, useSlot: true },
    {
      prop: 'bot_is_admin',
      label: t('qqGovernance.fields.botAdmin'),
      minWidth: 130,
      useSlot: true
    },
    { prop: 'progress', label: t('qqGovernance.v2.progress'), minWidth: 180, useSlot: true },
    { prop: 'valid_count', label: t('qqGovernance.v2.valid'), minWidth: 100 },
    { prop: 'review_count', label: t('qqGovernance.v2.review'), minWidth: 100 },
    { prop: 'invalid_count', label: t('qqGovernance.v2.invalid'), minWidth: 100, useSlot: true },
    { prop: 'snapshot_state', label: t('qqGovernance.v2.snapshot'), minWidth: 120, useSlot: true },
    { prop: 'last_synced_at', label: t('qqGovernance.v2.lastSync'), minWidth: 180, useSlot: true },
    { prop: 'operation', label: t('common.operation'), minWidth: 80, fixed: 'right', useSlot: true }
  ])
  const visiblePolicyColumns = computed(() =>
    policyColumns.value.filter((column) => column.visible !== false && column.checked !== false)
  )
  const visibleGroupColumns = computed(() =>
    groupColumns.value.filter((column) => column.visible !== false && column.checked !== false)
  )

  const taskTable = useTable({
    core: {
      apiFn: fetchTaskTableData,
      apiParams: { page: 1, pageSize: 200 },
      paginationKey: { current: 'page', size: 'pageSize' },
      immediate: false,
      columnsFactory: () => [
        { prop: 'id', label: 'ID', width: 90, sortable: true },
        {
          prop: 'action_type',
          label: t('qqGovernance.fields.actionType'),
          width: 150,
          useSlot: true
        },
        { prop: 'group_id', label: t('qqGovernance.fields.groupId'), width: 130, sortable: true },
        { prop: 'qq', label: t('qqGovernance.fields.qq'), width: 140, sortable: true },
        { prop: 'status', label: t('qqGovernance.fields.status'), width: 130, sortable: true },
        {
          prop: 'retry_count',
          label: t('qqGovernance.fields.retryCount'),
          width: 110,
          sortable: true
        },
        {
          prop: 'last_error',
          label: t('qqGovernance.fields.error'),
          minWidth: 220,
          showOverflowTooltip: true
        },
        { prop: 'operation', label: t('common.operation'), width: 80, useSlot: true }
      ]
    }
  })

  const reviewTable = useTable({
    core: {
      apiFn: fetchReviewTableData,
      apiParams: { page: 1, pageSize: 200 },
      paginationKey: { current: 'page', size: 'pageSize' },
      immediate: false,
      columnsFactory: () => [
        { prop: 'group_id', label: t('qqGovernance.fields.groupId'), width: 130 },
        { prop: 'qq', label: t('qqGovernance.fields.qq'), width: 140 },
        { prop: 'decision', label: t('qqGovernance.fields.decision'), width: 130, useSlot: true },
        {
          prop: 'reason',
          label: t('qqGovernance.fields.reason'),
          minWidth: 300,
          showOverflowTooltip: true
        },
        {
          prop: 'created_at',
          label: t('common.createdAt'),
          minWidth: 180,
          sortable: true,
          formatter: (row) => formatTime(row.created_at)
        }
      ]
    }
  })

  const alertTable = useTable({
    core: {
      apiFn: fetchAlertTableData,
      apiParams: { page: 1, pageSize: 200 },
      paginationKey: { current: 'page', size: 'pageSize' },
      immediate: false,
      columnsFactory: () => [
        { prop: 'kind', label: t('qqGovernance.fields.status'), width: 140 },
        { prop: 'group_id', label: t('qqGovernance.fields.groupId'), width: 130 },
        {
          prop: 'message',
          label: t('qqGovernance.fields.message'),
          minWidth: 280,
          showOverflowTooltip: true
        },
        { prop: 'status', label: t('qqGovernance.fields.status'), width: 140, useSlot: true },
        { prop: 'operation', label: t('common.operation'), width: 80, useSlot: true }
      ]
    }
  })

  const {
    data: taskData,
    loading: taskLoading,
    pagination: taskPagination,
    columns: taskColumns,
    columnChecks: taskColumnChecks,
    handleSizeChange: taskHandleSizeChange,
    handleCurrentChange: taskHandleCurrentChange
  } = taskTable
  const {
    data: reviewData,
    loading: reviewLoading,
    pagination: reviewPagination,
    columns: reviewColumns,
    columnChecks: reviewColumnChecks,
    handleSizeChange: reviewHandleSizeChange,
    handleCurrentChange: reviewHandleCurrentChange
  } = reviewTable
  const {
    data: alertData,
    loading: alertLoading,
    pagination: alertPagination,
    columns: alertColumns,
    columnChecks: alertColumnChecks,
    handleSizeChange: alertHandleSizeChange,
    handleCurrentChange: alertHandleCurrentChange
  } = alertTable

  function resetPolicyForm() {
    Object.assign(policyForm, {
      group_id: '',
      enabled: true,
      allowed_corporation_ids: [],
      allowed_role_codes: [],
      auto_reject_unmatched: false,
      member_violation_policy: 'review_only',
      card_template: '',
      card_sync_enabled: false
    })
  }
  async function loadRules() {
    loading.policies = true
    try {
      policies.value = await fetchQQGovernancePolicies()
    } finally {
      loading.policies = false
    }
  }
  async function loadGroups() {
    loading.groups = true
    try {
      groups.value = await fetchQQGovernanceGroups()
    } finally {
      loading.groups = false
    }
  }
  async function loadOperations() {
    Object.assign(taskTable.searchParams, {
      page: 1,
      group_id: Number(operationFilters.groupId) || undefined,
      status: operationFilters.status || undefined,
      action_type: operationFilters.actionType || undefined
    })
    Object.assign(reviewTable.searchParams, {
      page: 1,
      group_id: Number(operationFilters.groupId) || undefined,
      decision: operationFilters.decision || undefined
    })
    Object.assign(alertTable.searchParams, {
      page: 1,
      status: operationFilters.status || undefined
    })
    const [, , , metricRows] = await Promise.all([
      taskTable.getData(),
      reviewTable.getData(),
      alertTable.getData(),
      fetchQQGovernanceMetrics()
    ])
    metrics.value = metricRows
  }
  function resetOperationFilters() {
    Object.assign(operationFilters, { groupId: '', status: '', actionType: '', decision: '' })
    void loadOperations()
  }
  async function loadSettings() {
    Object.assign(settings, await fetchQQGovernanceSettings())
    connection.value = await fetchQQGovernanceConnection()
  }
  async function loadOptions() {
    roleOptions.value = await fetchGetRoleDefinitions()
  }
  async function searchCorporations(query: string) {
    if (!query.trim()) return
    searchingCorporations.value = true
    try {
      corporationOptions.value = await searchQQGovernanceCorporations(query)
    } finally {
      searchingCorporations.value = false
    }
  }
  function formatCorporationLabel(corp: Api.QQGovernance.CorporationOption) {
    const name = corp.corporation_name || String(corp.corporation_id)
    return `${name} (${corp.corporation_id})`
  }
  async function refreshActive() {
    if (activeTab.value === 'rules') await Promise.all([loadRules(), loadGroups()])
    else if (activeTab.value === 'groups') await loadGroups()
    else if (activeTab.value === 'operations') await loadOperations()
    else await loadSettings()
  }
  watch(activeTab, () => void refreshActive(), { immediate: true })
  void loadOptions()
  function openPolicyDialog(policy?: Api.QQGovernance.Policy) {
    editingGroupId.value = policy?.group_id || 0
    resetPolicyForm()
    if (policy) {
      Object.assign(policyForm, {
        group_id: String(policy.group_id),
        enabled: policy.enabled,
        allowed_corporation_ids: policy.allowed_corporation_ids,
        allowed_role_codes: policy.allowed_role_codes,
        auto_reject_unmatched: policy.auto_reject_unmatched,
        member_violation_policy: policy.member_violation_policy,
        card_template: policy.card_template,
        card_sync_enabled: policy.card_sync_enabled
      })
      // Seed options with the rule's saved corporations so already-selected IDs
      // still show the "name (id)" label without requiring a fresh search.
      corporationOptions.value = policy.allowed_corporations?.length
        ? policy.allowed_corporations.map((corp: Api.QQGovernance.CorporationOption) => ({
            corporation_id: corp.corporation_id,
            corporation_name: corp.corporation_name
          }))
        : policy.allowed_corporation_ids.map((id: number) => ({
            corporation_id: id,
            corporation_name: ''
          }))
    } else {
      corporationOptions.value = []
    }
    policyDialog.value = true
  }
  async function savePolicy() {
    const groupId = Number(policyForm.group_id)
    if (!Number.isSafeInteger(groupId) || groupId <= 0)
      return ElMessage.error(t('qqGovernance.fields.groupId'))
    savingPolicy.value = true
    try {
      const payload = {
        enabled: policyForm.enabled,
        allowed_corporation_ids: policyForm.allowed_corporation_ids,
        allowed_role_codes: policyForm.allowed_role_codes,
        auto_reject_unmatched: policyForm.auto_reject_unmatched,
        member_violation_policy: policyForm.member_violation_policy,
        card_template: policyForm.card_template,
        card_sync_enabled: policyForm.card_sync_enabled
      }
      if (editingGroupId.value) await updateQQGovernancePolicy(editingGroupId.value, payload)
      else await saveQQGovernancePolicy({ group_id: groupId, ...payload })
      policyDialog.value = false
      await loadRules()
      ElMessage.success(t('qqGovernance.messages.saveSuccess'))
    } finally {
      savingPolicy.value = false
    }
  }
  async function removePolicy(id: number) {
    await deleteQQGovernancePolicy(id)
    await refreshActive()
  }
  async function reconcile(id: number) {
    const result = await triggerQQGovernanceReconcile(id)
    const messages: Record<Api.QQGovernance.ReconcileResult['status'], string> = {
      created: 'qqGovernance.messages.reconcileCreated',
      resumed: 'qqGovernance.messages.reconcileResumed',
      running: 'qqGovernance.messages.reconcileRunning',
      blocked: 'qqGovernance.messages.reconcileBlocked'
    }
    ElMessage.success(t(messages[result.status]))
    await refreshActive()
  }
  function rateLimitLabel(bucket: Api.QQGovernance.RateLimitBucket) {
    if (bucket.wait_ms > 0) {
      return t('qqGovernance.messages.rateLimitWaiting', { milliseconds: bucket.wait_ms })
    }
    return t('qqGovernance.messages.rateLimitAvailable', {
      tokens: bucket.tokens.toFixed(2),
      capacity: bucket.capacity
    })
  }
  async function recoverDisconnectedTasks() {
    recoveringDisconnectedTasks.value = true
    try {
      const result = await recoverQQGovernanceDisconnectedTasks()
      ElMessage.success(
        t('qqGovernance.messages.recoverDisconnectedSuccess', { count: result.recovered_tasks })
      )
      await Promise.all([loadSettings(), refreshActive(), taskTable.refreshData()])
    } finally {
      recoveringDisconnectedTasks.value = false
    }
  }
  async function retryTask(id: number) {
    await retryQQGovernanceTask(id)
    await taskTable.refreshData()
  }
  async function acknowledge(id: number) {
    await acknowledgeQQGovernanceAlert(id)
    await alertTable.refreshData()
  }
  async function saveSettings() {
    savingSettings.value = true
    try {
      Object.assign(settings, await updateQQGovernanceSettings(settings))
      ElMessage.success(t('qqGovernance.v2.settingsSaved'))
    } finally {
      savingSettings.value = false
    }
  }
  async function resetRisk() {
    await resetQQGovernanceRisk()
    await loadSettings()
    ElMessage.success(t('qqGovernance.messages.riskReset'))
  }
</script>

<style scoped>
  .qq-governance-page {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }
  .page-title {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
  }
  .operation-filters {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 8px 12px;
    margin-bottom: 12px;
  }
  .qq-governance-tabs :deep(.el-tabs__content) {
    overflow: visible;
  }
  .qq-governance-page :deep(.art-table-card .el-card__body) {
    display: flex;
    flex-direction: column;
    min-height: 0;
  }
  .qq-governance-page :deep(.art-table-card .art-table) {
    min-height: 0;
  }
  .qq-governance-tabs :deep(.el-tab-pane) {
    min-height: 0;
  }
  .operation-tabs :deep(.el-tab-pane) {
    display: flex;
    flex-direction: column;
    min-height: 0;
  }
  .operation-tabs :deep(.art-table-card),
  .operation-tabs :deep(.art-table-card .el-card__body) {
    min-height: 0;
  }
  .form-hint {
    margin-top: 6px;
    color: var(--el-text-color-secondary);
    font-size: 12px;
    line-height: 1.5;
  }
</style>
