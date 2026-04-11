<template>
  <div class="task-manager-esi-statuses">
    <div class="task-manager-esi-statuses__filters">
      <ElInput
        v-model="filterForm.character"
        :placeholder="t('taskManager.esi.filters.character')"
        clearable
        style="width: 240px"
        @change="handleSearch"
        @clear="handleSearch"
      />
      <ElButton type="primary" @click="handleSearch">{{ t('common.search') }}</ElButton>
      <ElButton @click="resetFilters">{{ t('common.reset') }}</ElButton>
    </div>

    <ElCard class="art-table-card" shadow="never">
      <ArtTableHeader
        v-model:columns="statusColumnChecks"
        :loading="statusLoading"
        @refresh="refreshData"
      >
        <template #left>
          <div
            class="task-manager-esi-statuses__header-group task-manager-esi-statuses__header-group--wrap"
          >
            <span class="font-medium">{{ t('taskManager.esi.sections.statuses') }}</span>
            <ElSelect
              v-model="filterForm.task_name"
              :placeholder="t('taskManager.esi.filters.taskName')"
              clearable
              filterable
              style="width: 180px"
              @change="handleSearch"
            >
              <ElOption
                v-for="task in tasks"
                :key="task.name"
                :label="task.description"
                :value="task.name"
              />
            </ElSelect>
            <ElSelect
              v-model="filterForm.status"
              :placeholder="t('taskManager.esi.filters.status')"
              clearable
              style="width: 140px"
              @change="handleSearch"
            >
              <ElOption :label="t('taskManager.esi.status.pending')" value="pending" />
              <ElOption :label="t('taskManager.esi.status.running')" value="running" />
              <ElOption :label="t('taskManager.esi.status.success')" value="success" />
              <ElOption :label="t('taskManager.esi.status.failed')" value="failed" />
              <ElOption :label="t('taskManager.esi.status.skipped')" value="skipped" />
            </ElSelect>
          </div>
        </template>
      </ArtTableHeader>

      <ArtTable
        :loading="statusLoading"
        :data="statusData"
        :columns="statusColumns"
        :pagination="pagination"
        visual-variant="ledger"
        @pagination:size-change="handleSizeChange"
        @pagination:current-change="handleCurrentChange"
      />
    </ElCard>
  </div>
</template>

<script setup lang="ts">
  import { ElButton, ElCard, ElInput, ElMessage, ElOption, ElSelect, ElTag } from 'element-plus'
  import { useI18n } from 'vue-i18n'
  import { h } from 'vue'
  import { formatTime } from '@utils/common'
  import {
    fetchESIRefreshStatuses,
    fetchESIRefreshTasks,
    runESIRefreshTask
  } from '@/api/esi-refresh'
  import { useTable } from '@/hooks/core/useTable'

  type TaskInfo = Api.ESIRefresh.TaskInfo
  type TaskStatus = Api.ESIRefresh.TaskStatus

  const { t } = useI18n()

  const priorityType = (priority: number) => {
    const map = {
      1: 'danger',
      10: 'warning',
      50: 'success',
      90: 'info'
    } as const

    return map[priority as keyof typeof map] ?? 'info'
  }

  const priorityLabel = (priority: number) => {
    const map = {
      1: t('taskManager.esi.priority.critical'),
      10: t('taskManager.esi.priority.high'),
      50: t('taskManager.esi.priority.normal'),
      90: t('taskManager.esi.priority.low')
    } as const

    return map[priority as keyof typeof map] ?? `P${priority}`
  }

  const statusType = (status: TaskStatus['status']) => {
    const map = {
      pending: 'info',
      running: 'warning',
      success: 'success',
      failed: 'danger',
      skipped: 'info'
    } as const

    return map[status]
  }

  const tasks = ref<TaskInfo[]>([])
  const runningTasks = ref(new Set<string>())
  const filterForm = reactive({ task_name: '', status: '', character: '' })

  const {
    columns: statusColumns,
    columnChecks: statusColumnChecks,
    data: statusData,
    loading: statusLoading,
    pagination,
    handleSizeChange,
    handleCurrentChange,
    refreshData,
    getData,
    searchParams
  } = useTable<typeof fetchESIRefreshStatuses>({
    core: {
      apiFn: fetchESIRefreshStatuses,
      apiParams: { current: 1, size: 20 },
      columnsFactory: () => [
        { type: 'index', width: 60, label: '#' },
        {
          prop: 'task_name',
          label: t('taskManager.columns.name'),
          width: 200,
          showOverflowTooltip: true
        },
        {
          prop: 'description',
          label: t('taskManager.columns.description'),
          minWidth: 160,
          showOverflowTooltip: true
        },
        {
          prop: 'character_name',
          label: t('common.name'),
          width: 180,
          showOverflowTooltip: true,
          formatter: (row: TaskStatus) => row.character_name || '-'
        },
        {
          prop: 'character_id',
          label: t('taskManager.esi.columns.characterId'),
          width: 120
        },
        {
          prop: 'priority',
          label: t('taskManager.esi.columns.priority'),
          width: 100,
          formatter: (row: TaskStatus) =>
            h(ElTag, { type: priorityType(row.priority), size: 'small', effect: 'plain' }, () =>
              priorityLabel(row.priority)
            )
        },
        {
          prop: 'status',
          label: t('taskManager.columns.status'),
          width: 100,
          formatter: (row: TaskStatus) =>
            h(ElTag, { type: statusType(row.status), size: 'small', effect: 'plain' }, () =>
              t(`taskManager.esi.status.${row.status}`)
            )
        },
        {
          prop: 'last_run',
          label: t('taskManager.columns.lastRun'),
          width: 180,
          formatter: (row: TaskStatus) => formatTime(row.last_run)
        },
        {
          prop: 'next_run',
          label: t('taskManager.esi.columns.nextRun'),
          width: 180,
          formatter: (row: TaskStatus) => formatTime(row.next_run)
        },
        {
          prop: 'error',
          label: t('taskManager.columns.error'),
          minWidth: 200,
          showOverflowTooltip: true,
          formatter: (row: TaskStatus) =>
            row.error
              ? h('span', { class: 'text-red-500' }, row.error)
              : h('span', { class: 'text-gray-400' }, '-')
        },
        {
          prop: 'actions',
          label: t('common.operation'),
          width: 100,
          fixed: 'right',
          formatter: (row: TaskStatus) =>
            h(
              ElButton,
              {
                size: 'small',
                type: 'primary',
                loading: runningTasks.value.has(`${row.task_name}_${row.character_id}`),
                onClick: () => handleRunTask(row)
              },
              () => t('taskManager.actions.run')
            )
        }
      ]
    }
  })

  async function loadTasks() {
    try {
      tasks.value = (await fetchESIRefreshTasks()) ?? []
    } catch {
      tasks.value = []
    }
  }

  function handleSearch() {
    Object.assign(searchParams, {
      current: 1,
      character: filterForm.character || undefined,
      task_name: filterForm.task_name || undefined,
      status: filterForm.status || undefined
    })
    getData()
  }

  function resetFilters() {
    filterForm.character = ''
    filterForm.task_name = ''
    filterForm.status = ''
    handleSearch()
  }

  async function handleRunTask(row: TaskStatus) {
    const key = `${row.task_name}_${row.character_id}`
    runningTasks.value.add(key)
    try {
      await runESIRefreshTask({ task_name: row.task_name, character_id: row.character_id })
      ElMessage.success(
        t('taskManager.esi.messages.characterTaskTriggered', {
          name: row.description,
          characterId: row.character_id
        })
      )
      refreshData()
    } catch {
      ElMessage.error(
        t('taskManager.esi.messages.characterTaskTriggerFailed', {
          name: row.description,
          characterId: row.character_id
        })
      )
    } finally {
      runningTasks.value.delete(key)
    }
  }

  onMounted(() => {
    void loadTasks()
  })
</script>

<style scoped>
  .task-manager-esi-statuses {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  .task-manager-esi-statuses__filters {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 12px;
  }

  .task-manager-esi-statuses__header-group {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .task-manager-esi-statuses__header-group--wrap {
    flex-wrap: wrap;
  }
</style>
