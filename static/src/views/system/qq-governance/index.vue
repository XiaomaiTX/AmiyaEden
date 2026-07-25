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
        <div class="toolbar"
          ><ElButton type="primary" @click="openPolicyDialog()">{{
            t('qqGovernance.actions.addPolicy')
          }}</ElButton></div
        >
        <ElCard class="art-table-card" shadow="never"
          ><ElTable v-loading="loading.policies" :data="policies" border stripe>
            <ElTableColumn :label="t('qqGovernance.v2.groupName')" min-width="180"
              ><template #default="{ row }">{{
                groupNames.get(row.group_id) || '-'
              }}</template></ElTableColumn
            >
            <ElTableColumn
              prop="group_id"
              :label="t('qqGovernance.fields.groupId')"
              width="150"
              sortable
            />
            <ElTableColumn :label="t('qqGovernance.fields.enabled')" width="100"
              ><template #default="{ row }"
                ><ElTag :type="row.enabled ? 'success' : 'info'">{{
                  row.enabled ? t('common.enable') : t('common.disable')
                }}</ElTag></template
              ></ElTableColumn
            >
            <ElTableColumn :label="t('qqGovernance.v2.memberCount')" width="120"
              ><template #default="{ row }">{{
                groupStatuses.get(row.group_id)?.member_count ?? '-'
              }}</template></ElTableColumn
            >
            <ElTableColumn :label="t('common.operation')" width="220" fixed="right"
              ><template #default="{ row }"
                ><ElButton text type="primary" @click="openPolicyDialog(row)">{{
                  t('qqGovernance.actions.edit')
                }}</ElButton
                ><ElButton text type="primary" @click="reconcile(row.group_id)">{{
                  t('qqGovernance.actions.reconcile')
                }}</ElButton
                ><ElPopconfirm
                  :title="t('qqGovernance.messages.deleteConfirm')"
                  @confirm="removePolicy(row.group_id)"
                  ><template #reference
                    ><ElButton text type="danger">{{
                      t('qqGovernance.actions.delete')
                    }}</ElButton></template
                  ></ElPopconfirm
                ></template
              ></ElTableColumn
            >
          </ElTable></ElCard
        >
      </ElTabPane>

      <ElTabPane :label="t('qqGovernance.v2.groups')" name="groups">
        <ElCard class="art-table-card" shadow="never"
          ><ElTable v-loading="loading.groups" :data="groups" border stripe>
            <ElTableColumn prop="group_name" :label="t('qqGovernance.v2.groupName')" min-width="170"
              ><template #default="{ row }">{{ row.group_name || '-' }}</template></ElTableColumn
            ><ElTableColumn prop="group_id" :label="t('qqGovernance.fields.groupId')" width="150" />
            <ElTableColumn :label="t('qqGovernance.v2.memberCount')" width="140"
              ><template #default="{ row }"
                >{{ row.member_count
                }}<span v-if="row.max_member_count"> / {{ row.max_member_count }}</span></template
              ></ElTableColumn
            >
            <ElTableColumn :label="t('qqGovernance.fields.botAdmin')" width="140"
              ><template #default="{ row }"
                ><ElTag v-if="row.bot_is_admin === true" type="success">{{
                  t('qqGovernance.values.yes')
                }}</ElTag
                ><ElTag v-else-if="row.bot_is_admin === false" type="danger">{{
                  t('qqGovernance.values.no')
                }}</ElTag
                ><ElTag v-else type="info">{{ t('qqGovernance.values.unknown') }}</ElTag></template
              ></ElTableColumn
            >
            <ElTableColumn :label="t('qqGovernance.v2.progress')" min-width="160"
              ><template #default="{ row }"
                ><span v-if="row.reconcile_run_status"
                  >{{ row.reconcile_processed }} / {{ row.reconcile_expected }}
                  <ElTag class="ml-2" size="small" :type="runTag(row.reconcile_run_status)">{{
                    runLabel(row.reconcile_run_status)
                  }}</ElTag></span
                ><span v-else>-</span></template
              ></ElTableColumn
            >
            <ElTableColumn
              prop="valid_count"
              :label="t('qqGovernance.v2.valid')"
              width="110"
            /><ElTableColumn
              prop="review_count"
              :label="t('qqGovernance.v2.review')"
              width="110"
            /><ElTableColumn :label="t('qqGovernance.v2.invalid')" width="110"
              ><template #default="{ row }">{{
                row.invalid_candidate_count + row.invalid_confirmed_count
              }}</template></ElTableColumn
            >
            <ElTableColumn :label="t('qqGovernance.v2.snapshot')" width="120"
              ><template #default="{ row }"
                ><ElTag :type="snapshotTag(row.snapshot_state)">{{
                  snapshotLabel(row.snapshot_state)
                }}</ElTag></template
              ></ElTableColumn
            ><ElTableColumn :label="t('qqGovernance.v2.lastSync')" min-width="180"
              ><template #default="{ row }">{{
                formatTime(row.last_synced_at) || '-'
              }}</template></ElTableColumn
            >
            <ElTableColumn :label="t('common.operation')" width="120" fixed="right"
              ><template #default="{ row }"
                ><ElButton text type="primary" @click="reconcile(row.group_id)">{{
                  t('qqGovernance.actions.reconcile')
                }}</ElButton></template
              ></ElTableColumn
            >
          </ElTable></ElCard
        >
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
        <ElTabs v-model="operationTab"
          ><ElTabPane :label="t('qqGovernance.v2.queue')" name="tasks"
            ><ElCard class="art-table-card" shadow="never"
              ><ElTable v-loading="loading.tasks" :data="tasks.list" border stripe
                ><ElTableColumn prop="id" label="ID" width="90" sortable /><ElTableColumn
                  prop="action_type"
                  :label="t('qqGovernance.fields.actionType')"
                  width="130"
                  sortable
                  ><template #default="{ row }">{{
                    actionLabel(row.action_type)
                  }}</template></ElTableColumn
                ><ElTableColumn
                  prop="group_id"
                  :label="t('qqGovernance.fields.groupId')"
                  width="130"
                  sortable
                /><ElTableColumn
                  prop="qq"
                  :label="t('qqGovernance.fields.qq')"
                  width="140"
                  sortable
                /><ElTableColumn
                  prop="status"
                  :label="t('qqGovernance.fields.status')"
                  width="130"
                  sortable
                /><ElTableColumn
                  prop="retry_count"
                  :label="t('qqGovernance.fields.retryCount')"
                  width="110"
                  sortable
                /><ElTableColumn
                  prop="last_error"
                  :label="t('qqGovernance.fields.error')"
                  min-width="220"
                  show-overflow-tooltip
                /><ElTableColumn :label="t('common.operation')" width="120"
                  ><template #default="{ row }"
                    ><ElButton
                      v-if="row.status === 'dead'"
                      text
                      type="primary"
                      @click="retryTask(row.id)"
                      >{{ t('qqGovernance.actions.retry') }}</ElButton
                    ></template
                  ></ElTableColumn
                ></ElTable
              ></ElCard
            ></ElTabPane
          >
          <ElTabPane :label="t('qqGovernance.v2.history')" name="reviews"
            ><ElCard class="art-table-card" shadow="never"
              ><ElTable v-loading="loading.reviews" :data="reviews.list" border stripe
                ><ElTableColumn
                  prop="group_id"
                  :label="t('qqGovernance.fields.groupId')"
                  width="130" /><ElTableColumn
                  prop="qq"
                  :label="t('qqGovernance.fields.qq')"
                  width="140" /><ElTableColumn
                  prop="decision"
                  :label="t('qqGovernance.fields.decision')"
                  width="130" /><ElTableColumn
                  prop="reason"
                  :label="t('qqGovernance.fields.reason')"
                  min-width="300"
                  show-overflow-tooltip /><ElTableColumn
                  prop="created_at"
                  :label="t('common.createdAt')"
                  min-width="180"
                  sortable /></ElTable></ElCard
          ></ElTabPane>
          <ElTabPane :label="t('qqGovernance.tabs.alerts')" name="alerts"
            ><ElCard class="art-table-card" shadow="never"
              ><ElTable v-loading="loading.alerts" :data="alerts.list" border stripe
                ><ElTableColumn
                  prop="kind"
                  :label="t('qqGovernance.fields.status')"
                  width="140"
                /><ElTableColumn
                  prop="group_id"
                  :label="t('qqGovernance.fields.groupId')"
                  width="130"
                /><ElTableColumn
                  prop="message"
                  :label="t('qqGovernance.fields.message')"
                  min-width="280"
                /><ElTableColumn
                  prop="status"
                  :label="t('qqGovernance.fields.status')"
                  width="140"
                /><ElTableColumn :label="t('common.operation')" width="120"
                  ><template #default="{ row }"
                    ><ElButton
                      v-if="row.status === 'open'"
                      text
                      type="primary"
                      @click="acknowledge(row.id)"
                      >{{ t('qqGovernance.actions.ack') }}</ElButton
                    ></template
                  ></ElTableColumn
                ></ElTable
              ></ElCard
            ></ElTabPane
          >
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
                }}</ElDescriptionsItem></ElDescriptions
              ><div class="mt-4"
                ><ElButton type="danger" plain @click="resetRisk">{{
                  t('qqGovernance.actions.resetRisk')
                }}</ElButton></div
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
              value="auto_kick_after_confirmed_mismatch" /></ElSelect></ElFormItem
        ><ElFormItem :label="t('qqGovernance.fields.cardTemplate')"
          ><ElInput
            v-model="policyForm.card_template"
            type="textarea"
            maxlength="60"
            show-word-limit
          /><div class="form-hint">{{ t('qqGovernance.v2.templateHint') }}</div></ElFormItem
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
    editingGroupId = ref(0)
  const policies = ref<Api.QQGovernance.Policy[]>([]),
    groups = ref<Api.QQGovernance.GroupStatus[]>([]),
    corporationOptions = ref<Api.QQGovernance.CorporationOption[]>([]),
    roleOptions = ref<Api.SystemManage.RoleDefinition[]>([]),
    searchingCorporations = ref(false)
  const tasks = ref<Api.QQGovernance.PageResult<Api.QQGovernance.ActionTask>>({
      list: [],
      total: 0,
      page: 1,
      page_size: 100
    }),
    reviews = ref<Api.QQGovernance.PageResult<Api.QQGovernance.Review>>({
      list: [],
      total: 0,
      page: 1,
      page_size: 100
    }),
    alerts = ref<Api.QQGovernance.PageResult<Api.QQGovernance.Alert>>({
      list: [],
      total: 0,
      page: 1,
      page_size: 100
    })
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
    connection = ref<Api.QQGovernance.Connection>({ connected: false, risk_level: 0 })
  const settings = reactive<Api.QQGovernance.Settings>({
    scan_interval_minutes: 15,
    mismatch_confirmations: 2,
    mismatch_observation_hours: 2
  })
  const loading = reactive({
    policies: false,
    groups: false,
    tasks: false,
    reviews: false,
    alerts: false
  })
  const operationFilters = reactive({ groupId: '', status: '', actionType: '', decision: '' })
  const policyForm = reactive({
    group_id: '',
    enabled: true,
    allowed_corporation_ids: [] as number[],
    allowed_role_codes: [] as string[],
    auto_reject_unmatched: false,
    member_violation_policy: 'review_only' as Api.QQGovernance.Policy['member_violation_policy'],
    card_template: ''
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
  function resetPolicyForm() {
    Object.assign(policyForm, {
      group_id: '',
      enabled: true,
      allowed_corporation_ids: [],
      allowed_role_codes: [],
      auto_reject_unmatched: false,
      member_violation_policy: 'review_only',
      card_template: ''
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
    loading.tasks = loading.reviews = loading.alerts = true
    try {
      const [taskRows, reviewRows, alertRows, metricRows] = await Promise.all([
        fetchQQGovernanceTasks({
          page: 1,
          page_size: 100,
          group_id: Number(operationFilters.groupId) || undefined,
          status: operationFilters.status || undefined,
          action_type: operationFilters.actionType || undefined
        }),
        fetchQQGovernanceReviews({
          page: 1,
          page_size: 100,
          group_id: Number(operationFilters.groupId) || undefined,
          decision: operationFilters.decision || undefined
        }),
        fetchQQGovernanceAlerts({
          page: 1,
          page_size: 100,
          status: operationFilters.status || undefined
        }),
        fetchQQGovernanceMetrics()
      ])
      tasks.value = taskRows
      reviews.value = reviewRows
      alerts.value = alertRows
      metrics.value = metricRows
    } finally {
      loading.tasks = loading.reviews = loading.alerts = false
    }
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
        card_template: policy.card_template
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
        card_template: policyForm.card_template
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
    await triggerQQGovernanceReconcile(id)
    ElMessage.success(t('qqGovernance.messages.reconcileSuccess'))
  }
  async function retryTask(id: number) {
    await retryQQGovernanceTask(id)
    await loadOperations()
  }
  async function acknowledge(id: number) {
    await acknowledgeQQGovernanceAlert(id)
    await loadOperations()
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
  .page-title,
  .toolbar {
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
  .toolbar {
    justify-content: flex-end;
    margin-bottom: 12px;
  }
  .qq-governance-tabs :deep(.el-tabs__content) {
    overflow: visible;
  }
  .form-hint {
    margin-top: 6px;
    color: var(--el-text-color-secondary);
    font-size: 12px;
    line-height: 1.5;
  }
</style>
