<template>
  <div class="ticket-page art-full-height">
    <div class="ticket-page__toolbar">
      <ElTabs v-model="activeTab" @tab-change="handleStatusTabChange">
        <ElTabPane :label="t('ticket.tabs.active')" name="active" />
        <ElTabPane :label="t('ticket.tabs.completed')" name="completed" />
      </ElTabs>
      <ElInput
        v-model="filters.keyword"
        :placeholder="t('ticket.filters.keyword')"
        style="width: 260px"
        clearable
      />
      <ElSelect
        v-model="filters.category_id"
        clearable
        :placeholder="t('ticket.filters.category')"
        style="width: 200px"
      >
        <ElOption
          v-for="category in categoryOptions"
          :key="category.id"
          :label="getCategoryLabel(category)"
          :value="category.id"
        />
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
    adminListTicketCategories,
    adminListTickets,
    adminUpdateTicketStatus
  } from '@/api/ticket'
  import { useTable } from '@/hooks/core/useTable'
  import { formatTime } from '@utils/common'
  import { ElButton, ElMessage, ElOption, ElSelect } from 'element-plus'
  import { useI18n } from 'vue-i18n'
  defineOptions({ name: 'TicketManagementPage' })
  type TicketManagementTab = 'active' | 'completed'
  type TicketStatusQuery = Api.Ticket.AdminTicketListParams['status']
  const { t, locale } = useI18n()
  const router = useRouter()
  const activeTab = ref<TicketManagementTab>('active')
  const statusTabs: Record<TicketManagementTab, Api.Ticket.TicketStatus[]> = {
    active: ['pending', 'in_progress'],
    completed: ['completed']
  }
  const getTabStatusQuery = (tab: TicketManagementTab): TicketStatusQuery =>
    statusTabs[tab].join(',') as TicketStatusQuery
  const categoryOptions = ref<Api.Ticket.TicketCategory[]>([])
  const filters = reactive<{ keyword: string; category_id?: number }>({
    keyword: '',
    category_id: undefined
  })
  const getCategoryLabel = (category: Api.Ticket.TicketCategory) =>
    locale.value.startsWith('zh') ? category.name : category.name_en || category.name
  const getTicketCategoryLabel = (ticket: Api.Ticket.TicketItem) =>
    locale.value.startsWith('zh')
      ? ticket.category_name || ticket.category_name_en || String(ticket.category_id)
      : ticket.category_name_en || ticket.category_name || String(ticket.category_id)
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
        status: getTabStatusQuery('active'),
        category_id: filters.category_id
      },
      columnsFactory: () => [
        { prop: 'id', label: 'ID', width: 80 },
        {
          prop: 'requester_name',
          label: t('ticket.columns.submitter'),
          minWidth: 160,
          formatter: (row) =>
            row.requester_name || row.requester_character_name || t('ticket.unknownUser')
        },
        {
          prop: 'category_name',
          label: t('ticket.columns.category'),
          minWidth: 140,
          formatter: (row) => getTicketCategoryLabel(row)
        },
        { prop: 'title', label: t('ticket.columns.title'), minWidth: 200 },
        {
          prop: 'description',
          label: t('ticket.columns.content'),
          minWidth: 260,
          formatter: (row) => h('span', { class: 'ticket-content-preview' }, row.description || '-')
        },
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
          prop: 'updated_at',
          label: t('common.updatedAt'),
          width: 180,
          formatter: (row) => h('span', {}, formatTime(row.updated_at))
        },
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
  const handleSearch = () => {
    Object.assign(searchParams, {
      current: 1,
      keyword: filters.keyword,
      status: getTabStatusQuery(activeTab.value),
      category_id: filters.category_id
    })
    getData()
  }
  const handleStatusTabChange = (name: string | number) => {
    activeTab.value = name as TicketManagementTab
    handleSearch()
  }
  const loadCategories = async () => {
    try {
      categoryOptions.value = await adminListTicketCategories()
    } catch (error: any) {
      ElMessage.error(error?.message || t('ticket.messages.loadFailed'))
    }
  }
  const openDetail = (id: number) =>
    router.push({ name: 'TicketAdminDetail', params: { id: String(id) } })
  onMounted(loadCategories)
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
  .ticket-page__toolbar :deep(.el-tabs__header) {
    margin: 0;
  }
  .ticket-content-preview {
    display: -webkit-box;
    overflow: hidden;
    line-height: 1.4;
    white-space: normal;
    -webkit-box-orient: vertical;
    -webkit-line-clamp: 2;
  }
</style>
