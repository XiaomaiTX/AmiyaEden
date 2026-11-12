<template>
  <div class="blacklist-page">
    <ElCard shadow="never" class="blacklist-summary">
      <template #header>
        <div class="blacklist-summary__header">
          <h2>{{ t('blacklist.summary.title') }}</h2>
          <ElButton
            type="primary"
            :loading="refreshing"
            :disabled="coolingDown"
            @click="handleRefresh"
          >
            {{ t('blacklist.refresh') }}
          </ElButton>
        </div>
      </template>
      <p class="blacklist-summary__hint">{{ t('blacklist.description') }}</p>
      <ElDescriptions v-if="summary" :column="2" border>
        <ElDescriptionsItem :label="t('blacklist.summary.finishedAt')">
          {{ formatTime(summary.finished_at) }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="t('blacklist.summary.operator')">
          {{ summary.operator_name || '-' }}
        </ElDescriptionsItem>
        <ElDescriptionsItem
          v-if="coolingDown && summary.next_refresh_at"
          :label="t('blacklist.summary.nextRefreshAt')"
          :span="2"
        >
          {{ formatTime(summary.next_refresh_at) }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="t('blacklist.summary.totals')">
          {{ t('blacklist.summary.totalsValue', summaryTotals) }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="t('blacklist.summary.changes')">
          {{ t('blacklist.summary.changesValue', summaryChanges) }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="t('blacklist.summary.unresolved')" :span="2">
          {{ t('blacklist.summary.unresolvedValue', summaryUnresolved) }}
        </ElDescriptionsItem>
        <ElDescriptionsItem
          v-if="summary.lookup_failed > 0"
          :label="t('blacklist.summary.lookupFailed')"
          :span="2"
        >
          {{ t('blacklist.lookupFailed', { count: summary.lookup_failed }) }}
        </ElDescriptionsItem>
      </ElDescriptions>
      <p v-else-if="summaryLoaded" class="blacklist-summary__empty">
        {{ t('blacklist.summary.never') }}
      </p>
    </ElCard>

    <div class="blacklist-page__toolbar">
      <ElSelect v-model="filters.kind" :placeholder="t('blacklist.filters.kind')" clearable>
        <ElOption
          v-for="kind in kindOptions"
          :key="kind"
          :label="t(`blacklist.kinds.${kind}`)"
          :value="kind"
        />
      </ElSelect>
      <ElInput
        v-model="filters.keyword"
        :placeholder="t('blacklist.filters.keyword')"
        clearable
        class="blacklist-page__keyword"
        @clear="handleSearch"
        @keyup.enter="handleSearch"
      />
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
  import { computed, h, onMounted, reactive, ref } from 'vue'
  import { useI18n } from 'vue-i18n'
  import { useNow } from '@vueuse/core'
  import { ElMessage, ElTag } from 'element-plus'
  import { fetchBlacklistEntries, fetchBlacklistSummary, refreshBlacklist } from '@/api/blacklist'
  import { useTable } from '@/hooks/core/useTable'
  import { formatTime } from '@utils/common'

  defineOptions({ name: 'SystemBlacklist' })

  type Entry = Api.Blacklist.Entry
  type Kind = Api.Blacklist.Kind

  const { t } = useI18n()
  const kindOptions: Kind[] = ['character', 'qq', 'discord']
  const filters = reactive<{ kind?: Kind; keyword: string }>({ kind: undefined, keyword: '' })
  const summary = ref<Api.Blacklist.RefreshSummary | null>(null)
  const summaryLoaded = ref(false)
  const refreshing = ref(false)
  const now = useNow({ interval: 1000 })
  const coolingDown = computed(
    () => !!summary.value?.next_refresh_at && new Date(summary.value.next_refresh_at) > now.value
  )

  const errorKey = (value: unknown) =>
    value instanceof Error && value.message.startsWith('blacklist.errors.')
      ? value.message
      : 'blacklist.errors.internal'
  const showError = (value: unknown) => ElMessage.error(t(errorKey(value)))

  const summaryTotals = computed(() => ({
    characters: summary.value?.characters ?? 0,
    qqs: summary.value?.qqs ?? 0,
    discords: summary.value?.discords ?? 0
  }))
  const summaryChanges = computed(() => ({
    added: summary.value?.added ?? 0,
    updated: summary.value?.updated ?? 0,
    removed: summary.value?.removed ?? 0
  }))
  const summaryUnresolved = computed(() => ({
    added: summary.value?.new_unresolved ?? 0,
    skipped: summary.value?.skipped_unresolved ?? 0
  }))

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
      apiFn: fetchBlacklistEntries,
      apiParams: { current: 1, size: 20 },
      columnsFactory: () => [
        {
          prop: 'kind',
          label: t('blacklist.columns.kind'),
          width: 120,
          formatter: (row: Entry) =>
            h(ElTag, { type: 'danger', effect: 'plain' }, () => t(`blacklist.kinds.${row.kind}`))
        },
        {
          prop: 'display',
          label: t('blacklist.columns.value'),
          minWidth: 240,
          formatter: (row: Entry) =>
            row.kind === 'character'
              ? t('blacklist.characterValue', { name: row.display, id: row.value })
              : row.display
        },
        {
          prop: 'reason',
          label: t('blacklist.columns.reason'),
          minWidth: 360,
          formatter: (row: Entry) => h('span', { class: 'blacklist-reason' }, row.reason || '-')
        }
      ]
    },
    hooks: { onError: (error) => ElMessage.error(t(errorKey(new Error(error.message)))) }
  })

  const handleSearch = () => {
    Object.assign(searchParams, {
      current: 1,
      kind: filters.kind || undefined,
      keyword: filters.keyword.trim() || undefined
    })
    getData()
  }

  async function loadSummary() {
    try {
      summary.value = await fetchBlacklistSummary()
      summaryLoaded.value = true
    } catch (error) {
      showError(error)
    }
  }

  async function handleRefresh() {
    refreshing.value = true
    try {
      const result = await refreshBlacklist()
      summary.value = result
      summaryLoaded.value = true
      const message = t('blacklist.refreshSuccess', {
        added: result.added,
        updated: result.updated,
        removed: result.removed
      })
      ElMessage.success(
        result.lookup_failed > 0
          ? `${message}\n${t('blacklist.lookupFailed', { count: result.lookup_failed })}`
          : message
      )
      await refreshData()
    } catch (error) {
      showError(error)
    } finally {
      refreshing.value = false
    }
  }

  onMounted(loadSummary)
</script>

<style scoped>
  .blacklist-page {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  .blacklist-summary__header {
    display: flex;
    flex-wrap: wrap;
    gap: 12px;
    align-items: center;
    justify-content: space-between;
  }

  .blacklist-summary__header h2 {
    margin: 0;
    font-size: 16px;
  }

  .blacklist-summary__hint,
  .blacklist-summary__empty {
    margin: 0 0 12px;
    color: var(--el-text-color-secondary);
  }

  .blacklist-summary__empty {
    margin-bottom: 0;
  }

  .blacklist-page__toolbar {
    display: flex;
    flex-wrap: wrap;
    gap: 12px;
    align-items: center;
  }

  .blacklist-page__toolbar .el-select {
    width: 160px;
  }

  .blacklist-page__keyword {
    width: 360px;
    max-width: 100%;
  }

  :deep(.blacklist-reason) {
    white-space: pre-line;
  }
</style>
