<template>
  <div class="task-manager-esi">
    <ElCard class="art-table-card" shadow="never">
      <ArtTableHeader
        v-model:columns="taskColumnChecks"
        :loading="tasksLoading"
        @refresh="loadTasks"
      >
        <template #left>
          <div class="task-manager-esi__header-group">
            <span class="font-medium">{{ t('taskManager.esi.sections.tasks') }}</span>
            <ElButton
              type="primary"
              :loading="runAllLoading"
              :disabled="!canRun"
              @click="handleRunAll"
            >
              {{ t('taskManager.actions.runAllEsi') }}
            </ElButton>
          </div>
        </template>
      </ArtTableHeader>

      <ArtTable :loading="tasksLoading" :data="tasks" :columns="taskColumns" />
    </ElCard>

    <ElDialog
      v-model="intervalDialogVisible"
      :title="t('taskManager.esi.actions.editInterval')"
      width="420px"
      destroy-on-close
      @closed="resetIntervalForm"
    >
      <ElForm label-width="140px">
        <ElFormItem :label="t('taskManager.columns.name')">
          <span>{{ intervalForm.taskName }}</span>
        </ElFormItem>
        <ElFormItem :label="t('taskManager.esi.fields.activeMinutes')">
          <ElInputNumber
            v-model="intervalForm.activeMinutes"
            :min="1"
            :step="1"
            controls-position="right"
          />
          <span class="ml-2 text-gray-400">{{ t('taskManager.intervalUnits.m') }}</span>
        </ElFormItem>
        <ElFormItem :label="t('taskManager.esi.fields.inactiveMinutes')">
          <ElInputNumber
            v-model="intervalForm.inactiveMinutes"
            :min="1"
            :step="1"
            controls-position="right"
          />
          <span class="ml-2 text-gray-400">{{ t('taskManager.intervalUnits.m') }}</span>
        </ElFormItem>
      </ElForm>
      <template #footer>
        <ElButton @click="intervalDialogVisible = false">
          {{ t('common.cancel') }}
        </ElButton>
        <ElButton type="primary" :loading="intervalSaving" @click="handleSaveInterval">
          {{ t('common.confirm') }}
        </ElButton>
      </template>
    </ElDialog>
  </div>
</template>

<script setup lang="ts">
  import {
    ElButton,
    ElCard,
    ElDialog,
    ElForm,
    ElFormItem,
    ElInputNumber,
    ElMessage,
    ElMessageBox,
    ElTag
  } from 'element-plus'
  import { useI18n } from 'vue-i18n'
  import { h, ref, computed, reactive } from 'vue'
  import {
    fetchESIRefreshTasks,
    runESIRefreshAll,
    runESIRefreshTaskByName,
    updateESITaskInterval
  } from '@/api/esi-refresh'
  import { useTableColumns } from '@/hooks/core/useTableColumns'
  import { useUserStore } from '@/store/modules/user'
  import { useCorpCapability } from '@/hooks/core/useCorpCapability'

  type TaskInfo = Api.ESIRefresh.TaskInfo

  const { t } = useI18n()
  const userStore = useUserStore()
  const { hasCapability } = useCorpCapability()

  // Running ESI refresh tasks requires the same `system.task.run` capability
  // as any other task; the route meta only enforces the read key.
  const canRun = computed(() => hasCapability('system.task.run'))

  const canEditInterval = computed(() => {
    const roles = userStore.info?.roles ?? []
    return roles.includes('super_admin')
  })

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

  const runningByName = ref(new Set<string>())
  const tasks = ref<TaskInfo[]>([])
  const tasksLoading = ref(false)
  const runAllLoading = ref(false)

  const intervalDialogVisible = ref(false)
  const intervalSaving = ref(false)
  const intervalForm = reactive({
    taskName: '',
    activeMinutes: 360,
    inactiveMinutes: 1440
  })

  function openIntervalDialog(row: TaskInfo) {
    intervalForm.taskName = row.name
    intervalForm.activeMinutes = row.active_interval_minutes
    intervalForm.inactiveMinutes = row.inactive_interval_minutes
    intervalDialogVisible.value = true
  }

  function resetIntervalForm() {
    intervalForm.taskName = ''
    intervalForm.activeMinutes = 360
    intervalForm.inactiveMinutes = 1440
  }

  async function handleSaveInterval() {
    intervalSaving.value = true
    try {
      await updateESITaskInterval(intervalForm.taskName, {
        active_minutes: intervalForm.activeMinutes,
        inactive_minutes: intervalForm.inactiveMinutes
      })
      ElMessage.success(t('taskManager.messages.scheduleUpdated'))
      intervalDialogVisible.value = false
      await loadTasks()
    } catch {
      ElMessage.error(t('taskManager.messages.scheduleUpdateFailed'))
    } finally {
      intervalSaving.value = false
    }
  }

  const { columns: taskColumns, columnChecks: taskColumnChecks } = useTableColumns<TaskInfo>(() => [
    { type: 'index', width: 60, label: '#' },
    { prop: 'name', label: t('taskManager.columns.name'), width: 200, showOverflowTooltip: true },
    {
      prop: 'description',
      label: t('taskManager.columns.description'),
      minWidth: 180,
      showOverflowTooltip: true
    },
    {
      prop: 'priority',
      label: t('taskManager.esi.columns.priority'),
      width: 100,
      formatter: (row: TaskInfo) =>
        h(ElTag, { type: priorityType(row.priority), size: 'small', effect: 'plain' }, () =>
          priorityLabel(row.priority)
        )
    },
    {
      prop: 'active_interval',
      label: t('taskManager.esi.columns.activeInterval'),
      width: 120
    },
    {
      prop: 'inactive_interval',
      label: t('taskManager.esi.columns.inactiveInterval'),
      width: 120
    },
    {
      prop: 'required_scopes',
      label: t('taskManager.esi.columns.requiredScopes'),
      minWidth: 260,
      formatter: (row: TaskInfo) =>
        row.required_scopes?.length
          ? h(
              'div',
              { class: 'flex flex-wrap gap-1 py-1' },
              row.required_scopes.map((scope) =>
                h(ElTag, { size: 'small', effect: 'plain', key: scope }, () => scope)
              )
            )
          : h('span', { class: 'text-gray-400' }, t('taskManager.esi.messages.noScopes'))
    },
    {
      prop: 'actions',
      label: t('common.operation'),
      width: 200,
      fixed: 'right',
      formatter: (row: TaskInfo) => {
        const children = [
          h(
            ElButton,
            {
              size: 'small',
              type: 'primary',
              disabled: !canRun.value,
              loading: runningByName.value.has(row.name),
              onClick: () => handleRunTaskByName(row)
            },
            () => t('taskManager.actions.run')
          )
        ]
        if (canEditInterval.value) {
          children.push(
            h(
              ElButton,
              {
                size: 'small',
                type: 'warning',
                style: { marginLeft: '6px' },
                onClick: () => openIntervalDialog(row)
              },
              () => t('taskManager.esi.actions.editInterval')
            )
          )
        }
        return h('div', { class: 'flex items-center' }, children)
      }
    }
  ])

  async function loadTasks() {
    tasksLoading.value = true
    try {
      tasks.value = (await fetchESIRefreshTasks()) ?? []
    } catch {
      tasks.value = []
    } finally {
      tasksLoading.value = false
    }
  }

  async function handleRunTaskByName(row: TaskInfo) {
    try {
      await ElMessageBox.confirm(
        t('taskManager.esi.confirm.runTaskByName', { name: row.description }),
        t('common.tips'),
        {
          confirmButtonText: t('common.confirm'),
          cancelButtonText: t('common.cancel'),
          type: 'warning'
        }
      )
    } catch {
      return
    }

    runningByName.value.add(row.name)
    try {
      await runESIRefreshTaskByName({ task_name: row.name })
      ElMessage.success(t('taskManager.esi.messages.taskTriggeredAll', { name: row.description }))
    } catch {
      ElMessage.error(t('taskManager.esi.messages.taskTriggerFailedAll', { name: row.description }))
    } finally {
      runningByName.value.delete(row.name)
    }
  }

  async function handleRunAll() {
    try {
      await ElMessageBox.confirm(t('taskManager.confirm.runAllEsi'), t('common.tips'), {
        confirmButtonText: t('common.confirm'),
        cancelButtonText: t('common.cancel'),
        type: 'warning'
      })
    } catch {
      return
    }

    runAllLoading.value = true
    try {
      await runESIRefreshAll()
      ElMessage.success(t('taskManager.messages.esiRefreshTriggered'))
    } catch {
      ElMessage.error(t('taskManager.esi.messages.refreshTriggerFailed'))
    } finally {
      runAllLoading.value = false
    }
  }

  onMounted(() => {
    void loadTasks()
  })
</script>

<style scoped>
  .task-manager-esi {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  .task-manager-esi__header-group {
    display: flex;
    align-items: center;
    gap: 12px;
  }
</style>
