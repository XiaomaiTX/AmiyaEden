<template>
  <ElTag :type="tagType">{{ t(`ticket.priority.${priority}`) }}</ElTag>
</template>

<script setup lang="ts">
  import { ElTag } from 'element-plus'
  import { computed } from 'vue'
  import { useI18n } from 'vue-i18n'

  const props = defineProps<{
    priority: Api.Ticket.TicketPriority
  }>()

  const { t } = useI18n()

  const tagType = computed(() => {
    if (props.priority === 'high') return 'danger'
    if (props.priority === 'medium') return 'warning'
    return 'info'
  })
</script>

