import { useCallback, useEffect, useMemo, useState } from 'react'
import {
  createGalaxyRegistryEntry,
  endGalaxyRegistryEntry,
  fetchAdminGalaxyRegistryEntries,
  fetchGalaxyRegistrySystems,
  forceEndAdminGalaxyRegistryEntry,
  revalidateAdminGalaxyRegistryEntry,
} from '@/api/galaxy-registry'
import { Button } from '@/components/ui/button'
import { DataTable, type ColumnDef } from '@/components/ui/data-table'
import { Input } from '@/components/ui/input'
import { notifyError, notifySuccess } from '@/feedback'
import { useI18n } from '@/i18n'
import { formatTime } from '@/lib/format'
import { useSessionStore } from '@/stores'
import type {
  GalaxyRegistryEntryItem,
  GalaxyRegistrySystemItem,
  GalaxyRegistrySystemsResponse,
} from '@/types/api/galaxy-registry'

function defaultExpectedEnd() {
  const date = new Date(Date.now() + 60 * 60 * 1000)
  const offset = date.getTimezoneOffset() * 60_000
  return new Date(date.getTime() - offset).toISOString().slice(0, 16)
}

export function DashboardGalaxyRegistryPage() {
  const { t } = useI18n()
  const roles = useSessionStore((state) => state.roles)
  const canCaptain = roles.includes('captain') || roles.includes('super_admin')
  const canAdmin = roles.includes('admin') || roles.includes('super_admin')
  const [systems, setSystems] = useState<GalaxyRegistrySystemsResponse | null>(null)
  const [entries, setEntries] = useState<GalaxyRegistryEntryItem[]>([])
  const [entryTotal, setEntryTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)
  const [expectedEnd, setExpectedEnd] = useState(defaultExpectedEnd)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const [systemResponse, entryResponse] = await Promise.all([
        fetchGalaxyRegistrySystems(),
        canAdmin
          ? fetchAdminGalaxyRegistryEntries({ current: page, size: pageSize })
          : Promise.resolve({ list: [], total: 0, page: 1, pageSize }),
      ])
      setSystems(systemResponse)
      setEntries(entryResponse.list ?? [])
      setEntryTotal(entryResponse.total ?? 0)
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : t('common.fetchFail'))
    } finally {
      setLoading(false)
    }
  }, [canAdmin, page, pageSize, t])

  useEffect(() => {
    const timer = window.setTimeout(() => void load(), 0)
    return () => window.clearTimeout(timer)
  }, [load])

  const act = useCallback(
    async (action: () => Promise<unknown>) => {
      try {
        await action()
        notifySuccess(t('galaxyRegistry.actionSuccess'))
        await load()
      } catch (cause) {
        notifyError(cause instanceof Error ? cause.message : t('galaxyRegistry.actionFailed'))
      }
    },
    [load, t]
  )

  const systemColumns = useMemo<ColumnDef<GalaxyRegistrySystemItem, unknown>[]>(
    () => [
      { accessorKey: 'solar_system_name', header: t('galaxyRegistry.system') },
      { accessorKey: 'region_name', header: t('galaxyRegistry.region') },
      { accessorKey: 'security', header: t('galaxyRegistry.security') },
      { accessorKey: 'status', header: t('common.status') },
      {
        id: 'captain',
        header: t('galaxyRegistry.captain'),
        cell: ({ row }) => row.original.active_entry?.captain_character_name ?? '-',
      },
      {
        id: 'expected',
        header: t('galaxyRegistry.expectedEnd'),
        cell: ({ row }) => formatTime(row.original.active_entry?.expected_end_at),
      },
      {
        id: 'actions',
        header: t('common.operation'),
        cell: ({ row }) => {
          const item = row.original
          if (!canCaptain) return null
          if (item.active_entry?.is_mine) {
            return (
              <Button size="sm" variant="outline" onClick={() => void act(() => endGalaxyRegistryEntry(item.active_entry!.entry_id))}>
                {t('galaxyRegistry.end')}
              </Button>
            )
          }
          if (item.status === 'idle' || item.status === 'overdue') {
            return (
              <Button
                size="sm"
                onClick={() =>
                  void act(() =>
                    createGalaxyRegistryEntry({
                      system_config_id: item.system_config_id,
                      expected_end_at: new Date(expectedEnd).toISOString(),
                    })
                  )
                }
              >
                {t('galaxyRegistry.register')}
              </Button>
            )
          }
          return null
        },
      },
    ],
    [act, canCaptain, expectedEnd, t]
  )

  const entryColumns = useMemo<ColumnDef<GalaxyRegistryEntryItem, unknown>[]>(
    () => [
      { accessorKey: 'solar_system_name', header: t('galaxyRegistry.system') },
      { accessorKey: 'captain_character_name', header: t('galaxyRegistry.captain') },
      { accessorKey: 'status', header: t('common.status') },
      { accessorKey: 'validation_status', header: t('galaxyRegistry.validation') },
      { id: 'started', header: t('galaxyRegistry.startedAt'), cell: ({ row }) => formatTime(row.original.actual_start_at) },
      {
        id: 'adminActions',
        header: t('common.operation'),
        cell: ({ row }) => (
          <div className="flex gap-2">
            {row.original.status === 'active' ? (
              <Button size="sm" variant="outline" onClick={() => void act(() => forceEndAdminGalaxyRegistryEntry(row.original.id))}>
                {t('galaxyRegistry.forceEnd')}
              </Button>
            ) : null}
            <Button size="sm" variant="outline" onClick={() => void act(() => revalidateAdminGalaxyRegistryEntry(row.original.id))}>
              {t('galaxyRegistry.revalidate')}
            </Button>
          </div>
        ),
      },
    ],
    [act, t]
  )

  return (
    <section className="space-y-5">
      <div>
        <h1 className="text-xl font-semibold">{t('galaxyRegistry.title')}</h1>
        <p className="text-sm text-muted-foreground">{t('galaxyRegistry.subtitle')}</p>
      </div>
      {systems ? (
        <div className="grid gap-3 sm:grid-cols-3">
          <div className="rounded-lg border p-4">{t('galaxyRegistry.idle')}: {systems.summary.idle_count}</div>
          <div className="rounded-lg border p-4">{t('galaxyRegistry.busy')}: {systems.summary.busy_count}</div>
          <div className="rounded-lg border p-4">{t('galaxyRegistry.overdue')}: {systems.summary.overdue_count}</div>
        </div>
      ) : null}
      {canCaptain ? (
        <label className="block max-w-sm space-y-2">
          <span className="text-sm font-medium">{t('galaxyRegistry.expectedEnd')}</span>
          <Input type="datetime-local" value={expectedEnd} onChange={(event) => setExpectedEnd(event.target.value)} />
        </label>
      ) : null}
      <DataTable
        columns={systemColumns}
        data={systems?.items ?? []}
        getRowId={(row) => String(row.system_config_id)}
        loading={loading}
        error={error}
        loadingText={t('common.loading')}
        emptyText={t('common.empty')}
      />
      {canAdmin ? (
        <div className="space-y-3">
          <h2 className="text-lg font-semibold">{t('galaxyRegistry.adminEntries')}</h2>
          <DataTable
            columns={entryColumns}
            data={entries}
            getRowId={(row) => String(row.id)}
            loading={loading}
            error={error}
            loadingText={t('common.loading')}
            emptyText={t('common.empty')}
            pagination={{
              page,
              pageSize,
              total: entryTotal,
              onPageChange: setPage,
              onPageSizeChange: (size) => { setPageSize(size); setPage(1) },
              previousLabel: t('common.previous'),
              nextLabel: t('common.next'),
              pageSizeLabel: t('common.pageSize'),
            }}
          />
        </div>
      ) : null}
    </section>
  )
}
