import { useEffect, useMemo, useState } from 'react'
import {
  fetchESIRefreshStatuses,
  fetchESIRefreshTasks,
  fetchTaskHistory,
  fetchTasks,
  runESIRefreshAll,
  runESIRefreshTask,
  runESIRefreshTaskByName,
  runTask,
  updateTaskSchedule,
} from '@/api/task-manager'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { useI18n } from '@/i18n'
import { useSessionStore } from '@/stores'
import type { TaskInfo, TaskStatus } from '@/types/api/esi-refresh'
import type { TaskExecutionItem, TaskItem } from '@/types/api/task-manager'
import { getErrorMessage, ShopDialog, formatDateTime } from './shop-page-utils'

type TaskTab = 'tasks' | 'esi-statuses' | 'history'
type ScheduleMode = 'cron' | 'every'
type ScheduleIntervalUnit = 'm' | 'h'

type ScheduleFormState = {
  name: string
  displayName: string
  mode: ScheduleMode
  cronExpr: string
  defaultCron: string
  intervalValue: number
  intervalUnit: ScheduleIntervalUnit
}

const defaultScheduleForm: ScheduleFormState = {
  name: '',
  displayName: '',
  mode: 'cron',
  cronExpr: '',
  defaultCron: '',
  intervalValue: 1,
  intervalUnit: 'm',
}

const everyExprPattern = /^@every\s+(\d+)([mh])$/i

function taskDisplayKey(taskName: string) {
  return `taskManager.taskNames.${taskName}`
}

function taskDisplayName(t: ReturnType<typeof useI18n>['t'], task: Pick<TaskItem, 'name'>) {
  const key = taskDisplayKey(task.name)
  const translated = t(key)
  return translated === key ? task.name : translated
}

function taskDescription(t: ReturnType<typeof useI18n>['t'], task: Pick<TaskItem, 'name' | 'description'>) {
  const key = `taskManager.taskDescriptions.${task.name}`
  const translated = t(key)
  return translated === key ? task.description || task.name : translated
}

function categoryTone(category: TaskItem['category']) {
  switch (category) {
    case 'esi':
      return 'bg-sky-100 text-sky-700 dark:bg-sky-500/10 dark:text-sky-300'
    case 'operation':
      return 'bg-amber-100 text-amber-700 dark:bg-amber-500/10 dark:text-amber-300'
    default:
      return 'bg-muted text-muted-foreground'
  }
}

function typeTone(type: TaskItem['type']) {
  return type === 'recurring'
    ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-300'
    : 'bg-slate-100 text-slate-700 dark:bg-slate-500/10 dark:text-slate-300'
}

type TaskLastStatus = NonNullable<TaskItem['last_execution']>['status']

function statusTone(status?: TaskLastStatus) {
  switch (status) {
    case 'running':
      return 'bg-amber-100 text-amber-700 dark:bg-amber-500/10 dark:text-amber-300'
    case 'success':
      return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-300'
    case 'failed':
      return 'bg-rose-100 text-rose-700 dark:bg-rose-500/10 dark:text-rose-300'
    default:
      return 'bg-muted text-muted-foreground'
  }
}

function taskStatusTone(status: TaskStatus['status']) {
  switch (status) {
    case 'pending':
      return 'bg-slate-100 text-slate-700 dark:bg-slate-500/10 dark:text-slate-300'
    case 'running':
      return 'bg-amber-100 text-amber-700 dark:bg-amber-500/10 dark:text-amber-300'
    case 'success':
      return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-300'
    case 'failed':
      return 'bg-rose-100 text-rose-700 dark:bg-rose-500/10 dark:text-rose-300'
    default:
      return 'bg-muted text-muted-foreground'
  }
}

function triggerTone(trigger: TaskExecutionItem['trigger']) {
  return trigger === 'cron'
    ? 'bg-slate-100 text-slate-700 dark:bg-slate-500/10 dark:text-slate-300'
    : 'bg-amber-100 text-amber-700 dark:bg-amber-500/10 dark:text-amber-300'
}

function formatDuration(durationMs?: number) {
  if (durationMs == null) return '-'
  if (durationMs < 1000) return `${durationMs} ms`
  const seconds = durationMs / 1000
  if (seconds < 60) {
    return `${seconds.toFixed(seconds >= 10 ? 0 : 1)} s`
  }
  const minutes = Math.floor(seconds / 60)
  const remainingSeconds = Math.round(seconds % 60)
  if (minutes < 60) {
    return `${minutes}m ${remainingSeconds}s`
  }
  const hours = Math.floor(minutes / 60)
  const remainingMinutes = minutes % 60
  return `${hours}h ${remainingMinutes}m`
}

function priorityTone(priority: number) {
  switch (priority) {
    case 1:
      return 'bg-rose-100 text-rose-700 dark:bg-rose-500/10 dark:text-rose-300'
    case 10:
      return 'bg-amber-100 text-amber-700 dark:bg-amber-500/10 dark:text-amber-300'
    case 50:
      return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-300'
    default:
      return 'bg-slate-100 text-slate-700 dark:bg-slate-500/10 dark:text-slate-300'
  }
}

function priorityLabel(t: ReturnType<typeof useI18n>['t'], priority: number) {
  const key =
    priority === 1
      ? 'taskManager.esi.priority.critical'
      : priority === 10
        ? 'taskManager.esi.priority.high'
        : priority === 50
          ? 'taskManager.esi.priority.normal'
          : priority === 90
            ? 'taskManager.esi.priority.low'
            : `P${priority}`

  const translated = t(key)
  return translated === key ? `P${priority}` : translated
}

function runTaskOptions(t: ReturnType<typeof useI18n>['t'], task: TaskItem) {
  if (task.type === 'triggered') {
    return t('taskManager.messages.eventTriggered')
  }

  return task.cron_expr || task.default_cron || '-'
}

export function SystemTaskManagerPage() {
  const { t } = useI18n()
  const roles = useSessionStore((state) => state.roles)
  const [activeTab, setActiveTab] = useState<TaskTab>('tasks')

  return (
    <section className="space-y-4">
      <div className="rounded-lg border bg-card p-5">
        <div className="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
          <div>
            <h1 className="text-xl font-semibold">{t('taskManager.title')}</h1>
            <p className="mt-1 text-sm text-muted-foreground">{t('taskManager.subtitle')}</p>
          </div>
        </div>
      </div>

      <div className="flex flex-wrap gap-2 rounded-lg border bg-card p-2">
        {([
          ['tasks', t('taskManager.tabs.tasks')],
          ['esi-statuses', t('taskManager.tabs.esiStatuses')],
          ['history', t('taskManager.tabs.history')],
        ] as const).map(([key, label]) => (
          <Button
            key={key}
            type="button"
            variant={activeTab === key ? 'default' : 'outline'}
            onClick={() => setActiveTab(key)}
          >
            {label}
          </Button>
        ))}
      </div>

      {activeTab === 'tasks' ? <TasksPanel t={t} roles={roles} /> : null}
      {activeTab === 'esi-statuses' ? <EsiStatusesPanel t={t} /> : null}
      {activeTab === 'history' ? <HistoryPanel t={t} /> : null}
    </section>
  )
}

function TasksPanel({ t, roles }: { t: ReturnType<typeof useI18n>['t']; roles: string[] }) {
  const canRun = true
  const canUpdateSchedule = roles.includes('super_admin')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [tasks, setTasks] = useState<TaskItem[]>([])
  const [refreshSeed, setRefreshSeed] = useState(0)
  const [runningTaskNames, setRunningTaskNames] = useState<string[]>([])
  const [scheduleDialogOpen, setScheduleDialogOpen] = useState(false)
  const [scheduleSaving, setScheduleSaving] = useState(false)
  const [scheduleForm, setScheduleForm] = useState<ScheduleFormState>(defaultScheduleForm)

  const loadTasks = async () => {
    setLoading(true)
    setError(null)
    try {
      setTasks(await fetchTasks())
    } catch (caughtError) {
      setError(getErrorMessage(caughtError, t('taskManager.loadTasksFailed')))
      setTasks([])
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    const timer = window.setTimeout(() => {
      void loadTasks()
    }, 0)
    return () => window.clearTimeout(timer)
  }, [refreshSeed])

  const openScheduleDialog = (task: TaskItem) => {
    const activeCronExpr = task.cron_expr || task.default_cron
    const everyMatch = activeCronExpr.match(everyExprPattern)
    setScheduleForm({
      name: task.name,
      displayName: taskDisplayName(t, task),
      mode: everyMatch ? 'every' : 'cron',
      cronExpr: activeCronExpr,
      defaultCron: task.default_cron,
      intervalValue: everyMatch ? Number(everyMatch[1]) : 1,
      intervalUnit: everyMatch ? (everyMatch[2].toLowerCase() as ScheduleIntervalUnit) : 'm',
    })
    setScheduleDialogOpen(true)
  }

  const submitScheduleUpdate = async () => {
    const cronExpr =
      scheduleForm.mode === 'every'
        ? `@every ${scheduleForm.intervalValue}${scheduleForm.intervalUnit}`
        : scheduleForm.cronExpr.trim()

    if (!cronExpr) {
      setError(t('taskManager.messages.scheduleRequired'))
      return
    }

    setScheduleSaving(true)
    setError(null)
    try {
      await updateTaskSchedule(scheduleForm.name, { cron_expr: cronExpr })
      setScheduleDialogOpen(false)
      setRefreshSeed((current) => current + 1)
    } catch (caughtError) {
      setError(getErrorMessage(caughtError, t('taskManager.messages.scheduleUpdateFailed')))
    } finally {
      setScheduleSaving(false)
    }
  }

  const handleRunTask = async (task: TaskItem) => {
    if (!window.confirm(t('taskManager.confirm.runTask', { name: taskDisplayName(t, task) }))) {
      return
    }

    setRunningTaskNames((current) => [...current, task.name])
    setError(null)
    try {
      await runTask(task.name)
      setRefreshSeed((current) => current + 1)
    } catch (caughtError) {
      setError(getErrorMessage(caughtError, t('taskManager.messages.taskTriggerFailed')))
    } finally {
      setRunningTaskNames((current) => current.filter((name) => name !== task.name))
    }
  }

  return (
    <>
      <div className="overflow-hidden rounded-lg border bg-card">
        <div className="border-b px-4 py-3 text-sm font-medium">
          {t('taskManager.tabs.tasks')}
        </div>
        <div className="overflow-x-auto">
          {error ? <p className="px-4 py-3 text-sm text-destructive">{error}</p> : null}
          {loading ? <p className="px-4 py-3 text-sm text-muted-foreground">{t('taskManager.loading')}</p> : null}
          <table className="min-w-full text-sm">
            <thead>
              <tr className="border-b bg-muted/40 text-left">
                <th className="px-3 py-2">#</th>
                <th className="px-3 py-2">{t('taskManager.columns.name')}</th>
                <th className="px-3 py-2">{t('taskManager.columns.description')}</th>
                <th className="px-3 py-2">{t('taskManager.columns.category')}</th>
                <th className="px-3 py-2">{t('taskManager.columns.type')}</th>
                <th className="px-3 py-2">{t('taskManager.columns.schedule')}</th>
                <th className="px-3 py-2">{t('taskManager.columns.lastRun')}</th>
                <th className="px-3 py-2">{t('taskManager.columns.lastStatus')}</th>
                <th className="px-3 py-2">{t('common.operation')}</th>
              </tr>
            </thead>
            <tbody>
              {tasks.map((task, index) => (
                <tr key={task.name} className="border-b align-top">
                  <td className="px-3 py-2">{index + 1}</td>
                  <td className="px-3 py-2 font-medium">{taskDisplayName(t, task)}</td>
                  <td className="px-3 py-2 text-muted-foreground">{taskDescription(t, task)}</td>
                  <td className="px-3 py-2">
                    <span
                      className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${categoryTone(
                        task.category
                      )}`}
                    >
                      {t(`taskManager.category.${task.category}`)}
                    </span>
                  </td>
                  <td className="px-3 py-2">
                    <span
                      className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${typeTone(
                        task.type
                      )}`}
                    >
                      {t(`taskManager.type.${task.type}`)}
                    </span>
                  </td>
                  <td className="px-3 py-2">
                    <div className="flex items-center gap-2">
                      {task.type === 'triggered' ? (
                        <span className="inline-flex rounded-full bg-slate-100 px-2 py-0.5 text-xs font-medium text-slate-700 dark:bg-slate-500/10 dark:text-slate-300">
                          {runTaskOptions(t, task)}
                        </span>
                      ) : (
                        <span className="font-mono text-xs">{runTaskOptions(t, task)}</span>
                      )}
                      {task.type === 'recurring' && canUpdateSchedule ? (
                        <Button type="button" size="sm" variant="outline" onClick={() => openScheduleDialog(task)}>
                          {t('taskManager.actions.editSchedule')}
                        </Button>
                      ) : null}
                    </div>
                  </td>
                  <td className="px-3 py-2">{formatDateTime(task.last_execution?.started_at)}</td>
                  <td className="px-3 py-2">
                    <span
                      className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${statusTone(
                        task.last_execution?.status
                      )}`}
                    >
                      {task.last_execution?.status
                        ? t(`taskManager.status.${task.last_execution.status}`)
                        : '-'}
                    </span>
                  </td>
                  <td className="px-3 py-2">
                    {task.runnable && canRun ? (
                      <Button
                        type="button"
                        size="sm"
                        variant="outline"
                        disabled={runningTaskNames.includes(task.name)}
                        onClick={() => void handleRunTask(task)}
                      >
                        {t('taskManager.actions.run')}
                      </Button>
                    ) : (
                      <span className="text-xs text-muted-foreground">
                        {t('taskManager.messages.eventTriggered')}
                      </span>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      <ShopDialog
        open={scheduleDialogOpen}
        title={t('taskManager.actions.editSchedule')}
        onClose={() => setScheduleDialogOpen(false)}
        closeLabel={t('common.close')}
        widthClass="max-w-xl"
        footer={
          <>
            <Button type="button" variant="outline" onClick={() => setScheduleDialogOpen(false)} disabled={scheduleSaving}>
              {t('common.cancel')}
            </Button>
            <Button type="button" onClick={() => void submitScheduleUpdate()} disabled={scheduleSaving}>
              {scheduleSaving ? t('taskManager.saving') : t('common.confirm')}
            </Button>
          </>
        }
      >
        <div className="space-y-4">
          <div className="text-sm">
            <span className="text-muted-foreground">{t('taskManager.columns.name')}: </span>
            <span>{scheduleForm.displayName}</span>
          </div>
          <div className="text-sm">
            <span className="text-muted-foreground">{t('taskManager.fields.defaultCron')}: </span>
            <span className="font-mono">{scheduleForm.defaultCron || '-'}</span>
          </div>
          <div className="flex flex-wrap gap-2">
            <Button
              type="button"
              variant={scheduleForm.mode === 'cron' ? 'default' : 'outline'}
              onClick={() => setScheduleForm((current) => ({ ...current, mode: 'cron' }))}
            >
              {t('taskManager.mode.cron')}
            </Button>
            <Button
              type="button"
              variant={scheduleForm.mode === 'every' ? 'default' : 'outline'}
              onClick={() => setScheduleForm((current) => ({ ...current, mode: 'every' }))}
            >
              {t('taskManager.mode.every')}
            </Button>
          </div>
          {scheduleForm.mode === 'every' ? (
            <div className="grid gap-4 md:grid-cols-2">
              <label className="space-y-2">
                <span className="text-sm text-muted-foreground">{t('taskManager.fields.intervalValue')}</span>
                <Input
                  type="number"
                  min={1}
                  step={1}
                  value={String(scheduleForm.intervalValue)}
                  onChange={(event) =>
                    setScheduleForm((current) => ({ ...current, intervalValue: Number(event.target.value) }))
                  }
                />
              </label>
              <label className="space-y-2">
                <span className="text-sm text-muted-foreground">{t('taskManager.fields.intervalUnit')}</span>
                <select
                  className="h-10 rounded-md border border-input bg-background px-3 text-sm"
                  value={scheduleForm.intervalUnit}
                  onChange={(event) =>
                    setScheduleForm((current) => ({
                      ...current,
                      intervalUnit: event.target.value as ScheduleIntervalUnit,
                    }))
                  }
                >
                  <option value="m">{t('taskManager.intervalUnits.m')}</option>
                  <option value="h">{t('taskManager.intervalUnits.h')}</option>
                </select>
              </label>
            </div>
          ) : (
            <label className="space-y-2 block">
              <span className="text-sm text-muted-foreground">{t('taskManager.fields.cronExpr')}</span>
              <Input
                value={scheduleForm.cronExpr}
                onChange={(event) => setScheduleForm((current) => ({ ...current, cronExpr: event.target.value }))}
                placeholder={t('taskManager.placeholders.cronExpr')}
              />
            </label>
          )}
        </div>
      </ShopDialog>
    </>
  )
}

function EsiStatusesPanel({ t }: { t: ReturnType<typeof useI18n>['t'] }) {
  const [tasks, setTasks] = useState<TaskInfo[]>([])
  const [loadingTasks, setLoadingTasks] = useState(true)
  const [loadingStatuses, setLoadingStatuses] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [statusRows, setStatusRows] = useState<TaskStatus[]>([])
  const [taskName, setTaskName] = useState('')
  const [character, setCharacter] = useState('')
  const [status, setStatus] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)
  const [total, setTotal] = useState(0)
  const [runningKeys, setRunningKeys] = useState<string[]>([])
  const [refreshSeed, setRefreshSeed] = useState(0)

  const loadTasks = async () => {
    setLoadingTasks(true)
    try {
      setTasks(await fetchESIRefreshTasks())
    } catch {
      setTasks([])
    } finally {
      setLoadingTasks(false)
    }
  }

  const loadStatuses = async () => {
    setLoadingStatuses(true)
    setError(null)
    try {
      const response = await fetchESIRefreshStatuses({
        current: page,
        size: pageSize,
        character: character.trim() || undefined,
        task_name: taskName || undefined,
        status: status || undefined,
      })
      setStatusRows(response.list ?? [])
      setTotal(response.total ?? 0)
      setPage(response.page ?? page)
      setPageSize(response.pageSize ?? pageSize)
    } catch (caughtError) {
      setError(getErrorMessage(caughtError, t('taskManager.esi.messages.loadStatusesFailed')))
      setStatusRows([])
      setTotal(0)
    } finally {
      setLoadingStatuses(false)
    }
  }

  useEffect(() => {
    void loadTasks()
  }, [])

  useEffect(() => {
    const timer = window.setTimeout(() => {
      void loadStatuses()
    }, 0)
    return () => window.clearTimeout(timer)
  }, [page, pageSize, refreshSeed, taskName, character, status])

  const pageCount = useMemo(() => Math.max(1, Math.ceil(total / pageSize) || 1), [pageSize, total])

  const handleRunTaskByName = async (task: TaskInfo) => {
    if (!window.confirm(t('taskManager.esi.confirm.runTaskByName', { name: task.description }))) {
      return
    }

    setError(null)
    try {
      await runESIRefreshTaskByName({ task_name: task.name })
      setRefreshSeed((current) => current + 1)
    } catch (caughtError) {
      setError(getErrorMessage(caughtError, t('taskManager.esi.messages.taskTriggerFailedAll')))
    }
  }

  const handleRunAll = async () => {
    if (!window.confirm(t('taskManager.confirm.runAllEsi'))) {
      return
    }

    setError(null)
    try {
      await runESIRefreshAll()
      setRefreshSeed((current) => current + 1)
    } catch (caughtError) {
      setError(getErrorMessage(caughtError, t('taskManager.esi.messages.refreshTriggerFailed')))
    }
  }

  const handleRunTask = async (row: TaskStatus) => {
    if (!window.confirm(t('taskManager.esi.confirm.runTaskByName', { name: row.description }))) {
      return
    }

    const key = `${row.task_name}_${row.character_id}`
    setRunningKeys((current) => [...current, key])
    setError(null)
    try {
      await runESIRefreshTask({ task_name: row.task_name, character_id: row.character_id })
      setRefreshSeed((current) => current + 1)
    } catch (caughtError) {
      setError(getErrorMessage(caughtError, t('taskManager.esi.messages.characterTaskTriggerFailed')))
    } finally {
      setRunningKeys((current) => current.filter((item) => item !== key))
    }
  }

  return (
    <div className="space-y-4">
      <div className="overflow-hidden rounded-lg border bg-card">
        <div className="border-b px-4 py-3 text-sm font-medium">
          {t('taskManager.esi.sections.tasks')}
        </div>
        <div className="flex flex-wrap items-center gap-3 px-4 py-4">
          <Button type="button" variant="outline" onClick={() => void handleRunAll()}>
            {t('taskManager.actions.runAllEsi')}
          </Button>
          {tasks.map((task) => (
            <Button key={task.name} type="button" variant="outline" onClick={() => void handleRunTaskByName(task)}>
              {task.description}
            </Button>
          ))}
        </div>
      </div>

      <div className="rounded-lg border bg-card p-4">
        <div className="flex flex-wrap items-center gap-3">
          <span className="text-sm font-medium">{t('taskManager.esi.sections.statuses')}</span>
          <Input
            className="w-60"
            value={character}
            onChange={(event) => setCharacter(event.target.value)}
            placeholder={t('taskManager.esi.filters.character')}
          />
          <select
            className="h-10 rounded-md border border-input bg-background px-3 text-sm"
            value={taskName}
            onChange={(event) => setTaskName(event.target.value)}
          >
            <option value="">{t('taskManager.esi.filters.taskName')}</option>
            {tasks.map((task) => (
              <option key={task.name} value={task.name}>
                {task.description}
              </option>
            ))}
          </select>
          <select
            className="h-10 rounded-md border border-input bg-background px-3 text-sm"
            value={status}
            onChange={(event) => setStatus(event.target.value)}
          >
            <option value="">{t('taskManager.esi.filters.status')}</option>
            <option value="pending">{t('taskManager.esi.status.pending')}</option>
            <option value="running">{t('taskManager.esi.status.running')}</option>
            <option value="success">{t('taskManager.esi.status.success')}</option>
            <option value="failed">{t('taskManager.esi.status.failed')}</option>
            <option value="skipped">{t('taskManager.esi.status.skipped')}</option>
          </select>
        </div>
      </div>

      <div className="overflow-hidden rounded-lg border bg-card">
        <div className="border-b px-4 py-3 text-sm font-medium">
          {t('taskManager.esi.sections.statuses')}
        </div>
        <div className="overflow-x-auto">
          {error ? <p className="px-4 py-3 text-sm text-destructive">{error}</p> : null}
          {(loadingTasks || loadingStatuses) ? (
            <p className="px-4 py-3 text-sm text-muted-foreground">{t('taskManager.loading')}</p>
          ) : null}
          <table className="min-w-full text-sm">
            <thead>
              <tr className="border-b bg-muted/40 text-left">
                <th className="px-3 py-2">#</th>
                <th className="px-3 py-2">{t('taskManager.columns.name')}</th>
                <th className="px-3 py-2">{t('taskManager.columns.description')}</th>
                <th className="px-3 py-2">{t('common.name')}</th>
                <th className="px-3 py-2">{t('taskManager.esi.columns.characterId')}</th>
                <th className="px-3 py-2">{t('taskManager.esi.columns.priority')}</th>
                <th className="px-3 py-2">{t('taskManager.columns.status')}</th>
                <th className="px-3 py-2">{t('taskManager.columns.lastRun')}</th>
                <th className="px-3 py-2">{t('taskManager.esi.columns.nextRun')}</th>
                <th className="px-3 py-2">{t('taskManager.columns.error')}</th>
                <th className="px-3 py-2">{t('common.operation')}</th>
              </tr>
            </thead>
            <tbody>
              {statusRows.map((row, index) => {
                const key = `${row.task_name}_${row.character_id}`
                return (
                  <tr key={key} className="border-b align-top">
                    <td className="px-3 py-2">{index + 1}</td>
                    <td className="px-3 py-2">{row.task_name}</td>
                    <td className="px-3 py-2">{row.description}</td>
                    <td className="px-3 py-2">{row.character_name || '-'}</td>
                    <td className="px-3 py-2">{row.character_id}</td>
                    <td className="px-3 py-2">
                      <span
                        className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${priorityTone(
                          row.priority
                        )}`}
                      >
                        {priorityLabel(t, row.priority)}
                      </span>
                    </td>
                    <td className="px-3 py-2">
                      <span
                        className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${taskStatusTone(
                          row.status
                        )}`}
                      >
                        {t(`taskManager.esi.status.${row.status}`)}
                      </span>
                    </td>
                    <td className="px-3 py-2">{formatDateTime(row.last_run)}</td>
                    <td className="px-3 py-2">{formatDateTime(row.next_run)}</td>
                    <td className="px-3 py-2 text-xs text-destructive">{row.error || '-'}</td>
                    <td className="px-3 py-2">
                      <Button
                        type="button"
                        size="sm"
                        variant="outline"
                        disabled={runningKeys.includes(key)}
                        onClick={() => void handleRunTask(row)}
                      >
                        {t('taskManager.actions.run')}
                      </Button>
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>
      </div>

      <div className="flex flex-wrap items-center gap-3 text-sm">
        <span>
          {page}/{pageCount}
        </span>
        <Button type="button" size="sm" variant="outline" onClick={() => setPage((current) => Math.max(1, current - 1))} disabled={page <= 1}>
          {t('welfareMy.pagination.prev')}
        </Button>
        <Button
          type="button"
          size="sm"
          variant="outline"
          onClick={() => setPage((current) => current + 1)}
          disabled={statusRows.length < pageSize || page * pageSize >= total}
        >
          {t('welfareMy.pagination.next')}
        </Button>
        <label className="flex items-center gap-2">
          <span>{t('welfareMy.pageSize')}</span>
          <select
            className="h-8 rounded-md border border-input bg-background px-2 text-sm"
            value={pageSize}
            onChange={(event) => {
              setPageSize(Number(event.target.value))
              setPage(1)
            }}
          >
            {[10, 20, 50].map((size) => (
              <option key={size} value={size}>
                {size}
              </option>
            ))}
          </select>
        </label>
      </div>
    </div>
  )
}

function HistoryPanel({ t }: { t: ReturnType<typeof useI18n>['t'] }) {
  const [taskOptions, setTaskOptions] = useState<TaskItem[]>([])
  const [loadingTasks, setLoadingTasks] = useState(true)
  const [loadingHistory, setLoadingHistory] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [rows, setRows] = useState<TaskExecutionItem[]>([])
  const [taskName, setTaskName] = useState('')
  const [status, setStatus] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)
  const [total, setTotal] = useState(0)

  const loadTasks = async () => {
    setLoadingTasks(true)
    try {
      setTaskOptions(await fetchTasks())
    } catch {
      setTaskOptions([])
    } finally {
      setLoadingTasks(false)
    }
  }

  const loadHistory = async () => {
    setLoadingHistory(true)
    setError(null)
    try {
      const response = await fetchTaskHistory({
        current: page,
        size: pageSize,
        task_name: taskName || undefined,
        status: status || undefined,
      })
      setRows(response.list ?? [])
      setTotal(response.total ?? 0)
      setPage(response.page ?? page)
      setPageSize(response.pageSize ?? pageSize)
    } catch (caughtError) {
      setError(getErrorMessage(caughtError, t('taskManager.loadHistoryFailed')))
      setRows([])
      setTotal(0)
    } finally {
      setLoadingHistory(false)
    }
  }

  useEffect(() => {
    void loadTasks()
  }, [])

  useEffect(() => {
    const timer = window.setTimeout(() => {
      void loadHistory()
    }, 0)
    return () => window.clearTimeout(timer)
  }, [page, pageSize, taskName, status])

  const pageCount = useMemo(() => Math.max(1, Math.ceil(total / pageSize) || 1), [pageSize, total])

  return (
    <div className="space-y-4">
      <div className="rounded-lg border bg-card p-4">
        <div className="flex flex-wrap items-center gap-3">
          <span className="text-sm font-medium">{t('taskManager.tabs.history')}</span>
          <select
            className="h-10 rounded-md border border-input bg-background px-3 text-sm"
            value={taskName}
            onChange={(event) => setTaskName(event.target.value)}
          >
            <option value="">{t('taskManager.filters.taskName')}</option>
            {taskOptions.map((task) => (
              <option key={task.name} value={task.name}>
                {taskDisplayName(t, task)}
              </option>
            ))}
          </select>
          <select
            className="h-10 rounded-md border border-input bg-background px-3 text-sm"
            value={status}
            onChange={(event) => setStatus(event.target.value)}
          >
            <option value="">{t('taskManager.filters.status')}</option>
            <option value="running">{t('taskManager.status.running')}</option>
            <option value="success">{t('taskManager.status.success')}</option>
            <option value="failed">{t('taskManager.status.failed')}</option>
          </select>
        </div>
      </div>

      <div className="overflow-hidden rounded-lg border bg-card">
        <div className="border-b px-4 py-3 text-sm font-medium">
          {t('taskManager.tabs.history')}
        </div>
        <div className="overflow-x-auto">
          {error ? <p className="px-4 py-3 text-sm text-destructive">{error}</p> : null}
          {(loadingTasks || loadingHistory) ? (
            <p className="px-4 py-3 text-sm text-muted-foreground">{t('taskManager.loading')}</p>
          ) : null}
          <table className="min-w-full text-sm">
            <thead>
              <tr className="border-b bg-muted/40 text-left">
                <th className="px-3 py-2">#</th>
                <th className="px-3 py-2">{t('taskManager.columns.name')}</th>
                <th className="px-3 py-2">{t('taskManager.columns.trigger')}</th>
                <th className="px-3 py-2">{t('taskManager.columns.triggeredBy')}</th>
                <th className="px-3 py-2">{t('taskManager.columns.triggeredById')}</th>
                <th className="px-3 py-2">{t('taskManager.columns.status')}</th>
                <th className="px-3 py-2">{t('taskManager.columns.startedAt')}</th>
                <th className="px-3 py-2">{t('taskManager.columns.duration')}</th>
                <th className="px-3 py-2">{t('taskManager.columns.error')}</th>
                <th className="px-3 py-2">{t('taskManager.columns.summary')}</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((row, index) => (
                <tr key={row.id} className="border-b align-top">
                  <td className="px-3 py-2">{index + 1}</td>
                  <td className="px-3 py-2">{taskDisplayName(t, { name: row.task_name })}</td>
                  <td className="px-3 py-2">
                    <span
                      className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${triggerTone(
                        row.trigger
                      )}`}
                    >
                      {t(`taskManager.trigger.${row.trigger}`)}
                    </span>
                  </td>
                  <td className="px-3 py-2">{row.triggered_by_name || '-'}</td>
                  <td className="px-3 py-2">{row.triggered_by ?? '-'}</td>
                  <td className="px-3 py-2">
                    <span
                      className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${taskStatusTone(
                        row.status
                      )}`}
                    >
                      {t(`taskManager.status.${row.status}`)}
                    </span>
                  </td>
                  <td className="px-3 py-2">{formatDateTime(row.started_at)}</td>
                  <td className="px-3 py-2">{formatDuration(row.duration_ms)}</td>
                  <td className="px-3 py-2 text-xs text-destructive">{row.error || '-'}</td>
                  <td className="px-3 py-2">{row.summary || '-'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      <div className="flex flex-wrap items-center gap-3 text-sm">
        <span>
          {page}/{pageCount}
        </span>
        <Button type="button" size="sm" variant="outline" onClick={() => setPage((current) => Math.max(1, current - 1))} disabled={page <= 1}>
          {t('welfareMy.pagination.prev')}
        </Button>
        <Button
          type="button"
          size="sm"
          variant="outline"
          onClick={() => setPage((current) => current + 1)}
          disabled={rows.length < pageSize || page * pageSize >= total}
        >
          {t('welfareMy.pagination.next')}
        </Button>
        <label className="flex items-center gap-2">
          <span>{t('welfareMy.pageSize')}</span>
          <select
            className="h-8 rounded-md border border-input bg-background px-2 text-sm"
            value={pageSize}
            onChange={(event) => {
              setPageSize(Number(event.target.value))
              setPage(1)
            }}
          >
            {[10, 20, 50].map((size) => (
              <option key={size} value={size}>
                {size}
              </option>
            ))}
          </select>
        </label>
      </div>
    </div>
  )
}
