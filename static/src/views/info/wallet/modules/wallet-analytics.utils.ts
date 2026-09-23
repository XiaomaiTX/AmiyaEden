/**
 * 钱包收支分析的图表数据工具
 *
 * 全部为纯函数：把 /info/wallet/analytics 的响应整理成共享图表组件所需的数据结构。
 * 日界口径与后端一致，均按 EVE 时间（UTC）自然日；可得范围一律取自后端返回的
 * available_from / available_to，不硬编码 ESI 的 30 天窗口。
 *
 * @module views/info/wallet/modules/wallet-analytics.utils
 */

export type WalletAnalyticsPreset = '1d' | '7d' | '30d' | '90d' | 'all'

export interface WalletAnalyticsRange {
  /** 起始日（含），YYYY-MM-DD */
  from: string
  /** 结束日（含），YYYY-MM-DD */
  to: string
}

export interface WalletAnalyticsFlowSeries {
  labels: string[]
  income: number[]
  expense: number[]
}

export interface WalletAnalyticsRefTypeSeries {
  /** 交易类型原始值（ref_type），展示名由页面按既有回退链翻译 */
  refTypes: string[]
  values: number[]
}

/** 时间范围预设，顺序即界面按钮顺序 */
export const WALLET_ANALYTICS_PRESETS: WalletAnalyticsPreset[] = ['1d', '7d', '30d', '90d', 'all']

/** 类型构成图默认展示的条目数 */
export const WALLET_ANALYTICS_TOP_REF_TYPES = 8

const PRESET_DAYS: Record<Exclude<WalletAnalyticsPreset, 'all'>, number> = {
  '1d': 1,
  '7d': 7,
  '30d': 30,
  '90d': 90
}

const ISO_DAY_PATTERN = /^\d{4}-\d{2}-\d{2}$/
const MILLISECONDS_PER_DAY = 24 * 60 * 60 * 1000

/** 解析 YYYY-MM-DD（按 UTC 零点）为时间戳，非法输入返回 null */
export const parseIsoDay = (day: string): number | null => {
  if (!ISO_DAY_PATTERN.test(day)) {
    return null
  }
  const parsed = Date.parse(`${day}T00:00:00Z`)
  return Number.isNaN(parsed) ? null : parsed
}

/** 按 UTC 自然日偏移 YYYY-MM-DD */
export const shiftIsoDay = (day: string, offsetDays: number): string => {
  const base = parseIsoDay(day)
  if (base === null) {
    return day
  }
  return new Date(base + offsetDays * MILLISECONDS_PER_DAY).toISOString().slice(0, 10)
}

/** 图表 X 轴短标签：YYYY-MM-DD -> MM-DD */
export const formatDayLabel = (day: string): string => (day.length >= 10 ? day.slice(5, 10) : day)

/**
 * 解析预设时间范围。
 *
 * 终点取库内最新流水日期（availableTo），向前推固定天数后不早于 availableFrom，
 * 因此范围始终落在后端给出的可分析区间内。没有任何流水时返回 null。
 */
export const resolveAnalyticsRange = (
  preset: WalletAnalyticsPreset,
  availableFrom: string,
  availableTo: string
): WalletAnalyticsRange | null => {
  if (!availableFrom || !availableTo) {
    return null
  }
  if (preset === 'all') {
    return { from: availableFrom, to: availableTo }
  }
  const from = shiftIsoDay(availableTo, -(PRESET_DAYS[preset] - 1))
  return { from: from < availableFrom ? availableFrom : from, to: availableTo }
}

const sortByDayAsc = (
  points: Api.EveInfo.WalletAnalyticsDailyPoint[]
): Api.EveInfo.WalletAnalyticsDailyPoint[] =>
  [...points].sort((left, right) => left.date.localeCompare(right.date))

/** 每日日终余额折线数值（按日期升序） */
export const buildBalanceSeries = (points: Api.EveInfo.WalletAnalyticsDailyPoint[]): number[] =>
  sortByDayAsc(points).map((point) => point.balance)

/**
 * 每日收支双向对比数据（按日期升序）。
 * 支出以正数返回：双向柱状图组件内部会把负向数据翻到零轴下方。
 */
export const buildDailyFlowSeries = (
  points: Api.EveInfo.WalletAnalyticsDailyPoint[]
): WalletAnalyticsFlowSeries => {
  const sorted = sortByDayAsc(points)
  return {
    labels: sorted.map((point) => formatDayLabel(point.date)),
    income: sorted.map((point) => point.income),
    expense: sorted.map((point) => point.expense)
  }
}

const roundUpToNiceMagnitude = (value: number): number => {
  if (!Number.isFinite(value) || value <= 0) {
    return 0
  }
  const magnitude = 10 ** Math.floor(Math.log10(value))
  const normalized = value / magnitude
  const nice = normalized <= 1 ? 1 : normalized <= 2 ? 2 : normalized <= 5 ? 5 : 10
  return nice * magnitude
}

/**
 * 双向柱状图的对称 Y 轴边界：取收支最大值并向上取整到 1/2/5 的整数次幂，
 * 保证柱体不被裁切的同时让坐标轴刻度保持整数级可读值。
 */
export const resolveFlowAxisBound = (income: number[], expense: number[]): number =>
  roundUpToNiceMagnitude(Math.max(0, ...income, ...expense))

/** 收入 / 支出类型构成 Top N（按金额降序，仅保留该方向有金额的类型） */
export const buildRefTypeSeries = (
  breakdown: Api.EveInfo.WalletAnalyticsRefTypeItem[],
  direction: 'income' | 'expense',
  limit: number = WALLET_ANALYTICS_TOP_REF_TYPES
): WalletAnalyticsRefTypeSeries => {
  const ranked = breakdown
    .map((item) => ({
      refType: item.ref_type,
      amount: direction === 'income' ? item.income : item.expense
    }))
    .filter((item) => item.amount > 0)
    .sort((left, right) => right.amount - left.amount)
    .slice(0, limit)

  return {
    refTypes: ranked.map((item) => item.refType),
    values: ranked.map((item) => item.amount)
  }
}
