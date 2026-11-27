<template>
  <div class="space-y-3">
    <div class="flex flex-wrap items-center justify-between gap-2">
      <h2 class="font-semibold">{{ t('qqOfficial.requests') }} · {{ group.alias || group.id }}</h2>
      <div class="flex flex-wrap gap-2">
        <ElSelect
          v-model="searchParams.status"
          :aria-label="t('qqOfficial.status')"
          :placeholder="t('qqOfficial.status')"
          clearable
          class="w-44"
          @change="getData()"
        >
          <ElOption
            v-for="value in requestStatuses"
            :key="value"
            :label="stateLabel(value)"
            :value="value"
          />
        </ElSelect>
        <ElButton
          v-if="canManage"
          type="primary"
          :disabled="busy || !ready || !group.present || !group.authorized"
          @click="queueRefresh"
          >{{ t('qqOfficial.requestRefresh') }}</ElButton
        >
      </div>
    </div>
    <ElAlert
      v-if="status && !status.approval_writes_enabled"
      :title="t('qqOfficial.writesOff')"
      type="info"
      :closable="false"
    />
    <ElCard class="art-table-card" shadow="never">
      <ArtTableHeader v-model:columns="columnChecks" :loading="loading" @refresh="refreshData" />
      <ElAlert v-if="error" :title="t('qqOfficial.loadFailed')" type="error" :closable="false" />
      <ArtTable
        :columns="columns"
        :data="data"
        :loading="loading"
        :pagination="pagination"
        visual-variant="ledger"
        row-key="id"
        @pagination:current-change="handleCurrentChange"
      >
        <template #actions="{ row }">
          <div class="flex flex-wrap gap-2">
            <ElButton v-if="row.task_id" link type="primary" @click="emit('task', row.task_id)">{{
              t('qqOfficial.taskDetail')
            }}</ElButton>
            <template v-if="canManage && canDecideQQRequest(status, group, row)">
              <ElButton
                link
                type="primary"
                :disabled="busy"
                @click="confirmDecision(row, 'approve')"
                >{{ t('qqOfficial.approve') }}</ElButton
              >
              <ElButton
                link
                type="danger"
                :disabled="busy"
                @click="confirmDecision(row, 'decline')"
                >{{ t('qqOfficial.decline') }}</ElButton
              >
            </template>
          </div>
        </template>
      </ArtTable>
    </ElCard>
  </div>
</template>

<script setup lang="ts">
  import { computed } from 'vue'
  import { ElMessage, ElMessageBox } from 'element-plus'
  import { useI18n } from 'vue-i18n'
  import {
    decideQQOfficialJoinRequest,
    listQQOfficialJoinRequests,
    refreshQQOfficialJoinRequests
  } from '@/api/generated/client'
  import type {
    ListQQOfficialJoinRequestsParams,
    QQOfficialGroup,
    QQOfficialJoinDecision,
    QQOfficialJoinRequest,
    QQOfficialStatus
  } from '@/api/generated/client'
  import { unwrapContract } from '@/api/response'
  import { useTable } from '@/hooks/core/useTable'
  import {
    canDecideQQRequest,
    isQQDispatchReady,
    requestStatuses
  } from '@/hooks/qq-official-helpers'
  import { qqReadOptions, type useQQOfficialActions } from '@/hooks/useQQOfficialConsole'
  import { formatTime } from '@/utils/common/time'

  const props = defineProps<{
    group: QQOfficialGroup
    status: QQOfficialStatus | null
    actions: ReturnType<typeof useQQOfficialActions>
  }>()
  const emit = defineEmits<{ task: [id: string] }>()
  const { t } = useI18n()
  const { busy, canManage } = props.actions
  const ready = computed(() => isQQDispatchReady(props.status))
  const stateLabel = (state: string) => t(`qqOfficial.state.${state}`, t('qqOfficial.unknownState'))
  const {
    data,
    error,
    loading,
    columns,
    columnChecks,
    pagination,
    searchParams,
    refreshData,
    getData,
    handleCurrentChange
  } = useTable({
    core: {
      apiFn: (query: ListQQOfficialJoinRequestsParams) =>
        unwrapContract(listQQOfficialJoinRequests(props.group.id, query, qqReadOptions)),
      apiParams: { current: 1, size: 200 },
      columnsFactory: () => [
        {
          prop: 'username',
          label: t('qqOfficial.username'),
          minWidth: 150,
          showOverflowTooltip: true
        },
        {
          prop: 'applied_at',
          label: t('qqOfficial.applied'),
          minWidth: 160,
          formatter: (row) => formatTime(row.applied_at)
        },
        {
          prop: 'expires_at',
          label: t('qqOfficial.expires'),
          minWidth: 160,
          formatter: (row) => formatTime(row.expires_at)
        },
        {
          prop: 'status',
          label: t('qqOfficial.status'),
          minWidth: 120,
          formatter: (row) => stateLabel(row.status)
        },
        { prop: 'actions', label: t('common.operation'), useSlot: true, minWidth: 180 }
      ]
    }
  })
  const queueRefresh = async () => {
    const id = props.group.id
    const receipt = await props.actions.enqueue(`refresh:${id}`, (key) =>
      refreshQQOfficialJoinRequests(id, { idempotency_key: key }, qqReadOptions)
    )
    if (receipt) emit('task', receipt.task_id)
  }
  const confirmDecision = async (
    request: QQOfficialJoinRequest,
    decision: QQOfficialJoinDecision
  ) => {
    try {
      await ElMessageBox.confirm(
        t(`qqOfficial.${decision}Confirm`, { name: request.username }),
        t(`qqOfficial.${decision}`),
        {
          type: 'warning',
          confirmButtonText: t('common.confirm'),
          cancelButtonText: t('common.cancel')
        }
      )
    } catch {
      return
    }
    if (!canDecideQQRequest(props.status, props.group, request)) {
      ElMessage.warning(t('qqOfficial.conflict'))
      return
    }
    const receipt = await props.actions.enqueue(`decision:${request.id}:${decision}`, (key) =>
      decideQQOfficialJoinRequest(request.id, { decision, idempotency_key: key }, qqReadOptions)
    )
    if (receipt) emit('task', receipt.task_id)
    refreshData()
  }
</script>
