<template>
  <ElCard class="mt-4" shadow="never">
    <template #header>
      <div class="flex flex-wrap items-center justify-between gap-2">
        <h2 class="font-medium">{{ t('fleet.qqPing.results') }}</h2>
        <div class="flex flex-wrap gap-2">
          <ElButton size="small" :disabled="loading" @click="refreshData">
            {{ t('fleet.qqPing.reload') }}
          </ElButton>
          <ElButton
            type="warning"
            size="small"
            :loading="sending"
            :disabled="queuedCount !== null"
            @click="queue"
          >
            {{ t(queueError ? 'fleet.qqPing.retry' : 'fleet.qqPing.queue') }}
          </ElButton>
          <ElButton v-if="queuedCount !== null" size="small" @click="prepareAnother">
            {{ t('fleet.qqPing.prepareAnother') }}
          </ElButton>
        </div>
      </div>
    </template>
    <ElAlert
      v-if="queuedCount !== null"
      class="mb-2"
      :title="t('fleet.qqPing.queued', { count: queuedCount })"
      type="success"
      :closable="false"
    />
    <ElAlert v-if="queueError" class="mb-2" :title="t(queueError)" type="error" :closable="false" />
    <ElAlert
      v-if="error"
      class="mb-2"
      :title="t('fleet.qqPing.loadFailed')"
      type="error"
      :closable="false"
    />
    <ArtTable
      :columns="columns"
      :data="data"
      :loading="loading"
      :pagination="pagination"
      :pagination-options="{ pageSizes: [100] }"
      :empty-text="error ? t('fleet.qqPing.loadFailed') : t('fleet.qqPing.empty')"
      :show-table-header="false"
      visual-variant="ledger"
      row-key="id"
      @pagination:current-change="handleCurrentChange"
    />
  </ElCard>
</template>

<script setup lang="ts">
  import { useIntervalFn } from '@vueuse/core'
  import { useI18n } from 'vue-i18n'
  import {
    listFleetQQPingResults,
    sendFleetQQPing,
    type ListFleetQQPingResultsParams
  } from '@/api/generated/client'
  import { unwrapContract } from '@/api/response'
  import { useTable } from '@/hooks/core/useTable'
  import { shouldPollQQTask } from '@/hooks/qq-official-helpers'
  import { formatTime } from '@/utils/common/time'
  import { createFleetQQPingIntent, fleetQQPingErrorKey } from './qqPing'

  const props = defineProps<{ fleetId: string }>()
  const { t } = useI18n()
  const intent = createFleetQQPingIntent()
  const sending = ref(false)
  const queuedCount = ref<number | null>(null)
  const queueError = ref('')
  const quiet = { showErrorMessage: false }
  const stateLabel = (state?: string) =>
    state ? t(`qqOfficial.state.${state}`, t('qqOfficial.unknownState')) : '—'

  const {
    columns,
    data,
    error,
    loading,
    pagination,
    handleCurrentChange,
    refreshData,
    refreshCreate,
    refreshSoft
  } = useTable({
    core: {
      apiFn: (params: ListFleetQQPingResultsParams) =>
        unwrapContract(listFleetQQPingResults(props.fleetId, params, quiet)),
      // This endpoint caps page size at 100, below the shared ledger preset of 200.
      apiParams: { current: 1, size: 100 },
      columnsFactory: () => [
        {
          prop: 'status',
          label: t('common.status'),
          minWidth: 130,
          formatter: (row) => stateLabel(row.status)
        },
        {
          prop: 'reason',
          label: t('qqOfficial.reason'),
          minWidth: 155,
          formatter: (row) => stateLabel(row.reason)
        },
        {
          prop: 'created_at',
          label: t('qqOfficial.created'),
          minWidth: 170,
          formatter: (row) => formatTime(row.created_at)
        },
        {
          prop: 'finished_at',
          label: t('qqOfficial.finished'),
          minWidth: 170,
          formatter: (row) => formatTime(row.finished_at)
        }
      ]
    }
  })

  const { pause, resume } = useIntervalFn(() => {
    if (
      !document.hidden &&
      !loading.value &&
      data.value.some((row) => shouldPollQQTask(row.status))
    ) {
      void refreshSoft()
    }
  }, 5000)
  onActivated(resume)
  onDeactivated(pause)

  async function queue() {
    const request = intent.submit((idempotency_key) =>
      unwrapContract(sendFleetQQPing(props.fleetId, { idempotency_key }, quiet))
    )
    if (!request) return
    sending.value = true
    queueError.value = ''
    try {
      const receipt = await request
      queuedCount.value = receipt.tasks.length
      void refreshCreate()
    } catch (error) {
      queueError.value = fleetQQPingErrorKey(error)
    } finally {
      sending.value = false
    }
  }

  function prepareAnother() {
    if (intent.resetAfterAcceptance()) queuedCount.value = null
  }
</script>
