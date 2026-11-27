<template>
  <ElDialog
    :model-value="true"
    :title="t('qqOfficial.taskDetail')"
    :close-on-click-modal="!busy"
    :close-on-press-escape="!busy"
    :show-close="!busy"
    width="min(900px, 95vw)"
    @close="emit('close')"
  >
    <div v-loading="loading" class="min-h-16 space-y-3">
      <ElAlert v-if="error" :title="t('qqOfficial.loadFailed')" type="error" :closable="false" />
      <template v-if="task">
        <dl class="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
          <div v-for="[label, value] in detailRows" :key="label">
            <dt class="text-sm text-g-600">{{ t(`qqOfficial.${label}`) }}</dt>
            <dd class="break-all">{{ value ?? '—' }}</dd>
          </div>
        </dl>
        <h3 class="font-semibold">{{ t('qqOfficial.attempts') }}</h3>
        <ArtTable :columns="attemptColumns" :data="task.attempts" />
      </template>
    </div>
    <template #footer>
      <ElButton @click="refresh">{{ t('qqOfficial.localReload') }}</ElButton>
      <ElButton
        v-if="canManage && task && canCancelQQTask(task.status)"
        type="danger"
        :loading="busy"
        @click="confirmCancel"
        >{{ t('qqOfficial.cancelTask') }}</ElButton
      >
    </template>
  </ElDialog>
</template>

<script setup lang="ts">
  import { computed } from 'vue'
  import { ElMessageBox } from 'element-plus'
  import { useI18n } from 'vue-i18n'
  import { cancelQQOfficialTask, getQQOfficialTask } from '@/api/generated/client'
  import type { QQOfficialTaskAttempt } from '@/api/generated/client'
  import { unwrapContract } from '@/api/response'
  import { canCancelQQTask } from '@/hooks/qq-official-helpers'
  import {
    qqReadOptions,
    qqTaskPolling,
    useQQOfficialRead,
    type useQQOfficialActions
  } from '@/hooks/useQQOfficialConsole'
  import type { ColumnOption } from '@/types/component'
  import { formatTime } from '@/utils/common/time'

  const props = defineProps<{ id: string; actions: ReturnType<typeof useQQOfficialActions> }>()
  const emit = defineEmits<{ close: [] }>()
  const { t } = useI18n()
  const { busy, canManage } = props.actions
  const {
    data: task,
    loading,
    error,
    refresh,
    replace
  } = useQQOfficialRead(
    (signal) => unwrapContract(getQQOfficialTask(props.id, { signal, ...qqReadOptions })),
    10_000,
    qqTaskPolling
  )
  const stateLabel = (state?: string) =>
    state ? t(`qqOfficial.state.${state}`, t('qqOfficial.unknownState')) : '—'
  const detailRows = computed(() => {
    const value = task.value
    if (!value) return []
    return [
      ['taskId', value.id],
      ['groupId', value.group_id],
      ['kind', stateLabel(value.kind)],
      ['source', stateLabel(value.source)],
      ['status', stateLabel(value.status)],
      ['reason', stateLabel(value.reason)],
      ['trace', value.trace_id],
      ['platformCode', value.upstream_code],
      ['created', formatTime(value.created_at)],
      ['expires', formatTime(value.expires_at)],
      ['nextRun', formatTime(value.next_run_at)],
      ['finished', formatTime(value.finished_at)]
    ]
  })
  const attemptColumns = computed<ColumnOption<QQOfficialTaskAttempt>[]>(() => [
    {
      prop: 'at',
      label: t('qqOfficial.created'),
      minWidth: 160,
      formatter: (row) => formatTime(row.at)
    },
    {
      prop: 'finished_at',
      label: t('qqOfficial.finished'),
      minWidth: 160,
      formatter: (row) => formatTime(row.finished_at)
    },
    {
      prop: 'outcome',
      label: t('qqOfficial.outcome'),
      minWidth: 125,
      formatter: (row) => stateLabel(row.outcome)
    },
    { prop: 'platform_code', label: t('qqOfficial.platformCode'), minWidth: 105 },
    { prop: 'http_status', label: t('qqOfficial.httpStatus'), minWidth: 105 },
    { prop: 'trace_id', label: t('qqOfficial.trace'), minWidth: 170, showOverflowTooltip: true }
  ])
  const confirmCancel = async () => {
    try {
      await ElMessageBox.confirm(t('qqOfficial.cancelConfirm'), t('qqOfficial.cancelTask'), {
        type: 'warning',
        confirmButtonText: t('common.confirm'),
        cancelButtonText: t('common.cancel')
      })
    } catch {
      return
    }
    if (!task.value || !canCancelQQTask(task.value.status)) return
    const cancelled = await props.actions.run(() =>
      unwrapContract(cancelQQOfficialTask(props.id, qqReadOptions))
    )
    if (cancelled) replace(cancelled)
    else await refresh()
  }
</script>
