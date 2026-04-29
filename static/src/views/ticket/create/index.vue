<template>
  <div class="ticket-create-page">
    <ElCard>
      <ElForm :model="form" label-width="110px">
        <ElFormItem :label="t('ticket.form.category')">
          <ElSelect v-model="form.category_id" :placeholder="t('ticket.form.category')" style="width: 320px">
            <ElOption v-for="item in categories" :key="item.id" :label="item.name" :value="item.id" />
          </ElSelect>
        </ElFormItem>
        <ElFormItem :label="t('ticket.form.title')">
          <ElInput v-model="form.title" maxlength="200" />
        </ElFormItem>
        <ElFormItem :label="t('ticket.form.priority')">
          <ElSelect v-model="form.priority" style="width: 180px">
            <ElOption :label="t('ticket.priority.low')" value="low" />
            <ElOption :label="t('ticket.priority.medium')" value="medium" />
            <ElOption :label="t('ticket.priority.high')" value="high" />
          </ElSelect>
        </ElFormItem>
        <ElFormItem :label="t('ticket.form.description')">
          <ElInput v-model="form.description" type="textarea" :rows="6" />
        </ElFormItem>
        <ElFormItem>
          <ElButton type="primary" :loading="submitting" @click="submit">{{ t('ticket.submit') }}</ElButton>
        </ElFormItem>
      </ElForm>
    </ElCard>
  </div>
</template>

<script setup lang="ts">
  import { createTicket, listTicketCategories } from '@/api/ticket'
  import { ElMessage } from 'element-plus'
  import { useI18n } from 'vue-i18n'

  defineOptions({ name: 'TicketCreatePage' })

  const { t } = useI18n()
  const router = useRouter()

  const submitting = ref(false)
  const categories = ref<Api.Ticket.TicketCategory[]>([])
  const form = reactive<Api.Ticket.CreateTicketParams>({
    category_id: 0,
    title: '',
    description: '',
    priority: 'medium'
  })

  const loadCategories = async () => {
    categories.value = await listTicketCategories()
    if (!form.category_id && categories.value.length > 0) {
      form.category_id = categories.value[0].id
    }
  }

  const submit = async () => {
    if (!form.category_id || !form.title.trim() || !form.description.trim()) {
      ElMessage.warning(t('ticket.messages.required'))
      return
    }
    submitting.value = true
    try {
      await createTicket(form)
      ElMessage.success(t('ticket.messages.created'))
      router.push({ name: 'TicketMyList' })
    } catch (error: any) {
      ElMessage.error(error?.message || t('ticket.messages.createFailed'))
    } finally {
      submitting.value = false
    }
  }

  onMounted(loadCategories)
</script>

