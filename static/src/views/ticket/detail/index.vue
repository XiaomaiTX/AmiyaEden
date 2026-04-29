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
      </template>
      <p class="ticket-detail-desc">{{ ticket.description }}</p>
    </ElCard>

    <ElCard>
      <template #header>{{ t('ticket.replies') }}</template>
      <div class="ticket-reply-list">
        <TicketReplyItem v-for="item in replies" :key="item.id" :reply="item" />
      </div>
      <ElInput v-model="content" type="textarea" :rows="3" />
      <div class="ticket-detail-actions">
        <ElButton type="primary" :loading="submitting" @click="submitReply">{{ t('ticket.reply') }}</ElButton>
      </div>
    </ElCard>
  </div>
</template>

<script setup lang="ts">
  import { addMyTicketReply, getMyTicket, listMyTicketReplies } from '@/api/ticket'
  import TicketPriorityBadge from '@/components/ticket/TicketPriorityBadge.vue'
  import TicketReplyItem from '@/components/ticket/TicketReplyItem.vue'
  import TicketStatusBadge from '@/components/ticket/TicketStatusBadge.vue'
  import { ElMessage } from 'element-plus'
  import { useI18n } from 'vue-i18n'

  defineOptions({ name: 'TicketDetailPage' })

  const { t } = useI18n()
  const route = useRoute()

  const ticketId = computed(() => Number(route.params.id))
  const loading = ref(false)
  const submitting = ref(false)
  const ticket = ref<Api.Ticket.TicketItem | null>(null)
  const replies = ref<Api.Ticket.TicketReply[]>([])
  const content = ref('')

  const loadData = async () => {
    loading.value = true
    try {
      const [ticketData, replyData] = await Promise.all([
        getMyTicket(ticketId.value),
        listMyTicketReplies(ticketId.value)
      ])
      ticket.value = ticketData
      replies.value = replyData
    } catch (error: any) {
      ElMessage.error(error?.message || t('ticket.messages.loadFailed'))
    } finally {
      loading.value = false
    }
  }

  const submitReply = async () => {
    if (!content.value.trim()) {
      return
    }
    submitting.value = true
    try {
      await addMyTicketReply(ticketId.value, { content: content.value })
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

