<template>
  <ElCard class="art-table-card" shadow="never">
    <ArtTableHeader v-model:columns="columnChecks" :loading="loading" @refresh="refreshData">
      <template #left>
        <ElButton
          v-if="canManage"
          :disabled="!ready || busy"
          type="primary"
          @click="verifyOpen = true"
          >{{ t('qqOfficial.verifyGroup') }}</ElButton
        >
      </template>
    </ArtTableHeader>
    <ElAlert v-if="error" :title="t('qqOfficial.loadFailed')" type="error" :closable="false" />
    <ArtTable
      :columns="columns"
      :data="data"
      :loading="loading"
      :pagination="pagination"
      row-key="id"
      @pagination:current-change="handleCurrentChange"
      @pagination:size-change="handleSizeChange"
    >
      <template #alias="{ row }">
        <div>{{ row.alias || '—' }}</div>
        <div class="text-xs text-g-600">{{ row.id }}</div>
      </template>
      <template #actions="{ row }">
        <div class="flex flex-wrap gap-x-2">
          <ElButton v-if="canManage" link :disabled="busy" @click="edit(row)">{{
            t('qqOfficial.editGroup')
          }}</ElButton>
          <ElButton
            v-if="canManage"
            link
            :disabled="
              busy || !ready || !row.present || !row.authorized || row.push_state === 'blocked'
            "
            @click="queueTest(row.id)"
            >{{ t('qqOfficial.sendTest') }}</ElButton
          >
          <ElButton link @click="emit('requests', row)">{{ t('qqOfficial.requests') }}</ElButton>
        </div>
      </template>
    </ArtTable>
  </ElCard>
  <ElDialog
    v-model="editOpen"
    :title="t('qqOfficial.editGroup')"
    :close-on-click-modal="!busy"
    :close-on-press-escape="!busy"
    :show-close="!busy"
    width="min(520px, 95vw)"
  >
    <ElForm label-position="top" :disabled="busy">
      <ElFormItem :label="t('qqOfficial.alias')"
        ><ElInput v-model="draft.alias" maxlength="80"
      /></ElFormItem>
      <div class="mb-3 flex flex-wrap gap-x-2">
        <ElCheckbox
          v-model="draft.authorized"
          :disabled="!draft.authorized && !editing?.present"
          @change="clearModesOnOptOut"
          >{{ t('qqOfficial.optIn') }}</ElCheckbox
        >
        <ElCheckbox
          v-model="draft.notification_enabled"
          :disabled="!draft.notification_enabled && !canEnableNotifications"
          >{{ t('qqOfficial.notifications') }}</ElCheckbox
        >
        <ElCheckbox
          v-model="draft.approval_enabled"
          :disabled="!draft.approval_enabled && !canEnableApproval"
          >{{ t('qqOfficial.approval') }}</ElCheckbox
        >
      </div>
      <ElFormItem :label="t('qqOfficial.fleetTarget')"
        ><ElInput v-model="draft.fleet" inputmode="numeric"
      /></ElFormItem>
      <ElFormItem :label="t('qqOfficial.structureTarget')"
        ><ElInput v-model="draft.structure" inputmode="numeric"
      /></ElFormItem>
      <p class="text-sm text-g-600">{{ t('qqOfficial.targetHint') }}</p>
    </ElForm>
    <template #footer>
      <ElButton :disabled="busy" @click="editOpen = false">{{ t('common.cancel') }}</ElButton>
      <ElButton
        type="primary"
        :disabled="!patch || Object.keys(patch).length === 0"
        :loading="busy"
        @click="saveGroup"
        >{{ t('common.save') }}</ElButton
      >
    </template>
  </ElDialog>
  <ElDialog
    v-model="verifyOpen"
    :title="t('qqOfficial.verifyGroup')"
    :close-on-click-modal="!busy"
    :close-on-press-escape="!busy"
    :show-close="!busy"
    width="min(520px, 95vw)"
    @closed="clearVerify"
  >
    <p class="mb-3">{{ t('qqOfficial.verifyHint') }}</p>
    <ElForm label-position="top">
      <ElFormItem :label="t('qqOfficial.openid')"
        ><ElInput
          v-model="openid"
          :disabled="busy"
          autocomplete="off"
          maxlength="128"
          @input="actions.clearKey('verify')"
      /></ElFormItem>
    </ElForm>
    <template #footer>
      <ElButton :disabled="busy" @click="verifyOpen = false">{{ t('common.cancel') }}</ElButton>
      <ElButton
        type="primary"
        :disabled="!ready || !isQQGroupOpenID(openid)"
        :loading="busy"
        @click="verifyGroup"
        >{{ t('qqOfficial.verifyGroup') }}</ElButton
      >
    </template>
  </ElDialog>
</template>

<script setup lang="ts">
  import { computed, reactive, ref, shallowRef } from 'vue'
  import { ElMessage } from 'element-plus'
  import { useI18n } from 'vue-i18n'
  import {
    listQQOfficialGroups,
    testQQOfficialGroup,
    updateQQOfficialGroup,
    verifyQQOfficialGroup
  } from '@/api/generated/client'
  import type {
    ListQQOfficialGroupsParams,
    QQOfficialGroup,
    QQOfficialGroupUpdate,
    QQOfficialStatus
  } from '@/api/generated/client'
  import { unwrapContract } from '@/api/response'
  import { useTable } from '@/hooks/core/useTable'
  import {
    isQQDispatchReady,
    isQQGroupOpenID,
    parseQQCorporationTarget
  } from '@/hooks/qq-official-helpers'
  import { qqReadOptions, type useQQOfficialActions } from '@/hooks/useQQOfficialConsole'

  const props = defineProps<{
    status: QQOfficialStatus | null
    actions: ReturnType<typeof useQQOfficialActions>
  }>()
  const emit = defineEmits<{ task: [id: string]; requests: [group: QQOfficialGroup] }>()
  const { t } = useI18n()
  const { busy, canManage } = props.actions
  const ready = computed(() => isQQDispatchReady(props.status))
  const editOpen = ref(false)
  const verifyOpen = ref(false)
  const openid = ref('')
  const editing = shallowRef<QQOfficialGroup>()
  const draft = reactive({
    alias: '',
    authorized: false,
    notification_enabled: false,
    approval_enabled: false,
    fleet: '',
    structure: ''
  })
  const flags = [
    ['present', 'present'],
    ['authorized', 'optIn'],
    ['notification_enabled', 'notifications'],
    ['test_verified', 'testVerified'],
    ['approval_enabled', 'approval']
  ] as const
  const stateLabel = (state: string) => t(`qqOfficial.state.${state}`, t('qqOfficial.unknownState'))

  const {
    data,
    loading,
    error,
    columns,
    columnChecks,
    pagination,
    refreshData,
    handleCurrentChange,
    handleSizeChange
  } = useTable({
    core: {
      apiFn: (query: ListQQOfficialGroupsParams) =>
        unwrapContract(listQQOfficialGroups(query, qqReadOptions)),
      apiParams: { current: 1, size: 20 },
      columnsFactory: () => [
        { prop: 'alias', label: t('qqOfficial.alias'), useSlot: true, minWidth: 240 },
        ...flags.map(([key, label]) => ({
          prop: key,
          label: t(`qqOfficial.${label}`),
          minWidth: 115,
          formatter: (group: QQOfficialGroup) => t(group[key] ? 'qqOfficial.yes' : 'qqOfficial.no')
        })),
        {
          prop: 'push_state',
          label: t('qqOfficial.pushState'),
          minWidth: 120,
          formatter: (group) => stateLabel(group.push_state)
        },
        {
          prop: 'approval_capability',
          label: t('qqOfficial.approvalCapability'),
          minWidth: 125,
          formatter: (group) => stateLabel(group.approval_capability)
        },
        { prop: 'actions', label: t('common.operation'), useSlot: true, minWidth: 230 }
      ]
    }
  })
  const canEnableNotifications = computed(
    () => draft.authorized && editing.value?.test_verified && editing.value.push_state === 'allowed'
  )
  const canEnableApproval = computed(
    () =>
      draft.authorized &&
      props.status?.approval_writes_enabled &&
      editing.value?.approval_capability === 'available'
  )
  const clearModesOnOptOut = () => {
    if (!draft.authorized) {
      draft.notification_enabled = false
      draft.approval_enabled = false
    }
  }
  const patch = computed<QQOfficialGroupUpdate | undefined>(() => {
    const group = editing.value
    const fleet = parseQQCorporationTarget(draft.fleet)
    const structure = parseQQCorporationTarget(draft.structure)
    if (
      !group ||
      fleet === undefined ||
      structure === undefined ||
      (draft.authorized && (!group.present || !draft.alias.trim()))
    )
      return undefined
    const update: QQOfficialGroupUpdate = {}
    if (draft.alias.trim() !== group.alias) update.alias = draft.alias.trim()
    for (const key of ['authorized', 'notification_enabled', 'approval_enabled'] as const) {
      if (draft[key] !== group[key]) update[key] = draft[key]
    }
    if (fleet !== (group.fleet_corporation_id ?? 0)) update.fleet_corporation_id = fleet
    if (structure !== (group.structure_corporation_id ?? 0))
      update.structure_corporation_id = structure
    return update
  })
  const edit = (group: QQOfficialGroup) => {
    editing.value = group
    Object.assign(draft, {
      alias: group.alias,
      authorized: group.authorized,
      notification_enabled: group.notification_enabled,
      approval_enabled: group.approval_enabled,
      fleet: String(group.fleet_corporation_id ?? ''),
      structure: String(group.structure_corporation_id ?? '')
    })
    editOpen.value = true
  }
  const saveGroup = async () => {
    const id = editing.value?.id
    const update = patch.value
    if (!id || !update || !Object.keys(update).length) return
    const saved = await props.actions.run(() =>
      unwrapContract(updateQQOfficialGroup(id, update, qqReadOptions))
    )
    if (saved) {
      editOpen.value = false
      ElMessage.success(t('qqOfficial.saved'))
      refreshData()
    }
  }
  const queueTest = async (id: string) => {
    const receipt = await props.actions.enqueue(`test:${id}`, (key) =>
      testQQOfficialGroup(id, { idempotency_key: key }, qqReadOptions)
    )
    if (receipt) emit('task', receipt.task_id)
  }
  const clearVerify = () => {
    openid.value = ''
    props.actions.clearKey('verify')
  }
  const verifyGroup = async () => {
    if (!ready.value || !isQQGroupOpenID(openid.value)) return
    const receipt = await props.actions.enqueue('verify', (key) =>
      verifyQQOfficialGroup(
        { group_openid: openid.value.trim(), idempotency_key: key },
        qqReadOptions
      )
    )
    if (receipt) {
      verifyOpen.value = false
      clearVerify()
      emit('task', receipt.task_id)
    }
  }
</script>
