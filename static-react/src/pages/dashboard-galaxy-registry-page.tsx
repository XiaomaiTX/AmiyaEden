import { useCallback, useEffect, useMemo, useState } from 'react'
import {
  createAdminGalaxyRegistrySystem, createGalaxyRegistryEntry, deleteAdminGalaxyRegistrySystem,
  endGalaxyRegistryEntry, fetchAdminGalaxyRegistryAnalytics, fetchAdminGalaxyRegistryEntries,
  fetchAdminGalaxyRegistrySystems, fetchGalaxyRegistrySystems, fetchMyGalaxyRegistryEntries,
  forceEndAdminGalaxyRegistryEntry, revalidateAdminGalaxyRegistryEntry, searchGalaxyRegistrySdeSystems,
  updateAdminGalaxyRegistryEntryValidation, updateAdminGalaxyRegistrySystem, updateGalaxyRegistryEntryExpectedEndAt,
} from '@/api/galaxy-registry'
import { Button } from '@/components/ui/button'
import { DataTable, type ColumnDef } from '@/components/ui/data-table'
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'
import { Switch } from '@/components/ui/switch'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { confirmAction, notifyError, notifySuccess, notifyWarning } from '@/feedback'
import { useI18n } from '@/i18n'
import { formatTime } from '@/lib/format'
import { useSessionStore } from '@/stores'
import type {
  GalaxyRegistryAdminAnalytics, GalaxyRegistryAdminSystem, GalaxyRegistryEntryItem,
  GalaxyRegistryEntryListParams, GalaxyRegistryEntryPage, GalaxyRegistrySdeSystem,
  GalaxyRegistrySystemItem, GalaxyRegistrySystemsResponse,
} from '@/types/api/galaxy-registry'

type AdminDraft = GalaxyRegistryAdminSystem & { localId: string; isNew: boolean; dirty: boolean }
type EntryDialog = { id: number; expectedEnd: string; actualStartAt: string | null }
type EntryFilters = Pick<GalaxyRegistryEntryListParams, 'keyword' | 'status' | 'validation_status'> & { page: number; pageSize: number }
const emptyPage: GalaxyRegistryEntryPage = { list: [], total: 0, page: 1, pageSize: 20 }
const defaultFilters: EntryFilters = { page: 1, pageSize: 20, keyword: '', status: '', validation_status: '' }

function toDateInput(value: Date) {
  const offset = value.getTimezoneOffset() * 60_000
  return new Date(value.getTime() - offset).toISOString().slice(0, 16)
}

function isAllowedEnd(value: string, start: Date) {
  const end = new Date(value)
  return !Number.isNaN(end.valueOf()) && end > start && end <= new Date(start.getTime() + 2 * 60 * 60 * 1000)
}

export function DashboardGalaxyRegistryPage() {
  const { t } = useI18n()
  const roles = useSessionStore((state) => state.roles)
  const canCaptain = roles.includes('captain') || roles.includes('super_admin')
  const canAdmin = roles.includes('admin') || roles.includes('super_admin')
  const [tab, setTab] = useState('current')
  const [systems, setSystems] = useState<GalaxyRegistrySystemsResponse | null>(null)
  const [myPage, setMyPage] = useState<GalaxyRegistryEntryPage>(emptyPage)
  const [adminPage, setAdminPage] = useState<GalaxyRegistryEntryPage>(emptyPage)
  const [myFilters, setMyFilters] = useState<EntryFilters>(defaultFilters)
  const [adminFilters, setAdminFilters] = useState<EntryFilters>(defaultFilters)
  const [adminSystems, setAdminSystems] = useState<AdminDraft[]>([])
  const [sdeResults, setSdeResults] = useState<GalaxyRegistrySdeSystem[]>([])
  const [sdeKeyword, setSdeKeyword] = useState('')
  const [analytics, setAnalytics] = useState<GalaxyRegistryAdminAnalytics | null>(null)
  const [analyticsRange, setAnalyticsRange] = useState({ start: '', end: '' })
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [entryDialog, setEntryDialog] = useState<EntryDialog | null>(null)
  const [validationDialog, setValidationDialog] = useState<GalaxyRegistryEntryItem | null>(null)
  const [validation, setValidation] = useState<'valid' | 'violation'>('valid')
  const [violationReason, setViolationReason] = useState('')

  const loadSystems = useCallback(async () => { setSystems(await fetchGalaxyRegistrySystems()) }, [])
  const loadCaptain = useCallback(async () => {
    setMyPage(await fetchMyGalaxyRegistryEntries({ current: myFilters.page, size: myFilters.pageSize, status: myFilters.status, validation_status: myFilters.validation_status }))
  }, [myFilters])
  const loadAdmin = useCallback(async () => {
    const [entries, configured, nextAnalytics] = await Promise.all([
      fetchAdminGalaxyRegistryEntries({ current: adminFilters.page, size: adminFilters.pageSize, keyword: adminFilters.keyword, status: adminFilters.status, validation_status: adminFilters.validation_status }),
      fetchAdminGalaxyRegistrySystems(),
      fetchAdminGalaxyRegistryAnalytics({ start_date: analyticsRange.start, end_date: analyticsRange.end }),
    ])
    setAdminPage(entries)
    setAdminSystems((current) => current.some((row) => row.dirty) ? current : configured.map((row) => ({ ...row, localId: String(row.id), isNew: false, dirty: false })))
    setAnalytics(nextAnalytics)
  }, [adminFilters, analyticsRange])
  const loadActiveTab = useCallback(async () => {
    setLoading(true); setError(null)
    try {
      if (tab === 'current') await loadSystems()
      if (tab === 'captain' && canCaptain) await loadCaptain()
      if (tab === 'admin' && canAdmin) await loadAdmin()
    } catch (cause) { setError(cause instanceof Error ? cause.message : t('common.fetchFail')) } finally { setLoading(false) }
  }, [canAdmin, canCaptain, loadAdmin, loadCaptain, loadSystems, t, tab])
  useEffect(() => { const timer = window.setTimeout(() => void loadActiveTab(), 0); return () => window.clearTimeout(timer) }, [loadActiveTab])

  const act = useCallback(async (action: () => Promise<unknown>, refresh: 'tab' | 'all' = 'tab') => {
    try {
      await action()
      notifySuccess(t('galaxyRegistry.actionSuccess'))
      if (refresh === 'all') { await Promise.all([loadSystems(), canAdmin ? loadAdmin() : Promise.resolve(), canCaptain ? loadCaptain() : Promise.resolve()]) } else await loadActiveTab()
    } catch (cause) { notifyError(cause instanceof Error ? cause.message : t('galaxyRegistry.actionFailed')) }
  }, [canAdmin, canCaptain, loadActiveTab, loadAdmin, loadCaptain, loadSystems, t])

  const updateDraft = (localId: string, changes: Partial<AdminDraft>) => setAdminSystems((rows) => rows.map((row) => row.localId === localId ? { ...row, ...changes, dirty: true } : row))
  const addSde = (item: GalaxyRegistrySdeSystem) => setAdminSystems((rows) => rows.some((row) => row.solar_system_id === item.solar_system_id) ? rows : [...rows, { ...item, id: 0, note: '', min_bounty_amount: 0, is_enabled: true, created_at: '', updated_at: '', localId: `new-${item.solar_system_id}`, isNew: true, dirty: true }])
  const removeDraft = useCallback(async (row: AdminDraft) => {
    if (row.isNew) { setAdminSystems((rows) => rows.filter((item) => item.localId !== row.localId)); return }
    if (!await confirmAction({ title: t('feedback.confirmTitle'), message: t('feedback.confirmMessage'), confirmText: t('common.confirm'), cancelText: t('common.cancel') })) return
    await act(async () => { await deleteAdminGalaxyRegistrySystem(row.id); setAdminSystems([]) }, 'all')
  }, [act, t])
  const saveDrafts = async () => act(async () => {
    for (const row of adminSystems) {
      if (row.isNew) await createAdminGalaxyRegistrySystem({ solar_system_id: row.solar_system_id, note: row.note, min_bounty_amount: row.min_bounty_amount, is_enabled: row.is_enabled })
      else if (row.dirty) await updateAdminGalaxyRegistrySystem(row.id, { note: row.note, min_bounty_amount: row.min_bounty_amount, is_enabled: row.is_enabled })
    }
    setAdminSystems([])
  }, 'all')
  const submitExpectedEnd = async () => {
    if (!entryDialog || !isAllowedEnd(entryDialog.expectedEnd, entryDialog.actualStartAt ? new Date(entryDialog.actualStartAt) : new Date())) { notifyWarning(t('galaxyRegistry.actionFailed')); return }
    await act(async () => {
      if (entryDialog.id < 0) await createGalaxyRegistryEntry({ system_config_id: -entryDialog.id, expected_end_at: new Date(entryDialog.expectedEnd).toISOString() })
      else await updateGalaxyRegistryEntryExpectedEndAt(entryDialog.id, { expected_end_at: new Date(entryDialog.expectedEnd).toISOString() })
      setEntryDialog(null)
    }, 'all')
  }

  const systemColumns = useMemo<ColumnDef<GalaxyRegistrySystemItem, unknown>[]>(() => [
    { accessorKey: 'solar_system_name', header: t('galaxyRegistry.system') }, { accessorKey: 'region_name', header: t('galaxyRegistry.region') },
    { accessorKey: 'status', header: t('common.status') }, { accessorKey: 'note', header: t('galaxyRegistry.fields.note') },
    { id: 'captain', header: t('galaxyRegistry.captain'), cell: ({ row }) => row.original.active_entry?.captain_character_name ?? '-' },
    { id: 'end', header: t('galaxyRegistry.expectedEnd'), cell: ({ row }) => formatTime(row.original.active_entry?.expected_end_at) },
    { id: 'actions', header: t('common.operation'), cell: ({ row }) => {
      const entry = row.original.active_entry
      if (!canCaptain) return null
      if (entry?.is_mine) return <div className="flex gap-2"><Button size="sm" variant="outline" onClick={() => setEntryDialog({ id: entry.entry_id, expectedEnd: toDateInput(new Date(entry.expected_end_at)), actualStartAt: entry.actual_start_at })}>{t('common.edit')}</Button><Button size="sm" variant="outline" onClick={() => void act(() => endGalaxyRegistryEntry(entry.entry_id), 'all')}>{t('galaxyRegistry.end')}</Button></div>
      return row.original.status === 'idle' || row.original.status === 'overdue' ? <Button size="sm" onClick={() => setEntryDialog({ id: -row.original.system_config_id, expectedEnd: toDateInput(new Date(Date.now() + 60 * 60 * 1000)), actualStartAt: null })}>{t('galaxyRegistry.register')}</Button> : null
    } },
  ], [act, canCaptain, t])
  const entryColumns = useMemo<ColumnDef<GalaxyRegistryEntryItem, unknown>[]>(() => [
    { accessorKey: 'solar_system_name', header: t('galaxyRegistry.system') }, { accessorKey: 'captain_character_name', header: t('galaxyRegistry.captain') },
    { accessorKey: 'status', header: t('common.status') }, { accessorKey: 'validation_status', header: t('galaxyRegistry.validation') },
    { id: 'started', header: t('galaxyRegistry.startedAt'), cell: ({ row }) => formatTime(row.original.actual_start_at) },
    { id: 'actions', header: t('common.operation'), cell: ({ row }) => tab === 'captain' && row.original.status === 'active' ? <div className="flex gap-2"><Button size="sm" variant="outline" onClick={() => setEntryDialog({ id: row.original.id, expectedEnd: toDateInput(new Date(row.original.expected_end_at)), actualStartAt: row.original.actual_start_at })}>{t('common.edit')}</Button><Button size="sm" variant="outline" onClick={() => void act(() => endGalaxyRegistryEntry(row.original.id), 'all')}>{t('galaxyRegistry.end')}</Button></div> : tab === 'admin' ? <div className="flex gap-2"><Button size="sm" variant="outline" onClick={() => void act(() => revalidateAdminGalaxyRegistryEntry(row.original.id), 'all')}>{t('galaxyRegistry.revalidate')}</Button>{row.original.status === 'active' && <Button size="sm" variant="outline" onClick={() => void act(() => forceEndAdminGalaxyRegistryEntry(row.original.id), 'all')}>{t('galaxyRegistry.forceEnd')}</Button>}<Button size="sm" variant="outline" onClick={() => { setValidationDialog(row.original); setValidation(row.original.validation_status === 'violation' ? 'violation' : 'valid'); setViolationReason(row.original.violation_reason ?? '') }}>{t('common.edit')}</Button></div> : null },
  ], [act, t, tab])
  const adminSystemColumns = useMemo<ColumnDef<AdminDraft, unknown>[]>(() => [
    { accessorKey: 'solar_system_name', header: t('galaxyRegistry.system') }, { accessorKey: 'region_name', header: t('galaxyRegistry.region') },
    { id: 'note', header: t('galaxyRegistry.fields.note'), cell: ({ row }) => <Input value={row.original.note} onChange={(event) => updateDraft(row.original.localId, { note: event.target.value })} /> },
    { id: 'bounty', header: t('galaxyRegistry.fields.minBounty'), cell: ({ row }) => <Input type="number" min="0" value={row.original.min_bounty_amount} onChange={(event) => updateDraft(row.original.localId, { min_bounty_amount: Math.max(0, Number(event.target.value) || 0) })} /> },
    { id: 'enabled', header: t('galaxyRegistry.fields.enabled'), cell: ({ row }) => <Switch checked={row.original.is_enabled} onCheckedChange={(value) => updateDraft(row.original.localId, { is_enabled: value })} /> },
    { id: 'dirty', header: t('common.status'), cell: ({ row }) => row.original.isNew ? t('galaxyRegistry.statuses.pending') : row.original.dirty ? t('common.edit') : '-' },
    { id: 'actions', header: t('common.operation'), cell: ({ row }) => <Button size="sm" variant="outline" onClick={() => void removeDraft(row.original)}>{t('common.delete')}</Button> },
  ], [removeDraft, t])
  const tableProps = { loading, error, loadingText: t('common.loading'), emptyText: t('common.empty') }
  const pagination = (filters: EntryFilters, page: GalaxyRegistryEntryPage, setFilters: (updater: (current: EntryFilters) => EntryFilters) => void) => ({ page: filters.page, pageSize: filters.pageSize, total: page.total, onPageChange: (next: number) => setFilters((current) => ({ ...current, page: next })), onPageSizeChange: (pageSize: number) => setFilters((current) => ({ ...current, page: 1, pageSize })), previousLabel: t('common.previous'), nextLabel: t('common.next'), pageSizeLabel: t('common.pageSize') })
  const filters = (value: EntryFilters, setValue: (updater: (current: EntryFilters) => EntryFilters) => void, keyword = false) => <div className="flex flex-wrap gap-2">{keyword && <Input placeholder={t('galaxyRegistry.fields.keyword')} value={value.keyword ?? ''} onChange={(event) => setValue((current) => ({ ...current, page: 1, keyword: event.target.value }))} />}<NativeSelect value={value.status} onChange={(event) => setValue((current) => ({ ...current, page: 1, status: event.target.value as EntryFilters['status'] }))}><NativeSelectOption value="">{t('common.all')}</NativeSelectOption><NativeSelectOption value="active">{t('galaxyRegistry.statuses.active')}</NativeSelectOption><NativeSelectOption value="completed">{t('galaxyRegistry.statuses.completed')}</NativeSelectOption></NativeSelect><NativeSelect value={value.validation_status} onChange={(event) => setValue((current) => ({ ...current, page: 1, validation_status: event.target.value as EntryFilters['validation_status'] }))}><NativeSelectOption value="">{t('common.all')}</NativeSelectOption><NativeSelectOption value="pending">{t('galaxyRegistry.statuses.pending')}</NativeSelectOption><NativeSelectOption value="valid">{t('galaxyRegistry.statuses.valid')}</NativeSelectOption><NativeSelectOption value="violation">{t('galaxyRegistry.statuses.violation')}</NativeSelectOption></NativeSelect><Button variant="outline" onClick={() => void loadActiveTab()}>{t('common.search')}</Button></div>

  return <section className="space-y-5"><header><h1 className="text-xl font-semibold">{t('galaxyRegistry.title')}</h1><p className="text-sm text-muted-foreground">{t('galaxyRegistry.subtitle')}</p></header><span className="sr-only">{t('galaxyRegistry.adminEntries')}</span><Tabs value={tab} onValueChange={setTab}><TabsList><TabsTrigger value="current">{t('galaxyRegistry.tabs.current')}</TabsTrigger>{canCaptain && <TabsTrigger value="captain">{t('galaxyRegistry.tabs.captain')}</TabsTrigger>}{canAdmin && <TabsTrigger value="admin">{t('galaxyRegistry.tabs.admin')}</TabsTrigger>}</TabsList><TabsContent value="current" className="space-y-4">{systems && <div className="grid gap-3 sm:grid-cols-3"><div className="rounded border p-3">{t('galaxyRegistry.idle')}: {systems.summary.idle_count}</div><div className="rounded border p-3">{t('galaxyRegistry.busy')}: {systems.summary.busy_count}</div><div className="rounded border p-3">{t('galaxyRegistry.overdue')}: {systems.summary.overdue_count}</div></div>}<DataTable columns={systemColumns} data={systems?.items ?? []} getRowId={(row) => String(row.system_config_id)} {...tableProps} /></TabsContent><TabsContent value="captain" className="space-y-4">{filters(myFilters, setMyFilters)}<DataTable columns={entryColumns} data={myPage.list} getRowId={(row) => String(row.id)} {...tableProps} pagination={pagination(myFilters, myPage, setMyFilters)} /></TabsContent><TabsContent value="admin" className="space-y-5"><div className="flex flex-wrap gap-2"><Input placeholder={t('galaxyRegistry.fields.keyword')} value={sdeKeyword} onChange={(event) => setSdeKeyword(event.target.value)} /><Button onClick={() => void act(async () => setSdeResults(await searchGalaxyRegistrySdeSystems({ keyword: sdeKeyword, limit: 20 })), 'tab')}>{t('common.search')}</Button><Button variant="outline" disabled={!adminSystems.some((row) => row.dirty)} onClick={() => void saveDrafts()}>{t('common.save')}</Button></div>{sdeResults.map((item) => <div key={item.solar_system_id} className="flex items-center justify-between rounded border p-2"><span>{item.solar_system_name}</span><Button size="sm" disabled={adminSystems.some((row) => row.solar_system_id === item.solar_system_id)} onClick={() => addSde(item)}>{t('galaxyRegistry.register')}</Button></div>)}<DataTable columns={adminSystemColumns} data={adminSystems} getRowId={(row) => row.localId} {...tableProps} />{analytics && <div className="space-y-3"><div className="flex flex-wrap gap-2"><Input type="date" value={analyticsRange.start} onChange={(event) => setAnalyticsRange((range) => ({ ...range, start: event.target.value }))} /><Input type="date" value={analyticsRange.end} onChange={(event) => setAnalyticsRange((range) => ({ ...range, end: event.target.value }))} /><Button variant="outline" onClick={() => void loadActiveTab()}>{t('common.search')}</Button></div><div className="grid gap-3 sm:grid-cols-2"><div className="rounded border p-3">{t('galaxyRegistry.analytics.recent7d')}: {analytics.recent_7d.valid_count} / {analytics.recent_7d.entry_count}</div><div className="rounded border p-3">{t('galaxyRegistry.analytics.recent30d')}: {analytics.recent_30d.valid_count} / {analytics.recent_30d.entry_count}</div></div></div>}{filters(adminFilters, setAdminFilters, true)}<DataTable columns={entryColumns} data={adminPage.list} getRowId={(row) => String(row.id)} {...tableProps} pagination={pagination(adminFilters, adminPage, setAdminFilters)} /></TabsContent></Tabs><Dialog open={entryDialog !== null} onOpenChange={(open) => !open && setEntryDialog(null)}><DialogContent><DialogHeader><DialogTitle>{t('galaxyRegistry.expectedEnd')}</DialogTitle></DialogHeader><Input type="datetime-local" value={entryDialog?.expectedEnd ?? ''} onChange={(event) => setEntryDialog((value) => value ? { ...value, expectedEnd: event.target.value } : null)} /><DialogFooter><Button variant="outline" onClick={() => setEntryDialog(null)}>{t('common.cancel')}</Button><Button onClick={() => void submitExpectedEnd()}>{t('common.save')}</Button></DialogFooter></DialogContent></Dialog><Dialog open={validationDialog !== null} onOpenChange={(open) => !open && setValidationDialog(null)}><DialogContent><DialogHeader><DialogTitle>{t('galaxyRegistry.validation')}</DialogTitle></DialogHeader><NativeSelect value={validation} onChange={(event) => setValidation(event.target.value as 'valid' | 'violation')}><NativeSelectOption value="valid">{t('galaxyRegistry.statuses.valid')}</NativeSelectOption><NativeSelectOption value="violation">{t('galaxyRegistry.statuses.violation')}</NativeSelectOption></NativeSelect>{validation === 'violation' && <Input placeholder={t('galaxyRegistry.fields.violationReason')} value={violationReason} onChange={(event) => setViolationReason(event.target.value)} />}<DialogFooter><Button variant="outline" onClick={() => setValidationDialog(null)}>{t('common.cancel')}</Button><Button onClick={() => void act(async () => { if (validationDialog) await updateAdminGalaxyRegistryEntryValidation(validationDialog.id, { validation_status: validation, violation_reason: validation === 'violation' ? violationReason || undefined : undefined }); setValidationDialog(null) }, 'all')}>{t('common.save')}</Button></DialogFooter></DialogContent></Dialog></section>
}
