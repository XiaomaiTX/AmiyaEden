<template>
  <div class="art-card h-128 p-5 mb-5 max-sm:mb-4">
    <div class="art-card-header">
      <div class="title">
        <h4>{{ title }}</h4>
        <p>
          {{ $t('console.papChart.recentMonths', { count: chartData.length }) }}
        </p>
      </div>
    </div>
    <div v-if="chartData.length > 0" class="mt-2">
      <ArtLineChart
        height="13rem"
        :data="chartValues"
        :xAxisData="chartLabels"
        :showAxisLine="false"
      />
    </div>
    <div v-else class="flex-cc h-[calc(100%-40px)] text-g-500 text-sm">
      {{ $t('console.papChart.empty') }}
    </div>
  </div>
</template>

<script setup lang="ts">
  import { useI18n } from 'vue-i18n'
  import ArtLineChart from '@/components/core/charts/art-line-chart/index.vue'
  import { buildPapTrendSeries } from './pap-chart.utils'

  const { t } = useI18n()
  const props = defineProps<{
    title: string
    data: Api.Dashboard.PapMonthly[]
  }>()

  const chartData = computed(() => {
    return buildPapTrendSeries(props.data)
  })

  const chartLabels = computed(() => {
    return chartData.value.map((d) =>
      t('console.papChart.monthLabel', {
        year: d.year,
        month: String(d.month).padStart(2, '0')
      })
    )
  })

  const chartValues = computed(() => {
    return chartData.value.map((d) => d.total_pap)
  })
</script>
