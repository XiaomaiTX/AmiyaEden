<template>
  <div class="ticket-detail-page" v-loading="loading">
    <ElCard v-if="ticket">
      <template #header>
        <div class="ticket-detail-header">
          <span>#{{ ticket.id }} {{ ticket.title }}</span>
          <div class="ticket-detail-header__right">
            <TicketStatusBadge :status="ticket.status" />
            <TicketPriorityBadge :priority="ticket.priority" />
          </div>
        </div>
        <div class="ticket-detail-header__controls">
          <ElSelect v-model="editStatus" style="width: 180px">
            <ElOption :label="t('ticket.status.pending')" value="pending" />
            <ElOption :label="t('ticket.status.in_progress')" value="in_progress" />
            <ElOption :label="t('ticket.status.completed')" value="completed" />
          </ElSelect>
          <ElSelect v-model="editPriority" style="width: 180px">
            <ElOption :label="t('ticket.priority.unassigned')" value="unassigned" />
            <ElOption :label="t('ticket.priority.low')" value="low" />
            <ElOption :label="t('ticket.priority.medium')" value="medium" />
            <ElOption :label="t('ticket.priority.high')" value="high" />
          </ElSelect>
          <ElButton
            type="primary"
            :loading="savingMeta"
            :disabled="!hasMetaChanges"
            @click="saveMeta"
          >
            {{ t('common.save') }}
          </ElButton>
        </div>
      </template>
      <p class="ticket-detail-desc">{{ ticket.description }}</p>
    </ElCard>

    <ElCard>
      <template #header>{{ t('ticket.replies') }}</template>
      <div class="ticket-reply-list">
        <TicketReplyItem v-for="item in replies" :key="item.id" :reply="item" />
      </div>
      <ElCheckbox v-model="isInternal">{{ t('ticket.internalNote') }}</ElCheckbox>
      <ElInput v-model="content" type="textarea" :rows="3" />
      <div class="ticket-detail-actions">
        <ElButton type="primary" :loading="submitting" @click="submitReply">{{
          t('ticket.reply')
        }}</ElButton>
      </div>
    </ElCard>

    <ElCard class="art-table-card" shadow="never">
      <template #header>{{ t('ticket.statusHistory') }}</template>
      <ArtTable :data="histories" :columns="historyColumns" />
    </ElCard>
  </div>
</template>

<script setup lang="ts">
  import {
    adminAddTicketReply,
    adminGetTicket,
    adminListTicketReplies,
    adminListTicketStatusHistory,
    adminUpdateTicketPriority,
    adminUpdateTicketStatus
  } from '@/api/ticket'
  import TicketPriorityBadge from '@/components/ticket/TicketPriorityBadge.vue'
  import TicketReplyItem from '@/components/ticket/TicketReplyItem.vue'
  import TicketStatusBadge from '@/components/ticket/TicketStatusBadge.vue'
  import { ElMessage } from 'element-plus'
  import { useI18n } from 'vue-i18n'

  defineOptions({ name: 'TicketAdminDetailPage' })

  const { t } = useI18n()
  const route = useRoute()
  const ticketId = computed(() => Number(route.params.id))

  const loading = ref(false)
  const submitting = ref(false)
  const savingMeta = ref(false)
  const ticket = ref<Api.Ticket.TicketItem | null>(null)
  const replies = ref<Api.Ticket.TicketReply[]>([])
  const histories = ref<Api.Ticket.TicketStatusHistory[]>([])
  const editStatus = ref<Api.Ticket.TicketStatus>('pending')
  const editPriority = ref<Api.Ticket.TicketPriority>('unassigned')
  const hasMetaChanges = computed(
    () =>
      !!ticket.value &&
      (editStatus.value !== ticket.value.status || editPriority.value !== ticket.value.priority)
  )

  const isTicketStatus = (value?: string): value is Api.Ticket.TicketStatus =>
    value === 'pending' || value === 'in_progress' || value === 'completed'

  const renderHistoryStatus = (status?: string) => {
    if (!isTicketStatus(status)) {
      return '-'
    }
    return h(TicketStatusBadge, { status })
  }

  const historyColumns = computed(() => [
    {
      prop: 'from_status',
      label: t('ticket.columns.fromStatus'),
      width: 160,
      formatter: (row: Api.Ticket.TicketStatusHistory) => renderHistoryStatus(row.from_status)
    },
    {
      prop: 'to_status',
      label: t('ticket.columns.toStatus'),
      width: 160,
      formatter: (row: Api.Ticket.TicketStatusHistory) => renderHistoryStatus(row.to_status)
    },
    {
      prop: 'changed_by_nickname',
      label: t('ticket.columns.operator'),
      width: 140,
      formatter: (row: Api.Ticket.TicketStatusHistory) => row.changed_by_nickname || '-'
    },
    { prop: 'changed_at', label: t('common.time') }
  ])
  const content = ref('')
  const isInternal = ref(false)

  const loadData = async () => {
    loading.value = true
    try {
      const [ticketData, replyData, historyData] = await Promise.all([
        adminGetTicket(ticketId.value),
        adminListTicketReplies(ticketId.value),
        adminListTicketStatusHistory(ticketId.value)
      ])
      ticket.value = ticketData
      editStatus.value = ticketData.status
      editPriority.value = ticketData.priority
      replies.value = replyData
      histories.value = historyData
    } catch (error: any) {
      ElMessage.error(error?.message || t('ticket.messages.loadFailed'))
    } finally {
      loading.value = false
    }
  }

  const saveMeta = async () => {
    if (!ticket.value || !hasMetaChanges.value) {
      return
    }
    savingMeta.value = true
    try {
      if (editStatus.value !== ticket.value.status) {
        await adminUpdateTicketStatus(ticketId.value, { status: editStatus.value })
      }
      if (editPriority.value !== ticket.value.priority) {
        await adminUpdateTicketPriority(ticketId.value, { priority: editPriority.value })
      }
      await loadData()
      ElMessage.success(t('ticket.messages.updated'))
    } catch (error: any) {
      ElMessage.error(error?.message || t('ticket.messages.updateFailed'))
    } finally {
      savingMeta.value = false
    }
  }

  const submitReply = async () => {
    if (!content.value.trim()) {
      return
    }
    submitting.value = true
    try {
      await adminAddTicketReply(ticketId.value, {
        content: content.value,
        is_internal: isInternal.value
      })
      content.value = ''
      await loadData()
      ElMessage.success(t('ticket.messages.replyAdded'))
    } catch (error: any) {
      ElMessage.error(error?.message || t('ticket.messages.replyFailed'))
    } finally {
      submitting.value = false
    }
  }

  onMounted(loadData)
</script>

<style scoped>
  .ticket-detail-page {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  .ticket-detail-header {
    display: flex;
    justify-content: space-between;
    gap: 12px;
    align-items: center;
  }

  .ticket-detail-header__right {
    display: flex;
    gap: 8px;
  }

  .ticket-detail-header__controls {
    display: flex;
    gap: 12px;
    align-items: center;
    margin-top: 12px;
    flex-wrap: wrap;
  }

  .ticket-detail-desc {
    white-space: pre-wrap;
    line-height: 1.6;
    margin: 0;
  }

  .ticket-reply-list {
    display: flex;
    flex-direction: column;
    gap: 8px;
    margin-bottom: 12px;
  }

  .ticket-detail-actions {
    margin-top: 12px;
  }
</style>
