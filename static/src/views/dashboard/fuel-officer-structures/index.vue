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
        row.fuel_per_hour != null ? row.fuel_per_hour : '-'
    },
    {
      prop: 'fuel_to_month_end',
      label: t('corporationStructures.table.fuelToMonthEnd'),
      minWidth: 160,
      formatter: (row: Api.Dashboard.CorporationStructureRow) =>
        row.fuel_to_month_end != null ? row.fuel_to_month_end : '-'
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
