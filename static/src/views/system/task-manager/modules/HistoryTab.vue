<template>
  <div class="task-manager-history">
    <div class="task-manager-history__filters">
      <ElSelect
        v-model="filters.task_name"
        :placeholder="t('taskManager.filters.taskName')"
        clearable
        filterable
        style="width: 220px"
        @change="handleSearch"
      >
        <ElOption
          v-for="task in taskOptions"
          :key="task.name"
          :label="taskDisplayName(task.name)"
          :value="task.name"
        />
      </ElSelect>
      <ElSelect
        v-model="filters.status"
        :placeholder="t('taskManager.filters.status')"
        clearable
        style="width: 160px"
        @change="handleSearch"
      >
        <ElOption :label="t('taskManager.status.running')" value="running" />
        <ElOption :label="t('taskManager.status.success')" value="success" />
        <ElOption :label="t('taskManager.status.failed')" value="failed" />
      </ElSelect>
      <ElButton type="primary" @click="handleSearch">{{ t('common.search') }}</ElButton>
      <ElButton @click="resetFilters">{{ t('common.reset') }}</ElButton>
    </div>

    <ElCard class="art-table-card" shadow="never">
      <ArtTableHeader v-model:columns="columnChecks" :loading="loading" @refresh="refreshData" />

      <ArtTable
        :loading="loading"
        :data="historyData"
        :columns="columns"
        :pagination="pagination"
        visual-variant="ledger"
        @pagination:size-change="handleSizeChange"
        @pagination:current-change="handleCurrentChange"
      />
    </ElCard>
  </div>
</template>

<script setup lang="ts">
  import { fetchTaskHistory, fetchTasks } from '@/api/task-manager'
  import { getTaskDisplayName } from '../task-labels'
  import { formatTime } from '@utils/common'
  import { ElButton, ElCard, ElOption, ElSelect, ElTag, ElTooltip } from 'element-plus'
  import { h } from 'vue'
  import { useI18n } from 'vue-i18n'
  import { useTable } from '@/hooks/core/useTable'

  const { t } = useI18n()

  const taskOptions = ref<Api.TaskManager.TaskItem[]>([])
  const filters = reactive({
    task_name: '',
    status: ''
  })

  const taskDisplayName = (taskName: string) => getTaskDisplayName(t, taskName, taskName)

  const triggerTagType = (trigger: Api.TaskManager.TaskExecutionItem['trigger']) => {
    const map = {
      cron: 'info',
      manual: 'warning'
    } as const

    return map[trigger]
  }

  const statusTagType = (status: Api.TaskManager.TaskExecutionItem['status']) => {
    const map = {
      running: 'warning',
      success: 'success',
      failed: 'danger'
    } as const

    return map[status]
  }

  const formatDuration = (durationMs?: number) => {
    if (durationMs == null) {
      return '-'
    }
    if (durationMs < 1000) {
      return `${durationMs} ms`
    }

    const seconds = durationMs / 1000
    if (seconds < 60) {
      return `${seconds.toFixed(seconds >= 10 ? 0 : 1)} s`
    }

    const minutes = Math.floor(seconds / 60)
    const remainingSeconds = Math.round(seconds % 60)
    if (minutes < 60) {
      return `${minutes}m ${remainingSeconds}s`
    }

    const hours = Math.floor(minutes / 60)
    const remainingMinutes = minutes % 60
    return `${hours}h ${remainingMinutes}m`
  }

  const triggeredByLabel = (row: Api.TaskManager.TaskExecutionItem) => {
    return row.triggered_by_name || '-'
  }

  const {
    columns,
    columnChecks,
    data: historyData,
    loading,
    pagination,
    refreshData,
    handleSizeChange,
    handleCurrentChange,
    getData,
    searchParams
  } = useTable<typeof fetchTaskHistory>({
    core: {
      apiFn: fetchTaskHistory,
      apiParams: { current: 1, size: 200 },
      columnsFactory: () => [
        { type: 'index', width: 60, label: '#' },
        {
          prop: 'task_name',
          label: t('taskManager.columns.name'),
          minWidth: 180,
          showOverflowTooltip: true,
          formatter: (row: Api.TaskManager.TaskExecutionItem) => taskDisplayName(row.task_name)
        },
        {
          prop: 'trigger',
          label: t('taskManager.columns.trigger'),
          width: 110,
          formatter: (row: Api.TaskManager.TaskExecutionItem) =>
            h(ElTag, { type: triggerTagType(row.trigger), size: 'small', effect: 'plain' }, () =>
              t(`taskManager.trigger.${row.trigger}`)
            )
        },
        {
          prop: 'triggered_by_name',
          label: t('taskManager.columns.triggeredBy'),
          width: 150,
          formatter: (row: Api.TaskManager.TaskExecutionItem) => triggeredByLabel(row)
        },
        {
          prop: 'triggered_by',
          label: t('taskManager.columns.triggeredById'),
          width: 120,
          formatter: (row: Api.TaskManager.TaskExecutionItem) =>
            row.triggered_by != null ? row.triggered_by : '-'
        },
        {
          prop: 'status',
          label: t('taskManager.columns.status'),
          width: 110,
          formatter: (row: Api.TaskManager.TaskExecutionItem) =>
            h(ElTag, { type: statusTagType(row.status), size: 'small', effect: 'plain' }, () =>
              t(`taskManager.status.${row.status}`)
            )
        },
        {
          prop: 'started_at',
          label: t('taskManager.columns.startedAt'),
          width: 190,
          formatter: (row: Api.TaskManager.TaskExecutionItem) => formatTime(row.started_at)
        },
        {
          prop: 'duration_ms',
          label: t('taskManager.columns.duration'),
          width: 120,
          formatter: (row: Api.TaskManager.TaskExecutionItem) => formatDuration(row.duration_ms)
        },
        {
          prop: 'error',
          label: t('taskManager.columns.error'),
          minWidth: 220,
          showOverflowTooltip: true,
          formatter: (row: Api.TaskManager.TaskExecutionItem) => {
            if (!row.error) {
              return h('span', { class: 'text-gray-400' }, '-')
            }

            return h(
              ElTooltip,
              { content: row.error, placement: 'top-start' },
              {
                default: () => h('span', { class: 'text-red-500' }, row.error)
              }
            )
          }
        },
        {
          prop: 'summary',
          label: t('taskManager.columns.summary'),
          minWidth: 200,
          showOverflowTooltip: true,
          formatter: (row: Api.TaskManager.TaskExecutionItem) => row.summary || '-'
        }
      ]
    }
  })

  async function loadTaskOptions() {
    try {
      taskOptions.value = (await fetchTasks()) ?? []
    } catch {
      taskOptions.value = []
    }
  }

  function handleSearch() {
    Object.assign(searchParams, {
      current: 1,
      task_name: filters.task_name || undefined,
      status: filters.status || undefined
    })
    getData()
  }

  function resetFilters() {
    filters.task_name = ''
    filters.status = ''
    handleSearch()
  }

  onMounted(() => {
    void loadTaskOptions()
  })
</script>

<style scoped>
  .task-manager-history {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .task-manager-history__filters {
    display: flex;
    flex-wrap: wrap;
    gap: 12px;
    align-items: center;
  }
</style>
