import { useCallback, useEffect, useMemo, useState } from 'react'
import {
  createAdminGalaxyRegistrySystem, createGalaxyRegistryEntry,
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
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { notifyError, notifySuccess, notifyWarning } from '@/feedback'
import { useI18n } from '@/i18n'
import { formatTime } from '@/lib/format'
import { useSessionStore } from '@/stores'
import type { GalaxyRegistryAdminSystem, GalaxyRegistryEntryItem, GalaxyRegistrySdeSystem, GalaxyRegistrySystemItem, GalaxyRegistrySystemsResponse } from '@/types/api/galaxy-registry'

type AdminDraft = GalaxyRegistryAdminSystem & { localId: string; isNew: boolean; dirty: boolean }
const emptyPage = { list: [] as GalaxyRegistryEntryItem[], total: 0, page: 1, pageSize: 20 }
function dateInput(value = new Date(Date.now() + 60 * 60 * 1000)) { const offset = value.getTimezoneOffset() * 60_000; return new Date(value.getTime() - offset).toISOString().slice(0, 16) }
function isWithinTwoHours(value: string, start = new Date()) { const parsed = new Date(value); return !Number.isNaN(parsed.valueOf()) && parsed > start && parsed <= new Date(start.getTime() + 2 * 60 * 60 * 1000) }

export function DashboardGalaxyRegistryPage() {
  const { t } = useI18n(); const roles = useSessionStore((state) => state.roles)
  const canCaptain = roles.includes('captain') || roles.includes('super_admin'); const canAdmin = roles.includes('admin') || roles.includes('super_admin')
  const [tab, setTab] = useState('current'); const [systems, setSystems] = useState<GalaxyRegistrySystemsResponse | null>(null)
  const [myPage, setMyPage] = useState(emptyPage); const [adminPage, setAdminPage] = useState(emptyPage)
  const [myQuery, setMyQuery] = useState<{ page: number; status: '' | 'active' | 'completed'; validation_status: '' | 'pending' | 'valid' | 'violation' }>({ page: 1, status: '', validation_status: '' }); const [adminQuery] = useState<{ page: number; keyword: string; status: '' | 'active' | 'completed'; validation_status: '' | 'pending' | 'valid' | 'violation' }>({ page: 1, keyword: '', status: '', validation_status: '' })
  const [adminSystems, setAdminSystems] = useState<AdminDraft[]>([]); const [sde, setSde] = useState<GalaxyRegistrySdeSystem[]>([]); const [sdeKeyword, setSdeKeyword] = useState('')
  const [analytics, setAnalytics] = useState<Awaited<ReturnType<typeof fetchAdminGalaxyRegistryAnalytics>> | null>(null)
  const [loading, setLoading] = useState(false); const [error, setError] = useState<string | null>(null)
  const [entryDialog, setEntryDialog] = useState<{ id: number; expectedEnd: string } | null>(null); const [validationDialog, setValidationDialog] = useState<GalaxyRegistryEntryItem | null>(null)
  const [validation, setValidation] = useState<'valid' | 'violation'>('valid'); const [violationReason, setViolationReason] = useState('')

  const loadSystems = useCallback(async () => setSystems(await fetchGalaxyRegistrySystems()), [])
  const loadTab = useCallback(async () => {
    setLoading(true); setError(null)
    try {
      if (tab === 'current') await loadSystems()
      if (tab === 'captain' && canCaptain) { const page = await fetchMyGalaxyRegistryEntries({ current: myQuery.page, size: 20, status: myQuery.status, validation_status: myQuery.validation_status }); setMyPage(page) }
      if (tab === 'admin' && canAdmin) { const [entries, configured, nextAnalytics] = await Promise.all([fetchAdminGalaxyRegistryEntries({ current: adminQuery.page, size: 20, keyword: adminQuery.keyword, status: adminQuery.status, validation_status: adminQuery.validation_status }), fetchAdminGalaxyRegistrySystems(), fetchAdminGalaxyRegistryAnalytics()]); setAdminPage(entries); setAdminSystems(configured.map((item) => ({ ...item, localId: String(item.id), isNew: false, dirty: false }))); setAnalytics(nextAnalytics) }
    } catch (cause) { setError(cause instanceof Error ? cause.message : t('common.fetchFail')) } finally { setLoading(false) }
  }, [adminQuery, canAdmin, canCaptain, loadSystems, myQuery, t, tab])
  useEffect(() => { const timer = window.setTimeout(() => void loadTab(), 0); return () => window.clearTimeout(timer) }, [loadTab])
  const act = useCallback(async (action: () => Promise<unknown>, reload = true) => { try { await action(); notifySuccess(t('galaxyRegistry.actionSuccess')); if (reload) await loadTab(); } catch (cause) { notifyError(cause instanceof Error ? cause.message : t('galaxyRegistry.actionFailed')) } }, [loadTab, t])
  const submitExpectedEnd = async () => { if (!entryDialog || !isWithinTwoHours(entryDialog.expectedEnd, entryDialog.id < 0 ? new Date() : new Date())) { notifyWarning(t('galaxyRegistry.actionFailed')); return }; await act(async () => { if (entryDialog.id < 0) await createGalaxyRegistryEntry({ system_config_id: -entryDialog.id, expected_end_at: new Date(entryDialog.expectedEnd).toISOString() }); else await updateGalaxyRegistryEntryExpectedEndAt(entryDialog.id, { expected_end_at: new Date(entryDialog.expectedEnd).toISOString() }); setEntryDialog(null) }, tab === 'current') }
  const addSde = (item: GalaxyRegistrySdeSystem) => setAdminSystems((rows) => rows.some((row) => row.solar_system_id === item.solar_system_id) ? rows : [...rows, { ...item, id: 0, note: '', min_bounty_amount: 0, is_enabled: true, created_at: '', updated_at: '', localId: `new-${item.solar_system_id}`, isNew: true, dirty: true }])
  const saveDrafts = async () => act(async () => { for (const row of adminSystems) { if (row.isNew) await createAdminGalaxyRegistrySystem({ solar_system_id: row.solar_system_id, note: row.note, min_bounty_amount: row.min_bounty_amount, is_enabled: row.is_enabled }); else if (row.dirty) await updateAdminGalaxyRegistrySystem(row.id, { note: row.note, min_bounty_amount: row.min_bounty_amount, is_enabled: row.is_enabled }) }; await loadSystems() })
  const systemColumns = useMemo<ColumnDef<GalaxyRegistrySystemItem, unknown>[]>(() => [{ accessorKey: 'solar_system_name', header: t('galaxyRegistry.system') }, { accessorKey: 'region_name', header: t('galaxyRegistry.region') }, { accessorKey: 'status', header: t('common.status') }, { id: 'captain', header: t('galaxyRegistry.captain'), cell: ({ row }) => row.original.active_entry?.captain_character_name ?? '-' }, { id: 'end', header: t('galaxyRegistry.expectedEnd'), cell: ({ row }) => formatTime(row.original.active_entry?.expected_end_at) }, { id: 'actions', header: t('common.operation'), cell: ({ row }) => !canCaptain ? null : row.original.active_entry?.is_mine ? <div className="flex gap-2"><Button size="sm" variant="outline" onClick={() => setEntryDialog({ id: row.original.active_entry!.entry_id, expectedEnd: dateInput(new Date(row.original.active_entry!.expected_end_at)) })}>{t('common.edit')}</Button><Button size="sm" variant="outline" onClick={() => void act(() => endGalaxyRegistryEntry(row.original.active_entry!.entry_id))}>{t('galaxyRegistry.end')}</Button></div> : (row.original.status === 'idle' || row.original.status === 'overdue') ? <Button size="sm" onClick={() => setEntryDialog({ id: -row.original.system_config_id, expectedEnd: dateInput() })}>{t('galaxyRegistry.register')}</Button> : null }], [act, canCaptain, t])
  const entryColumns = useMemo<ColumnDef<GalaxyRegistryEntryItem, unknown>[]>(() => [{ accessorKey: 'solar_system_name', header: t('galaxyRegistry.system') }, { accessorKey: 'captain_character_name', header: t('galaxyRegistry.captain') }, { accessorKey: 'status', header: t('common.status') }, { accessorKey: 'validation_status', header: t('galaxyRegistry.validation') }, { id: 'started', header: t('galaxyRegistry.startedAt'), cell: ({ row }) => formatTime(row.original.actual_start_at) }, { id: 'actions', header: t('common.operation'), cell: ({ row }) => tab === 'captain' && row.original.status === 'active' ? <div className="flex gap-2"><Button size="sm" variant="outline" onClick={() => setEntryDialog({ id: row.original.id, expectedEnd: dateInput(new Date(row.original.expected_end_at)) })}>{t('common.edit')}</Button><Button size="sm" variant="outline" onClick={() => void act(() => endGalaxyRegistryEntry(row.original.id))}>{t('galaxyRegistry.end')}</Button></div> : tab === 'admin' ? <div className="flex gap-2"><Button size="sm" variant="outline" onClick={() => void act(() => revalidateAdminGalaxyRegistryEntry(row.original.id))}>{t('galaxyRegistry.revalidate')}</Button>{row.original.status === 'active' ? <Button size="sm" variant="outline" onClick={() => void act(() => forceEndAdminGalaxyRegistryEntry(row.original.id))}>{t('galaxyRegistry.forceEnd')}</Button> : null}<Button size="sm" variant="outline" onClick={() => { setValidationDialog(row.original); setValidation(row.original.validation_status === 'violation' ? 'violation' : 'valid'); setViolationReason(row.original.violation_reason ?? '') }}>{t('common.edit')}</Button></div> : null }], [act, t, tab])
  const tableProps = { loading, error, loadingText: t('common.loading'), emptyText: t('common.empty') }
  const currentEntries = tab === 'captain' ? myPage : adminPage
  return (
    <section className="space-y-5">
      <header><h1 className="text-xl font-semibold">{t('galaxyRegistry.title')}</h1><p className="text-sm text-muted-foreground">{t('galaxyRegistry.subtitle')}</p></header>
      <span className="sr-only">{t('galaxyRegistry.adminEntries')}</span>
      <Tabs value={tab} onValueChange={setTab}>
        <TabsList><TabsTrigger value="current">{t('common.current')}</TabsTrigger>{canCaptain && <TabsTrigger value="captain">{t('galaxyRegistry.captain')}</TabsTrigger>}{canAdmin && <TabsTrigger value="admin">{t('common.admin')}</TabsTrigger>}</TabsList>
        <TabsContent value="current" className="space-y-4">
          {systems && <div className="grid gap-3 sm:grid-cols-3"><div className="rounded border p-3">{t('galaxyRegistry.idle')}: {systems.summary.idle_count}</div><div className="rounded border p-3">{t('galaxyRegistry.busy')}: {systems.summary.busy_count}</div><div className="rounded border p-3">{t('galaxyRegistry.overdue')}: {systems.summary.overdue_count}</div></div>}
          <DataTable columns={systemColumns} data={systems?.items ?? []} getRowId={(row) => String(row.system_config_id)} {...tableProps} />
        </TabsContent>
        <TabsContent value="captain" className="space-y-4">
          <div className="flex gap-2"><NativeSelect value={myQuery.status} onChange={(event) => setMyQuery((value) => ({ ...value, page: 1, status: event.target.value as '' | 'active' | 'completed' }))}><NativeSelectOption value="">{t('common.all')}</NativeSelectOption><NativeSelectOption value="active">active</NativeSelectOption><NativeSelectOption value="completed">completed</NativeSelectOption></NativeSelect><Button variant="outline" onClick={() => void loadTab()}>{t('common.search')}</Button></div>
          <DataTable columns={entryColumns} data={currentEntries.list} getRowId={(row) => String(row.id)} {...tableProps} pagination={{ page: myQuery.page, pageSize: 20, total: myPage.total, onPageChange: (page) => setMyQuery((value) => ({ ...value, page })), onPageSizeChange: () => undefined, previousLabel: t('common.previous'), nextLabel: t('common.next'), pageSizeLabel: t('common.pageSize') }} />
        </TabsContent>
        <TabsContent value="admin" className="space-y-5">
          <div className="flex gap-2"><Input value={sdeKeyword} onChange={(event) => setSdeKeyword(event.target.value)} /><Button onClick={() => void act(async () => setSde(await searchGalaxyRegistrySdeSystems({ keyword: sdeKeyword, limit: 20 })), false)}>{t('common.search')}</Button><Button variant="outline" disabled={!adminSystems.some((row) => row.dirty)} onClick={() => void saveDrafts()}>{t('common.save')}</Button></div>
          {sde.map((item) => <div key={item.solar_system_id} className="flex items-center justify-between rounded border p-2"><span>{item.solar_system_name}</span><Button size="sm" onClick={() => addSde(item)}>{t('common.add')}</Button></div>)}
          <DataTable columns={entryColumns} data={adminPage.list} getRowId={(row) => String(row.id)} {...tableProps} />
          {analytics && <div className="grid gap-3 sm:grid-cols-2"><div className="rounded border p-3">7d: {analytics.recent_7d.entry_count}</div><div className="rounded border p-3">30d: {analytics.recent_30d.entry_count}</div></div>}
        </TabsContent>
      </Tabs>
      <Dialog open={entryDialog !== null} onOpenChange={(open) => !open && setEntryDialog(null)}><DialogContent><DialogHeader><DialogTitle>{t('galaxyRegistry.expectedEnd')}</DialogTitle></DialogHeader><Input type="datetime-local" value={entryDialog?.expectedEnd ?? ''} onChange={(event) => setEntryDialog((value) => value ? { ...value, expectedEnd: event.target.value } : null)} /><DialogFooter><Button variant="outline" onClick={() => setEntryDialog(null)}>{t('common.cancel')}</Button><Button onClick={() => void submitExpectedEnd()}>{t('common.save')}</Button></DialogFooter></DialogContent></Dialog>
      <Dialog open={validationDialog !== null} onOpenChange={(open) => !open && setValidationDialog(null)}><DialogContent><DialogHeader><DialogTitle>{t('galaxyRegistry.validation')}</DialogTitle></DialogHeader><NativeSelect value={validation} onChange={(event) => setValidation(event.target.value as 'valid' | 'violation')}><NativeSelectOption value="valid">valid</NativeSelectOption><NativeSelectOption value="violation">violation</NativeSelectOption></NativeSelect>{validation === 'violation' && <Input value={violationReason} onChange={(event) => setViolationReason(event.target.value)} />}<DialogFooter><Button variant="outline" onClick={() => setValidationDialog(null)}>{t('common.cancel')}</Button><Button onClick={() => void act(async () => { if (validationDialog) await updateAdminGalaxyRegistryEntryValidation(validationDialog.id, { validation_status: validation, violation_reason: validation === 'violation' ? violationReason || undefined : undefined }); setValidationDialog(null) })}>{t('common.save')}</Button></DialogFooter></DialogContent></Dialog>
    </section>
  )
}
