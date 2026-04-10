<template>
  <div class="task-manager-page">
    <ElTabs v-model="activeTab">
      <ElTabPane :label="t('taskManager.tabs.tasks')" name="tasks" />
      <ElTabPane :label="t('taskManager.tabs.esiStatuses')" name="esi-statuses" />
      <ElTabPane :label="t('taskManager.tabs.history')" name="history" />
    </ElTabs>

    <TasksTab v-if="activeTab === 'tasks'" />
    <EsiStatusesTab v-if="activeTab === 'esi-statuses'" />
    <HistoryTab v-if="activeTab === 'history'" />
  </div>
</template>

<script setup lang="ts">
  import { ElTabPane, ElTabs } from 'element-plus'
  import { useI18n } from 'vue-i18n'
  import EsiStatusesTab from './modules/EsiStatusesTab.vue'
  import HistoryTab from './modules/HistoryTab.vue'
  import TasksTab from './modules/TasksTab.vue'

  defineOptions({ name: 'TaskManager' })

  const { t } = useI18n()
  const activeTab = ref('tasks')
</script>

<style scoped>
  .task-manager-page {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }
</style>
