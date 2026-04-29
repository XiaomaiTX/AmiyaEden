<template>
  <ElTag :type="tagType">{{ t(`ticket.status.${status}`) }}</ElTag>
</template>

<script setup lang="ts">
  import { ElTag } from 'element-plus'
  import { computed } from 'vue'
  import { useI18n } from 'vue-i18n'

  const props = defineProps<{
    status: Api.Ticket.TicketStatus
  }>()

  const { t } = useI18n()

  const tagType = computed(() => {
    if (props.status === 'pending') return 'warning'
    if (props.status === 'in_progress') return 'primary'
    return 'success'
  })
</script>

