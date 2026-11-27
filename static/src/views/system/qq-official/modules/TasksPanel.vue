<template>
  <div class="space-y-3">
    <div class="flex flex-wrap gap-2">
      <ElSelect
        v-for="[key, options] in filters"
        :key="key"
        v-model="searchParams[key]"
        :aria-label="t(`qqOfficial.${key}`)"
        :placeholder="t(`qqOfficial.${key}`)"
        clearable
        class="w-44"
        @change="getData()"
      >
        <ElOption v-for="value in options" :key="value" :label="stateLabel(value)" :value="value" />
      </ElSelect>
    </div>
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
        <template #id="{ row }"
          ><ElButton link type="primary" @click="emit('task', row.id)">{{
            row.id
          }}</ElButton></template
        >
      </ArtTable>
    </ElCard>
  </div>
</template>

<script setup lang="ts">
  import { useI18n } from 'vue-i18n'
  import { listQQOfficialTasks } from '@/api/generated/client'
  import type { ListQQOfficialTasksParams, QQOfficialTask } from '@/api/generated/client'
  import { unwrapContract } from '@/api/response'
  import { useTable } from '@/hooks/core/useTable'
  import { taskKinds, taskSources, taskStatuses } from '@/hooks/qq-official-helpers'
  import { qqReadOptions } from '@/hooks/useQQOfficialConsole'
  import { formatTime } from '@/utils/common/time'

  const emit = defineEmits<{ task: [id: string] }>()
  const { t } = useI18n()
  const filters = [
    ['status', taskStatuses],
    ['kind', taskKinds],
    ['source', taskSources]
  ] as const
  const stateLabel = (state?: string) =>
    state ? t(`qqOfficial.state.${state}`, t('qqOfficial.unknownState')) : '—'
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
      apiFn: (query: ListQQOfficialTasksParams) =>
        unwrapContract(listQQOfficialTasks(query, qqReadOptions)),
      apiParams: { current: 1, size: 200 },
      columnsFactory: () => [
        { prop: 'id', label: t('qqOfficial.taskId'), useSlot: true, minWidth: 290 },
        ...(['kind', 'source', 'status', 'reason'] as const).map((prop) => ({
          prop,
          label: t(`qqOfficial.${prop}`),
          minWidth: 140,
          formatter: (task: QQOfficialTask) => stateLabel(task[prop])
        })),
        {
          prop: 'created_at',
          label: t('qqOfficial.created'),
          minWidth: 165,
          formatter: (task) => formatTime(task.created_at)
        },
        {
          prop: 'expires_at',
          label: t('qqOfficial.expires'),
          minWidth: 165,
          formatter: (task) => formatTime(task.expires_at)
        }
      ]
    }
  })
</script>
