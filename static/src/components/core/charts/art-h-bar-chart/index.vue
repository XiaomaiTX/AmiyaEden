<!-- 水平柱状图 -->
<template>
  <div
    ref="chartRef"
    class="relative w-full"
    :style="{ height: props.height }"
    v-loading="props.loading"
  ></div>
</template>

<script setup lang="ts">
  import { useChartOps, useChartComponent } from '@/hooks/core/useChart'
  import { getCssVar } from '@/utils/ui'
  import { graphic, type EChartsOption } from '@/plugins/echarts'
  import type { BarChartProps, BarDataItem } from '@/types/component/chart'

  defineOptions({ name: 'ArtHBarChart' })

  const props = withDefaults(defineProps<BarChartProps>(), {
    // 基础配置
    height: useChartOps().chartHeight,
    loading: false,
    isEmpty: false,
    colors: () => useChartOps().colors,

    // 数据配置
    data: () => [0, 0, 0, 0, 0, 0, 0],
    xAxisData: () => [],
    barWidth: '36%',
    // 类目较少时（如按类型构成仅几个类目）限制柱体厚度，避免百分比折算后过粗
    barMaxWidth: 14,
    stack: false,
    reverseCategoryAxis: false,

    // 轴线显示配置
    showAxisLabel: true,
    showAxisLine: true,
    showSplitLine: true,

    // 交互配置
    showTooltip: true,
    showLegend: false,
    legendPosition: 'bottom'
  })

  // 判断是否为多数据
  const isMultipleData = computed(() => {
    return (
      Array.isArray(props.data) &&
      props.data.length > 0 &&
      typeof props.data[0] === 'object' &&
      'name' in props.data[0]
    )
  })

  // 获取颜色配置
  const getColor = (customColor?: string, index?: number) => {
    if (customColor) return customColor

    if (index !== undefined) {
      return props.colors![index % props.colors!.length]
    }

    // 默认渐变色
    return new graphic.LinearGradient(0, 0, 1, 0, [
      {
        offset: 0,
        color: getCssVar('--el-color-primary')
      },
      {
        offset: 1,
        color: getCssVar('--el-color-primary-light-4')
      }
    ])
  }

  // 创建渐变色
  const createGradientColor = (color: string) => {
    return new graphic.LinearGradient(0, 0, 1, 0, [
      {
        offset: 0,
        color: color
      },
      {
        offset: 1,
        color: color
      }
    ])
  }

  // 获取基础样式配置
  const getBaseItemStyle = (
    color: string | InstanceType<typeof graphic.LinearGradient> | undefined
  ) => ({
    borderRadius: 4,
    color: typeof color === 'string' ? createGradientColor(color) : color
  })

  // 创建系列配置
  const createSeriesItem = (config: {
    name?: string
    data: number[]
    color?: string | InstanceType<typeof graphic.LinearGradient>
    barWidth?: string | number
    barMaxWidth?: number
    stack?: string
  }) => {
    const animationConfig = getAnimationConfig()

    return {
      name: config.name,
      data: config.data,
      type: 'bar' as const,
      stack: config.stack,
      itemStyle: getBaseItemStyle(config.color),
      barWidth: config.barWidth || props.barWidth,
      barMaxWidth: config.barMaxWidth ?? props.barMaxWidth,
      ...animationConfig
    }
  }

  // 使用新的图表组件抽象
  const {
    chartRef,
    getAxisLineStyle,
    getAxisLabelStyle,
    getAxisTickStyle,
    getSplitLineStyle,
    getAnimationConfig,
    getTooltipStyle,
    getLegendStyle,
    getGridWithLegend
  } = useChartComponent({
    props,
    checkEmpty: () => {
      // 检查单数据情况
      if (Array.isArray(props.data) && typeof props.data[0] === 'number') {
        const singleData = props.data as number[]
        return !singleData.length || singleData.every((val) => val === 0)
      }

      // 检查多数据情况
      if (Array.isArray(props.data) && typeof props.data[0] === 'object') {
        const multiData = props.data as BarDataItem[]
        return (
          !multiData.length ||
          multiData.every((item) => !item.data?.length || item.data.every((val) => val === 0))
        )
      }

      return true
    },
    watchSources: [() => props.data, () => props.xAxisData, () => props.colors],
    generateOptions: (): EChartsOption => {
      const options: EChartsOption = {
        grid: getGridWithLegend(props.showLegend && isMultipleData.value, props.legendPosition, {
          top: 15,
          // 右侧留白：数值轴刻度（如 30.00 B）在 grid.right=0 时会被容器右缘裁切
          right: 24,
          left: 0
        }),
        tooltip: props.showTooltip ? getTooltipStyle('axis', getTooltipValueOptions()) : undefined,
        xAxis: {
          type: 'value',
          axisTick: getAxisTickStyle(),
          axisLine: getAxisLineStyle(props.showAxisLine),
          axisLabel: getValueAxisLabelStyle(),
          splitLine: getSplitLineStyle(props.showSplitLine)
        },
        yAxis: {
          type: 'category',
          data: props.xAxisData,
          inverse: props.reverseCategoryAxis,
          axisTick: getAxisTickStyle(),
          axisLabel: getAxisLabelStyle(props.showAxisLabel),
          axisLine: getAxisLineStyle(props.showAxisLine)
        }
      }

      // 添加图例配置
      if (props.showLegend && isMultipleData.value) {
        options.legend = getLegendStyle(props.legendPosition)
      }

      // 生成系列数据
      if (isMultipleData.value) {
        const multiData = props.data as BarDataItem[]
        options.series = multiData.map((item, index) => {
          const computedColor = getColor(props.colors[index], index)

          return createSeriesItem({
            name: item.name,
            data: item.data,
            color: computedColor,
            barWidth: item.barWidth,
            barMaxWidth: item.barWidth ? undefined : props.barMaxWidth,
            stack: props.stack ? item.stack || 'total' : undefined
          })
        })
      } else {
        // 单数据情况
        const singleData = props.data as number[]
        const computedColor = getColor()

        options.series = [
          createSeriesItem({
            data: singleData,
            color: computedColor
          })
        ]
      }

      return options
    }
  })

  /**
   * 数值轴刻度样式
   *
   * 传入 `valueFormatter` 时用它渲染刻度标签（如 ISK 智能缩写 1.50 B），
   * 避免原始长数字（2,600,000,000）在窄轴上互相重叠；未传入则沿用默认样式。
   */
  const getValueAxisLabelStyle = () => {
    const baseStyle = getAxisLabelStyle(props.showAxisLabel)
    const { valueFormatter } = props
    if (!valueFormatter) {
      return baseStyle
    }
    return {
      ...baseStyle,
      // 刻度过密时自动隐藏，避免标签互相重叠
      hideOverlap: true,
      formatter: (value: number) => valueFormatter(Number(value))
    }
  }

  /**
   * 提示框数值格式化
   *
   * 传入 `valueFormatter` 时同步用于悬浮提示中的数值（如 ISK 智能缩写 1.50 B），
   * 与数值轴刻度保持一致；未传入则保留 ECharts 默认渲染。
   */
  const getTooltipValueOptions = () => {
    const { valueFormatter } = props
    if (!valueFormatter) {
      return {}
    }
    return {
      valueFormatter: (value: unknown) => valueFormatter(Number(value))
    }
  }
</script>
