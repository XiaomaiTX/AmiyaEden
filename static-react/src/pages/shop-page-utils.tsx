/* eslint-disable react-refresh/only-export-components */
import type { ReactNode } from 'react'
import { cn } from '@/lib/utils'

export function getErrorMessage(error: unknown, fallback: string) {
  return error instanceof Error && error.message ? error.message : fallback
}

export function formatDateTime(value: string | null | undefined) {
  if (!value) {
    return '-'
  }

  const parsed = new Date(value)
  return Number.isNaN(parsed.getTime()) ? value : parsed.toLocaleString()
}

const coinFormatter = new Intl.NumberFormat('en-US')

export function formatCoin(value: number | null | undefined) {
  if (value === null || value === undefined) {
    return '-'
  }

  return coinFormatter.format(value)
}

export function formatSignedCoin(value: number | null | undefined) {
  if (value === null || value === undefined) {
    return '-'
  }

  const prefix = value > 0 ? '+' : ''
  return `${prefix}${formatCoin(value)}`
}

export function formatContact(
  t: (key: string, vars?: Record<string, string | number>) => string,
  qq: string,
  discordId: string
) {
  if (qq) return `${t('common.qq')}: ${qq}`
  if (discordId) return `${t('common.discord')}: ${discordId}`
  return '-'
}

export function getLimitPeriodLabel(
  t: (key: string, vars?: Record<string, string | number>) => string,
  value: 'forever' | 'daily' | 'weekly' | 'monthly'
) {
  const key = `shop.period.${value}`
  const translated = t(key)
  return translated === key ? value : translated
}

export function productStatusClass(status: number) {
  return status === 1
    ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-300'
    : 'bg-rose-100 text-rose-700 dark:bg-rose-500/10 dark:text-rose-300'
}

export function orderStatusClass(status: string) {
  switch (status) {
    case 'requested':
      return 'bg-amber-100 text-amber-700 dark:bg-amber-500/10 dark:text-amber-300'
    case 'delivered':
      return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-300'
    case 'rejected':
      return 'bg-rose-100 text-rose-700 dark:bg-rose-500/10 dark:text-rose-300'
    default:
      return 'bg-muted text-muted-foreground'
  }
}

export function refTypeClass(value: string) {
  switch (value) {
    case 'shop_purchase':
      return 'bg-sky-100 text-sky-700 dark:bg-sky-500/10 dark:text-sky-300'
    case 'shop_refund':
      return 'bg-amber-100 text-amber-700 dark:bg-amber-500/10 dark:text-amber-300'
    default:
      return 'bg-muted text-muted-foreground'
  }
}

export function ShopBadge({
  className,
  children,
}: {
  className?: string
  children: ReactNode
}) {
  return (
    <span
      className={cn(
        'inline-flex rounded-full px-2 py-0.5 text-xs font-medium leading-5',
        className
      )}
    >
      {children}
    </span>
  )
}

export function ShopDialog({
  open,
  title,
  description,
  children,
  footer,
  onClose,
  closeLabel,
  widthClass = 'max-w-lg',
}: {
  open: boolean
  title: string
  description?: string
  children: ReactNode
  footer?: ReactNode
  onClose: () => void
  closeLabel: string
  widthClass?: string
}) {
  if (!open) {
    return null
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
      <div
        aria-label={title}
        aria-modal="true"
        className={cn('w-full rounded-lg border bg-card p-5 shadow-xl', widthClass)}
        role="dialog"
      >
        <div className="flex items-start justify-between gap-4">
          <div>
            <h2 className="text-lg font-semibold">{title}</h2>
            {description ? <p className="mt-1 text-sm text-muted-foreground">{description}</p> : null}
          </div>
          <button
            type="button"
            className="rounded-md px-2 py-1 text-sm text-muted-foreground hover:bg-muted"
            aria-label={closeLabel}
            onClick={onClose}
          >
            ×
          </button>
        </div>
        <div className="mt-4 space-y-4">{children}</div>
        {footer ? <div className="mt-5 flex justify-end gap-3">{footer}</div> : null}
      </div>
    </div>
  )
}
