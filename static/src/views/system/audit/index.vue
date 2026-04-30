<template>
  <div class="audit-admin-page art-full-height">
    <ElTabs v-model="activeTab">
      <ElTabPane :label="$t('auditAdmin.tabs.events')" name="events">
        <ElCard class="art-table-card" shadow="never">
          <ArtTableHeader v-model:columns="columnChecks" :loading="loading" @refresh="refreshData">
            <template #left>
              <ElInput
                v-model="filterForm.keyword"
                :placeholder="$t('auditAdmin.placeholders.keyword')"
                clearable
                style="width: 220px"
                @keyup.enter="handleSearch"
              />
              <ElSelect
                v-model="filterForm.category"
                :placeholder="$t('auditAdmin.filters.category')"
                clearable
                style="width: 160px"
              >
                <ElOption value="permission" label="permission" />
                <ElOption value="fuxi_wallet" label="fuxi_wallet" />
                <ElOption value="config" label="config" />
                <ElOption value="approval" label="approval" />
                <ElOption value="task_ops" label="task_ops" />
                <ElOption value="security" label="security" />
              </ElSelect>
              <ElSelect
                v-model="filterForm.result"
                :placeholder="$t('auditAdmin.filters.result')"
                clearable
                style="width: 140px"
              >
                <ElOption :label="$t('auditAdmin.results.success')" value="success" />
                <ElOption :label="$t('auditAdmin.results.failed')" value="failed" />
              </ElSelect>
              <ElInput
                v-model="filterForm.request_id"
                :placeholder="$t('auditAdmin.placeholders.requestId')"
                clearable
                style="width: 240px"
                @keyup.enter="handleSearch"
              />
              <ElButton type="primary" @click="handleSearch">{{ $t('common.search') }}</ElButton>
              <ElButton @click="handleReset">{{ $t('common.reset') }}</ElButton>
            </template>
          </ArtTableHeader>

          <ArtTable
            :loading="loading"
            :data="data"
            :columns="columns"
            :pagination="pagination"
            visual-variant="ledger"
            @pagination:size-change="handleSizeChange"
            @pagination:current-change="handleCurrentChange"
          />
        </ElCard>
      </ElTabPane>

      <ElTabPane :label="$t('auditAdmin.tabs.exports')" name="exports">
        <ElCard class="art-table-card" shadow="never">
          <ElAlert
            :title="$t('auditAdmin.export.unavailableTitle')"
            :description="$t('auditAdmin.export.unavailableDescription')"
            type="info"
            :closable="false"
            show-icon
          />
        </ElCard>
      </ElTabPane>
    </ElTabs>

    <ElDrawer v-model="detailVisible" :title="$t('auditAdmin.detailTitle')" size="42%">
      <div v-if="currentEvent">
        <ElDescriptions :column="1" border>
          <ElDescriptionsItem :label="$t('auditAdmin.columns.eventId')">{{
            currentEvent.event_id
          }}</ElDescriptionsItem>
          <ElDescriptionsItem :label="$t('auditAdmin.columns.time')">{{
            formatTime(currentEvent.occurred_at)
          }}</ElDescriptionsItem>
          <ElDescriptionsItem :label="$t('auditAdmin.columns.category')">{{
            currentEvent.category
          }}</ElDescriptionsItem>
          <ElDescriptionsItem :label="$t('auditAdmin.columns.action')">{{
            currentEvent.action
          }}</ElDescriptionsItem>
          <ElDescriptionsItem :label="$t('auditAdmin.columns.result')">
            <ElTag :type="currentEvent.result === 'success' ? 'success' : 'danger'">
              {{
                currentEvent.result === 'success'
                  ? $t('auditAdmin.results.success')
                  : $t('auditAdmin.results.failed')
              }}
            </ElTag>
          </ElDescriptionsItem>
          <ElDescriptionsItem :label="$t('auditAdmin.columns.actorUserId')">{{
            currentEvent.actor_user_id || '-'
          }}</ElDescriptionsItem>
          <ElDescriptionsItem :label="$t('auditAdmin.columns.targetUserId')">{{
            currentEvent.target_user_id || '-'
          }}</ElDescriptionsItem>
          <ElDescriptionsItem :label="$t('auditAdmin.columns.requestId')">{{
            currentEvent.request_id || '-'
          }}</ElDescriptionsItem>
        </ElDescriptions>

        <div class="details-block">
          <div class="details-title">{{ $t('auditAdmin.columns.details') }}</div>
          <pre>{{ prettyDetails(currentEvent.details_json) }}</pre>
        </div>
      </div>
    </ElDrawer>
  </div>
</template>

<script setup lang="ts">
  import {
    ElAlert,
    ElButton,
    ElCard,
    ElDescriptions,
    ElDescriptionsItem,
    ElDrawer,
    ElInput,
    ElOption,
    ElSelect,
    ElTabPane,
    ElTabs,
    ElTag
  } from 'element-plus'
  import { useI18n } from 'vue-i18n'
  import { formatTime } from '@utils/common'
  import { useTable } from '@/hooks/core/useTable'
  import { adminListAuditEvents } from '@/api/audit'

  defineOptions({ name: 'SystemAudit' })
  const { t } = useI18n()

  type AuditEvent = Api.Audit.AuditEvent

  const activeTab = ref('events')
  const filterForm = reactive<Api.Audit.AuditEventSearchParams>({
    keyword: '',
    category: '',
    result: undefined,
    request_id: ''
  })

  const detailVisible = ref(false)
  const currentEvent = ref<AuditEvent | null>(null)

  const openDetail = (row: AuditEvent) => {
    currentEvent.value = row
    detailVisible.value = true
  }

  const prettyDetails = (raw: string) => {
    if (!raw) return '{}'
    try {
      return JSON.stringify(JSON.parse(raw), null, 2)
    } catch {
      return raw
    }
  }

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
      apiFn: adminListAuditEvents,
      apiParams: { current: 1, size: 200 },
      columnsFactory: () => [
        { type: 'index', label: '#', width: 60 },
        {
          prop: 'occurred_at',
          label: t('auditAdmin.columns.time'),
          minWidth: 180,
          formatter: (row: AuditEvent) => h('span', {}, formatTime(row.occurred_at))
        },
        { prop: 'category', label: t('auditAdmin.columns.category'), width: 130 },
        { prop: 'action', label: t('auditAdmin.columns.action'), minWidth: 180 },
        { prop: 'actor_user_id', label: t('auditAdmin.columns.actorUserId'), width: 120 },
        { prop: 'target_user_id', label: t('auditAdmin.columns.targetUserId'), width: 120 },
        {
          prop: 'result',
          label: t('auditAdmin.columns.result'),
          width: 100,
          formatter: (row: AuditEvent) =>
            h(
              ElTag,
              { size: 'small', type: row.result === 'success' ? 'success' : 'danger' },
              () =>
                row.result === 'success'
                  ? t('auditAdmin.results.success')
                  : t('auditAdmin.results.failed')
            )
        },
        {
          prop: 'request_id',
          label: t('auditAdmin.columns.requestId'),
          minWidth: 180,
          showOverflowTooltip: true
        },
        {
          prop: 'resource_id',
          label: t('auditAdmin.columns.resourceId'),
          minWidth: 140,
          showOverflowTooltip: true
        },
        {
          prop: 'operation',
          label: t('common.operation'),
          width: 100,
          fixed: 'right',
          formatter: (row: AuditEvent) =>
            h(ElButton, { type: 'primary', text: true, onClick: () => openDetail(row) }, () =>
              t('auditAdmin.actions.detail')
            )
        }
      ]
    }
  })

  const handleSearch = () => {
    Object.assign(searchParams, {
      category: filterForm.category || undefined,
      result: filterForm.result || undefined,
      request_id: filterForm.request_id || undefined,
      keyword: filterForm.keyword || undefined
    })
    getData()
  }

  const handleReset = () => {
    filterForm.keyword = ''
    filterForm.category = ''
    filterForm.result = undefined
    filterForm.request_id = ''
    Object.assign(searchParams, {
      category: undefined,
      result: undefined,
      request_id: undefined,
      keyword: undefined
    })
    getData()
  }
</script>

<style scoped lang="scss">
  .audit-admin-page {
    .details-block {
      margin-top: 16px;
      .details-title {
        font-weight: 600;
        margin-bottom: 8px;
      }
      pre {
        margin: 0;
        background: var(--el-fill-color-light);
        border-radius: 6px;
        padding: 12px;
        overflow: auto;
      }
    }
  }
</style>
