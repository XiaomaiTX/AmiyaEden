<template>
  <div class="ticket-page">
    <div class="ticket-page__toolbar">
      <ElInput
        v-model="filters.keyword"
        :placeholder="t('ticket.filters.keyword')"
        style="width: 260px"
        clearable
      />
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
    </div>

    <ElTable :data="list" v-loading="loading">
      <ElTableColumn prop="id" label="ID" width="80" />
      <ElTableColumn prop="user_id" :label="t('ticket.columns.submitter')" width="100" />
      <ElTableColumn prop="title" :label="t('ticket.columns.title')" min-width="200" />
      <ElTableColumn :label="t('ticket.columns.status')" width="180">
        <template #default="{ row }">
          <ElSelect v-model="row.status" size="small" @change="(val) => updateStatus(row.id, val)">
            <ElOption :label="t('ticket.status.pending')" value="pending" />
            <ElOption :label="t('ticket.status.in_progress')" value="in_progress" />
            <ElOption :label="t('ticket.status.completed')" value="completed" />
          </ElSelect>
        </template>
      </ElTableColumn>
      <ElTableColumn :label="t('ticket.columns.priority')" width="170">
        <template #default="{ row }">
          <ElSelect
            v-model="row.priority"
            size="small"
            @change="(val) => updatePriority(row.id, val)"
          >
            <ElOption :label="t('ticket.priority.low')" value="low" />
            <ElOption :label="t('ticket.priority.medium')" value="medium" />
            <ElOption :label="t('ticket.priority.high')" value="high" />
          </ElSelect>
        </template>
      </ElTableColumn>
      <ElTableColumn prop="updated_at" :label="t('common.updatedAt')" width="180" />
      <ElTableColumn :label="t('common.operation')" width="120" fixed="right">
        <template #default="{ row }">
          <ElButton link type="primary" @click="openDetail(row.id)">{{
            t('ticket.viewDetail')
          }}</ElButton>
        </template>
      </ElTableColumn>
    </ElTable>
  </div>
</template>

<script setup lang="ts">
  import {
    adminListTickets,
    adminUpdateTicketPriority,
    adminUpdateTicketStatus
  } from '@/api/ticket'
  import { ElMessage } from 'element-plus'
  import { useI18n } from 'vue-i18n'

  defineOptions({ name: 'TicketManagementPage' })

  const { t } = useI18n()
  const router = useRouter()

  const loading = ref(false)
  const list = ref<Api.Ticket.TicketItem[]>([])
  const filters = reactive<{ keyword: string; status: Api.Ticket.TicketStatus | '' }>({
    keyword: '',
    status: ''
  })

  const loadTickets = async () => {
    loading.value = true
    try {
      const data = await adminListTickets({
        current: 1,
        size: 100,
        keyword: filters.keyword,
        status: filters.status
      })
      list.value = data.list
    } catch (error: any) {
      ElMessage.error(error?.message || t('ticket.messages.loadFailed'))
    } finally {
      loading.value = false
    }
  }

  const updateStatus = async (id: number, status: Api.Ticket.TicketStatus) => {
    try {
      await adminUpdateTicketStatus(id, { status })
      ElMessage.success(t('ticket.messages.updated'))
    } catch (error: any) {
      ElMessage.error(error?.message || t('ticket.messages.updateFailed'))
      await loadTickets()
    }
  }

  const updatePriority = async (id: number, priority: Api.Ticket.TicketPriority) => {
    try {
      await adminUpdateTicketPriority(id, { priority })
      ElMessage.success(t('ticket.messages.updated'))
    } catch (error: any) {
      ElMessage.error(error?.message || t('ticket.messages.updateFailed'))
      await loadTickets()
    }
  }

  const openDetail = (id: number) =>
    router.push({ name: 'TicketAdminDetail', params: { id: String(id) } })

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
