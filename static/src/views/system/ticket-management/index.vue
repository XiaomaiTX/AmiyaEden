<template>
  <div class="ticket-page art-full-height">
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
      <ElButton type="primary" @click="handleSearch">{{ t('common.search') }}</ElButton>
    </div>

    <ElCard class="art-table-card" shadow="never">
      <ArtTableHeader v-model:columns="columnChecks" :loading="loading" @refresh="refreshData" />
      <ArtTable
        :loading="loading"
        :data="data"
        :columns="columns"
        :pagination="pagination"
        @pagination:size-change="handleSizeChange"
        @pagination:current-change="handleCurrentChange"
      />
    </ElCard>
  </div>
</template>

<script setup lang="ts">
  import {
    adminListTickets,
    adminUpdateTicketPriority,
    adminUpdateTicketStatus
  } from '@/api/ticket'
  import { useTable } from '@/hooks/core/useTable'
  import { ElButton, ElMessage, ElOption, ElSelect } from 'element-plus'
  import { useI18n } from 'vue-i18n'

  defineOptions({ name: 'TicketManagementPage' })

  const { t } = useI18n()
  const router = useRouter()

  const filters = reactive<{ keyword: string; status: Api.Ticket.TicketStatus | '' }>({
    keyword: '',
    status: ''
  })

  const {
    columns,
    columnChecks,
    data,
    loading,
    pagination,
    searchParams,
    getData,
    refreshData,
    refreshUpdate,
    handleSizeChange,
    handleCurrentChange
  } = useTable({
    core: {
      apiFn: adminListTickets,
      apiParams: {
        current: 1,
        size: 20,
        keyword: filters.keyword,
        status: filters.status
      },
      columnsFactory: () => [
        { prop: 'id', label: 'ID', width: 80 },
        { prop: 'user_id', label: t('ticket.columns.submitter'), width: 100 },
        { prop: 'title', label: t('ticket.columns.title'), minWidth: 200 },
        {
          prop: 'status',
          label: t('ticket.columns.status'),
          width: 180,
          formatter: (row) =>
            h(
              ElSelect,
              {
                modelValue: row.status,
                size: 'small',
                onChange: (val: Api.Ticket.TicketStatus) => updateStatus(row.id, val)
              },
              () => [
                h(ElOption, { label: t('ticket.status.pending'), value: 'pending' }),
                h(ElOption, { label: t('ticket.status.in_progress'), value: 'in_progress' }),
                h(ElOption, { label: t('ticket.status.completed'), value: 'completed' })
              ]
            )
        },
        {
          prop: 'priority',
          label: t('ticket.columns.priority'),
          width: 170,
          formatter: (row) =>
            h(
              ElSelect,
              {
                modelValue: row.priority,
                size: 'small',
                onChange: (val: Api.Ticket.TicketPriority) => updatePriority(row.id, val)
              },
              () => [
                h(ElOption, { label: t('ticket.priority.low'), value: 'low' }),
                h(ElOption, { label: t('ticket.priority.medium'), value: 'medium' }),
                h(ElOption, { label: t('ticket.priority.high'), value: 'high' })
              ]
            )
        },
        { prop: 'updated_at', label: t('common.updatedAt'), width: 180 },
        {
          prop: 'operation',
          label: t('common.operation'),
          width: 120,
          fixed: 'right',
          formatter: (row) =>
            h(
              ElButton,
              {
                link: true,
                type: 'primary',
                onClick: () => openDetail(row.id)
              },
              () => t('ticket.viewDetail')
            )
        }
      ]
    }
  })

  const updateStatus = async (id: number, status: Api.Ticket.TicketStatus) => {
    try {
      await adminUpdateTicketStatus(id, { status })
      ElMessage.success(t('ticket.messages.updated'))
      await refreshUpdate()
    } catch (error: any) {
      ElMessage.error(error?.message || t('ticket.messages.updateFailed'))
      await refreshData()
    }
  }

  const updatePriority = async (id: number, priority: Api.Ticket.TicketPriority) => {
    try {
      await adminUpdateTicketPriority(id, { priority })
      ElMessage.success(t('ticket.messages.updated'))
      await refreshUpdate()
    } catch (error: any) {
      ElMessage.error(error?.message || t('ticket.messages.updateFailed'))
      await refreshData()
    }
  }

  const handleSearch = () => {
    Object.assign(searchParams, {
      current: 1,
      keyword: filters.keyword,
      status: filters.status
    })
    getData()
  }

  const openDetail = (id: number) =>
    router.push({ name: 'TicketAdminDetail', params: { id: String(id) } })
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
