<template>
  <div class="ticket-page">
    <div>
      <ElButton type="primary" @click="openCreate">{{ t('ticket.category.create') }}</ElButton>
    </div>
    <ElTable :data="list" v-loading="loading">
      <ElTableColumn prop="id" label="ID" width="80" />
      <ElTableColumn prop="name" :label="t('common.name')" min-width="180" />
      <ElTableColumn prop="name_en" label="EN Name" min-width="180" />
      <ElTableColumn prop="sort_order" :label="t('ticket.category.sortOrder')" width="120" />
      <ElTableColumn :label="t('ticket.category.enabled')" width="120">
        <template #default="{ row }">
          <ElTag :type="row.enabled ? 'success' : 'info'">{{ row.enabled ? 'ON' : 'OFF' }}</ElTag>
        </template>
      </ElTableColumn>
      <ElTableColumn :label="t('common.operation')" width="180" fixed="right">
        <template #default="{ row }">
          <ElButton link type="primary" @click="openEdit(row)">{{ t('common.edit') }}</ElButton>
          <ElButton link type="danger" @click="remove(row.id)">{{ t('common.delete') }}</ElButton>
        </template>
      </ElTableColumn>
    </ElTable>

    <ElDialog v-model="visible" :title="editingId ? t('common.edit') : t('ticket.category.create')">
      <ElForm :model="form" label-width="120px">
        <ElFormItem :label="t('common.name')">
          <ElInput v-model="form.name" />
        </ElFormItem>
        <ElFormItem label="EN Name">
          <ElInput v-model="form.name_en" />
        </ElFormItem>
        <ElFormItem :label="t('common.reason')">
          <ElInput v-model="form.description" />
        </ElFormItem>
        <ElFormItem :label="t('ticket.category.sortOrder')">
          <ElInputNumber v-model="form.sort_order" :min="0" />
        </ElFormItem>
        <ElFormItem :label="t('ticket.category.enabled')">
          <ElSwitch v-model="form.enabled" />
        </ElFormItem>
      </ElForm>
      <template #footer>
        <ElButton @click="visible = false">{{ t('common.cancel') }}</ElButton>
        <ElButton type="primary" @click="save">{{ t('common.save') }}</ElButton>
      </template>
    </ElDialog>
  </div>
</template>

<script setup lang="ts">
  import {
    adminCreateTicketCategory,
    adminDeleteTicketCategory,
    adminListTicketCategories,
    adminUpdateTicketCategory
  } from '@/api/ticket'
  import { ElMessage } from 'element-plus'
  import { useI18n } from 'vue-i18n'

  defineOptions({ name: 'TicketCategoriesPage' })

  const { t } = useI18n()
  const loading = ref(false)
  const visible = ref(false)
  const editingId = ref(0)
  const list = ref<Api.Ticket.TicketCategory[]>([])
  const form = reactive<Api.Ticket.UpsertCategoryParams>({
    name: '',
    name_en: '',
    description: '',
    sort_order: 0,
    enabled: true
  })

  const resetForm = () => {
    form.name = ''
    form.name_en = ''
    form.description = ''
    form.sort_order = 0
    form.enabled = true
  }

  const loadCategories = async () => {
    loading.value = true
    try {
      list.value = await adminListTicketCategories()
    } catch (error: any) {
      ElMessage.error(error?.message || t('ticket.messages.loadFailed'))
    } finally {
      loading.value = false
    }
  }

  const openCreate = () => {
    editingId.value = 0
    resetForm()
    visible.value = true
  }

  const openEdit = (item: Api.Ticket.TicketCategory) => {
    editingId.value = item.id
    form.name = item.name
    form.name_en = item.name_en
    form.description = item.description
    form.sort_order = item.sort_order
    form.enabled = item.enabled
    visible.value = true
  }

  const save = async () => {
    if (!form.name.trim() || !form.name_en.trim()) {
      ElMessage.warning(t('ticket.messages.required'))
      return
    }
    try {
      if (editingId.value) {
        await adminUpdateTicketCategory(editingId.value, form)
      } else {
        await adminCreateTicketCategory(form)
      }
      visible.value = false
      ElMessage.success(t('ticket.messages.updated'))
      await loadCategories()
    } catch (error: any) {
      ElMessage.error(error?.message || t('ticket.messages.updateFailed'))
    }
  }

  const remove = async (id: number) => {
    try {
      await adminDeleteTicketCategory(id)
      ElMessage.success(t('ticket.messages.deleted'))
      await loadCategories()
    } catch (error: any) {
      ElMessage.error(error?.message || t('ticket.messages.deleteFailed'))
    }
  }

  onMounted(loadCategories)
</script>

<style scoped>
  .ticket-page {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }
</style>
