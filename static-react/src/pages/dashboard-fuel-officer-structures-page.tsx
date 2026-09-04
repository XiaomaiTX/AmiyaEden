import { useCallback, useEffect, useMemo, useState } from 'react'
import { fetchMyAssignedCorporationStructures } from '@/api/corporation-structures'
import { DataTable, type ColumnDef } from '@/components/ui/data-table'
import { useI18n } from '@/i18n'
import { formatTime, fuelExpiryMonthOffset } from '@/lib/format'
import type { CorporationStructureRow } from '@/types/api/dashboard'

export function DashboardFuelOfficerStructuresPage() {
  const { t } = useI18n()
  const [rows, setRows] = useState<CorporationStructureRow[]>([])
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const formatEstimate = useCallback(
    (row: CorporationStructureRow, field: 'fuel_per_hour' | 'fuel_to_month_end') => {
      if (!row.fuel_estimate_incomplete) {
        const value = row[field]
        if (value == null) {
          return '-'
        }
        // +N 徽标表示目标月底（fuel_expires 所在月）距当前月还有几个整月
        if (field === 'fuel_to_month_end' && row.fuel_expires) {
          const monthOffset = fuelExpiryMonthOffset(row.fuel_expires)
          if (monthOffset != null && monthOffset >= 1) {
            return (
              <>
                {value}
                <span className="ml-1.5 inline-flex items-center rounded-full bg-muted px-1.5 py-0.5 text-[10px] font-medium text-muted-foreground">
                  +{monthOffset}
                </span>
              </>
            )
          }
        }
        return value
      }
      const keys: Record<string, string> = {
        authorization_required: 'fuelEstimateAuthorizationRequired',
        activity_mapping_required: 'fuelEstimateActivityMappingRequired',
        module_mismatch: 'fuelEstimateModuleMismatch',
        rate_unavailable: 'fuelEstimateRateUnavailable',
        ambiguous_module: 'fuelEstimateAmbiguousModule',
      }
      return t(
        `corporationStructures.table.${keys[row.fuel_estimate_status ?? ''] ?? 'fuelEstimateIncomplete'}`
      )
    },
    [t]
  )

  const columns = useMemo<ColumnDef<CorporationStructureRow, unknown>[]>(
    () => [
      { accessorKey: 'corporation_name', header: t('corporationStructures.table.corporation') },
      { accessorKey: 'name', header: t('corporationStructures.table.name') },
      { accessorKey: 'system_name', header: t('corporationStructures.table.system') },
      { accessorKey: 'state', header: t('corporationStructures.table.state') },
      { accessorKey: 'fuel_remaining', header: t('corporationStructures.table.fuelRemaining') },
      {
        id: 'fuel_per_hour',
        header: t('corporationStructures.table.fuelPerHour'),
        cell: ({ row }) => formatEstimate(row.original, 'fuel_per_hour'),
      },
      {
        id: 'fuel_to_month_end',
        header: t('corporationStructures.table.fuelToMonthEnd'),
        cell: ({ row }) => formatEstimate(row.original, 'fuel_to_month_end'),
      },
      { accessorKey: 'state_timer_end', header: t('corporationStructures.table.timerEnd') },
      {
        id: 'updated_at',
        header: t('corporationStructures.table.updatedAt'),
        cell: ({ row }) =>
          row.original.updated_at
            ? formatTime(new Date(row.original.updated_at * 1000).toISOString())
            : '-',
      },
    ],
    [formatEstimate, t]
  )

  useEffect(() => {
    let cancelled = false
    const timer = window.setTimeout(() => {
      setLoading(true)
      setError(null)
      void fetchMyAssignedCorporationStructures({ page, page_size: pageSize })
        .then((response) => {
          if (!cancelled) {
            setRows(response.items ?? [])
            setTotal(response.total ?? 0)
          }
        })
        .catch((cause: unknown) => {
          if (!cancelled) {
            setError(cause instanceof Error ? cause.message : t('common.fetchFail'))
          }
        })
        .finally(() => {
          if (!cancelled) {
            setLoading(false)
          }
        })
    }, 0)
    return () => {
      cancelled = true
      window.clearTimeout(timer)
    }
  }, [page, pageSize, t])

  return (
    <section className="space-y-4">
      <div>
        <h1 className="text-xl font-semibold">{t('nav.dashboard.fuelOfficerStructures')}</h1>
      </div>
      <DataTable
        columns={columns}
        data={rows}
        getRowId={(row) => String(row.structure_id)}
        loading={loading}
        error={error}
        loadingText={t('common.loading')}
        emptyText={t('common.empty')}
        pagination={{
          page,
          pageSize,
          total,
          onPageChange: setPage,
          onPageSizeChange: (size) => {
            setPageSize(size)
            setPage(1)
          },
          previousLabel: t('common.previous'),
          nextLabel: t('common.next'),
          pageSizeLabel: t('common.pageSize'),
        }}
      />
    </section>
  )
}
