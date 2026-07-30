import { useCallback, useEffect, useMemo, useState } from 'react'
import {
  acknowledgeQQGovernanceAlert,
  fetchQQGovernanceAlerts,
  fetchQQGovernanceConnection,
  fetchQQGovernanceGroups,
  fetchQQGovernanceMetrics,
  fetchQQGovernancePolicies,
  fetchQQGovernanceSettings,
  fetchQQGovernanceTasks,
  recoverQQGovernanceDisconnectedTasks,
  resetQQGovernanceRisk,
  retryQQGovernanceTask,
  triggerQQGovernanceReconcile,
  updateQQGovernanceSettings,
} from '@/api/qq-governance'
import { Button } from '@/components/ui/button'
import { DataTable, type ColumnDef } from '@/components/ui/data-table'
import { Input } from '@/components/ui/input'
import { notifyError, notifySuccess } from '@/feedback'
import { useI18n } from '@/i18n'
import type {
  QQActionTask,
  QQAlert,
  QQConnection,
  QQGroupStatus,
  QQMetrics,
  QQPolicy,
  QQSettings,
} from '@/types/api/qq-governance'

type Tab = 'overview' | 'policies' | 'tasks' | 'alerts' | 'settings'

export function SystemQQGovernancePage() {
  const { t } = useI18n()
  const [tab, setTab] = useState<Tab>('overview')
  const [groups, setGroups] = useState<QQGroupStatus[]>([])
  const [policies, setPolicies] = useState<QQPolicy[]>([])
  const [tasks, setTasks] = useState<QQActionTask[]>([])
  const [alerts, setAlerts] = useState<QQAlert[]>([])
  const [metrics, setMetrics] = useState<QQMetrics | null>(null)
  const [connection, setConnection] = useState<QQConnection | null>(null)
  const [settings, setSettings] = useState<QQSettings | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const [groupData, policyData, taskData, alertData, metricData, connectionData, settingsData] =
        await Promise.all([
          fetchQQGovernanceGroups(),
          fetchQQGovernancePolicies(),
          fetchQQGovernanceTasks({ page: 1, page_size: 200 }),
          fetchQQGovernanceAlerts({ page: 1, page_size: 200 }),
          fetchQQGovernanceMetrics(),
          fetchQQGovernanceConnection(),
          fetchQQGovernanceSettings(),
        ])
      setGroups(groupData ?? [])
      setPolicies(policyData ?? [])
      setTasks(taskData.list ?? [])
      setAlerts(alertData.list ?? [])
      setMetrics(metricData)
      setConnection(connectionData)
      setSettings(settingsData)
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : t('common.fetchFail'))
    } finally {
      setLoading(false)
    }
  }, [t])

  useEffect(() => {
    const timer = window.setTimeout(() => void load(), 0)
    return () => window.clearTimeout(timer)
  }, [load])

  const act = useCallback(async (action: () => Promise<unknown>) => {
    try {
      await action()
      notifySuccess(t('qqGovernance.actionSuccess'))
      await load()
    } catch (cause) {
      notifyError(cause instanceof Error ? cause.message : t('qqGovernance.actionFailed'))
    }
  }, [load, t])

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
  ], [t])

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

  return (
    <section className="space-y-5">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h1 className="text-xl font-semibold">{t('qqGovernance.title')}</h1>
          <p className="text-sm text-muted-foreground">{t('qqGovernance.subtitle')}</p>
        </div>
        <Button variant="outline" onClick={() => void load()}>{t('common.refresh')}</Button>
      </div>
      <div className="flex flex-wrap gap-2">
        {(['overview', 'policies', 'tasks', 'alerts', 'settings'] as Tab[]).map((item) => (
          <Button key={item} size="sm" variant={tab === item ? 'default' : 'outline'} onClick={() => setTab(item)}>
            {t(`qqGovernance.tabs.${item}`)}
          </Button>
        ))}
      </div>

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
          <div className="flex gap-2">
            <Button variant="outline" onClick={() => void act(recoverQQGovernanceDisconnectedTasks)}>{t('qqGovernance.recover')}</Button>
            <Button variant="destructive" onClick={() => void act(resetQQGovernanceRisk)}>{t('qqGovernance.resetRisk')}</Button>
          </div>
          <DataTable columns={groupColumns} data={groups} getRowId={(row) => String(row.group_id)} {...tableProps} />
        </div>
      ) : null}
      {tab === 'policies' ? <DataTable columns={policyColumns} data={policies} getRowId={(row) => String(row.id)} {...tableProps} /> : null}
      {tab === 'tasks' ? <DataTable columns={taskColumns} data={tasks} getRowId={(row) => String(row.id)} {...tableProps} /> : null}
      {tab === 'alerts' ? <DataTable columns={alertColumns} data={alerts} getRowId={(row) => String(row.id)} {...tableProps} /> : null}
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
    </section>
  )
}
