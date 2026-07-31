import {
  Table,
  TableHeader,
  TableBody,
  TableRow,
  TableHead,
  TableCell,
} from '@/components/ui/table'
import { useI18n } from '@/i18n'
import { formatIskSmart } from '@/lib/isk'
import { formatTime } from '@/lib/format'
import type { DashboardSrpItem } from '@/types/api/dashboard'
import { cn } from '@/lib/utils'

interface DashboardSrpListProps {
  list: DashboardSrpItem[]
  className?: string
}

type StatusTone = 'primary' | 'success' | 'warning' | 'danger' | 'info'

const REVIEW_STATUS_TONE: Record<string, StatusTone> = {
  submitted: 'warning',
  approved: 'success',
  rejected: 'danger',
}

const PAYOUT_STATUS_TONE: Record<string, StatusTone> = {
  notpaid: 'info',
  paid: 'success',
}

const REVIEW_STATUS_KEYS = new Set(['submitted', 'approved', 'rejected'])
const PAYOUT_STATUS_KEYS = new Set(['notpaid', 'paid'])

function StatusBadge({ label, tone }: { label: string; tone: StatusTone }) {
  return (
    <span
      className={cn(
        'inline-flex items-center rounded border px-1.5 py-0.5 text-[10px] font-medium uppercase',
        toneClass(tone)
      )}
    >
      {label}
    </span>
  )
}

function toneClass(tone: StatusTone): string {
  switch (tone) {
    case 'primary':
      return 'border-primary/40 text-primary'
    case 'success':
      return 'border-success/40 text-success-foreground'
    case 'warning':
      return 'border-amber-500/40 text-amber-600 dark:text-amber-400'
    case 'danger':
      return 'border-destructive/40 text-destructive'
    case 'info':
    default:
      return 'border-muted-foreground/30 text-muted-foreground'
  }
}

export function DashboardSrpList({ list, className }: DashboardSrpListProps) {
  const { t } = useI18n()

  const resolveReviewLabel = (status: string): string =>
    REVIEW_STATUS_KEYS.has(status) ? t(`dashboardConsole.srpList.reviewStatus.${status}`) : status

  const resolvePayoutLabel = (status: string): string =>
    PAYOUT_STATUS_KEYS.has(status) ? t(`dashboardConsole.srpList.payoutStatus.${status}`) : status

  return (
    <section className={cn('rounded-lg border bg-card p-5', className)}>
      <header className="mb-4 flex items-center justify-between gap-2">
        <div>
          <h4 className="text-sm font-semibold">{t('dashboardConsole.srpList.title')}</h4>
          <p className="text-xs text-muted-foreground">
            {t('dashboardConsole.srpList.recentCount', { count: list.length })}
          </p>
        </div>
      </header>

      {list.length === 0 ? (
        <div className="flex h-30 items-center justify-center text-sm text-muted-foreground">
          {t('dashboardConsole.srpList.empty')}
        </div>
      ) : (
        <div className="overflow-x-auto">
          <Table className="w-full min-w-[860px] border-collapse text-sm">
            <TableHeader>
              <TableRow className="border-b border-border text-left text-xs text-muted-foreground">
                <TableHead className="px-3 py-2 font-medium">
                  {t('dashboardConsole.srpList.columns.character')}
                </TableHead>
                <TableHead className="px-3 py-2 font-medium">
                  {t('dashboardConsole.srpList.columns.ship')}
                </TableHead>
                <TableHead className="px-3 py-2 font-medium">
                  {t('dashboardConsole.srpList.columns.system')}
                </TableHead>
                <TableHead className="px-3 py-2 font-medium">
                  {t('dashboardConsole.srpList.lossTime')}
                </TableHead>
                <TableHead className="px-3 py-2 text-right font-medium">
                  {t('dashboardConsole.srpList.recommendedAmount')}
                </TableHead>
                <TableHead className="px-3 py-2 text-right font-medium">
                  {t('dashboardConsole.srpList.finalAmount')}
                </TableHead>
                <TableHead className="px-3 py-2 text-center font-medium">
                  {t('dashboardConsole.srpList.columns.reviewStatus')}
                </TableHead>
                <TableHead className="px-3 py-2 text-center font-medium">
                  {t('dashboardConsole.srpList.columns.payoutStatus')}
                </TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {list.map((row) => {
                const reviewTone = REVIEW_STATUS_TONE[row.review_status] ?? 'info'
                const payoutTone = PAYOUT_STATUS_TONE[row.payout_status] ?? 'info'
                return (
                  <TableRow key={row.id} className="border-b border-border last:border-b-0">
                    <TableCell className="px-3 py-2">{row.character_name}</TableCell>
                    <TableCell className="px-3 py-2">{row.ship_name}</TableCell>
                    <TableCell className="px-3 py-2">{row.solar_system_name}</TableCell>
                    <TableCell className="px-3 py-2">{formatTime(row.killmail_time)}</TableCell>
                    <TableCell className="px-3 py-2 text-right text-muted-foreground">
                      {formatIskSmart(row.recommended_amount)}
                    </TableCell>
                    <TableCell className="px-3 py-2 text-right font-medium">
                      {formatIskSmart(row.final_amount)}
                    </TableCell>
                    <TableCell className="px-3 py-2 text-center">
                      <StatusBadge
                        label={resolveReviewLabel(row.review_status)}
                        tone={reviewTone}
                      />
                    </TableCell>
                    <TableCell className="px-3 py-2 text-center">
                      <StatusBadge
                        label={resolvePayoutLabel(row.payout_status)}
                        tone={payoutTone}
                      />
                    </TableCell>
                  </TableRow>
                )
              })}
            </TableBody>
          </Table>
        </div>
      )}
    </section>
  )
}
