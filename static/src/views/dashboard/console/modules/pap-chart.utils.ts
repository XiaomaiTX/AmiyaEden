export type PapMonthly = Api.Dashboard.PapMonthly

export const PAP_TREND_MONTHS = 12

const buildMonthKey = (year: number, month: number): string => {
  return `${year}-${String(month).padStart(2, '0')}`
}

const shiftMonth = (year: number, month: number, offset: number): PapMonthly => {
  const date = new Date(Date.UTC(year, month - 1 + offset, 1))
  return {
    year: date.getUTCFullYear(),
    month: date.getUTCMonth() + 1,
    total_pap: 0
  }
}

export const buildPapTrendSeries = (data: PapMonthly[]): PapMonthly[] => {
  if (data.length === 0) {
    return []
  }

  const sorted = [...data].sort((left, right) => {
    if (left.year !== right.year) {
      return left.year - right.year
    }
    return left.month - right.month
  })

  const latest = sorted[sorted.length - 1]
  const values = new Map<string, number>()

  for (const item of sorted) {
    values.set(buildMonthKey(item.year, item.month), item.total_pap)
  }

  const start = shiftMonth(latest.year, latest.month, -(PAP_TREND_MONTHS - 1))
  const series: PapMonthly[] = []

  for (let index = 0; index < PAP_TREND_MONTHS; index += 1) {
    const month = shiftMonth(start.year, start.month, index)
    series.push({
      year: month.year,
      month: month.month,
      total_pap: values.get(buildMonthKey(month.year, month.month)) ?? 0
    })
  }

  return series
}

export const formatPapTrendMonthLabel = (item: PapMonthly): string => {
  return `${item.year}-${String(item.month).padStart(2, '0')}`
}
