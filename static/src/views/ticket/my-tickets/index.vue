<template>
  <div class="ticket-page">
    <div class="ticket-page__toolbar">
      <ElSelect
        v-model="filters.status"
        clearable
        :placeholder="t('ticket.filters.status')"
        style="width: 180px"
      >
        <ElOption :label="t('ticket.status.pending')" value="pending" />
        <ElOption :label="t('ticket.status.in_progress')" value="in_progress" />
        <ElOption :label="t('ticket.status.completed')" value="completed" />
      </ElSelect>
      <ElButton type="primary" @click="loadTickets">{{ t('common.search') }}</ElButton>
      <ElButton @click="goCreate">{{ t('ticket.createTicket') }}</ElButton>
    </div>

    <ElTable :data="list" v-loading="loading">
      <ElTableColumn prop="id" label="ID" width="80" />
      <ElTableColumn prop="title" :label="t('ticket.columns.title')" min-width="220" />
      <ElTableColumn :label="t('ticket.columns.status')" width="120">
        <template #default="{ row }">
          <TicketStatusBadge :status="row.status" />
        </template>
      </ElTableColumn>
      <ElTableColumn :label="t('ticket.columns.priority')" width="120">
        <template #default="{ row }">
          <TicketPriorityBadge :priority="row.priority" />
        </template>
      </ElTableColumn>
      <ElTableColumn prop="updated_at" :label="t('common.updatedAt')" width="180" />
      <ElTableColumn :label="t('common.operation')" width="120" fixed="right">
        <template #default="{ row }">
          <ElButton link type="primary" @click="goDetail(row.id)">{{
            t('ticket.viewDetail')
          }}</ElButton>
        </template>
      </ElTableColumn>
    </ElTable>
  </div>
</template>

<script setup lang="ts">
  import { listMyTickets } from '@/api/ticket'
  import TicketPriorityBadge from '@/components/ticket/TicketPriorityBadge.vue'
  import TicketStatusBadge from '@/components/ticket/TicketStatusBadge.vue'
  import { ElMessage } from 'element-plus'
  import { useI18n } from 'vue-i18n'

  defineOptions({ name: 'TicketMyTickets' })

  const { t } = useI18n()
  const router = useRouter()

  const loading = ref(false)
  const list = ref<Api.Ticket.TicketItem[]>([])
  const filters = ref<{ status?: Api.Ticket.TicketStatus | '' }>({ status: '' })

  const loadTickets = async () => {
    loading.value = true
    try {
      const res = await listMyTickets({ current: 1, size: 50, status: filters.value.status })
      list.value = res.list
    } catch (error: any) {
      ElMessage.error(error?.message || t('ticket.messages.loadFailed'))
    } finally {
      loading.value = false
    }
  }

  const goCreate = () => router.push({ name: 'TicketCreate' })
  const goDetail = (id: number) => router.push({ name: 'TicketDetail', params: { id: String(id) } })

  onMounted(loadTickets)
</script>

<style scoped>
  .ticket-page {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  .ticket-page__toolbar {
    display: flex;
    gap: 12px;
    align-items: center;
    flex-wrap: wrap;
  }
</style>
