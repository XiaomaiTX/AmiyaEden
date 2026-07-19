<template>
  <div class="qq-governance-page art-full-height">
    <ElCard class="mb-4" shadow="never">
      <div class="flex items-center justify-between gap-4">
        <div>
          <div class="text-base font-semibold">{{ t('qqGovernance.title') }}</div>
          <div class="mt-1 text-sm text-gray-500">{{ t('qqGovernance.description') }}</div>
        </div>
        <div class="flex gap-2">
          <ElButton @click="loadAll">{{ t('common.refresh') }}</ElButton>
          <ElButton type="danger" plain @click="resetRisk">{{
            t('qqGovernance.actions.resetRisk')
          }}</ElButton>
        </div>
      </div>
    </ElCard>

    <ElTabs v-model="activeTab" @tab-change="loadActiveTab">
      <ElTabPane :label="t('qqGovernance.tabs.policies')" name="policies">
        <ElCard shadow="never">
          <template #header
            ><div class="flex justify-end"
              ><ElButton type="primary" @click="openPolicyDialog()">{{
                t('qqGovernance.actions.addPolicy')
              }}</ElButton></div
            ></template
          >
          <ElTable v-loading="loading.policies" :data="policies" border stripe>
            <ElTableColumn prop="group_id" :label="t('qqGovernance.fields.groupId')" width="150" />
            <ElTableColumn :label="t('qqGovernance.fields.enabled')" width="100"
              ><template #default="{ row }"
                ><ElTag :type="row.enabled ? 'success' : 'info'">{{
                  row.enabled ? t('common.enable') : t('common.disable')
                }}</ElTag></template
              ></ElTableColumn
            >
            <ElTableColumn
              prop="member_violation_policy"
              :label="t('qqGovernance.fields.violationPolicy')"
              min-width="180"
            />
            <ElTableColumn
              prop="scan_interval_minutes"
              :label="t('qqGovernance.fields.scanInterval')"
              width="180"
            />
            <ElTableColumn
              prop="card_template"
              :label="t('qqGovernance.fields.cardTemplate')"
              min-width="220"
              show-overflow-tooltip
            />
            <ElTableColumn :label="t('common.operation')" width="260" fixed="right"
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
          </ElTable>
        </ElCard>
      </ElTabPane>

      <ElTabPane :label="t('qqGovernance.tabs.members')" name="members">
        <ElTabs v-model="memberTab">
          <ElTabPane :label="t('qqGovernance.tabs.members')" name="states"
            ><ElTable v-loading="loading.members" :data="members.list" border stripe
              ><ElTableColumn
                prop="group_id"
                :label="t('qqGovernance.fields.groupId')"
                width="130" /><ElTableColumn
                prop="qq"
                :label="t('qqGovernance.fields.qq')"
                width="150" /><ElTableColumn
                prop="status"
                :label="t('qqGovernance.fields.status')"
                min-width="180" /><ElTableColumn
                prop="mismatch_count"
                :label="t('qqGovernance.fields.confirmations')"
                width="120" /><ElTableColumn
                prop="unknown_count"
                label="UNKNOWN"
                width="110" /><ElTableColumn
                prop="last_checked_at"
                :label="t('common.updatedAt')"
                min-width="180" /></ElTable
          ></ElTabPane>
          <ElTabPane :label="t('qqGovernance.fields.decision')" name="reviews"
            ><ElTable v-loading="loading.reviews" :data="reviews.list" border stripe
              ><ElTableColumn
                prop="group_id"
                :label="t('qqGovernance.fields.groupId')"
                width="130" /><ElTableColumn
                prop="qq"
                :label="t('qqGovernance.fields.qq')"
                width="150" /><ElTableColumn
                prop="decision"
                :label="t('qqGovernance.fields.decision')"
                width="140" /><ElTableColumn
                prop="reason"
                :label="t('qqGovernance.fields.reason')"
                min-width="260"
                show-overflow-tooltip /><ElTableColumn
                prop="created_at"
                :label="t('common.createdAt')"
                min-width="180" /></ElTable
          ></ElTabPane>
        </ElTabs>
      </ElTabPane>

      <ElTabPane :label="t('qqGovernance.tabs.tasks')" name="tasks"
        ><ElTable v-loading="loading.tasks" :data="tasks.list" border stripe
          ><ElTableColumn prop="id" label="ID" width="90" /><ElTableColumn
            prop="action_type"
            :label="t('qqGovernance.fields.actionType')"
            width="130"
          /><ElTableColumn
            prop="group_id"
            :label="t('qqGovernance.fields.groupId')"
            width="130"
          /><ElTableColumn
            prop="qq"
            :label="t('qqGovernance.fields.qq')"
            width="140"
          /><ElTableColumn
            prop="status"
            :label="t('qqGovernance.fields.status')"
            width="130"
          /><ElTableColumn
            prop="retry_count"
            :label="t('qqGovernance.fields.retryCount')"
            width="110"
          /><ElTableColumn
            prop="last_error"
            :label="t('qqGovernance.fields.error')"
            min-width="220"
            show-overflow-tooltip
          /><ElTableColumn :label="t('common.operation')" width="140"
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
        ></ElTabPane
      >
      <ElTabPane :label="t('qqGovernance.tabs.alerts')" name="alerts"
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
            prop="qq"
            :label="t('qqGovernance.fields.qq')"
            width="140"
          /><ElTableColumn
            prop="message"
            :label="t('qqGovernance.fields.message')"
            min-width="280"
          /><ElTableColumn
            prop="status"
            :label="t('qqGovernance.fields.status')"
            width="140"
          /><ElTableColumn :label="t('common.operation')" width="140"
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
        ></ElTabPane
      >
      <ElTabPane :label="t('qqGovernance.tabs.runtime')" name="runtime"
        ><ElRow :gutter="16"
          ><ElCol :span="8"
            ><ElCard shadow="never"
              ><div class="text-sm text-gray-500">{{ t('qqGovernance.fields.connection') }}</div
              ><div class="mt-2 text-xl">{{
                metrics.connected
                  ? t('qqGovernance.values.connected')
                  : t('qqGovernance.values.disconnected')
              }}</div></ElCard
            ></ElCol
          ><ElCol :span="8"
            ><ElCard shadow="never"
              ><div class="text-sm text-gray-500">{{ t('qqGovernance.fields.failureRate') }}</div
              ><div class="mt-2 text-xl">{{ percent(metrics.failure_rate) }}</div></ElCard
            ></ElCol
          ><ElCol :span="8"
            ><ElCard shadow="never"
              ><div class="text-sm text-gray-500">{{ t('qqGovernance.fields.dead') }}</div
              ><div class="mt-2 text-xl">{{ metrics.dead }}</div></ElCard
            ></ElCol
          ></ElRow
        ></ElTabPane
      >
    </ElTabs>

    <ElDialog
      v-model="policyDialog"
      :title="t('qqGovernance.title')"
      width="620px"
      destroy-on-close
    >
      <ElForm :model="policyForm" label-width="160px">
        <ElFormItem :label="t('qqGovernance.fields.groupId')"
          ><ElInputNumber
            v-model="policyForm.group_id"
            :disabled="editingGroupId > 0"
            :min="1"
            class="!w-full"
        /></ElFormItem>
        <ElFormItem :label="t('qqGovernance.fields.enabled')"
          ><ElSwitch v-model="policyForm.enabled"
        /></ElFormItem>
        <ElFormItem :label="t('qqGovernance.fields.corporations')"
          ><ElSelect
            v-model="policyForm.allowed_corporation_ids"
            multiple
            filterable
            allow-create
            default-first-option
            class="!w-full"
            :placeholder="t('qqGovernance.placeholders.corporations')"
        /></ElFormItem>
        <ElFormItem :label="t('qqGovernance.fields.roles')"
          ><ElSelect
            v-model="policyForm.allowed_role_codes"
            multiple
            class="!w-full"
            :placeholder="t('qqGovernance.placeholders.roles')"
            ><ElOption
              v-for="role in roleOptions"
              :key="role"
              :label="role"
              :value="role" /></ElSelect
        ></ElFormItem>
        <ElFormItem :label="t('qqGovernance.fields.autoReject')"
          ><ElSwitch v-model="policyForm.auto_reject_unmatched"
        /></ElFormItem>
        <ElFormItem :label="t('qqGovernance.fields.violationPolicy')"
          ><ElSelect v-model="policyForm.member_violation_policy" class="!w-full"
            ><ElOption :label="t('qqGovernance.values.reviewOnly')" value="review_only" /><ElOption
              :label="t('qqGovernance.values.autoKick')"
              value="auto_kick_after_confirmed_mismatch" /></ElSelect
        ></ElFormItem>
        <ElFormItem :label="t('qqGovernance.fields.scanInterval')"
          ><ElInputNumber
            v-model="policyForm.scan_interval_minutes"
            :min="15"
            :max="360"
            :step="15"
        /></ElFormItem>
        <ElFormItem :label="t('qqGovernance.fields.confirmations')"
          ><ElInputNumber v-model="policyForm.mismatch_confirmations" :min="2" :max="3"
        /></ElFormItem>
        <ElFormItem :label="t('qqGovernance.fields.observation')"
          ><ElInputNumber v-model="policyForm.mismatch_observation_hours" :min="1" :max="6"
        /></ElFormItem>
        <ElFormItem :label="t('qqGovernance.fields.cardTemplate')"
          ><ElInput
            v-model="policyForm.card_template"
            type="textarea"
            :placeholder="t('qqGovernance.placeholders.cardTemplate')"
        /></ElFormItem>
      </ElForm>
      <template #footer
        ><ElButton @click="policyDialog = false">{{ t('common.cancel') }}</ElButton
        ><ElButton type="primary" :loading="saving" @click="savePolicy">{{
          t('qqGovernance.actions.save')
        }}</ElButton></template
      >
    </ElDialog>
  </div>
</template>

<script setup lang="ts">
  import { reactive, ref } from 'vue'
  import { useI18n } from 'vue-i18n'
  import {
    acknowledgeQQGovernanceAlert,
    deleteQQGovernancePolicy,
    fetchQQGovernanceAlerts,
    fetchQQGovernanceMembers,
    fetchQQGovernanceMetrics,
    fetchQQGovernancePolicies,
    fetchQQGovernanceReviews,
    fetchQQGovernanceTasks,
    resetQQGovernanceRisk,
    retryQQGovernanceTask,
    saveQQGovernancePolicy,
    triggerQQGovernanceReconcile,
    updateQQGovernancePolicy
  } from '@/api/qq-governance'

  defineOptions({ name: 'QQGovernance' })
  const { t } = useI18n()
  const activeTab = ref('policies')
  const memberTab = ref('states')
  const policyDialog = ref(false)
  const saving = ref(false)
  const editingGroupId = ref(0)
  const policies = ref<Api.QQGovernance.Policy[]>([])
  const members = ref<Api.QQGovernance.PageResult<Api.QQGovernance.MemberState>>({
    list: [],
    total: 0,
    page: 1,
    page_size: 100
  })
  const reviews = ref<Api.QQGovernance.PageResult<Api.QQGovernance.Review>>({
    list: [],
    total: 0,
    page: 1,
    page_size: 100
  })
  const tasks = ref<Api.QQGovernance.PageResult<Api.QQGovernance.ActionTask>>({
    list: [],
    total: 0,
    page: 1,
    page_size: 100
  })
  const alerts = ref<Api.QQGovernance.PageResult<Api.QQGovernance.Alert>>({
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
  })
  const loading = reactive({
    policies: false,
    members: false,
    reviews: false,
    tasks: false,
    alerts: false
  })
  type PolicyForm = Omit<Api.QQGovernance.Policy, 'id' | 'updated_by' | 'updated_at'>
  const roleOptions = [
    'admin',
    'senior_fc',
    'fc',
    'srp',
    'shop_order_manage',
    'welfare',
    'captain',
    'mentor',
    'fuel_officer',
    'user',
    'guest'
  ]
  const defaultPolicy = (): PolicyForm => ({
    group_id: 0,
    enabled: true,
    allowed_corporation_ids: [],
    allowed_role_codes: [],
    auto_reject_unmatched: false,
    member_violation_policy: 'review_only',
    scan_interval_minutes: 15,
    mismatch_confirmations: 2,
    mismatch_observation_hours: 2,
    card_template: ''
  })
  const policyForm = reactive(defaultPolicy())
  const resetPolicyForm = (value: PolicyForm = defaultPolicy()) => Object.assign(policyForm, value)
  const percent = (value: number) => `${(value * 100).toFixed(1)}%`
  const loadPolicies = async () => {
    loading.policies = true
    try {
      policies.value = await fetchQQGovernancePolicies()
    } finally {
      loading.policies = false
    }
  }
  const loadMembers = async () => {
    loading.members = true
    try {
      members.value = await fetchQQGovernanceMembers({ page: 1, page_size: 100 })
    } finally {
      loading.members = false
    }
  }
  const loadReviews = async () => {
    loading.reviews = true
    try {
      reviews.value = await fetchQQGovernanceReviews({ page: 1, page_size: 100 })
    } finally {
      loading.reviews = false
    }
  }
  const loadTasks = async () => {
    loading.tasks = true
    try {
      tasks.value = await fetchQQGovernanceTasks({ page: 1, page_size: 100 })
    } finally {
      loading.tasks = false
    }
  }
  const loadAlerts = async () => {
    loading.alerts = true
    try {
      alerts.value = await fetchQQGovernanceAlerts({ page: 1, page_size: 100 })
    } finally {
      loading.alerts = false
    }
  }
  const loadRuntime = async () => {
    metrics.value = await fetchQQGovernanceMetrics()
  }
  const loadActiveTab = async () => {
    if (activeTab.value === 'policies') await loadPolicies
    if (activeTab.value === 'members') {
      await Promise.all([loadMembers(), loadReviews()])
    }
    if (activeTab.value === 'tasks') await loadTasks
    if (activeTab.value === 'alerts') await loadAlerts
    if (activeTab.value === 'runtime') await loadRuntime
  }
  const loadAll = async () => {
    await Promise.all([
      loadPolicies(),
      loadMembers(),
      loadReviews(),
      loadTasks(),
      loadAlerts(),
      loadRuntime()
    ])
  }
  const openPolicyDialog = (policy?: Api.QQGovernance.Policy) => {
    editingGroupId.value = policy?.group_id ?? 0
    resetPolicyForm(
      policy
        ? {
            group_id: policy.group_id,
            enabled: policy.enabled,
            allowed_corporation_ids: policy.allowed_corporation_ids,
            allowed_role_codes: policy.allowed_role_codes,
            auto_reject_unmatched: policy.auto_reject_unmatched,
            member_violation_policy: policy.member_violation_policy,
            scan_interval_minutes: policy.scan_interval_minutes,
            mismatch_confirmations: policy.mismatch_confirmations,
            mismatch_observation_hours: policy.mismatch_observation_hours,
            card_template: policy.card_template
          }
        : defaultPolicy()
    )
    policyDialog.value = true
  }
  const savePolicy = async () => {
    saving.value = true
    try {
      const allowedCorporationIDs = policyForm.allowed_corporation_ids.map((value) => Number(value))
      if (editingGroupId.value) {
        await updateQQGovernancePolicy(editingGroupId.value, {
          enabled: policyForm.enabled,
          allowed_corporation_ids: allowedCorporationIDs,
          allowed_role_codes: policyForm.allowed_role_codes,
          auto_reject_unmatched: policyForm.auto_reject_unmatched,
          member_violation_policy: policyForm.member_violation_policy,
          scan_interval_minutes: policyForm.scan_interval_minutes,
          mismatch_confirmations: policyForm.mismatch_confirmations,
          mismatch_observation_hours: policyForm.mismatch_observation_hours,
          card_template: policyForm.card_template
        })
      } else {
        await saveQQGovernancePolicy({
          ...policyForm,
          allowed_corporation_ids: allowedCorporationIDs
        })
      }
      ElMessage.success(t('qqGovernance.messages.saveSuccess'))
      policyDialog.value = false
      await loadPolicies()
    } finally {
      saving.value = false
    }
  }
  const removePolicy = async (groupId: number) => {
    await deleteQQGovernancePolicy(groupId)
    await loadPolicies()
  }
  const reconcile = async (groupId: number) => {
    await triggerQQGovernanceReconcile(groupId)
    ElMessage.success(t('qqGovernance.messages.reconcileSuccess'))
  }
  const retryTask = async (id: number) => {
    await retryQQGovernanceTask(id)
    await loadTasks()
  }
  const acknowledge = async (id: number) => {
    await acknowledgeQQGovernanceAlert(id)
    await loadAlerts()
  }
  const resetRisk = async () => {
    await resetQQGovernanceRisk()
    ElMessage.success(t('qqGovernance.messages.riskReset'))
    await loadRuntime()
  }
  void loadAll()
</script>
