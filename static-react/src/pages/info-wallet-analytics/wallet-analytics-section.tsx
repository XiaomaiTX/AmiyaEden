import { useEffect, useMemo, useState } from 'react'
import { Bar, BarChart, CartesianGrid, Line, LineChart, XAxis, YAxis } from 'recharts'
import { fetchInfoWalletAnalytics } from '@/api/eve-info'
import { Button } from '@/components/ui/button'
import {
  ChartContainer,
  ChartLegend,
  ChartLegendContent,
  ChartTooltip,
  ChartTooltipContent,
  type ChartConfig,
} from '@/components/ui/chart'
import { Input } from '@/components/ui/input'
import { useI18n } from '@/i18n'
import { formatIskPlain, formatIskSmart } from '@/lib/isk'
import { cn } from '@/lib/utils'
import type { WalletAnalyticsResponse } from '@/types/api/eve-info'
import {
  WALLET_ANALYTICS_PRESETS,
  buildBalanceSeries,
  buildDailyFlowSeries,
  buildRefTypeSeries,
  resolveAnalyticsRange,
  resolveFlowAxisBound,
  type WalletAnalyticsPreset,
  type WalletAnalyticsRange,
} from './wallet-analytics-series'

/** 预设按钮文案 key（显式映射，避免把 '7d' 这类以数字开头的值拼进 i18n key） */
const PRESET_LABEL_KEYS: Record<WalletAnalyticsPreset, string> = {
  '1d': 'infoWallet.analytics.preset1d',
  '7d': 'infoWallet.analytics.preset7d',
  '30d': 'infoWallet.analytics.preset30d',
  '90d': 'infoWallet.analytics.preset90d',
  all: 'infoWallet.analytics.presetAll',
}

/**
 * 每日收支配色：收入绿、支出红
 *
 * 取值为固定十六进制色（同 Tailwind green-600 / red-500），与 Vue 端的每日收支图保持一致，
 * 明暗两种主题下均可读。
 */
const DAILY_FLOW_COLORS = {
  income: '#16a34a',
  expense: '#ef4444',
} as const

function getErrorMessage(error: unknown, fallback: string) {
  return error instanceof Error && error.message ? error.message : fallback
}

interface WalletAnalyticsSectionProps {
  /** 当前人物 ID，由页面的人物切换器提供 */
  characterId: number | null
}

export function WalletAnalyticsSection({ characterId }: WalletAnalyticsSectionProps) {
  const { t } = useI18n()
  const [loading, setLoading] = useState(Boolean(characterId))
  const [error, setError] = useState<string | null>(null)
  const [analytics, setAnalytics] = useState<WalletAnalyticsResponse | null>(null)
  /** 已就位数据所属的人物，用于切换人物时避免展示上一个人物的图表 */
  const [loadedCharacterId, setLoadedCharacterId] = useState<number | null>(null)
  /** null 表示用户手动选定的自定义区间，此时不高亮任何预设 */
  const [activePreset, setActivePreset] = useState<WalletAnalyticsPreset | null>('all')
  const [dateRange, setDateRange] = useState<WalletAnalyticsRange | null>(null)

  const summary = analytics?.summary ?? null
  const availableFrom = summary?.available_from ?? ''
  const availableTo = summary?.available_to ?? ''
  const hasAvailableRange = Boolean(availableFrom) && Boolean(availableTo)
  const hasCurrentAnalytics = Boolean(analytics) && loadedCharacterId === characterId

  useEffect(() => {
    let cancelled = false

    const load = async () => {
      if (!characterId) return

      setLoading(true)
      setError(null)

      try {
        const data = await fetchInfoWalletAnalytics({ character_id: characterId })
        if (cancelled) return
        setAnalytics(data)
        setLoadedCharacterId(characterId)
        setActivePreset('all')
        setDateRange(
          resolveAnalyticsRange('all', data.summary.available_from, data.summary.available_to)
        )
      } catch (loadError) {
        if (cancelled) return
        setAnalytics(null)
        setLoadedCharacterId(characterId)
        setDateRange(null)
        setError(getErrorMessage(loadError, t('infoWallet.analytics.loadFailed')))
      } finally {
        if (!cancelled) setLoading(false)
      }
    }

    void load()
    return () => {
      cancelled = true
    }
  }, [characterId, t])

  const applyRange = async (range: WalletAnalyticsRange) => {
    if (!characterId) return
    setLoading(true)
    setError(null)
    try {
      const data = await fetchInfoWalletAnalytics({
        character_id: characterId,
        from: range.from,
        to: range.to,
      })
      setAnalytics(data)
      setLoadedCharacterId(characterId)
    } catch (loadError) {
      setAnalytics(null)
      setDateRange(null)
      setError(getErrorMessage(loadError, t('infoWallet.analytics.loadFailed')))
    } finally {
      setLoading(false)
    }
  }

  const clampToAvailable = (from: string, to: string): WalletAnalyticsRange | null => {
    if (!availableFrom || !availableTo) return null
    const start = from < availableFrom ? availableFrom : from
    const end = to > availableTo ? availableTo : to
    return end < start ? null : { from: start, to: end }
  }

  const handlePresetChange = (preset: WalletAnalyticsPreset) => {
    const range = resolveAnalyticsRange(preset, availableFrom, availableTo)
    if (!range) return
    setActivePreset(preset)
    setDateRange(range)
    void applyRange(range)
  }

  const handleDateChange = (edge: 'from' | 'to', value: string) => {
    if (!value) return
    const current = dateRange ?? resolveAnalyticsRange('all', availableFrom, availableTo)
    if (!current) return
    const next = clampToAvailable(
      edge === 'from' ? value : current.from,
      edge === 'to' ? value : current.to
    )
    if (!next) return
    setActivePreset(null)
    setDateRange(next)
    void applyRange(next)
  }

  const balanceValues = useMemo(
    () => buildBalanceSeries(analytics?.daily_series ?? []),
    [analytics?.daily_series]
  )
  const flow = useMemo(
    () => buildDailyFlowSeries(analytics?.daily_series ?? []),
    [analytics?.daily_series]
  )
  const axisBound = resolveFlowAxisBound(flow.income, flow.expense)

  const balanceChartData = flow.labels.map((label, index) => ({
    label,
    balance: balanceValues[index] ?? 0,
  }))
  const flowChartData = flow.labels.map((label, index) => ({
    label,
    income: flow.income[index] ?? 0,
    expense: -(flow.expense[index] ?? 0),
  }))
  const incomeRefSeries = buildRefTypeSeries(analytics?.ref_type_breakdown ?? [], 'income')
  const expenseRefSeries = buildRefTypeSeries(analytics?.ref_type_breakdown ?? [], 'expense')
  const incomeRefData = incomeRefSeries.refTypes.map((refType, index) => ({
    label: refType,
    value: incomeRefSeries.values[index] ?? 0,
  }))
  const expenseRefData = expenseRefSeries.refTypes.map((refType, index) => ({
    label: refType,
    value: expenseRefSeries.values[index] ?? 0,
  }))

  const balanceChartConfig = {
    balance: { label: t('infoWallet.analytics.balanceTrend'), color: 'var(--primary)' },
  } satisfies ChartConfig
  const flowChartConfig = {
    income: { label: t('infoWallet.analytics.income'), color: DAILY_FLOW_COLORS.income },
    expense: { label: t('infoWallet.analytics.expense'), color: DAILY_FLOW_COLORS.expense },
  } satisfies ChartConfig
  const refTypeChartConfig = {
    value: { label: t('infoWallet.analytics.amount'), color: 'var(--primary)' },
  } satisfies ChartConfig

  const stats = summary
    ? [
        {
          key: 'income',
          label: t('infoWallet.analytics.totalIncome'),
          value: formatIskSmart(summary.total_income),
          detail: formatIskPlain(summary.total_income),
          className: 'text-primary',
        },
        {
          key: 'expense',
          label: t('infoWallet.analytics.totalExpense'),
          value: formatIskSmart(summary.total_expense),
          detail: formatIskPlain(summary.total_expense),
          className: 'text-destructive',
        },
        {
          key: 'net',
          label: t('infoWallet.analytics.totalNet'),
          value: formatIskSmart(summary.total_net),
          detail: formatIskPlain(summary.total_net),
          className: summary.total_net >= 0 ? 'text-primary' : 'text-destructive',
        },
        {
          key: 'tax',
          label: t('infoWallet.analytics.totalTax'),
          value: formatIskSmart(summary.total_tax),
          detail: formatIskPlain(summary.total_tax),
          className: 'text-foreground',
        },
        {
          key: 'opening',
          label: t('infoWallet.analytics.openingBalance'),
          value: formatIskSmart(summary.opening_balance),
          detail: formatIskPlain(summary.opening_balance),
          className: 'text-foreground',
        },
        {
          key: 'closing',
          label: t('infoWallet.analytics.closingBalance'),
          value: formatIskSmart(summary.closing_balance),
          detail: formatIskPlain(summary.closing_balance),
          className: 'text-foreground',
        },
      ]
    : []

  const renderIskTooltip = (value: unknown, name: unknown) => (
    <>
      <span className="text-muted-foreground">{String(name)}</span>
      <span className="font-mono font-medium tabular-nums">{formatIskSmart(Number(value))}</span>
    </>
  )

  return (
    <section className="space-y-4 rounded-lg border bg-card p-4">
      <header className="flex flex-wrap items-start justify-between gap-3">
        <div className="flex flex-wrap items-center gap-2">
          <span className="text-sm font-semibold">{t('infoWallet.analytics.title')}</span>
          {WALLET_ANALYTICS_PRESETS.map((preset) => (
            <Button
              key={preset}
              size="sm"
              variant={activePreset === preset ? 'default' : 'outline'}
              isDisabled={!hasAvailableRange}
              onClick={() => handlePresetChange(preset)}
            >
              {t(PRESET_LABEL_KEYS[preset])}
            </Button>
          ))}
          <Input
            type="date"
            className="h-8 w-36"
            aria-label={t('infoWallet.analytics.rangeStart')}
            min={availableFrom || undefined}
            max={availableTo || undefined}
            value={dateRange?.from ?? ''}
            onChange={(event) => handleDateChange('from', event.target.value)}
          />
          <span className="text-xs text-muted-foreground">~</span>
          <Input
            type="date"
            className="h-8 w-36"
            aria-label={t('infoWallet.analytics.rangeEnd')}
            min={availableFrom || undefined}
            max={availableTo || undefined}
            value={dateRange?.to ?? ''}
            onChange={(event) => handleDateChange('to', event.target.value)}
          />
        </div>
        <div className="text-right text-xs leading-5 text-muted-foreground">
          {dateRange ? (
            <p>
              {t('infoWallet.analytics.rangeSummary', {
                from: dateRange.from,
                to: dateRange.to,
                count: summary?.entry_count ?? 0,
              })}
            </p>
          ) : null}
          <p>{t('infoWallet.analytics.utcNote')}</p>
        </div>
      </header>

      {!characterId ? (
        <p className="py-12 text-center text-sm text-muted-foreground">
          {t('infoWallet.analytics.selectCharacter')}
        </p>
      ) : error ? (
        <p className="py-12 text-center text-sm text-destructive">{error}</p>
      ) : !hasCurrentAnalytics ? (
        <p className="py-12 text-center text-sm text-muted-foreground">{t('infoWallet.loading')}</p>
      ) : !hasAvailableRange ? (
        <p className="py-12 text-center text-sm text-muted-foreground">
          {t('infoWallet.analytics.noData')}
        </p>
      ) : (
        <div className={cn('space-y-4', loading && 'opacity-60')} aria-busy={loading}>
          <div className="grid grid-cols-2 gap-3 lg:grid-cols-3 xl:grid-cols-6">
            {stats.map((stat) => (
              <div key={stat.key} className="rounded-lg border px-3 py-2">
                <p className="text-xs text-muted-foreground">{stat.label}</p>
                <p className={cn('mt-1 text-base font-medium', stat.className)}>{stat.value}</p>
                <p className="mt-0.5 text-xs break-all text-muted-foreground">{stat.detail}</p>
              </div>
            ))}
          </div>

          <div className="grid grid-cols-1 gap-4 xl:grid-cols-2">
            <div>
              <p className="mb-1 text-sm text-muted-foreground">
                {t('infoWallet.analytics.balanceTrend')}
              </p>
              <div className="h-56">
                <ChartContainer config={balanceChartConfig} className="h-full w-full">
                  <LineChart
                    data={balanceChartData}
                    margin={{ left: 8, right: 8, top: 8, bottom: 8 }}
                  >
                    <CartesianGrid vertical={false} strokeDasharray="3 3" />
                    <XAxis dataKey="label" tickLine={false} axisLine={false} minTickGap={8} />
                    <YAxis
                      tickLine={false}
                      axisLine={false}
                      width={64}
                      tickFormatter={(value: string | number) => formatIskSmart(Number(value))}
                    />
                    <ChartTooltip content={<ChartTooltipContent formatter={renderIskTooltip} />} />
                    <Line
                      dataKey="balance"
                      type="monotone"
                      stroke="var(--color-balance)"
                      strokeWidth={2}
                      dot={false}
                    />
                  </LineChart>
                </ChartContainer>
              </div>
            </div>

            <div>
              <p className="mb-1 text-sm text-muted-foreground">
                {t('infoWallet.analytics.dailyFlow')}
              </p>
              <div className="h-56">
                <ChartContainer config={flowChartConfig} className="h-full w-full">
                  <BarChart
                    data={flowChartData}
                    stackOffset="sign"
                    margin={{ left: 8, right: 8, top: 8, bottom: 8 }}
                  >
                    <CartesianGrid vertical={false} strokeDasharray="3 3" />
                    <XAxis dataKey="label" tickLine={false} axisLine={false} minTickGap={8} />
                    <YAxis
                      tickLine={false}
                      axisLine={false}
                      width={64}
                      domain={[-axisBound, axisBound]}
                      tickFormatter={(value: string | number) => formatIskSmart(Number(value))}
                    />
                    <ChartTooltip content={<ChartTooltipContent formatter={renderIskTooltip} />} />
                    <ChartLegend content={<ChartLegendContent />} />
                    <Bar
                      dataKey="income"
                      stackId="flow"
                      fill="var(--color-income)"
                      maxBarSize={14}
                    />
                    <Bar
                      dataKey="expense"
                      stackId="flow"
                      fill="var(--color-expense)"
                      maxBarSize={14}
                    />
                  </BarChart>
                </ChartContainer>
              </div>
            </div>

            <div>
              <p className="mb-1 text-sm text-muted-foreground">
                {t('infoWallet.analytics.incomeByType')}
              </p>
              <div className="h-56">
                <ChartContainer config={refTypeChartConfig} className="h-full w-full">
                  <BarChart
                    data={incomeRefData}
                    layout="vertical"
                    margin={{ left: 8, right: 16, top: 8, bottom: 8 }}
                  >
                    <XAxis type="number" hide />
                    <YAxis
                      type="category"
                      dataKey="label"
                      width={150}
                      tickLine={false}
                      axisLine={false}
                    />
                    <ChartTooltip content={<ChartTooltipContent formatter={renderIskTooltip} />} />
                    <Bar dataKey="value" fill="var(--color-value)" radius={4} maxBarSize={14} />
                  </BarChart>
                </ChartContainer>
              </div>
            </div>

            <div>
              <p className="mb-1 text-sm text-muted-foreground">
                {t('infoWallet.analytics.expenseByType')}
              </p>
              <div className="h-56">
                <ChartContainer config={refTypeChartConfig} className="h-full w-full">
                  <BarChart
                    data={expenseRefData}
                    layout="vertical"
                    margin={{ left: 8, right: 16, top: 8, bottom: 8 }}
                  >
                    <XAxis type="number" hide />
                    <YAxis
                      type="category"
                      dataKey="label"
                      width={150}
                      tickLine={false}
                      axisLine={false}
                    />
                    <ChartTooltip content={<ChartTooltipContent formatter={renderIskTooltip} />} />
                    <Bar dataKey="value" fill="var(--color-value)" radius={4} maxBarSize={14} />
                  </BarChart>
                </ChartContainer>
              </div>
            </div>
          </div>
        </div>
      )}
    </section>
  )
}
