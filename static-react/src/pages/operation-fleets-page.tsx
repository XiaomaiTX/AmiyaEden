import { useCallback, useEffect, useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { fetchMyCharacters } from '@/api/auth'
import {
  createFleet,
  deleteFleet,
  fetchFleetList,
  refreshFleetESI,
  issuePap,
  syncESIFleetMembers,
  updateFleet,
} from '@/api/fleet'
import { fetchFleetConfigList } from '@/api/fleet-config'
import { Button } from '@/components/ui/button'
import { DataTable, type ColumnDef } from '@/components/ui/data-table'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Textarea } from '@/components/ui/textarea'
import { Checkbox } from '@/components/ui/checkbox'
import { useI18n } from '@/i18n'
import { useSessionStore } from '@/stores'
import type { EveCharacter } from '@/types/api/auth'
import type { FleetAutoSrpMode, FleetImportance, FleetItem } from '@/types/api/fleet'
import type { FleetConfigItem } from '@/types/api/fleet-config'
import { formatDateTime, getErrorMessage, ShopBadge, ShopDialog } from './shop-page-utils'

type FleetFormState = {
  title: string
  description: string
  importance: FleetImportance
  pap_count: number
  character_id: number | ''
  start_at: string
  end_at: string
  fleet_config_id: number | ''
  send_ping: boolean
  auto_srp_mode: FleetAutoSrpMode
}

const defaultFormState: FleetFormState = {
  title: '',
  description: '',
  importance: 'other',
  pap_count: 1,
  character_id: '',
  start_at: '',
  end_at: '',
  fleet_config_id: '',
  send_ping: true,
  auto_srp_mode: 'disabled',
}

function toLocalInputValue(value: string) {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ''
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`
}

function toIsoFromLocalInput(value: string) {
  if (!value) return ''
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toISOString()
}

function importanceBadgeClass(value: FleetImportance) {
  switch (value) {
    case 'cta':
      return 'bg-amber-100 text-amber-700 dark:bg-amber-500/10 dark:text-amber-300'
    case 'strat_op':
      return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-300'
    default:
      return 'bg-slate-100 text-slate-700 dark:bg-slate-500/10 dark:text-slate-300'
  }
}

export function OperationFleetsPage() {
  const { t } = useI18n()
  const navigate = useNavigate()
  const roles = useSessionStore((state) => state.roles)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [fleets, setFleets] = useState<FleetItem[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)
  const [importance, setImportance] = useState('')
  const [refreshSeed, setRefreshSeed] = useState(0)
  const [characters, setCharacters] = useState<EveCharacter[]>([])
  const [fleetConfigs, setFleetConfigs] = useState<FleetConfigItem[]>([])
  const [dialogOpen, setDialogOpen] = useState(false)
  const [saving, setSaving] = useState(false)
  const [editingFleet, setEditingFleet] = useState<FleetItem | null>(null)
  const [papFleetId, setPapFleetId] = useState<string | null>(null)
  const [form, setForm] = useState<FleetFormState>(defaultFormState)

  const canManageFleet = roles.some((role) =>
    ['super_admin', 'admin', 'fc', 'senior_fc'].includes(role)
  )
  const canDeleteFleet = roles.some((role) => ['super_admin', 'admin'].includes(role))

  const loadData = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const response = await fetchFleetList({
        current: page,
        size: pageSize,
        importance: importance || undefined,
      })
      setFleets(response.list ?? [])
      setTotal(response.total ?? 0)
      setPage(response.page ?? page)
      setPageSize(response.pageSize ?? pageSize)
    } catch (caughtError) {
      setError(getErrorMessage(caughtError, t('fleet.manage.loadFailed')))
      setFleets([])
      setTotal(0)
    } finally {
      setLoading(false)
    }
  }, [importance, page, pageSize, t])

  const loadCharacters = useCallback(async () => {
    try {
      setCharacters(await fetchMyCharacters())
    } catch {
      setCharacters([])
    }
  }, [])

  const loadFleetConfigs = useCallback(async () => {
    try {
      const response = await fetchFleetConfigList({ current: 1, size: 100 })
      setFleetConfigs(response.list ?? [])
    } catch {
      setFleetConfigs([])
    }
  }, [])

  useEffect(() => {
    const timer = window.setTimeout(() => {
      void loadData()
    }, 0)
    return () => window.clearTimeout(timer)
  }, [loadData, refreshSeed])

  useEffect(() => {
    const timer = window.setTimeout(() => {
      void loadCharacters()
      void loadFleetConfigs()
    }, 0)
    return () => window.clearTimeout(timer)
  }, [loadCharacters, loadFleetConfigs])

  const openCreateDialog = useCallback(() => {
    setEditingFleet(null)
    setForm(defaultFormState)
    setDialogOpen(true)
  }, [])

  const openEditDialog = useCallback((fleet: FleetItem) => {
    setEditingFleet(fleet)
    setForm({
      title: fleet.title,
      description: fleet.description ?? '',
      importance: fleet.importance,
      pap_count: fleet.pap_count,
      character_id: fleet.fc_character_id,
      start_at: toLocalInputValue(fleet.start_at),
      end_at: toLocalInputValue(fleet.end_at),
      fleet_config_id: fleet.fleet_config_id ?? '',
      send_ping: true,
      auto_srp_mode: fleet.auto_srp_mode,
    })
    setDialogOpen(true)
  }, [])

  const submit = useCallback(async () => {
    if (!form.title.trim()) {
      setError(t('fleet.manage.requiredTitle'))
      return
    }
    if (!form.character_id) {
      setError(t('fleet.manage.requiredFc'))
      return
    }
    if (!form.start_at || !form.end_at) {
      setError(t('fleet.manage.requiredTimeRange'))
      return
    }

    setSaving(true)
    setError(null)
    try {
      const payload = {
        title: form.title.trim(),
        description: form.description.trim() || undefined,
        importance: form.importance,
        pap_count: form.pap_count,
        character_id: Number(form.character_id),
        start_at: toIsoFromLocalInput(form.start_at),
        end_at: toIsoFromLocalInput(form.end_at),
        fleet_config_id: form.fleet_config_id ? Number(form.fleet_config_id) : null,
        auto_srp_mode: form.auto_srp_mode,
      }

      if (editingFleet) {
        await updateFleet(editingFleet.id, payload)
        setError(null)
      } else {
        await createFleet({
          ...payload,
          send_ping: form.send_ping,
        })
      }

      setDialogOpen(false)
      setEditingFleet(null)
      setRefreshSeed((current) => current + 1)
    } catch (caughtError) {
      setError(getErrorMessage(caughtError, t('fleet.manage.saveFailed')))
    } finally {
      setSaving(false)
    }
  }, [editingFleet, form, t])

  const handleDelete = useCallback(
    async (fleet: FleetItem) => {
      if (!window.confirm(t('fleet.manage.deleteConfirm', { title: fleet.title }))) {
        return
      }

      try {
        await deleteFleet(fleet.id)
        setRefreshSeed((current) => current + 1)
      } catch (caughtError) {
        setError(getErrorMessage(caughtError, t('fleet.manage.deleteFailed')))
      }
    },
    [t]
  )

  const handleIssuePap = useCallback(
    async (fleet: FleetItem) => {
      if (
        !window.confirm(
          t('fleet.manage.issuePapConfirm', {
            title: fleet.title,
          })
        )
      ) {
        return
      }

      setPapFleetId(fleet.id)
      try {
        if (!fleet.esi_fleet_id) {
          await refreshFleetESI(fleet.id)
        }
        await syncESIFleetMembers(fleet.id)
        await issuePap(fleet.id)
        setRefreshSeed((current) => current + 1)
      } catch (caughtError) {
        setError(getErrorMessage(caughtError, t('fleet.manage.issuePapFailed')))
      } finally {
        setPapFleetId(null)
      }
    },
    [t]
  )

  const importanceLabel = useCallback(
    (value: FleetImportance) => {
      const key = `fleet.importance.${value}`
      const translated = t(key)
      return translated === key ? value : translated
    },
    [t]
  )

  const columns = useMemo<ColumnDef<FleetItem>[]>(
    () => [
      {
        accessorKey: 'title',
        header: t('fleet.fields.title'),
        cell: ({ row }) => (
          <>
            <Button
              type="button"
              variant="link"
              className="h-auto p-0 font-medium"
              onClick={() => navigate(`/operation/fleet-detail/${row.original.id}`)}
            >
              {row.original.title}
            </Button>
            <div className="line-clamp-2 text-xs text-muted-foreground">
              {row.original.description || '-'}
            </div>
          </>
        ),
      },
      {
        accessorKey: 'importance',
        header: t('fleet.fields.importance'),
        cell: ({ row }) => (
          <ShopBadge className={importanceBadgeClass(row.original.importance)}>
            {importanceLabel(row.original.importance)}
          </ShopBadge>
        ),
      },
      {
        id: 'fc',
        header: t('fleet.fields.fc'),
        cell: ({ row }) => (
          <>
            <div>{row.original.fc_display_name || row.original.fc_character_name}</div>
            <div className="text-xs text-muted-foreground">#{row.original.fc_character_id}</div>
          </>
        ),
      },
      {
        id: 'timeRange',
        header: t('fleet.fields.timeRange'),
        cell: ({ row }) => (
          <span className="text-xs text-muted-foreground">
            {formatDateTime(row.original.start_at)} <span className="mx-1">~</span>{' '}
            {formatDateTime(row.original.end_at)}
          </span>
        ),
      },
      { accessorKey: 'pap_count', header: t('fleet.fields.papCount') },
      {
        accessorKey: 'updated_at',
        header: t('common.updatedAt'),
        cell: ({ row }) => formatDateTime(row.original.updated_at),
      },
      {
        id: 'actions',
        header: t('common.operation'),
        cell: ({ row }) => (
          <div className="flex flex-wrap gap-2">
            <Button
              type="button"
              size="sm"
              variant="outline"
              onClick={() => navigate(`/operation/fleet-detail/${row.original.id}`)}
            >
              {t('fleet.manage.detail')}
            </Button>
            {canManageFleet ? (
              <Button
                type="button"
                size="sm"
                variant="outline"
                onClick={() => openEditDialog(row.original)}
              >
                {t('common.edit')}
              </Button>
            ) : null}
            {canManageFleet ? (
              <Button
                type="button"
                size="sm"
                variant="outline"
                onClick={() => void handleIssuePap(row.original)}
                isDisabled={papFleetId === row.original.id}
              >
                {t('fleet.pap.issue')}
              </Button>
            ) : null}
            {canDeleteFleet ? (
              <Button
                type="button"
                size="sm"
                variant="outline"
                onClick={() => void handleDelete(row.original)}
              >
                {t('common.delete')}
              </Button>
            ) : null}
          </div>
        ),
      },
    ],
    [
      canDeleteFleet,
      canManageFleet,
      handleDelete,
      handleIssuePap,
      importanceLabel,
      navigate,
      openEditDialog,
      papFleetId,
      t,
    ]
  )

  return (
    <section className="space-y-4">
      <div className="rounded-lg border bg-card p-5">
        <div className="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
          <div>
            <h1 className="text-xl font-semibold">{t('fleet.manage.title')}</h1>
            <p className="mt-1 text-sm text-muted-foreground">{t('fleet.manage.subtitle')}</p>
          </div>
          <div className="flex flex-wrap items-end gap-3">
            <label className="space-y-1">
              <span className="text-sm text-muted-foreground">{t('fleet.fields.importance')}</span>
              <Select
                selectedKey={String(importance ?? '')}
                onSelectionChange={(key) => ((value) => {
                  setImportance(value)
                  setPage(1)
                })(String(key))}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem id="">{t('fleet.manage.allImportance')}</SelectItem>
                  <SelectItem id="strat_op">{importanceLabel('strat_op')}</SelectItem>
                  <SelectItem id="cta">{importanceLabel('cta')}</SelectItem>
                  <SelectItem id="other">{importanceLabel('other')}</SelectItem>
                </SelectContent>
              </Select>
            </label>
            <Button
              type="button"
              variant="outline"
              onClick={() => setRefreshSeed((current) => current + 1)}
            >
              {t('common.refresh')}
            </Button>
            <Button type="button" onClick={openCreateDialog} isDisabled={!canManageFleet}>
              {t('fleet.manage.create')}
            </Button>
          </div>
        </div>
      </div>

      {error ? <p className="text-sm text-destructive">{error}</p> : null}
      {loading ? (
        <p className="text-sm text-muted-foreground">{t('fleet.manage.loading')}</p>
      ) : null}

      <DataTable
        columns={columns}
        data={fleets}
        getRowId={(fleet) => String(fleet.id)}
        loading={loading}
        error={error}
        loadingText={t('fleet.manage.loading')}
        emptyText={t('fleet.manage.empty')}
        pagination={{
          page,
          pageSize,
          total,
          onPageChange: setPage,
          onPageSizeChange: (nextPageSize) => {
            setPageSize(nextPageSize)
            setPage(1)
          },
          pageSizeOptions: [10, 20, 50],
          previousLabel: t('welfareMy.pagination.prev'),
          nextLabel: t('welfareMy.pagination.next'),
          pageSizeLabel: t('welfareMy.pageSize'),
        }}
      />

      <ShopDialog
        open={dialogOpen}
        title={editingFleet ? t('fleet.manage.edit') : t('fleet.manage.create')}
        onClose={() => {
          setDialogOpen(false)
          setEditingFleet(null)
          setForm(defaultFormState)
        }}
        closeLabel={t('common.close')}
        widthClass="max-w-3xl"
        footer={
          <>
            <Button
              type="button"
              variant="outline"
              onClick={() => {
                setDialogOpen(false)
                setEditingFleet(null)
                setForm(defaultFormState)
              }}
              isDisabled={saving}
            >
              {t('common.cancel')}
            </Button>
            <Button type="button" onClick={() => void submit()} isDisabled={saving}>
              {saving ? t('fleet.manage.saving') : t('common.confirm')}
            </Button>
          </>
        }
      >
        <div className="grid gap-4 md:grid-cols-2">
          <label className="space-y-2 md:col-span-2">
            <span className="text-sm text-muted-foreground">{t('fleet.fields.title')}</span>
            <Input
              value={form.title}
              onChange={(event) =>
                setForm((current) => ({ ...current, title: event.target.value }))
              }
            />
          </label>
          <label className="space-y-2 md:col-span-2">
            <span className="text-sm text-muted-foreground">{t('fleet.fields.description')}</span>
            <Textarea
              className="min-h-24"
              value={form.description}
              onChange={(event) =>
                setForm((current) => ({ ...current, description: event.target.value }))
              }
            />
          </label>
          <label className="space-y-2">
            <span className="text-sm text-muted-foreground">{t('fleet.fields.importance')}</span>
            <Select
              selectedKey={String(form.importance ?? '')}
              onSelectionChange={(key) => ((value) =>
                setForm((current) => ({
                  ...current,
                  importance: value as FleetImportance,
                })))(String(key))}
            >
              <SelectTrigger className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem id="strat_op">{importanceLabel('strat_op')}</SelectItem>
                <SelectItem id="cta">{importanceLabel('cta')}</SelectItem>
                <SelectItem id="other">{importanceLabel('other')}</SelectItem>
              </SelectContent>
            </Select>
          </label>
          <label className="space-y-2">
            <span className="text-sm text-muted-foreground">{t('fleet.fields.papCount')}</span>
            <Input
              type="number"
              min={0}
              value={String(form.pap_count)}
              onChange={(event) =>
                setForm((current) => ({ ...current, pap_count: Number(event.target.value) }))
              }
            />
          </label>
          <label className="space-y-2">
            <span className="text-sm text-muted-foreground">{t('fleet.fields.fc')}</span>
            <Select
              selectedKey={String(form.character_id ?? '')}
              onSelectionChange={(key) => ((value) =>
                setForm((current) => ({
                  ...current,
                  character_id: value ? Number(value) : '',
                })))(String(key))}
            >
              <SelectTrigger className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem id="">{t('fleet.fields.fcPlaceholder')}</SelectItem>
                {characters.map((character) => (
                  <SelectItem
                    key={character.character_id}
                    id={String(character.character_id ?? '')}
                  >
                    {character.character_name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </label>
          <label className="space-y-2">
            <span className="text-sm text-muted-foreground">{t('fleet.fields.fleetConfig')}</span>
            <Select
              selectedKey={String(form.fleet_config_id ?? '')}
              onSelectionChange={(key) => ((value) =>
                setForm((current) => ({
                  ...current,
                  fleet_config_id: value ? Number(value) : '',
                })))(String(key))}
            >
              <SelectTrigger className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem id="">{t('fleet.fields.fleetConfigPlaceholder')}</SelectItem>
                {fleetConfigs.map((config) => (
                  <SelectItem key={config.id} id={String(config.id ?? '')}>
                    {config.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </label>
          <label className="space-y-2">
            <span className="text-sm text-muted-foreground">{t('fleet.fields.autoSrpMode')}</span>
            <Select
              selectedKey={String(form.auto_srp_mode ?? '')}
              onSelectionChange={(key) => ((value) =>
                setForm((current) => ({
                  ...current,
                  auto_srp_mode: value as FleetAutoSrpMode,
                })))(String(key))}
            >
              <SelectTrigger className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem id="disabled">{t('fleet.autoSrp.disabled')}</SelectItem>
                <SelectItem id="submit_only">{t('fleet.autoSrp.submitOnly')}</SelectItem>
                <SelectItem id="auto_approve">{t('fleet.autoSrp.autoApprove')}</SelectItem>
              </SelectContent>
            </Select>
          </label>
          <label className="space-y-2">
            <span className="text-sm text-muted-foreground">{t('fleet.fields.startAt')}</span>
            <Input
              type="datetime-local"
              value={form.start_at}
              onChange={(event) =>
                setForm((current) => ({ ...current, start_at: event.target.value }))
              }
            />
          </label>
          <label className="space-y-2">
            <span className="text-sm text-muted-foreground">{t('fleet.fields.endAt')}</span>
            <Input
              type="datetime-local"
              value={form.end_at}
              onChange={(event) =>
                setForm((current) => ({ ...current, end_at: event.target.value }))
              }
            />
          </label>
          {!editingFleet ? (
            <label className="flex items-center gap-2 md:col-span-2">
              <Checkbox
                isSelected={form.send_ping}
                onChange={(checked) =>
                  setForm((current) => ({ ...current, send_ping: checked === true }))
                }
              />
              <span className="text-sm text-muted-foreground">{t('fleet.fields.sendPing')}</span>
            </label>
          ) : null}
        </div>
      </ShopDialog>
    </section>
  )
}
