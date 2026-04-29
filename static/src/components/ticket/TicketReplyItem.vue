<template>
  <div class="ticket-reply-item">
    <div class="ticket-reply-item__header">
      <span>#{{ reply.id }}</span>
      <span>{{ formatTime(reply.created_at) }}</span>
      <ElTag v-if="reply.is_internal" type="warning" size="small">{{
        t('ticket.internalNote')
      }}</ElTag>
    </div>
    <div class="ticket-reply-item__content">{{ reply.content }}</div>
  </div>
</template>

<script setup lang="ts">
  import { ElTag } from 'element-plus'
  import { useI18n } from 'vue-i18n'
  import { formatTime } from '@utils/common'

  defineProps<{
    reply: Api.Ticket.TicketReply
  }>()

  const { t } = useI18n()
</script>

<style scoped>
  .ticket-reply-item {
    padding: 12px;
    border: 1px solid var(--el-border-color-light);
    border-radius: 8px;
    background: var(--el-bg-color);
  }

  .ticket-reply-item__header {
    display: flex;
    gap: 12px;
    align-items: center;
    margin-bottom: 8px;
    font-size: 12px;
    color: var(--el-text-color-secondary);
  }

  .ticket-reply-item__content {
    white-space: pre-wrap;
    line-height: 1.5;
  }
</style>
