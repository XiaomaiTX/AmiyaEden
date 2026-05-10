<template>
  <div class="task-manager-esi-monitor">
    <ElCard class="art-table-card" shadow="never">
      <template #header>
        <div class="task-manager-esi-monitor__header">
          <div class="task-manager-esi-monitor__header-left">
            <span class="font-medium">{{ t('taskManager.tabs.esiMonitor') }}</span>
            <span class="text-xs text-gray-500">
              {{ t('taskManager.esi.monitor.generatedAt') }}:
              {{ formatTime(monitor?.generated_at) }}
            </span>
          </div>
          <div class="task-manager-esi-monitor__header-right">
            <ElSwitch
              v-model="autoRefreshEnabled"
              :active-text="t('taskManager.esi.monitor.autoRefresh')"
            />
            <ElButton :loading="loading" @click="() => loadMonitor(true)">
              {{ t('common.refresh') }}
            </ElButton>
          </div>
        </div>
      </template>

      <ElAlert
        v-if="errorMessage"
        type="error"
        :title="errorMessage"
        show-icon
        class="task-manager-esi-monitor__error"
      />

      <div class="task-manager-esi-monitor__cards">
        <div class="task-manager-esi-monitor__card">
          <div class="task-manager-esi-monitor__card-label">
            {{ t('taskManager.esi.monitor.cards.total') }}
          </div>
          <div class="task-manager-esi-monitor__card-value">{{ monitor?.overview.total ?? 0 }}</div>
        </div>
        <div class="task-manager-esi-monitor__card">
          <div class="task-manager-esi-monitor__card-label">
            {{ t('taskManager.esi.monitor.cards.healthy') }}
          </div>
          <div class="task-manager-esi-monitor__card-value">{{
            monitor?.overview.healthy ?? 0
          }}</div>
        </div>
        <div class="task-manager-esi-monitor__card">
          <div class="task-manager-esi-monitor__card-label">
            {{ t('taskManager.esi.monitor.cards.warning') }}
          </div>
          <div class="task-manager-esi-monitor__card-value">{{
            monitor?.overview.warning ?? 0
          }}</div>
        </div>
        <div class="task-manager-esi-monitor__card">
          <div class="task-manager-esi-monitor__card-label">
            {{ t('taskManager.esi.monitor.cards.critical') }}
          </div>
          <div class="task-manager-esi-monitor__card-value">{{
            monitor?.overview.critical ?? 0
          }}</div>
        </div>
        <div class="task-manager-esi-monitor__card">
          <div class="task-manager-esi-monitor__card-label">
            {{ t('taskManager.esi.monitor.cards.running') }}
          </div>
          <div class="task-manager-esi-monitor__card-value">{{
            monitor?.overview.running ?? 0
          }}</div>
        </div>
        <div class="task-manager-esi-monitor__card">
          <div class="task-manager-esi-monitor__card-label">
            {{ t('taskManager.esi.monitor.cards.failed') }}
          </div>
          <div class="task-manager-esi-monitor__card-value">{{
            monitor?.overview.failed ?? 0
          }}</div>
        </div>
        <div class="task-manager-esi-monitor__card">
          <div class="task-manager-esi-monitor__card-label">
            {{ t('taskManager.esi.monitor.cards.overdue') }}
          </div>
          <div class="task-manager-esi-monitor__card-value">{{
            monitor?.overview.overdue ?? 0
          }}</div>
        </div>
      </div>
    </ElCard>

    <ElCard class="art-table-card" shadow="never">
      <template #header>
        <span class="font-medium">{{ t('taskManager.esi.monitor.sections.taskPanels') }}</span>
      </template>
      <ArtTable :data="monitor?.task_panels ?? []" :columns="taskPanelColumns" :loading="loading" />
    </ElCard>

    <div class="task-manager-esi-monitor__grid">
      <ElCard class="art-table-card" shadow="never">
        <template #header>
          <span class="font-medium">{{ t('taskManager.esi.monitor.sections.failureTop') }}</span>
        </template>
        <ArtTable :data="monitor?.failure_top ?? []" :columns="failureColumns" :loading="loading" />
      </ElCard>

      <ElCard class="art-table-card" shadow="never">
        <template #header>
          <span class="font-medium">{{ t('taskManager.esi.monitor.sections.overdueTop') }}</span>
        </template>
        <ArtTable :data="monitor?.overdue_top ?? []" :columns="overdueColumns" :loading="loading" />
      </ElCard>
    </div>
  </div>
</template>

<script setup lang="ts">
  import { ElAlert, ElButton, ElCard, ElMessage, ElSwitch } from 'element-plus'
  import { useI18n } from 'vue-i18n'
  import { h } from 'vue'
  import { formatTime } from '@utils/common'
  import { fetchESIRefreshMonitor } from '@/api/esi-refresh'
  import { useTableColumns } from '@/hooks/core/useTableColumns'

  const { t } = useI18n()
  const monitor = ref<Api.ESIRefresh.MonitorResponse>()
  const loading = ref(false)
  const errorMessage = ref('')
  const autoRefreshEnabled = ref(true)
  const refreshTimer = ref<number>()

  const { columns: taskPanelColumns } = useTableColumns<Api.ESIRefresh.MonitorTaskPanelItem>(() => [
    { type: 'index', width: 60, label: '#' },
    {
      prop: 'task_name',
      label: t('taskManager.columns.name'),
      minWidth: 180,
      showOverflowTooltip: true
    },
    {
      prop: 'description',
      label: t('taskManager.columns.description'),
      minWidth: 180,
      showOverflowTooltip: true
    },
    { prop: 'total', label: t('taskManager.esi.monitor.cards.total'), width: 100 },
    { prop: 'failed', label: t('taskManager.esi.monitor.cards.failed'), width: 100 },
    { prop: 'overdue', label: t('taskManager.esi.monitor.cards.overdue'), width: 100 },
    { prop: 'running', label: t('taskManager.esi.monitor.cards.running'), width: 100 },
    {
      prop: 'success_rate',
      label: t('taskManager.esi.monitor.columns.successRate'),
      width: 130,
      formatter: (row: Api.ESIRefresh.MonitorTaskPanelItem) =>
        `${(row.success_rate * 100).toFixed(1)}%`
    },
    {
      prop: 'worst_lag_seconds',
      label: t('taskManager.esi.monitor.columns.worstLag'),
      width: 130
    }
  ])

  const { columns: failureColumns } = useTableColumns<Api.ESIRefresh.MonitorFailureItem>(() => [
    { type: 'index', width: 60, label: '#' },
    {
      prop: 'task_name',
      label: t('taskManager.columns.name'),
      width: 160,
      showOverflowTooltip: true
    },
    { prop: 'character_name', label: t('common.name'), width: 150, showOverflowTooltip: true },
    { prop: 'character_id', label: t('taskManager.esi.columns.characterId'), width: 120 },
    {
      prop: 'last_run',
      label: t('taskManager.columns.lastRun'),
      width: 180,
      formatter: (row: Api.ESIRefresh.MonitorFailureItem) => formatTime(row.last_run)
    },
    {
      prop: 'error',
      label: t('taskManager.columns.error'),
      minWidth: 220,
      showOverflowTooltip: true,
      formatter: (row: Api.ESIRefresh.MonitorFailureItem) =>
        row.error ? h('span', { class: 'text-red-500' }, row.error) : '-'
    }
  ])

  const { columns: overdueColumns } = useTableColumns<Api.ESIRefresh.MonitorOverdueItem>(() => [
    { type: 'index', width: 60, label: '#' },
    {
      prop: 'task_name',
      label: t('taskManager.columns.name'),
      width: 160,
      showOverflowTooltip: true
    },
    { prop: 'character_name', label: t('common.name'), width: 150, showOverflowTooltip: true },
    { prop: 'character_id', label: t('taskManager.esi.columns.characterId'), width: 120 },
    {
      prop: 'next_run',
      label: t('taskManager.esi.columns.nextRun'),
      width: 180,
      formatter: (row: Api.ESIRefresh.MonitorOverdueItem) => formatTime(row.next_run)
    },
    {
      prop: 'overdue_seconds',
      label: t('taskManager.esi.monitor.columns.overdueSeconds'),
      width: 140
    }
  ])

  async function loadMonitor(showErrorTip = false) {
    loading.value = true
    errorMessage.value = ''
    try {
      monitor.value = await fetchESIRefreshMonitor()
    } catch (error) {
      errorMessage.value = t('taskManager.esi.monitor.messages.loadFailed')
      if (showErrorTip) {
        ElMessage.error(
          error instanceof Error && error.message ? error.message : errorMessage.value
        )
      }
    } finally {
      loading.value = false
    }
  }

  function clearRefreshTimer() {
    if (refreshTimer.value) {
      window.clearInterval(refreshTimer.value)
      refreshTimer.value = undefined
    }
  }

  function setupAutoRefresh() {
    clearRefreshTimer()
    if (!autoRefreshEnabled.value) return
    refreshTimer.value = window.setInterval(() => {
      if (document.visibilityState === 'hidden') return
      void loadMonitor()
    }, 30_000)
  }

  function handleVisibilityChange() {
    if (document.visibilityState === 'hidden') {
      clearRefreshTimer()
      return
    }
    setupAutoRefresh()
  }

  watch(autoRefreshEnabled, () => {
    setupAutoRefresh()
  })

  onMounted(() => {
    void loadMonitor()
    setupAutoRefresh()
    document.addEventListener('visibilitychange', handleVisibilityChange)
  })

  onBeforeUnmount(() => {
    clearRefreshTimer()
    document.removeEventListener('visibilitychange', handleVisibilityChange)
  })
</script>

<style scoped>
  .task-manager-esi-monitor {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  .task-manager-esi-monitor__header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 12px;
    flex-wrap: wrap;
  }

  .task-manager-esi-monitor__header-left {
    display: flex;
    align-items: center;
    gap: 12px;
    flex-wrap: wrap;
  }

  .task-manager-esi-monitor__header-right {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .task-manager-esi-monitor__error {
    margin-bottom: 12px;
  }

  .task-manager-esi-monitor__cards {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(140px, 1fr));
    gap: 12px;
  }

  .task-manager-esi-monitor__card {
    border: 1px solid var(--el-border-color-light);
    border-radius: 8px;
    padding: 12px;
  }

  .task-manager-esi-monitor__card-label {
    font-size: 12px;
    color: var(--el-text-color-secondary);
  }

  .task-manager-esi-monitor__card-value {
    margin-top: 6px;
    font-size: 22px;
    font-weight: 600;
    color: var(--el-text-color-primary);
  }

  .task-manager-esi-monitor__grid {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 16px;
  }

  @media (max-width: 1200px) {
    .task-manager-esi-monitor__grid {
      grid-template-columns: 1fr;
    }
  }
</style>
