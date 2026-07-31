import { CartesianGrid, Line, LineChart, XAxis, YAxis } from 'recharts'
import {
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
  type ChartConfig,
} from '@/components/ui/chart'
import { useI18n } from '@/i18n'
import type { DashboardPapMonthly } from '@/types/api/dashboard'
import { buildPapTrendSeries } from './pap-trend'
import { cn } from '@/lib/utils'

interface DashboardPapChartProps {
  title: string
  data: DashboardPapMonthly[]
  className?: string
}

const chartConfig = {
  total_pap: {
    label: 'PAP',
    color: 'var(--primary)',
  },
} satisfies ChartConfig

export function DashboardPapChart({ title, data, className }: DashboardPapChartProps) {
  const { t } = useI18n()
  const series = buildPapTrendSeries(data)
  const chartData = series.map((item) => ({
    year: item.year,
    month: item.month,
    total_pap: item.total_pap,
    label: t('dashboardConsole.papChart.shortMonthLabel', {
      month: String(item.month).padStart(2, '0'),
    }),
  }))

  return (
    <section className={cn('flex h-128 flex-col rounded-lg border bg-card p-5', className)}>
      <header className="mb-2 flex items-center justify-between gap-2">
        <div>
          <h4 className="text-sm font-semibold">{title}</h4>
          <p className="text-xs text-muted-foreground">
            {t('dashboardConsole.papChart.recentMonths', { count: series.length })}
          </p>
        </div>
      </header>

      {series.length === 0 ? (
        <div className="flex flex-1 items-center justify-center text-sm text-muted-foreground">
          {t('dashboardConsole.papChart.empty')}
        </div>
      ) : (
        <div className="mt-2 flex-1">
          <ChartContainer config={chartConfig} className="h-full w-full">
            <LineChart data={chartData} margin={{ left: 8, right: 8, top: 8, bottom: 8 }}>
              <CartesianGrid vertical={false} strokeDasharray="3 3" />
              <XAxis
                dataKey="label"
                tickLine={false}
                axisLine={false}
                tickMargin={8}
                minTickGap={4}
              />
              <YAxis allowDecimals={false} tickLine={false} axisLine={false} width={28} />
              <ChartTooltip
                content={
                  <ChartTooltipContent
                    labelFormatter={(_, payload) => {
                      const entry = payload?.[0]?.payload as
                        | { year?: number; month?: number }
                        | undefined
                      if (!entry?.year || !entry?.month) {
                        return ''
                      }
                      return t('dashboardConsole.papChart.monthLabel', {
                        year: entry.year,
                        month: String(entry.month).padStart(2, '0'),
                      })
                    }}
                  />
                }
              />
              <Line
                dataKey="total_pap"
                type="monotone"
                stroke="var(--color-total_pap)"
                strokeWidth={2}
                dot={{ r: 3, fill: 'var(--color-total_pap)' }}
                activeDot={{ r: 4 }}
              />
            </LineChart>
          </ChartContainer>
        </div>
      )}
    </section>
  )
}
