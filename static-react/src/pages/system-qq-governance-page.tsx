import { useCallback, useEffect, useMemo, useState } from 'react'
import {
  acknowledgeQQGovernanceAlert,
  createQQGovernancePolicy,
  deleteQQGovernancePolicy,
  fetchQQGovernanceAlerts,
  fetchQQGovernanceConnection,
  fetchQQGovernanceGroups,
  fetchQQGovernanceMembers,
  fetchQQGovernanceMetrics,
  fetchQQGovernancePolicies,
  fetchQQGovernanceReviews,
  fetchQQGovernanceSettings,
  fetchQQGovernanceTasks,
  recoverQQGovernanceDisconnectedTasks,
  resetQQGovernanceRisk,
  retryQQGovernanceTask,
  searchQQGovernanceCorporations,
  updateQQGovernancePolicy,
  triggerQQGovernanceReconcile,
  updateQQGovernanceSettings,
} from '@/api/qq-governance'
import { Button } from '@/components/ui/button'
import { Combobox, ComboboxChip, ComboboxChips, ComboboxChipsInput, ComboboxContent, ComboboxEmpty, ComboboxItem, ComboboxList, ComboboxValue } from '@/components/ui/combobox'
import { DataTable, type ColumnDef } from '@/components/ui/data-table'
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Textarea } from '@/components/ui/textarea'
import { notifyError, notifySuccess } from '@/feedback'
import { useI18n } from '@/i18n'
import type {
  QQActionTask,
  QQAlert,
  QQConnection,
  QQCorporationOption,
  QQGroupStatus,
  QQMetrics,
  QQMemberState,
  QQPolicy,
  QQPageResult,
  QQReview,
  QQSettings,
} from '@/types/api/qq-governance'

type Tab = 'overview' | 'policies' | 'operations' | 'settings'
type OperationTab = 'tasks' | 'reviews' | 'alerts' | 'members'
type OperationFilters = { page: number; pageSize: number; groupID: string; qq: string; status: string; decision: string; actionType: string }
const emptyOperationPage = <T,>(): QQPageResult<T> => ({ list: [], total: 0, page: 1, page_size: 20 })
const defaultOperationFilters: OperationFilters = { page: 1, pageSize: 20, groupID: '', qq: '', status: '', decision: '', actionType: '' }

export function SystemQQGovernancePage() {
  const { t } = useI18n()
  const [tab, setTab] = useState<Tab>('overview')
  const [groups, setGroups] = useState<QQGroupStatus[]>([])
  const [policies, setPolicies] = useState<QQPolicy[]>([])
  const [tasks, setTasks] = useState<QQPageResult<QQActionTask>>(emptyOperationPage)
  const [members, setMembers] = useState<QQPageResult<QQMemberState>>(emptyOperationPage)
  const [reviews, setReviews] = useState<QQPageResult<QQReview>>(emptyOperationPage)
  const [alerts, setAlerts] = useState<QQPageResult<QQAlert>>(emptyOperationPage)
  const [metrics, setMetrics] = useState<QQMetrics | null>(null)
  const [connection, setConnection] = useState<QQConnection | null>(null)
  const [settings, setSettings] = useState<QQSettings | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [policyDialog, setPolicyDialog] = useState<QQPolicy | null | 'new'>(null)
  const [operationTab, setOperationTab] = useState<OperationTab>('tasks')
  const [operationFilters, setOperationFilters] = useState<OperationFilters>(defaultOperationFilters)
  const [corporations, setCorporations] = useState<QQCorporationOption[]>([])
  const [policyForm, setPolicyForm] = useState<{ group_id: string; enabled: boolean; allowed_corporation_ids: number[]; allowed_role_codes: string; auto_reject_unmatched: boolean; member_violation_policy: QQPolicy['member_violation_policy']; card_template: string; card_sync_enabled: boolean }>({ group_id: '', enabled: true, allowed_corporation_ids: [], allowed_role_codes: '', auto_reject_unmatched: false, member_violation_policy: 'review_only', card_template: '', card_sync_enabled: false })

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const [groupData, policyData, metricData, connectionData, settingsData] =
        await Promise.all([
          fetchQQGovernanceGroups(),
          fetchQQGovernancePolicies(),
          fetchQQGovernanceMetrics(),
          fetchQQGovernanceConnection(),
          fetchQQGovernanceSettings(),
        ])
      setGroups(groupData ?? [])
      setPolicies(policyData ?? [])
      setMetrics(metricData)
      setConnection(connectionData)
      setSettings(settingsData)
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : t('common.fetchFail'))
    } finally {
      setLoading(false)
    }
  }, [t])

  const loadOperations = useCallback(async () => {
    const numeric = (value: string) => value.trim() === '' ? undefined : Number(value)
    const common = {
      page: operationFilters.page,
      page_size: operationFilters.pageSize,
      group_id: numeric(operationFilters.groupID),
      qq: numeric(operationFilters.qq),
      status: operationFilters.status,
    }
    if ((common.group_id !== undefined && !Number.isSafeInteger(common.group_id)) || (common.qq !== undefined && !Number.isSafeInteger(common.qq))) return
    setLoading(true); setError(null)
    try {
      if (operationTab === 'tasks') setTasks(await fetchQQGovernanceTasks({ ...common, action_type: operationFilters.actionType }))
      if (operationTab === 'reviews') setReviews(await fetchQQGovernanceReviews({ ...common, decision: operationFilters.decision }))
      if (operationTab === 'alerts') setAlerts(await fetchQQGovernanceAlerts(common))
      if (operationTab === 'members') setMembers(await fetchQQGovernanceMembers(common))
    } catch (cause) { setError(cause instanceof Error ? cause.message : t('common.fetchFail')) } finally { setLoading(false) }
  }, [operationFilters, operationTab, t])

  useEffect(() => {
    const timer = window.setTimeout(() => void load(), 0)
    return () => window.clearTimeout(timer)
  }, [load])

  useEffect(() => {
    if (tab !== 'operations') return undefined
    const timer = window.setTimeout(() => void loadOperations(), 0)
    return () => window.clearTimeout(timer)
  }, [loadOperations, tab])

  const act = useCallback(async (action: () => Promise<unknown>) => {
    try {
      await action()
      notifySuccess(t('qqGovernance.actionSuccess'))
      await load()
      if (tab === 'operations') await loadOperations()
    } catch (cause) {
      notifyError(cause instanceof Error ? cause.message : t('qqGovernance.actionFailed'))
    }
  }, [load, loadOperations, t, tab])

  const mergeCorporations = useCallback((items: QQCorporationOption[]) => {
    setCorporations((current) => Array.from(new Map([...current, ...items].map((item) => [item.corporation_id, item])).values()))
  }, [])
  const searchCorporations = useCallback(async (keyword: string) => {
    if (keyword.trim().length < 2) return
    try { mergeCorporations(await searchQQGovernanceCorporations(keyword.trim())) } catch (cause) { notifyError(cause instanceof Error ? cause.message : t('qqGovernance.actionFailed')) }
  }, [mergeCorporations, t])
  const openPolicy = useCallback((policy: QQPolicy | 'new') => {
    if (policy === 'new') {
      setPolicyForm({ group_id: '', enabled: true, allowed_corporation_ids: [], allowed_role_codes: '', auto_reject_unmatched: false, member_violation_policy: 'review_only', card_template: '', card_sync_enabled: false })
      setPolicyDialog('new')
      return
    }
    mergeCorporations(policy.allowed_corporations ?? policy.allowed_corporation_ids.map((corporation_id) => ({ corporation_id, corporation_name: `#${corporation_id}` })))
    setPolicyForm({ group_id: String(policy.group_id), enabled: policy.enabled, allowed_corporation_ids: policy.allowed_corporation_ids, allowed_role_codes: policy.allowed_role_codes.join(','), auto_reject_unmatched: policy.auto_reject_unmatched, member_violation_policy: policy.member_violation_policy, card_template: policy.card_template, card_sync_enabled: policy.card_sync_enabled })
    setPolicyDialog(policy)
  }, [mergeCorporations])

  const groupColumns = useMemo<ColumnDef<QQGroupStatus, unknown>[]>(() => [
    { accessorKey: 'group_name', header: t('qqGovernance.group') },
    { accessorKey: 'group_id', header: t('qqGovernance.groupId') },
    { accessorKey: 'member_count', header: t('qqGovernance.members') },
    { accessorKey: 'snapshot_state', header: t('qqGovernance.snapshot') },
    {
      accessorKey: 'bot_is_admin',
      header: t('qqGovernance.botAdmin'),
      cell: ({ row }) =>
        row.original.bot_is_admin === null
          ? '-'
          : t(row.original.bot_is_admin ? 'common.yes' : 'common.no'),
    },
    {
      id: 'actions', header: t('common.operation'), cell: ({ row }) => (
        <Button size="sm" onClick={() => void act(() => triggerQQGovernanceReconcile(row.original.group_id))}>
          {t('qqGovernance.reconcile')}
        </Button>
      ),
    },
  ], [act, t])

  const policyColumns = useMemo<ColumnDef<QQPolicy, unknown>[]>(() => [
    { accessorKey: 'group_id', header: t('qqGovernance.groupId') },
    {
      accessorKey: 'enabled',
      header: t('qqGovernance.enabled'),
      cell: ({ row }) => t(row.original.enabled ? 'common.yes' : 'common.no'),
    },
    { accessorKey: 'allowed_role_codes', header: t('qqGovernance.roles'), cell: ({ row }) => row.original.allowed_role_codes.join(', ') || '-' },
    { accessorKey: 'allowed_corporation_ids', header: t('qqGovernance.corporations'), cell: ({ row }) => row.original.allowed_corporation_ids.join(', ') || '-' },
    { accessorKey: 'member_violation_policy', header: t('qqGovernance.violationPolicy') },
    { id: 'actions', header: t('common.operation'), cell: ({ row }) => <div className="flex gap-2"><Button size="sm" variant="outline" onClick={() => openPolicy(row.original)}>{t('common.edit')}</Button><Button size="sm" variant="destructive" onClick={() => void act(() => deleteQQGovernancePolicy(row.original.group_id))}>{t('common.delete')}</Button></div> },
  ], [act, openPolicy, t])

  const taskColumns = useMemo<ColumnDef<QQActionTask, unknown>[]>(() => [
    { accessorKey: 'id', header: t('qqGovernance.id') },
    { accessorKey: 'action_type', header: t('common.type') },
    { accessorKey: 'group_id', header: t('qqGovernance.groupId') },
    { accessorKey: 'qq', header: t('qqGovernance.qq') },
    { accessorKey: 'status', header: t('common.status') },
    { accessorKey: 'retry_cause', header: t('qqGovernance.retryCause') },
    {
      id: 'actions', header: t('common.operation'), cell: ({ row }) => (
        <Button size="sm" variant="outline" onClick={() => void act(() => retryQQGovernanceTask(row.original.id))}>
          {t('qqGovernance.retry')}
        </Button>
      ),
    },
  ], [act, t])

  const memberColumns = useMemo<ColumnDef<QQMemberState, unknown>[]>(() => [
    { accessorKey: 'group_id', header: t('qqGovernance.groupId') },
    { accessorKey: 'qq', header: t('qqGovernance.qq') },
    { accessorKey: 'status', header: t('common.status') },
    { accessorKey: 'target_card', header: t('qqGovernance.cardTemplate') },
    { accessorKey: 'mismatch_count', header: t('qqGovernance.confirmations') },
  ], [t])
  const reviewColumns = useMemo<ColumnDef<QQReview, unknown>[]>(() => [
    { accessorKey: 'group_id', header: t('qqGovernance.groupId') },
    { accessorKey: 'qq', header: t('qqGovernance.qq') },
    { accessorKey: 'decision', header: t('common.status') },
    { accessorKey: 'reason', header: t('common.reason') },
    { accessorKey: 'nickname', header: t('common.name') },
  ], [t])

  const alertColumns = useMemo<ColumnDef<QQAlert, unknown>[]>(() => [
    { accessorKey: 'kind', header: t('common.type') },
    { accessorKey: 'group_id', header: t('qqGovernance.groupId') },
    { accessorKey: 'qq', header: t('qqGovernance.qq') },
    { accessorKey: 'status', header: t('common.status') },
    { accessorKey: 'message', header: t('qqGovernance.message') },
    {
      id: 'actions', header: t('common.operation'), cell: ({ row }) => row.original.status !== 'acknowledged' ? (
        <Button size="sm" variant="outline" onClick={() => void act(() => acknowledgeQQGovernanceAlert(row.original.id))}>
          {t('qqGovernance.acknowledge')}
        </Button>
      ) : null,
    },
  ], [act, t])

  const tableProps = {
    loading,
    error,
    loadingText: t('common.loading'),
    emptyText: t('common.empty'),
    variant: 'ledger' as const,
  }
  const operationPage = operationTab === 'tasks' ? tasks : operationTab === 'reviews' ? reviews : operationTab === 'alerts' ? alerts : members
  const operationPagination = {
    page: operationFilters.page, pageSize: operationFilters.pageSize, total: operationPage.total,
    onPageChange: (page: number) => setOperationFilters((current) => ({ ...current, page })),
    onPageSizeChange: (pageSize: number) => setOperationFilters((current) => ({ ...current, page: 1, pageSize })),
    previousLabel: t('common.previous'), nextLabel: t('common.next'), pageSizeLabel: t('common.pageSize'),
  }

  return (
    <section className="space-y-5">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h1 className="text-xl font-semibold">{t('qqGovernance.title')}</h1>
          <p className="text-sm text-muted-foreground">{t('qqGovernance.subtitle')}</p>
        </div>
        <Button variant="outline" onClick={() => void (tab === 'operations' ? loadOperations() : load())}>{t('common.refresh')}</Button>
      </div>
      <Tabs value={tab} onValueChange={(value) => setTab(value as Tab)}>
      <TabsList>
        {(['overview', 'policies', 'operations', 'settings'] as Tab[]).map((item) => (
          <TabsTrigger key={item} value={item}>
            {t(`qqGovernance.tabs.${item}`)}
          </TabsTrigger>
        ))}
      </TabsList>
      </Tabs>

      {tab === 'overview' ? (
        <div className="space-y-4">
          <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
            <div className="rounded-lg border p-4">
              {t('qqGovernance.connected')}:{' '}
              {t(connection?.connected ? 'common.yes' : 'common.no')}
            </div>
            <div className="rounded-lg border p-4">{t('qqGovernance.risk')}: {connection?.risk_level ?? '-'}</div>
            <div className="rounded-lg border p-4">{t('qqGovernance.success')}: {metrics?.succeeded ?? 0}</div>
            <div className="rounded-lg border p-4">{t('qqGovernance.failed')}: {metrics?.failed ?? 0}</div>
          </div>
          {connection?.rate_limit && <div className="grid gap-3 sm:grid-cols-2"><div className="rounded-lg border p-4">{t('qqGovernance.globalLimit')}: {connection.rate_limit.global.tokens} / {connection.rate_limit.global.capacity}</div><div className="rounded-lg border p-4">{t('qqGovernance.groupLimit')}: {connection.rate_limit.groups.map((item) => `${item.group_id}: ${item.bucket.tokens} / ${item.bucket.capacity}`).join(', ') || '-'}</div></div>}
          <div className="flex gap-2">
            <Button variant="outline" onClick={() => void act(recoverQQGovernanceDisconnectedTasks)}>{t('qqGovernance.recover')}</Button>
            <Button variant="destructive" onClick={() => void act(resetQQGovernanceRisk)}>{t('qqGovernance.resetRisk')}</Button>
          </div>
          <DataTable columns={groupColumns} data={groups} getRowId={(row) => String(row.group_id)} {...tableProps} />
        </div>
      ) : null}
      {tab === 'policies' ? <div className="space-y-3"><Button onClick={() => openPolicy('new')}>{t('qqGovernance.addPolicy')}</Button><DataTable columns={policyColumns} data={policies} getRowId={(row) => String(row.id)} {...tableProps} /></div> : null}
      {tab === 'operations' ? <div className="space-y-3"><Tabs value={operationTab} onValueChange={(value) => { setOperationTab(value as OperationTab); setOperationFilters((current) => ({ ...current, page: 1 })) }}><TabsList><TabsTrigger value="tasks">{t('qqGovernance.tabs.tasks')}</TabsTrigger><TabsTrigger value="reviews">{t('qqGovernance.reviews')}</TabsTrigger><TabsTrigger value="alerts">{t('qqGovernance.tabs.alerts')}</TabsTrigger><TabsTrigger value="members">{t('qqGovernance.members')}</TabsTrigger></TabsList></Tabs><div className="flex flex-wrap gap-2"><Input type="number" placeholder={t('qqGovernance.groupId')} value={operationFilters.groupID} onChange={(event) => setOperationFilters((current) => ({ ...current, page: 1, groupID: event.target.value }))} /><Input type="number" placeholder={t('qqGovernance.qq')} value={operationFilters.qq} onChange={(event) => setOperationFilters((current) => ({ ...current, page: 1, qq: event.target.value }))} /><Input placeholder={t('common.status')} value={operationFilters.status} onChange={(event) => setOperationFilters((current) => ({ ...current, page: 1, status: event.target.value }))} />{operationTab === 'tasks' && <Input placeholder={t('common.type')} value={operationFilters.actionType} onChange={(event) => setOperationFilters((current) => ({ ...current, page: 1, actionType: event.target.value }))} />}{operationTab === 'reviews' && <Input placeholder={t('common.status')} value={operationFilters.decision} onChange={(event) => setOperationFilters((current) => ({ ...current, page: 1, decision: event.target.value }))} />}<Button variant="outline" onClick={() => void loadOperations()}>{t('common.search')}</Button></div>{operationTab === 'tasks' ? <DataTable columns={taskColumns} data={tasks.list} getRowId={(row) => String(row.id)} {...tableProps} pagination={operationPagination} /> : null}{operationTab === 'reviews' ? <DataTable columns={reviewColumns} data={reviews.list} getRowId={(row) => String(row.id)} {...tableProps} pagination={operationPagination} /> : null}{operationTab === 'alerts' ? <DataTable columns={alertColumns} data={alerts.list} getRowId={(row) => String(row.id)} {...tableProps} pagination={operationPagination} /> : null}{operationTab === 'members' ? <DataTable columns={memberColumns} data={members.list} getRowId={(row) => String(row.id)} {...tableProps} pagination={operationPagination} /> : null}</div> : null}
      {tab === 'settings' && settings ? (
        <div className="grid max-w-2xl gap-4 rounded-lg border p-5 sm:grid-cols-3">
          {([
            ['scan_interval_minutes', 'qqGovernance.scanInterval'],
            ['mismatch_confirmations', 'qqGovernance.confirmations'],
            ['mismatch_observation_hours', 'qqGovernance.observationHours'],
          ] as const).map(([field, label]) => (
            <label key={field} className="space-y-2">
              <span className="text-sm font-medium">{t(label)}</span>
              <Input type="number" value={settings[field]} onChange={(event) => setSettings({ ...settings, [field]: Number(event.target.value) })} />
            </label>
          ))}
          <div className="sm:col-span-3">
            <Button onClick={() => void act(() => updateQQGovernanceSettings(settings))}>{t('common.save')}</Button>
          </div>
        </div>
      ) : null}
      <Dialog open={policyDialog !== null} onOpenChange={(open) => !open && setPolicyDialog(null)}>
        <DialogContent>
          <DialogHeader><DialogTitle>{policyDialog === 'new' ? t('qqGovernance.addPolicy') : t('qqGovernance.editPolicy')}</DialogTitle></DialogHeader>
          <div className="space-y-3">
            <Input type="number" placeholder={t('qqGovernance.groupId')} value={policyForm.group_id} disabled={policyDialog !== 'new'} onChange={(event) => setPolicyForm({ ...policyForm, group_id: event.target.value })} />
            <Combobox<QQCorporationOption, true> multiple value={corporations.filter((item) => policyForm.allowed_corporation_ids.includes(item.corporation_id))} onValueChange={(value) => setPolicyForm((form) => ({ ...form, allowed_corporation_ids: value.map((item) => item.corporation_id) }))} onInputValueChange={(value) => void searchCorporations(value)} itemToStringLabel={(item) => item.corporation_name} itemToStringValue={(item) => String(item.corporation_id)}>
              <ComboboxChips><ComboboxValue>{(value: QQCorporationOption[]) => <>{value.map((item) => <ComboboxChip key={item.corporation_id}>{item.corporation_name}</ComboboxChip>)}<ComboboxChipsInput placeholder={value.length ? '' : t('qqGovernance.corporations')} /></>}</ComboboxValue></ComboboxChips>
              <ComboboxContent><ComboboxEmpty>{t('qqGovernance.noResults')}</ComboboxEmpty><ComboboxList>{corporations.map((item) => <ComboboxItem key={item.corporation_id} value={item}>{item.corporation_name}</ComboboxItem>)}</ComboboxList></ComboboxContent>
            </Combobox>
            <Input placeholder={t('qqGovernance.roles')} value={policyForm.allowed_role_codes} onChange={(event) => setPolicyForm({ ...policyForm, allowed_role_codes: event.target.value })} />
            <Select value={policyForm.member_violation_policy} onValueChange={(member_violation_policy) => setPolicyForm({ ...policyForm, member_violation_policy: member_violation_policy as QQPolicy['member_violation_policy'] })}><SelectTrigger className="w-full"><SelectValue placeholder={t('qqGovernance.violationPolicy')} /></SelectTrigger><SelectContent><SelectItem value="review_only">{t('qqGovernance.reviewOnly')}</SelectItem><SelectItem value="auto_kick_after_confirmed_mismatch">{t('qqGovernance.autoKick')}</SelectItem></SelectContent></Select>
            <Textarea placeholder={t('qqGovernance.cardTemplate')} value={policyForm.card_template} onChange={(event) => setPolicyForm({ ...policyForm, card_template: event.target.value })} />
            <label className="flex items-center gap-2"><Switch checked={policyForm.enabled} onCheckedChange={(enabled) => setPolicyForm({ ...policyForm, enabled })} />{t('qqGovernance.enabled')}</label>
            <label className="flex items-center gap-2"><Switch checked={policyForm.auto_reject_unmatched} onCheckedChange={(auto_reject_unmatched) => setPolicyForm({ ...policyForm, auto_reject_unmatched })} />{t('qqGovernance.autoReject')}</label>
            <label className="flex items-center gap-2"><Switch checked={policyForm.card_sync_enabled} onCheckedChange={(card_sync_enabled) => setPolicyForm({ ...policyForm, card_sync_enabled })} />{t('qqGovernance.cardSync')}</label>
          </div>
          <DialogFooter><Button variant="outline" onClick={() => setPolicyDialog(null)}>{t('common.cancel')}</Button><Button onClick={() => void act(async () => { const groupID = Number(policyForm.group_id); if (!Number.isSafeInteger(groupID) || groupID <= 0) throw new Error(t('qqGovernance.groupIdRequired')); const payload = { enabled: policyForm.enabled, allowed_corporation_ids: policyForm.allowed_corporation_ids, allowed_role_codes: policyForm.allowed_role_codes.split(',').map((value) => value.trim()).filter(Boolean), auto_reject_unmatched: policyForm.auto_reject_unmatched, member_violation_policy: policyForm.member_violation_policy, card_template: policyForm.card_template, card_sync_enabled: policyForm.card_sync_enabled }; if (policyDialog === 'new') await createQQGovernancePolicy({ group_id: groupID, ...payload }); else await updateQQGovernancePolicy(groupID, payload); setPolicyDialog(null) })}>{t('common.save')}</Button></DialogFooter>
        </DialogContent>
      </Dialog>
    </section>
  )
}
