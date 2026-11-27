import { useCallback, useState } from 'react'
import type { ColumnDef } from '@tanstack/react-table'
import {
  cancelQQOfficialTask,
  decideQQOfficialJoinRequest,
  getQQOfficialTask,
  listQQOfficialGroups,
  listQQOfficialJoinRequests,
  listQQOfficialTasks,
  refreshQQOfficialJoinRequests,
  testQQOfficialGroup,
  updateQQOfficialGroup,
  verifyQQOfficialGroup,
} from '@/api/generated/client'
import type {
  QQOfficialGroup,
  QQOfficialGroupUpdate,
  QQOfficialJoinDecision,
  QQOfficialJoinRequest,
  QQOfficialJoinRequestStatus,
  QQOfficialStatus,
  QQOfficialTask,
  QQOfficialTaskAttempt,
} from '@/api/generated/client'
import { unwrapContract } from '@/api/response'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { DataTable, DEFAULT_LEDGER_PAGE_SIZE } from '@/components/ui/data-table'
import { Dialog, DialogFooter, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { notifyError, notifySuccess } from '@/feedback'
import { useQQOfficialRead, type useQQOfficialActions } from '@/hooks/use-qq-official-console'
import { useI18n } from '@/i18n'
import { formatTime } from '@/lib/format'
import {
  canCancelQQTask,
  canDecideQQRequest,
  isQQDispatchReady,
  isQQGroupOpenID,
  parseQQCorporationTarget,
  qqStateLabel,
  requestStatuses,
  shouldPollQQTask,
  taskKinds,
  taskSources,
  taskStatuses,
} from '@/lib/qq-official'

type Actions = ReturnType<typeof useQQOfficialActions>
type OpenTask = (id: string) => void
type T = ReturnType<typeof useI18n>['t']
const panelClass = 'space-y-3 rounded-lg border bg-card p-4'
const toolbarClass = 'flex flex-wrap items-center justify-between gap-2'

function tableText(t: T, read: { error: boolean; loading: boolean }) {
  return {
    loading: read.loading,
    error: read.error ? t('qqOfficial.loadFailed') : null,
    loadingText: t('common.loading'),
    emptyText: t('common.empty'),
  }
}

function pager(
  t: T,
  page: number,
  pageSize: number,
  total: number,
  setPage: (page: number) => void,
  setSize?: (size: number) => void
) {
  return {
    page,
    pageSize,
    total,
    onPageChange: setPage,
    onPageSizeChange: (size: number) => {
      setSize?.(size)
      setPage(1)
    },
    previousLabel: t('common.previous'),
    nextLabel: t('common.next'),
    pageSizeLabel: t('common.pageSize'),
  }
}

function QQFilter<V extends string>({
  label,
  values,
  value,
  change,
}: {
  label: string
  values: readonly V[]
  value: V | undefined
  change: (value: V | undefined) => void
}) {
  const { t } = useI18n()
  return (
    <div className="grid gap-1">
      <span className="text-xs text-muted-foreground">{label}</span>
      <Select
        aria-label={label}
        selectedKey={value ?? 'all'}
        onSelectionChange={(key) => change(values.find((entry) => entry === key))}
      >
        <SelectTrigger>
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          <SelectItem id="all">{t('qqOfficial.all')}</SelectItem>
          {values.map((entry) => (
            <SelectItem key={entry} id={entry}>
              {qqStateLabel(t, entry)}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  )
}

export function QQConfirm({
  open,
  title,
  message,
  busy,
  close,
  confirm,
}: {
  open: boolean
  title: string
  message: string
  busy: boolean
  close: () => void
  confirm: () => void
}) {
  const { t } = useI18n()
  return (
    <Dialog
      isOpen={open}
      showCloseButton={!busy}
      isDismissable={!busy}
      onOpenChange={(next) => {
        if (!next && !busy) close()
      }}
    >
      <DialogTitle>{title}</DialogTitle>
      <p>{message}</p>
      <DialogFooter>
        <Button variant="outline" isDisabled={busy} onPress={close}>
          {t('common.cancel')}
        </Button>
        <Button variant="destructive" isDisabled={busy} onPress={confirm}>
          {t('common.confirm')}
        </Button>
      </DialogFooter>
    </Dialog>
  )
}

function QQGroupEditor({
  group,
  approvalWrites,
  actions,
  close,
  reload,
}: {
  group: QQOfficialGroup
  approvalWrites: boolean
  actions: Actions
  close: () => void
  reload: () => void
}) {
  const { t } = useI18n()
  const [alias, setAlias] = useState(group.alias)
  const [flags, setFlags] = useState({
    authorized: group.authorized,
    notification_enabled: group.notification_enabled,
    approval_enabled: group.approval_enabled,
  })
  const [fleet, setFleet] = useState(String(group.fleet_corporation_id ?? ''))
  const [structure, setStructure] = useState(String(group.structure_corporation_id ?? ''))
  const fleetID = parseQQCorporationTarget(fleet)
  const structureID = parseQQCorporationTarget(structure)
  const canEnable = {
    authorized: group.present,
    notification_enabled: flags.authorized && group.test_verified && group.push_state === 'allowed',
    approval_enabled:
      flags.authorized && approvalWrites && group.approval_capability === 'available',
  }
  const patch: QQOfficialGroupUpdate = {}
  if (alias.trim() !== group.alias) patch.alias = alias.trim()
  for (const key of ['authorized', 'notification_enabled', 'approval_enabled'] as const) {
    if (flags[key] !== group[key]) patch[key] = flags[key]
  }
  if (fleetID !== (group.fleet_corporation_id ?? 0)) patch.fleet_corporation_id = fleetID
  if (structureID !== (group.structure_corporation_id ?? 0))
    patch.structure_corporation_id = structureID
  const save = async () => {
    const result = await actions.run(() => unwrapContract(updateQQOfficialGroup(group.id, patch)))
    if (!result) return
    notifySuccess(t('qqOfficial.saved'))
    close()
    reload()
  }
  return (
    <Dialog
      isOpen
      isDismissable={!actions.busy}
      showCloseButton={!actions.busy}
      onOpenChange={(open) => {
        if (!open && !actions.busy) close()
      }}
    >
      <DialogTitle>{t('qqOfficial.editGroup')}</DialogTitle>
      <label className="grid gap-1">
        {t('qqOfficial.alias')}
        <Input maxLength={80} value={alias} onChange={(e) => setAlias(e.target.value)} />
      </label>
      {(
        [
          ['authorized', 'optIn'],
          ['notification_enabled', 'notifications'],
          ['approval_enabled', 'approval'],
        ] as const
      ).map(([key, label]) => (
        <Checkbox
          key={key}
          isSelected={flags[key]}
          isDisabled={actions.busy || (!flags[key] && !canEnable[key])}
          onChange={(value) =>
            setFlags(
              key === 'authorized' && !value
                ? { authorized: false, notification_enabled: false, approval_enabled: false }
                : { ...flags, [key]: value }
            )
          }
        >
          {t(`qqOfficial.${label}`)}
        </Checkbox>
      ))}
      <label className="grid gap-1">
        {t('qqOfficial.fleetTarget')}
        <Input inputMode="numeric" value={fleet} onChange={(e) => setFleet(e.target.value)} />
      </label>
      <label className="grid gap-1">
        {t('qqOfficial.structureTarget')}
        <Input
          inputMode="numeric"
          value={structure}
          onChange={(e) => setStructure(e.target.value)}
        />
      </label>
      <p className="text-muted-foreground">{t('qqOfficial.targetHint')}</p>
      <DialogFooter>
        <Button variant="outline" onPress={close} isDisabled={actions.busy}>
          {t('common.cancel')}
        </Button>
        <Button
          onPress={() => void save()}
          isDisabled={
            actions.busy ||
            fleetID === undefined ||
            structureID === undefined ||
            (flags.authorized && (!group.present || !alias.trim())) ||
            Object.keys(patch).length === 0
          }
        >
          {t('common.save')}
        </Button>
      </DialogFooter>
    </Dialog>
  )
}

export function QQGroupPanel({
  status,
  actions,
  openTask,
  openRequests,
}: {
  status: QQOfficialStatus | null
  actions: Actions
  openTask: OpenTask
  openRequests: (group: QQOfficialGroup) => void
}) {
  const { t } = useI18n()
  const [page, setPage] = useState(1)
  const [size, setSize] = useState(20)
  const [editing, setEditing] = useState<QQOfficialGroup>()
  const [verify, setVerify] = useState(false)
  const [openid, setOpenid] = useState('')
  const read = useQQOfficialRead(
    useCallback(
      (signal: AbortSignal) =>
        unwrapContract(listQQOfficialGroups({ current: page, size }, { signal })),
      [page, size]
    )
  )
  const ready = isQQDispatchReady(status)
  const columnFlags = [
    ['present', 'present'],
    ['authorized', 'optIn'],
    ['notification_enabled', 'notifications'],
    ['test_verified', 'testVerified'],
    ['approval_enabled', 'approval'],
  ] as const
  const queueTest = async (id: string) => {
    const receipt = await actions.enqueue(`test:${id}`, (key) =>
      testQQOfficialGroup(id, { idempotency_key: key })
    )
    if (receipt) openTask(receipt.task_id)
  }
  const verifyGroup = async () => {
    const receipt = await actions.enqueue('verify', (key) =>
      verifyQQOfficialGroup({ group_openid: openid.trim(), idempotency_key: key })
    )
    if (receipt) {
      setVerify(false)
      setOpenid('')
      openTask(receipt.task_id)
    }
  }
  const closeVerify = () => {
    setVerify(false)
    setOpenid('')
    actions.clearKey('verify')
  }
  const columns: ColumnDef<QQOfficialGroup>[] = [
    {
      id: 'alias',
      header: t('qqOfficial.alias'),
      cell: ({ row }) => (
        <>
          <div>{row.original.alias || '—'}</div>
          <div className="text-xs text-muted-foreground">{row.original.id}</div>
        </>
      ),
    },
    ...columnFlags.map(
      ([key, label]): ColumnDef<QQOfficialGroup> => ({
        id: key,
        header: t(`qqOfficial.${label}`),
        cell: ({ row }) => t(row.original[key] ? 'qqOfficial.yes' : 'qqOfficial.no'),
      })
    ),
    {
      id: 'push',
      header: t('qqOfficial.pushState'),
      cell: ({ row }) => qqStateLabel(t, row.original.push_state),
    },
    {
      id: 'capability',
      header: t('qqOfficial.approvalCapability'),
      cell: ({ row }) => qqStateLabel(t, row.original.approval_capability),
    },
    {
      id: 'actions',
      header: t('common.operation'),
      cell: ({ row: { original: group } }) => (
        <div className="flex flex-wrap gap-1">
          {actions.canManage && (
            <>
              <Button
                variant="ghost"
                size="sm"
                isDisabled={actions.busy}
                onPress={() => setEditing(group)}
              >
                {t('qqOfficial.editGroup')}
              </Button>
              <Button
                variant="ghost"
                size="sm"
                isDisabled={
                  actions.busy ||
                  !ready ||
                  !group.present ||
                  !group.authorized ||
                  group.push_state === 'blocked'
                }
                onPress={() => void queueTest(group.id)}
              >
                {t('qqOfficial.sendTest')}
              </Button>
            </>
          )}
          <Button variant="ghost" size="sm" onPress={() => openRequests(group)}>
            {t('qqOfficial.requests')}
          </Button>
        </div>
      ),
    },
  ]
  return (
    <div className={panelClass}>
      <div className={toolbarClass}>
        <h2 className="font-semibold">{t('qqOfficial.groups')}</h2>
        <div className="flex flex-wrap gap-2">
          <Button variant="outline" onPress={read.reload}>
            {t('qqOfficial.localReload')}
          </Button>
          {actions.canManage && (
            <Button isDisabled={!ready || actions.busy} onPress={() => setVerify(true)}>
              {t('qqOfficial.verifyGroup')}
            </Button>
          )}
        </div>
      </div>
      <DataTable
        columns={columns}
        data={read.data?.list ?? []}
        getRowId={(row) => row.id}
        {...tableText(t, read)}
        pagination={pager(t, page, size, read.data?.total ?? 0, setPage, setSize)}
      />
      {editing && (
        <QQGroupEditor
          key={editing.id}
          group={editing}
          approvalWrites={status?.approval_writes_enabled === true}
          actions={actions}
          close={() => setEditing(undefined)}
          reload={read.reload}
        />
      )}
      <Dialog
        isOpen={verify}
        showCloseButton={!actions.busy}
        isDismissable={!actions.busy}
        onOpenChange={(open) => {
          if (!open && !actions.busy) closeVerify()
        }}
      >
        <DialogTitle>{t('qqOfficial.verifyGroup')}</DialogTitle>
        <p>{t('qqOfficial.verifyHint')}</p>
        <label className="grid gap-1">
          {t('qqOfficial.openid')}
          <Input
            autoComplete="off"
            disabled={actions.busy}
            maxLength={128}
            value={openid}
            onChange={(e) => {
              setOpenid(e.target.value)
              actions.clearKey('verify')
            }}
          />
        </label>
        <DialogFooter>
          <Button variant="outline" isDisabled={actions.busy} onPress={closeVerify}>
            {t('common.cancel')}
          </Button>
          <Button
            isDisabled={!ready || actions.busy || !isQQGroupOpenID(openid)}
            onPress={() => void verifyGroup()}
          >
            {t('qqOfficial.verifyGroup')}
          </Button>
        </DialogFooter>
      </Dialog>
    </div>
  )
}

export function QQTasksPanel({ openTask }: { openTask: OpenTask }) {
  const { t } = useI18n()
  const [page, setPage] = useState(1)
  const [status, setStatus] = useState<(typeof taskStatuses)[number]>()
  const [kind, setKind] = useState<(typeof taskKinds)[number]>()
  const [source, setSource] = useState<(typeof taskSources)[number]>()
  const read = useQQOfficialRead(
    useCallback(
      (signal: AbortSignal) =>
        unwrapContract(
          listQQOfficialTasks(
            { current: page, size: DEFAULT_LEDGER_PAGE_SIZE, status, kind, source },
            { signal }
          )
        ),
      [kind, page, source, status]
    )
  )
  const columns: ColumnDef<QQOfficialTask>[] = [
    {
      id: 'id',
      header: t('qqOfficial.taskId'),
      cell: ({ row }) => (
        <Button variant="link" size="sm" onPress={() => openTask(row.original.id)}>
          {row.original.id}
        </Button>
      ),
    },
    ...(['kind', 'source', 'status', 'reason'] as const).map(
      (key): ColumnDef<QQOfficialTask> => ({
        id: key,
        header: t(`qqOfficial.${key}`),
        cell: ({ row }) => qqStateLabel(t, row.original[key]),
      })
    ),
    {
      id: 'created',
      header: t('qqOfficial.created'),
      cell: ({ row }) => formatTime(row.original.created_at),
    },
    {
      id: 'expires',
      header: t('qqOfficial.expires'),
      cell: ({ row }) => formatTime(row.original.expires_at),
    },
  ]
  return (
    <div className={panelClass}>
      <div className={toolbarClass}>
        <div className="flex flex-wrap gap-2">
          <QQFilter
            label={t('qqOfficial.status')}
            values={taskStatuses}
            value={status}
            change={(value) => {
              setPage(1)
              setStatus(value)
            }}
          />
          <QQFilter
            label={t('qqOfficial.kind')}
            values={taskKinds}
            value={kind}
            change={(value) => {
              setPage(1)
              setKind(value)
            }}
          />
          <QQFilter
            label={t('qqOfficial.source')}
            values={taskSources}
            value={source}
            change={(value) => {
              setPage(1)
              setSource(value)
            }}
          />
        </div>
        <Button variant="outline" onPress={read.reload}>
          {t('qqOfficial.localReload')}
        </Button>
      </div>
      <DataTable
        columns={columns}
        data={read.data?.list ?? []}
        getRowId={(row) => row.id}
        {...tableText(t, read)}
        variant="ledger"
        pagination={pager(t, page, DEFAULT_LEDGER_PAGE_SIZE, read.data?.total ?? 0, setPage)}
      />
    </div>
  )
}

const pollTask = (task: QQOfficialTask) => shouldPollQQTask(task.status)

export function QQTaskDetail({
  id,
  actions,
  close,
}: {
  id: string
  actions: Actions
  close: () => void
}) {
  const { t } = useI18n()
  const [confirm, setConfirm] = useState(false)
  const readTask = useCallback(
    (signal: AbortSignal) => unwrapContract(getQQOfficialTask(id, { signal })),
    [id]
  )
  const read = useQQOfficialRead(readTask, 10_000, pollTask)
  const task = read.data
  const cancel = async () => {
    const result = await actions.run(() => unwrapContract(cancelQQOfficialTask(id)))
    if (result) {
      read.replace(result)
      setConfirm(false)
    } else read.reload()
  }
  const attemptsColumns: ColumnDef<QQOfficialTaskAttempt>[] = [
    { id: 'at', header: t('qqOfficial.created'), cell: ({ row }) => formatTime(row.original.at) },
    {
      id: 'finished',
      header: t('qqOfficial.finished'),
      cell: ({ row }) => formatTime(row.original.finished_at),
    },
    {
      id: 'outcome',
      header: t('qqOfficial.outcome'),
      cell: ({ row }) => qqStateLabel(t, row.original.outcome),
    },
    { id: 'code', header: t('qqOfficial.platformCode'), accessorKey: 'platform_code' },
    { id: 'http', header: t('qqOfficial.httpStatus'), accessorKey: 'http_status' },
    { id: 'trace', header: t('qqOfficial.trace'), accessorKey: 'trace_id' },
  ]
  return (
    <Dialog
      isOpen
      isDismissable={!actions.busy}
      showCloseButton={!actions.busy}
      onOpenChange={(open) => {
        if (!open && !actions.busy) close()
      }}
      className="max-h-[90dvh] overflow-y-auto sm:max-w-4xl"
    >
      <DialogTitle>{t('qqOfficial.taskDetail')}</DialogTitle>
      {read.error && (
        <p role="alert" className="text-destructive">
          {t('qqOfficial.loadFailed')}
        </p>
      )}
      {read.loading && <p>{t('common.loading')}</p>}
      {task && (
        <>
          <dl className="grid gap-2 sm:grid-cols-3">
            {[
              ['taskId', task.id],
              ['groupId', task.group_id],
              ['kind', qqStateLabel(t, task.kind)],
              ['source', qqStateLabel(t, task.source)],
              ['status', qqStateLabel(t, task.status)],
              ['reason', qqStateLabel(t, task.reason)],
              ['trace', task.trace_id],
              ['platformCode', task.upstream_code],
              ['created', formatTime(task.created_at)],
              ['expires', formatTime(task.expires_at)],
              ['nextRun', formatTime(task.next_run_at)],
              ['finished', formatTime(task.finished_at)],
            ].map(([label, value]) => (
              <div key={label}>
                <dt className="text-muted-foreground">{t(`qqOfficial.${label}`)}</dt>
                <dd className="break-all">{value ?? '—'}</dd>
              </div>
            ))}
          </dl>
          <h3 className="font-semibold">{t('qqOfficial.attempts')}</h3>
          <DataTable
            columns={attemptsColumns}
            data={task.attempts}
            loadingText={t('common.loading')}
            emptyText={t('common.empty')}
          />
        </>
      )}
      {confirm && <p role="alert">{t('qqOfficial.cancelConfirm')}</p>}
      <DialogFooter>
        {confirm ? (
          <>
            <Button variant="outline" isDisabled={actions.busy} onPress={() => setConfirm(false)}>
              {t('common.cancel')}
            </Button>
            <Button
              variant="destructive"
              isDisabled={actions.busy || !task || !canCancelQQTask(task.status)}
              onPress={() => void cancel()}
            >
              {t('common.confirm')}
            </Button>
          </>
        ) : (
          <>
            <Button variant="outline" onPress={read.reload}>
              {t('qqOfficial.localReload')}
            </Button>
            {actions.canManage && task && canCancelQQTask(task.status) && (
              <Button
                variant="destructive"
                isDisabled={actions.busy}
                onPress={() => setConfirm(true)}
              >
                {t('qqOfficial.cancelTask')}
              </Button>
            )}
          </>
        )}
      </DialogFooter>
    </Dialog>
  )
}

export function QQRequestsPanel({
  group,
  status,
  actions,
  openTask,
}: {
  group: QQOfficialGroup
  status: QQOfficialStatus | null
  actions: Actions
  openTask: OpenTask
}) {
  const { t } = useI18n()
  const [page, setPage] = useState(1)
  const [filter, setFilter] = useState<QQOfficialJoinRequestStatus>()
  const [decision, setDecision] = useState<{
    request: QQOfficialJoinRequest
    action: QQOfficialJoinDecision
  }>()
  const read = useQQOfficialRead(
    useCallback(
      (signal: AbortSignal) =>
        unwrapContract(
          listQQOfficialJoinRequests(
            group.id,
            { current: page, size: DEFAULT_LEDGER_PAGE_SIZE, status: filter },
            { signal }
          )
        ),
      [filter, group.id, page]
    )
  )
  const refresh = async () => {
    const receipt = await actions.enqueue(`refresh:${group.id}`, (key) =>
      refreshQQOfficialJoinRequests(group.id, { idempotency_key: key })
    )
    if (receipt) openTask(receipt.task_id)
  }
  const submitDecision = async () => {
    if (!decision || !canDecideQQRequest(status, group, decision.request)) {
      notifyError(t('qqOfficial.conflict'))
      return
    }
    const { request, action } = decision
    const receipt = await actions.enqueue(`decision:${request.id}:${action}`, (key) =>
      decideQQOfficialJoinRequest(request.id, { decision: action, idempotency_key: key })
    )
    if (receipt) {
      setDecision(undefined)
      openTask(receipt.task_id)
    }
    read.reload()
  }
  const columns: ColumnDef<QQOfficialJoinRequest>[] = [
    { accessorKey: 'username', header: t('qqOfficial.username') },
    {
      id: 'applied',
      header: t('qqOfficial.applied'),
      cell: ({ row }) => formatTime(row.original.applied_at),
    },
    {
      id: 'expires',
      header: t('qqOfficial.expires'),
      cell: ({ row }) => formatTime(row.original.expires_at),
    },
    {
      id: 'status',
      header: t('qqOfficial.status'),
      cell: ({ row }) => qqStateLabel(t, row.original.status),
    },
    {
      id: 'actions',
      header: t('common.operation'),
      cell: ({ row: { original: request } }) => (
        <div className="flex flex-wrap gap-1">
          {request.task_id && (
            <Button
              variant="ghost"
              size="sm"
              onPress={() => request.task_id && openTask(request.task_id)}
            >
              {t('qqOfficial.taskDetail')}
            </Button>
          )}
          {actions.canManage &&
            canDecideQQRequest(status, group, request) &&
            (['approve', 'decline'] as const).map((action) => (
              <Button
                key={action}
                variant="outline"
                size="sm"
                isDisabled={actions.busy}
                onPress={() => setDecision({ request, action })}
              >
                {t(`qqOfficial.${action}`)}
              </Button>
            ))}
        </div>
      ),
    },
  ]
  return (
    <div className={panelClass}>
      <div className={toolbarClass}>
        <h2 className="font-semibold">
          {t('qqOfficial.requests')} · {group.alias || group.id}
        </h2>
        <div className="flex flex-wrap gap-2">
          <QQFilter
            label={t('qqOfficial.status')}
            values={requestStatuses}
            value={filter}
            change={(value) => {
              setPage(1)
              setFilter(value)
            }}
          />
          <Button variant="outline" onPress={read.reload}>
            {t('qqOfficial.localReload')}
          </Button>
          {actions.canManage && (
            <Button
              isDisabled={
                actions.busy || !isQQDispatchReady(status) || !group.present || !group.authorized
              }
              onPress={() => void refresh()}
            >
              {t('qqOfficial.requestRefresh')}
            </Button>
          )}
        </div>
      </div>
      {status && !status.approval_writes_enabled && (
        <p className="text-muted-foreground">{t('qqOfficial.writesOff')}</p>
      )}
      <DataTable
        columns={columns}
        data={read.data?.list ?? []}
        getRowId={(row) => row.id}
        {...tableText(t, read)}
        variant="ledger"
        pagination={pager(t, page, DEFAULT_LEDGER_PAGE_SIZE, read.data?.total ?? 0, setPage)}
      />
      {decision && (
        <QQConfirm
          open
          title={t(`qqOfficial.${decision.action}`)}
          message={t(`qqOfficial.${decision.action}Confirm`, { name: decision.request.username })}
          busy={actions.busy}
          close={() => setDecision(undefined)}
          confirm={() => void submitDecision()}
        />
      )}
    </div>
  )
}
