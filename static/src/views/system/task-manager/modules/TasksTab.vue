<template>
  <div class="task-manager-tasks">
    <ElCard class="art-table-card" shadow="never">
      <ArtTableHeader v-model:columns="columnChecks" :loading="loading" @refresh="loadTasks" />

      <ArtTable :loading="loading" :data="tasks" :columns="columns" />
    </ElCard>

    <EsiControls />

    <ElDialog
      v-model="scheduleDialogVisible"
      :title="t('taskManager.actions.editSchedule')"
      width="520px"
      destroy-on-close
      @closed="resetScheduleForm"
    >
      <ElForm label-width="120px">
        <ElFormItem :label="t('taskManager.columns.name')">
          <span>{{ scheduleForm.displayName }}</span>
        </ElFormItem>
        <ElFormItem :label="t('taskManager.fields.defaultCron')">
          <span class="font-mono text-sm">{{ scheduleForm.defaultCron || '-' }}</span>
        </ElFormItem>
        <ElFormItem :label="t('taskManager.fields.scheduleMode')">
          <ElRadioGroup v-model="scheduleForm.mode">
            <ElRadioButton value="cron">{{ t('taskManager.mode.cron') }}</ElRadioButton>
            <ElRadioButton value="every">{{ t('taskManager.mode.every') }}</ElRadioButton>
          </ElRadioGroup>
        </ElFormItem>
        <template v-if="scheduleForm.mode === 'every'">
          <ElFormItem :label="t('taskManager.fields.intervalValue')">
            <ElInputNumber v-model="scheduleForm.intervalValue" :min="1" :step="1" />
          </ElFormItem>
          <ElFormItem :label="t('taskManager.fields.intervalUnit')">
            <ElSelect v-model="scheduleForm.intervalUnit">
              <ElOption :label="t('taskManager.intervalUnits.m')" value="m" />
              <ElOption :label="t('taskManager.intervalUnits.h')" value="h" />
            </ElSelect>
          </ElFormItem>
        </template>
        <ElFormItem :label="t('taskManager.fields.cronExpr')">
          <ElInput
            v-model="scheduleForm.cronExpr"
            :placeholder="t('taskManager.placeholders.cronExpr')"
            :disabled="scheduleForm.mode === 'every'"
          />
        </ElFormItem>
      </ElForm>

      <template #footer>
        <ElButton @click="scheduleDialogVisible = false">{{ t('common.cancel') }}</ElButton>
        <ElButton type="primary" :loading="scheduleSaving" @click="handleScheduleUpdate">
          {{ t('common.confirm') }}
        </ElButton>
      </template>
    </ElDialog>
  </div>
</template>

<script setup lang="ts">
  import { useUserStore } from '@/store/modules/user'
  import { fetchTasks, runTask, updateTaskSchedule } from '@/api/task-manager'
  import EsiControls from './EsiControls.vue'
  import { getTaskDisplayDescription, getTaskDisplayName } from '../task-labels'
  import { isHttpError } from '@/utils/http/error'
  import { ApiStatus } from '@/utils/http/status'
  import { formatTime } from '@utils/common'
  import {
    ElButton,
    ElCard,
    ElDialog,
    ElForm,
    ElFormItem,
    ElInput,
    ElInputNumber,
    ElMessage,
    ElMessageBox,
    ElOption,
    ElRadioButton,
    ElRadioGroup,
    ElSelect,
    ElTag
  } from 'element-plus'
  import { computed, h } from 'vue'
  import { useI18n } from 'vue-i18n'
  import { useTableColumns } from '@/hooks/core/useTableColumns'

  const { t } = useI18n()
  const userStore = useUserStore()

  const tasks = ref<Api.TaskManager.TaskItem[]>([])
  const loading = ref(false)
  const scheduleDialogVisible = ref(false)
  const scheduleSaving = ref(false)
  const runningTaskNames = ref(new Set<string>())
  type ScheduleMode = 'cron' | 'every'
  type ScheduleIntervalUnit = 'm' | 'h'

  const createScheduleForm = () => ({
    name: '',
    displayName: '',
    mode: 'cron' as ScheduleMode,
    cronExpr: '',
    defaultCron: '',
    intervalValue: 1,
    intervalUnit: 'm' as ScheduleIntervalUnit
  })

  const scheduleForm = reactive(createScheduleForm())

  const everyExprPattern = /^@every\s+(\d+)([mh])$/i

  const canUpdateSchedule = computed(() => {
    const roles = userStore.info?.roles ?? []
    return roles.includes('super_admin')
  })

  const taskDisplayName = (task: Pick<Api.TaskManager.TaskItem, 'name'>) =>
    getTaskDisplayName(t, task.name, task.name)

  const taskDisplayDescription = (task: Pick<Api.TaskManager.TaskItem, 'name' | 'description'>) =>
    getTaskDisplayDescription(t, task.name, task.description || task.name)

  const categoryTagType = (category: Api.TaskManager.TaskItem['category']) => {
    const map = {
      esi: 'primary',
      operation: 'warning',
      system: 'info'
    } as const

    return map[category]
  }

  const typeTagType = (type: Api.TaskManager.TaskItem['type']) => {
    const map = {
      recurring: 'success',
      triggered: 'info'
    } as const

    return map[type]
  }

  const statusTagType = (status?: Api.TaskManager.TaskLastExecution['status']) => {
    const map = {
      running: 'warning',
      success: 'success',
      failed: 'danger'
    } as const

    return status ? map[status] : 'info'
  }

  const scheduleText = (task: Api.TaskManager.TaskItem) => {
    if (task.type === 'triggered') {
      return t('taskManager.messages.eventTriggered')
    }

    return task.cron_expr || task.default_cron || '-'
  }

  const canRunTask = (task: Api.TaskManager.TaskItem) => task.runnable

  const openScheduleDialog = (task: Api.TaskManager.TaskItem) => {
    resetScheduleForm()

    const activeCronExpr = task.cron_expr || task.default_cron
    const everyMatch = activeCronExpr.match(everyExprPattern)

    scheduleForm.name = task.name
    scheduleForm.displayName = taskDisplayName(task)
    scheduleForm.cronExpr = activeCronExpr
    scheduleForm.defaultCron = task.default_cron
    scheduleForm.mode = everyMatch ? 'every' : 'cron'
    scheduleForm.intervalValue = everyMatch ? Number(everyMatch[1]) : 1
    scheduleForm.intervalUnit = everyMatch ? (everyMatch[2].toLowerCase() as 'm' | 'h') : 'm'
    scheduleDialogVisible.value = true
  }

  const resetScheduleForm = () => {
    Object.assign(scheduleForm, createScheduleForm())
  }

  const renderScheduleCell = (task: Api.TaskManager.TaskItem) => {
    const scheduleNode =
      task.type === 'triggered'
        ? h(ElTag, { type: 'info', size: 'small', effect: 'plain' }, () => scheduleText(task))
        : h('span', { class: 'font-mono text-xs' }, scheduleText(task))

    const children = [scheduleNode]

    if (task.type === 'recurring' && canUpdateSchedule.value) {
      children.push(
        h(
          ElButton,
          {
            text: true,
            type: 'primary',
            size: 'small',
            onClick: () => openScheduleDialog(task)
          },
          () => t('taskManager.actions.editSchedule')
        )
      )
    }

    return h('div', { class: 'flex items-center gap-2 flex-wrap' }, children)
  }

  const renderLastStatus = (task: Api.TaskManager.TaskItem) => {
    const status = task.last_execution?.status
    if (!status) {
      return h('span', { class: 'text-gray-400' }, '-')
    }

    return h(ElTag, { type: statusTagType(status), size: 'small', effect: 'plain' }, () =>
      t(`taskManager.status.${status}`)
    )
  }

  const renderActions = (task: Api.TaskManager.TaskItem) => {
    if (!canRunTask(task)) {
      return h('span', { class: 'text-gray-400 text-xs' }, t('taskManager.messages.eventTriggered'))
    }

    return h(
      ElButton,
      {
        type: 'primary',
        size: 'small',
        loading: runningTaskNames.value.has(task.name),
        onClick: () => handleRunTask(task)
      },
      () => t('taskManager.actions.run')
    )
  }

  const { columns, columnChecks } = useTableColumns<Api.TaskManager.TaskItem>(() => [
    { type: 'index', width: 60, label: '#' },
    {
      prop: 'name',
      label: t('taskManager.columns.name'),
      minWidth: 180,
      showOverflowTooltip: true,
      formatter: (row: Api.TaskManager.TaskItem) => taskDisplayName(row)
    },
    {
      prop: 'description',
      label: t('taskManager.columns.description'),
      minWidth: 220,
      showOverflowTooltip: true,
      formatter: (row: Api.TaskManager.TaskItem) => taskDisplayDescription(row)
    },
    {
      prop: 'category',
      label: t('taskManager.columns.category'),
      width: 120,
      formatter: (row: Api.TaskManager.TaskItem) =>
        h(ElTag, { type: categoryTagType(row.category), size: 'small', effect: 'plain' }, () =>
          t(`taskManager.category.${row.category}`)
        )
    },
    {
      prop: 'type',
      label: t('taskManager.columns.type'),
      width: 120,
      formatter: (row: Api.TaskManager.TaskItem) =>
        h(ElTag, { type: typeTagType(row.type), size: 'small', effect: 'plain' }, () =>
          t(`taskManager.type.${row.type}`)
        )
    },
    {
      prop: 'schedule',
      label: t('taskManager.columns.schedule'),
      minWidth: 260,
      formatter: renderScheduleCell
    },
    {
      prop: 'last_execution.started_at',
      label: t('taskManager.columns.lastRun'),
      width: 190,
      formatter: (row: Api.TaskManager.TaskItem) => formatTime(row.last_execution?.started_at)
    },
    {
      prop: 'last_execution.status',
      label: t('taskManager.columns.lastStatus'),
      width: 140,
      formatter: renderLastStatus
    },
    {
      prop: 'actions',
      label: t('common.operation'),
      width: 140,
      fixed: 'right',
      formatter: renderActions
    }
  ])

  async function loadTasks() {
    loading.value = true
    try {
      tasks.value = (await fetchTasks()) ?? []
    } catch {
      tasks.value = []
    } finally {
      loading.value = false
    }
  }

  async function handleRunTask(task: Api.TaskManager.TaskItem) {
    try {
      await ElMessageBox.confirm(
        t('taskManager.confirm.runTask', { name: taskDisplayName(task) }),
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

    runningTaskNames.value.add(task.name)
    try {
      await runTask(task.name)
      ElMessage.success(t('taskManager.messages.taskTriggered', { name: taskDisplayName(task) }))
      window.setTimeout(() => {
        void loadTasks()
      }, 1200)
    } catch (error) {
      if (isHttpError(error) && error.code === ApiStatus.conflict) {
        ElMessage.error(t('taskManager.messages.taskAlreadyRunning'))
      } else {
        ElMessage.error(
          t('taskManager.messages.taskTriggerFailed', { name: taskDisplayName(task) })
        )
      }
    } finally {
      runningTaskNames.value.delete(task.name)
    }
  }

  async function handleScheduleUpdate() {
    const cronExpr =
      scheduleForm.mode === 'every'
        ? `@every ${scheduleForm.intervalValue}${scheduleForm.intervalUnit}`
        : scheduleForm.cronExpr.trim()

    if (!cronExpr) {
      ElMessage.error(t('taskManager.messages.scheduleRequired'))
      return
    }

    scheduleSaving.value = true
    try {
      await updateTaskSchedule(scheduleForm.name, { cron_expr: cronExpr })
      ElMessage.success(t('taskManager.messages.scheduleUpdated'))
      scheduleDialogVisible.value = false
      await loadTasks()
    } catch (error) {
      ElMessage.error(
        error instanceof Error && error.message
          ? error.message
          : t('taskManager.messages.scheduleUpdateFailed')
      )
    } finally {
      scheduleSaving.value = false
    }
  }

  onMounted(() => {
    void loadTasks()
  })
</script>

<style scoped>
  .task-manager-tasks {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }
</style>
