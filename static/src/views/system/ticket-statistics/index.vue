<template>
  <div class="ticket-stats-page" v-loading="loading">
    <ElRow :gutter="16">
      <ElCol :xs="24" :sm="12" :md="6">
        <ElCard
          ><div class="stat-item"
            ><span>{{ t('ticket.stats.total') }}</span
            ><strong>{{ stats?.total ?? 0 }}</strong></div
          ></ElCard
        >
      </ElCol>
      <ElCol :xs="24" :sm="12" :md="6">
        <ElCard
          ><div class="stat-item"
            ><span>{{ t('ticket.status.pending') }}</span
            ><strong>{{ stats?.status.pending ?? 0 }}</strong></div
          ></ElCard
        >
      </ElCol>
      <ElCol :xs="24" :sm="12" :md="6">
        <ElCard
          ><div class="stat-item"
            ><span>{{ t('ticket.status.in_progress') }}</span
            ><strong>{{ stats?.status.in_progress ?? 0 }}</strong></div
          ></ElCard
        >
      </ElCol>
      <ElCol :xs="24" :sm="12" :md="6">
        <ElCard
          ><div class="stat-item"
            ><span>{{ t('ticket.status.completed') }}</span
            ><strong>{{ stats?.status.completed ?? 0 }}</strong></div
          ></ElCard
        >
      </ElCol>
    </ElRow>
    <ElCard>
      <div class="trend-line">
        <span>{{ t('ticket.stats.recent7d') }}: {{ stats?.recent_7d ?? 0 }}</span>
        <span>{{ t('ticket.stats.recent30d') }}: {{ stats?.recent_30d ?? 0 }}</span>
      </div>
    </ElCard>
  </div>
</template>

<script setup lang="ts">
  import { adminTicketStatistics } from '@/api/ticket'
  import { ElMessage } from 'element-plus'
  import { useI18n } from 'vue-i18n'

  defineOptions({ name: 'TicketStatisticsPage' })

  const { t } = useI18n()
  const loading = ref(false)
  const stats = ref<Api.Ticket.Statistics | null>(null)

  const loadStats = async () => {
    loading.value = true
    try {
      stats.value = await adminTicketStatistics()
    } catch (error: any) {
      ElMessage.error(error?.message || t('ticket.messages.loadFailed'))
    } finally {
      loading.value = false
    }
  }

  onMounted(loadStats)
</script>

<style scoped>
  .ticket-stats-page {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  .stat-item {
    display: flex;
    justify-content: space-between;
    align-items: center;
    font-size: 14px;
  }

  .stat-item strong {
    font-size: 24px;
  }

  .trend-line {
    display: flex;
    gap: 20px;
  }
</style>
