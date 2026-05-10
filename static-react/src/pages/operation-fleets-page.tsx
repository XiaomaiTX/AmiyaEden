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
import { Input } from '@/components/ui/input'
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

  const canManageFleet = roles.some((role) => ['super_admin', 'admin', 'fc', 'senior_fc'].includes(role))
  const canDeleteFleet = roles.some((role) => ['super_admin', 'admin'].includes(role))

  const pageCount = useMemo(() => Math.max(1, Math.ceil(total / pageSize) || 1), [pageSize, total])

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
              <select
                className="h-10 rounded-md border border-input bg-background px-3 text-sm"
                value={importance}
                onChange={(event) => {
                  setImportance(event.target.value)
                  setPage(1)
                }}
              >
                <option value="">{t('fleet.manage.allImportance')}</option>
                <option value="strat_op">{importanceLabel('strat_op')}</option>
                <option value="cta">{importanceLabel('cta')}</option>
                <option value="other">{importanceLabel('other')}</option>
              </select>
            </label>
            <Button type="button" variant="outline" onClick={() => setRefreshSeed((current) => current + 1)}>
              {t('common.refresh')}
            </Button>
            <Button type="button" onClick={openCreateDialog} disabled={!canManageFleet}>
              {t('fleet.manage.create')}
            </Button>
          </div>
        </div>
      </div>

      {error ? <p className="text-sm text-destructive">{error}</p> : null}
      {loading ? <p className="text-sm text-muted-foreground">{t('fleet.manage.loading')}</p> : null}

      <div className="overflow-hidden rounded-lg border bg-card">
        <div className="border-b px-4 py-3 text-sm font-medium">
          {t('fleet.manage.title')} ({total})
        </div>
        <div className="overflow-x-auto">
          <table className="min-w-full text-sm">
            <thead>
              <tr className="border-b bg-muted/40 text-left">
                <th className="px-3 py-2">{t('fleet.fields.title')}</th>
                <th className="px-3 py-2">{t('fleet.fields.importance')}</th>
                <th className="px-3 py-2">{t('fleet.fields.fc')}</th>
                <th className="px-3 py-2">{t('fleet.fields.timeRange')}</th>
                <th className="px-3 py-2">{t('fleet.fields.papCount')}</th>
                <th className="px-3 py-2">{t('common.updatedAt')}</th>
                <th className="px-3 py-2">{t('common.operation')}</th>
              </tr>
            </thead>
            <tbody>
              {fleets.map((fleet) => (
                <tr key={fleet.id} className="border-b">
                  <td className="px-3 py-2">
                    <button
                      type="button"
                      className="font-medium text-left text-primary hover:underline"
                      onClick={() => navigate(`/operation/fleet-detail/${fleet.id}`)}
                    >
                      {fleet.title}
                    </button>
                    <div className="line-clamp-2 text-xs text-muted-foreground">{fleet.description || '-'}</div>
                  </td>
                  <td className="px-3 py-2">
                    <ShopBadge className={importanceBadgeClass(fleet.importance)}>
                      {importanceLabel(fleet.importance)}
                    </ShopBadge>
                  </td>
                  <td className="px-3 py-2">
                    <div>{fleet.fc_display_name || fleet.fc_character_name}</div>
                    <div className="text-xs text-muted-foreground">#{fleet.fc_character_id}</div>
                  </td>
                  <td className="px-3 py-2 text-xs text-muted-foreground">
                    {formatDateTime(fleet.start_at)} <span className="mx-1">~</span> {formatDateTime(fleet.end_at)}
                  </td>
                  <td className="px-3 py-2">{fleet.pap_count}</td>
                  <td className="px-3 py-2">{formatDateTime(fleet.updated_at)}</td>
                  <td className="px-3 py-2">
                    <div className="flex flex-wrap gap-2">
                      <Button type="button" size="sm" variant="outline" onClick={() => navigate(`/operation/fleet-detail/${fleet.id}`)}>
                        {t('fleet.manage.detail')}
                      </Button>
                      {canManageFleet ? (
                        <Button
                          type="button"
                          size="sm"
                          variant="outline"
                          onClick={() => openEditDialog(fleet)}
                        >
                          {t('common.edit')}
                        </Button>
                      ) : null}
                      {canManageFleet ? (
                        <Button
                          type="button"
                          size="sm"
                          variant="outline"
                          onClick={() => void handleIssuePap(fleet)}
                          disabled={papFleetId === fleet.id}
                        >
                          {t('fleet.pap.issue')}
                        </Button>
                      ) : null}
                      {canDeleteFleet ? (
                        <Button
                          type="button"
                          size="sm"
                          variant="outline"
                          onClick={() => void handleDelete(fleet)}
                        >
                          {t('common.delete')}
                        </Button>
                      ) : null}
                    </div>
                  </td>
                </tr>
              ))}
              {!loading && fleets.length === 0 ? (
                <tr>
                  <td className="px-3 py-6 text-center text-muted-foreground" colSpan={7}>
                    {t('fleet.manage.empty')}
                  </td>
                </tr>
              ) : null}
            </tbody>
          </table>
        </div>
      </div>

      <div className="flex flex-wrap items-center gap-3 text-sm">
        <span>
          {page}/{pageCount}
        </span>
        <Button type="button" variant="outline" size="sm" onClick={() => setPage((current) => Math.max(1, current - 1))} disabled={page <= 1}>
          {t('welfareMy.pagination.prev')}
        </Button>
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={() => setPage((current) => current + 1)}
          disabled={fleets.length < pageSize || page * pageSize >= total}
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
              disabled={saving}
            >
              {t('common.cancel')}
            </Button>
            <Button type="button" onClick={() => void submit()} disabled={saving}>
              {saving ? t('fleet.manage.saving') : t('common.confirm')}
            </Button>
          </>
        }
      >
        <div className="grid gap-4 md:grid-cols-2">
          <label className="space-y-2 md:col-span-2">
            <span className="text-sm text-muted-foreground">{t('fleet.fields.title')}</span>
            <Input value={form.title} onChange={(event) => setForm((current) => ({ ...current, title: event.target.value }))} />
          </label>
          <label className="space-y-2 md:col-span-2">
            <span className="text-sm text-muted-foreground">{t('fleet.fields.description')}</span>
            <textarea
              className="min-h-24 w-full rounded-lg border border-input bg-background px-3 py-2 text-sm outline-none"
              value={form.description}
              onChange={(event) => setForm((current) => ({ ...current, description: event.target.value }))}
            />
          </label>
          <label className="space-y-2">
            <span className="text-sm text-muted-foreground">{t('fleet.fields.importance')}</span>
            <select
              className="h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
              value={form.importance}
              onChange={(event) =>
                setForm((current) => ({
                  ...current,
                  importance: event.target.value as FleetImportance,
                }))
              }
            >
              <option value="strat_op">{importanceLabel('strat_op')}</option>
              <option value="cta">{importanceLabel('cta')}</option>
              <option value="other">{importanceLabel('other')}</option>
            </select>
          </label>
          <label className="space-y-2">
            <span className="text-sm text-muted-foreground">{t('fleet.fields.papCount')}</span>
            <Input
              type="number"
              min={0}
              value={String(form.pap_count)}
              onChange={(event) => setForm((current) => ({ ...current, pap_count: Number(event.target.value) }))}
            />
          </label>
          <label className="space-y-2">
            <span className="text-sm text-muted-foreground">{t('fleet.fields.fc')}</span>
            <select
              className="h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
              value={String(form.character_id)}
              onChange={(event) =>
                setForm((current) => ({
                  ...current,
                  character_id: event.target.value ? Number(event.target.value) : '',
                }))
              }
            >
              <option value="">{t('fleet.fields.fcPlaceholder')}</option>
              {characters.map((character) => (
                <option key={character.character_id} value={character.character_id}>
                  {character.character_name}
                </option>
              ))}
            </select>
          </label>
          <label className="space-y-2">
            <span className="text-sm text-muted-foreground">{t('fleet.fields.fleetConfig')}</span>
            <select
              className="h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
              value={String(form.fleet_config_id)}
              onChange={(event) =>
                setForm((current) => ({
                  ...current,
                  fleet_config_id: event.target.value ? Number(event.target.value) : '',
                }))
              }
            >
              <option value="">{t('fleet.fields.fleetConfigPlaceholder')}</option>
              {fleetConfigs.map((config) => (
                <option key={config.id} value={config.id}>
                  {config.name}
                </option>
              ))}
            </select>
          </label>
          <label className="space-y-2">
            <span className="text-sm text-muted-foreground">{t('fleet.fields.autoSrpMode')}</span>
            <select
              className="h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
              value={form.auto_srp_mode}
              onChange={(event) =>
                setForm((current) => ({
                  ...current,
                  auto_srp_mode: event.target.value as FleetAutoSrpMode,
                }))
              }
            >
              <option value="disabled">{t('fleet.autoSrp.disabled')}</option>
              <option value="submit_only">{t('fleet.autoSrp.submitOnly')}</option>
              <option value="auto_approve">{t('fleet.autoSrp.autoApprove')}</option>
            </select>
          </label>
          <label className="space-y-2">
            <span className="text-sm text-muted-foreground">{t('fleet.fields.startAt')}</span>
            <Input
              type="datetime-local"
              value={form.start_at}
              onChange={(event) => setForm((current) => ({ ...current, start_at: event.target.value }))}
            />
          </label>
          <label className="space-y-2">
            <span className="text-sm text-muted-foreground">{t('fleet.fields.endAt')}</span>
            <Input
              type="datetime-local"
              value={form.end_at}
              onChange={(event) => setForm((current) => ({ ...current, end_at: event.target.value }))}
            />
          </label>
          {!editingFleet ? (
            <label className="flex items-center gap-2 md:col-span-2">
              <input
                type="checkbox"
                checked={form.send_ping}
                onChange={(event) => setForm((current) => ({ ...current, send_ping: event.target.checked }))}
              />
              <span className="text-sm text-muted-foreground">{t('fleet.fields.sendPing')}</span>
            </label>
          ) : null}
        </div>
      </ShopDialog>
    </section>
  )
}
