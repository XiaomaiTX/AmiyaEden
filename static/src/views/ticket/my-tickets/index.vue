<template>
  <div class="ticket-page art-full-height">
    <div class="ticket-page__toolbar">
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
      <ElButton @click="goCreate">{{ t('ticket.createTicket') }}</ElButton>
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
  import { listMyTickets } from '@/api/ticket'
  import { useTable } from '@/hooks/core/useTable'
  import TicketPriorityBadge from '@/components/ticket/TicketPriorityBadge.vue'
  import TicketStatusBadge from '@/components/ticket/TicketStatusBadge.vue'
  import { ElButton } from 'element-plus'
  import { useI18n } from 'vue-i18n'

  defineOptions({ name: 'TicketMyTickets' })

  const { t } = useI18n()
  const router = useRouter()

  const filters = ref<{ status?: Api.Ticket.TicketStatus | '' }>({ status: '' })

  const {
    columns,
    columnChecks,
    data,
    loading,
    pagination,
    searchParams,
    getData,
    refreshData,
    handleSizeChange,
    handleCurrentChange
  } = useTable({
    core: {
      apiFn: listMyTickets,
      apiParams: {
        current: 1,
        size: 20,
        status: filters.value.status
      },
      columnsFactory: () => [
        { prop: 'id', label: 'ID', width: 80 },
        { prop: 'title', label: t('ticket.columns.title'), minWidth: 220 },
        {
          prop: 'status',
          label: t('ticket.columns.status'),
          width: 120,
          formatter: (row) => h(TicketStatusBadge, { status: row.status })
        },
        {
          prop: 'priority',
          label: t('ticket.columns.priority'),
          width: 120,
          formatter: (row) => h(TicketPriorityBadge, { priority: row.priority })
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
                onClick: () => goDetail(row.id)
              },
              () => t('ticket.viewDetail')
            )
        }
      ]
    }
  })

  const handleSearch = () => {
    Object.assign(searchParams, {
      current: 1,
      status: filters.value.status
    })
    getData()
  }

  const goCreate = () => router.push({ name: 'TicketCreate' })
  const goDetail = (id: number) => router.push({ name: 'TicketDetail', params: { id: String(id) } })
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
