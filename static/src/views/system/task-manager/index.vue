<template>
  <div class="task-manager-page">
    <ElAlert
      v-if="sdeStatus.has_update"
      type="warning"
      :closable="false"
      :title="t('taskManager.sdeStatus.updateAvailable')"
    />
    <ElAlert
      v-else
      type="info"
      :closable="false"
      :title="t('taskManager.sdeStatus.upToDate', { current: sdeStatus.current_version || '-' })"
    />

    <ElTabs v-model="activeTab">
      <ElTabPane :label="t('taskManager.tabs.tasks')" name="tasks" />
      <ElTabPane :label="t('taskManager.tabs.esiStatuses')" name="esi-statuses" />
      <ElTabPane :label="t('taskManager.tabs.esiMonitor')" name="esi-monitor" />
      <ElTabPane :label="t('taskManager.tabs.history')" name="history" />
    </ElTabs>

    <TasksTab v-if="activeTab === 'tasks'" />
    <EsiStatusesTab v-if="activeTab === 'esi-statuses'" />
    <EsiMonitorTab v-if="activeTab === 'esi-monitor'" />
    <HistoryTab v-if="activeTab === 'history'" />
  </div>
</template>

<script setup lang="ts">
  import { ElAlert, ElTabPane, ElTabs } from 'element-plus'
  import { useI18n } from 'vue-i18n'
  import { fetchSDEStatus } from '@/api/sys-config'
  import EsiMonitorTab from './modules/EsiMonitorTab.vue'
  import EsiStatusesTab from './modules/EsiStatusesTab.vue'
  import HistoryTab from './modules/HistoryTab.vue'
  import TasksTab from './modules/TasksTab.vue'

  defineOptions({ name: 'TaskManager' })

  const { t } = useI18n()
  const activeTab = ref('tasks')
  const sdeStatus = reactive<Api.SysConfig.SDEStatus>({
    current_version: '',
    latest_version: '',
    has_update: false,
    last_check_at: 0,
    last_check_success: false,
    last_check_error: '',
    last_update_at: 0,
    last_update_success: false,
    last_update_error: '',
    is_updating: false,
    update_stage: ''
  })

  async function loadSDEStatus() {
    try {
      const status = await fetchSDEStatus()
      Object.assign(sdeStatus, status)
    } catch {
      /* empty */
    }
  }

  onMounted(() => {
    void loadSDEStatus()
  })
</script>

<style scoped>
  .task-manager-page {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }
</style>
