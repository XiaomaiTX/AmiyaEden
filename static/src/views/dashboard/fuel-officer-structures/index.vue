<template>
  <div class="art-full-height">
    <ElCard shadow="never" class="art-table-card">
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
  import { ElMessage } from 'element-plus'
  import { computed, onMounted, reactive, ref } from 'vue'
  import { useI18n } from 'vue-i18n'
  import { fetchMyAssignedCorporationStructures } from '@/api/corporation-structures'
  import { formatTime } from '@/utils/common/time'
  import type { ColumnOption } from '@/types/component'

  defineOptions({ name: 'DashboardFuelOfficerStructures' })

  const { t } = useI18n()
  const loading = ref(false)
  const pagination = reactive({ current: 1, size: 20, total: 0 })
  const data = ref<Api.Dashboard.CorporationStructureRow[]>([])

  const columns = computed<ColumnOption<Api.Dashboard.CorporationStructureRow>[]>(() => [
    {
      prop: 'corporation_name',
      label: t('corporationStructures.table.corporation'),
      minWidth: 220
    },
    { prop: 'name', label: t('corporationStructures.table.name'), minWidth: 220 },
    { prop: 'system_name', label: t('corporationStructures.table.system'), minWidth: 200 },
    { prop: 'state', label: t('corporationStructures.table.state'), minWidth: 160 },
    {
      prop: 'fuel_remaining',
      label: t('corporationStructures.table.fuelRemaining'),
      minWidth: 140
    },
    {
      prop: 'fuel_per_hour',
      label: t('corporationStructures.table.fuelPerHour'),
      minWidth: 140,
      formatter: (row: Api.Dashboard.CorporationStructureRow) =>
        row.fuel_estimate_incomplete
          ? formatFuelEstimateStatus(row.fuel_estimate_status)
          : row.fuel_per_hour != null
            ? row.fuel_per_hour
            : '-'
    },
    {
      prop: 'fuel_to_month_end',
      label: t('corporationStructures.table.fuelToMonthEnd'),
      minWidth: 160,
      formatter: (row: Api.Dashboard.CorporationStructureRow) =>
        row.fuel_estimate_incomplete
          ? formatFuelEstimateStatus(row.fuel_estimate_status)
          : row.fuel_to_month_end != null
            ? row.fuel_to_month_end
            : '-'
    },
    {
      prop: 'state_timer_end',
      label: t('corporationStructures.table.timerEnd'),
      minWidth: 180
    },
    {
      prop: 'updated_at',
      label: t('corporationStructures.table.updatedAt'),
      minWidth: 180,
      formatter: (row: Api.Dashboard.CorporationStructureRow) =>
        row.updated_at ? formatTime(new Date(row.updated_at * 1000).toISOString()) : '-'
    }
  ])

  function formatFuelEstimateStatus(
    status: Api.Dashboard.CorporationStructureRow['fuel_estimate_status']
  ) {
    const keyByStatus: Record<string, string> = {
      authorization_required: 'fuelEstimateAuthorizationRequired',
      activity_mapping_required: 'fuelEstimateActivityMappingRequired',
      module_mismatch: 'fuelEstimateModuleMismatch',
      rate_unavailable: 'fuelEstimateRateUnavailable',
      ambiguous_module: 'fuelEstimateAmbiguousModule'
    }
    return t(`corporationStructures.table.${keyByStatus[status] || 'fuelEstimateIncomplete'}`)
  }

  async function loadData() {
    loading.value = true
    try {
      const res = await fetchMyAssignedCorporationStructures({
        page: pagination.current,
        page_size: pagination.size
      })
      data.value = res.items
      pagination.total = res.total
    } catch (error: any) {
      ElMessage.error(error?.message || t('common.fetchFail'))
    } finally {
      loading.value = false
    }
  }

  function handleSizeChange(size: number) {
    pagination.size = size
    pagination.current = 1
    void loadData()
  }

  function handleCurrentChange(current: number) {
    pagination.current = current
    void loadData()
  }

  onMounted(() => {
    void loadData()
  })
</script>
