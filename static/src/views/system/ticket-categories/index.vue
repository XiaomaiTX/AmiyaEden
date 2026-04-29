<template>
  <div class="ticket-page art-full-height">
    <div>
      <ElButton type="primary" @click="openCreate">{{ t('ticket.category.create') }}</ElButton>
    </div>
    <ElCard class="art-table-card" shadow="never">
      <ArtTableHeader v-model:columns="columnChecks" :loading="loading" @refresh="refreshData" />
      <ArtTable :loading="loading" :data="data" :columns="columns" />
    </ElCard>

    <ElDialog v-model="visible" :title="editingId ? t('common.edit') : t('ticket.category.create')">
      <ElForm :model="form" label-width="120px">
        <ElFormItem :label="t('common.name')">
          <ElInput v-model="form.name" />
        </ElFormItem>
        <ElFormItem :label="t('ticket.category.nameEn')">
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
  import { useTable } from '@/hooks/core/useTable'
  import { ElButton, ElMessage, ElTag } from 'element-plus'
  import { useI18n } from 'vue-i18n'

  defineOptions({ name: 'TicketCategoriesPage' })

  const { t } = useI18n()
  const visible = ref(false)
  const editingId = ref(0)
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

  const listTicketCategoriesTable = async (
    _params?: Api.Ticket.TicketListParams
  ): Promise<Api.Common.PaginatedResponse<Api.Ticket.TicketCategory>> => {
    void _params
    const list = await adminListTicketCategories()

    return {
      list,
      total: list.length,
      page: 1,
      pageSize: list.length || 10
    }
  }

  const { columns, columnChecks, data, loading, refreshData } = useTable({
    core: {
      apiFn: listTicketCategoriesTable,
      columnsFactory: () => [
        { prop: 'id', label: 'ID', width: 80 },
        { prop: 'name', label: t('common.name'), minWidth: 180 },
        { prop: 'name_en', label: t('ticket.category.nameEn'), minWidth: 180 },
        { prop: 'sort_order', label: t('ticket.category.sortOrder'), width: 120 },
        {
          prop: 'enabled',
          label: t('ticket.category.enabled'),
          width: 120,
          formatter: (row) =>
            h(ElTag, { type: row.enabled ? 'success' : 'info' }, () =>
              row.enabled ? t('ticket.category.on') : t('ticket.category.off')
            )
        },
        {
          prop: 'operation',
          label: t('common.operation'),
          width: 180,
          fixed: 'right',
          formatter: (row) =>
            h('div', {}, [
              h(ElButton, { link: true, type: 'primary', onClick: () => openEdit(row) }, () =>
                t('common.edit')
              ),
              h(ElButton, { link: true, type: 'danger', onClick: () => remove(row.id) }, () =>
                t('common.delete')
              )
            ])
        }
      ]
    }
  })

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
      await refreshData()
    } catch (error: any) {
      ElMessage.error(error?.message || t('ticket.messages.updateFailed'))
    }
  }

  const remove = async (id: number) => {
    try {
      await adminDeleteTicketCategory(id)
      ElMessage.success(t('ticket.messages.deleted'))
      await refreshData()
    } catch (error: any) {
      ElMessage.error(error?.message || t('ticket.messages.deleteFailed'))
    }
  }
</script>

<style scoped>
  .ticket-page {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }
</style>
