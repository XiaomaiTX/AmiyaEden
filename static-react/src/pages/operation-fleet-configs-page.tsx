import { useCallback, useEffect, useMemo, useState } from 'react'
import {
  createFleetConfig,
  deleteFleetConfig,
  fetchFleetConfigDetail,
  fetchFleetConfigEFT,
  fetchFleetConfigList,
  updateFleetConfig,
} from '@/api/fleet-config'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { useI18n } from '@/i18n'
import { useSessionStore } from '@/stores'
import type { FleetConfigItem, FittingReq } from '@/types/api/fleet-config'
import { formatDateTime, getErrorMessage, ShopDialog } from './shop-page-utils'

type FittingFormState = FittingReq

type FleetConfigFormState = {
  name: string
  description: string
  fittings: FittingFormState[]
}

const defaultFormState: FleetConfigFormState = {
  name: '',
  description: '',
  fittings: [],
}

export function OperationFleetConfigsPage() {
  const { t } = useI18n()
  const roles = useSessionStore((state) => state.roles)
  const canManage = roles.some((role) => ['super_admin', 'admin', 'senior_fc'].includes(role))
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [configs, setConfigs] = useState<FleetConfigItem[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)
  const [refreshSeed, setRefreshSeed] = useState(0)
  const [dialogOpen, setDialogOpen] = useState(false)
  const [dialogLoading, setDialogLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [editingConfig, setEditingConfig] = useState<FleetConfigItem | null>(null)
  const [form, setForm] = useState<FleetConfigFormState>(defaultFormState)

  const pageCount = useMemo(() => Math.max(1, Math.ceil(total / pageSize) || 1), [pageSize, total])

  const loadData = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const response = await fetchFleetConfigList({ current: page, size: pageSize })
      setConfigs(response.list ?? [])
      setTotal(response.total ?? 0)
      setPage(response.page ?? page)
      setPageSize(response.pageSize ?? pageSize)
    } catch (caughtError) {
      setError(getErrorMessage(caughtError, t('fleetConfig.loadFailed')))
      setConfigs([])
      setTotal(0)
    } finally {
      setLoading(false)
    }
  }, [page, pageSize, t])

  useEffect(() => {
    const timer = window.setTimeout(() => {
      void loadData()
    }, 0)
    return () => window.clearTimeout(timer)
  }, [loadData, refreshSeed])

  const openCreateDialog = useCallback(() => {
    setEditingConfig(null)
    setForm(defaultFormState)
    setDialogOpen(true)
  }, [])

  const openEditDialog = useCallback(async (config: FleetConfigItem) => {
    setEditingConfig(config)
    setDialogOpen(true)
    setDialogLoading(true)
    try {
      const [detail, eft] = await Promise.all([
        fetchFleetConfigDetail(config.id),
        fetchFleetConfigEFT(config.id, 'en').catch(() => null),
      ])
      const eftMap = new Map<number, string>()
      eft?.fittings?.forEach((item) => {
        eftMap.set(item.id, item.eft)
      })

      setForm({
        name: detail.name,
        description: detail.description ?? '',
        fittings: (detail.fittings ?? []).map((fitting) => ({
          id: fitting.id,
          fitting_name: fitting.fitting_name,
          eft: eftMap.get(fitting.id) ?? '',
          srp_amount: fitting.srp_amount,
        })),
      })
    } catch (caughtError) {
      setError(getErrorMessage(caughtError, t('fleetConfig.loadFailed')))
      setDialogOpen(false)
      setEditingConfig(null)
    } finally {
      setDialogLoading(false)
    }
  }, [t])

  const addFitting = useCallback(() => {
    setForm((current) => ({
      ...current,
      fittings: [...current.fittings, { fitting_name: '', eft: '', srp_amount: 0 }],
    }))
  }, [])

  const removeFitting = useCallback((index: number) => {
    setForm((current) => ({
      ...current,
      fittings: current.fittings.filter((_, currentIndex) => currentIndex !== index),
    }))
  }, [])

  const submit = useCallback(async () => {
    if (!form.name.trim()) {
      setError(t('fleetConfig.requiredName'))
      return
    }
    if (form.fittings.length === 0) {
      setError(t('fleetConfig.noFittings'))
      return
    }

    setSaving(true)
    setError(null)
    try {
      const payload = {
        name: form.name.trim(),
        description: form.description.trim() || undefined,
        fittings: form.fittings.map((fitting) => ({
          id: fitting.id,
          fitting_name: fitting.fitting_name.trim(),
          eft: fitting.eft.trim(),
          srp_amount: fitting.srp_amount,
        })),
      }

      if (editingConfig) {
        await updateFleetConfig(editingConfig.id, payload)
      } else {
        await createFleetConfig(payload)
      }

      setDialogOpen(false)
      setEditingConfig(null)
      setForm(defaultFormState)
      setRefreshSeed((current) => current + 1)
    } catch (caughtError) {
      setError(getErrorMessage(caughtError, t('fleetConfig.operationFailed')))
    } finally {
      setSaving(false)
    }
  }, [editingConfig, form, t])

  const handleDelete = useCallback(
    async (config: FleetConfigItem) => {
      if (!window.confirm(t('fleetConfig.deleteConfirm', { name: config.name }))) {
        return
      }

      try {
        await deleteFleetConfig(config.id)
        setRefreshSeed((current) => current + 1)
      } catch (caughtError) {
        setError(getErrorMessage(caughtError, t('fleetConfig.deleteFailed')))
      }
    },
    [t]
  )

  return (
    <section className="space-y-4">
      <div className="rounded-lg border bg-card p-5">
        <div className="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
          <div>
            <h1 className="text-xl font-semibold">{t('fleetConfig.title')}</h1>
            <p className="mt-1 text-sm text-muted-foreground">{t('fleetConfig.subtitle')}</p>
          </div>
          <div className="flex flex-wrap items-center gap-3">
            <Button type="button" variant="outline" onClick={() => setRefreshSeed((current) => current + 1)}>
              {t('common.refresh')}
            </Button>
            <Button type="button" onClick={openCreateDialog} disabled={!canManage}>
              {t('fleetConfig.create')}
            </Button>
          </div>
        </div>
      </div>

      {error ? <p className="text-sm text-destructive">{error}</p> : null}
      {loading ? <p className="text-sm text-muted-foreground">{t('fleetConfig.loading')}</p> : null}

      <div className="overflow-hidden rounded-lg border bg-card">
        <div className="border-b px-4 py-3 text-sm font-medium">
          {t('fleetConfig.title')} ({total})
        </div>
        <div className="overflow-x-auto">
          <table className="min-w-full text-sm">
            <thead>
              <tr className="border-b bg-muted/40 text-left">
                <th className="px-3 py-2">{t('fleetConfig.fields.name')}</th>
                <th className="px-3 py-2">{t('fleetConfig.fields.description')}</th>
                <th className="px-3 py-2">{t('fleetConfig.fields.fittings')}</th>
                <th className="px-3 py-2">{t('common.updatedAt')}</th>
                <th className="px-3 py-2">{t('common.operation')}</th>
              </tr>
            </thead>
            <tbody>
              {configs.map((config) => (
                <tr key={config.id} className="border-b">
                  <td className="px-3 py-2">
                    {canManage ? (
                      <button
                        type="button"
                        className="font-medium text-left text-primary hover:underline"
                        onClick={() => void openEditDialog(config)}
                      >
                        {config.name}
                      </button>
                    ) : (
                      <span className="font-medium">{config.name}</span>
                    )}
                  </td>
                  <td className="px-3 py-2">
                    <div className="line-clamp-2 text-sm">{config.description || '-'}</div>
                  </td>
                  <td className="px-3 py-2">{config.fittings?.length ?? 0}</td>
                  <td className="px-3 py-2">{formatDateTime(config.updated_at)}</td>
                  <td className="px-3 py-2">
                    <div className="flex flex-wrap gap-2">
                      <Button
                        type="button"
                        size="sm"
                        variant="outline"
                        onClick={() => void openEditDialog(config)}
                        disabled={!canManage}
                      >
                        {t('common.edit')}
                      </Button>
                      <Button type="button" size="sm" variant="outline" onClick={() => void handleDelete(config)} disabled={!canManage}>
                        {t('common.delete')}
                      </Button>
                    </div>
                  </td>
                </tr>
              ))}
              {!loading && configs.length === 0 ? (
                <tr>
                  <td className="px-3 py-6 text-center text-muted-foreground" colSpan={5}>
                    {t('fleetConfig.empty')}
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
          disabled={configs.length < pageSize || page * pageSize >= total}
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
        title={editingConfig ? t('fleetConfig.edit') : t('fleetConfig.create')}
        onClose={() => {
          setDialogOpen(false)
          setEditingConfig(null)
          setForm(defaultFormState)
        }}
        closeLabel={t('common.close')}
        widthClass="max-w-4xl"
        footer={
          <>
            <Button
              type="button"
              variant="outline"
              onClick={() => {
                setDialogOpen(false)
                setEditingConfig(null)
                setForm(defaultFormState)
              }}
              disabled={saving}
            >
              {t('common.cancel')}
            </Button>
            <Button type="button" onClick={() => void submit()} disabled={saving || dialogLoading}>
              {saving ? t('fleetConfig.saving') : t('common.confirm')}
            </Button>
          </>
        }
      >
        {dialogLoading ? <p className="text-sm text-muted-foreground">{t('fleetConfig.loadingDetail')}</p> : null}
        <div className="grid gap-4">
          <label className="space-y-2">
            <span className="text-sm text-muted-foreground">{t('fleetConfig.fields.name')}</span>
            <Input value={form.name} onChange={(event) => setForm((current) => ({ ...current, name: event.target.value }))} />
          </label>
          <label className="space-y-2">
            <span className="text-sm text-muted-foreground">{t('fleetConfig.fields.description')}</span>
            <textarea
              className="min-h-24 w-full rounded-lg border border-input bg-background px-3 py-2 text-sm outline-none"
              value={form.description}
              onChange={(event) => setForm((current) => ({ ...current, description: event.target.value }))}
            />
          </label>

          <div className="flex items-center justify-between gap-3">
            <h3 className="text-sm font-medium">{t('fleetConfig.fields.fittings')}</h3>
            <Button type="button" variant="outline" onClick={addFitting}>
              {t('fleetConfig.addFitting')}
            </Button>
          </div>

          <div className="space-y-3">
            {form.fittings.map((fitting, index) => (
              <div key={fitting.id ?? index} className="rounded-lg border p-3">
                <div className="flex items-center justify-between gap-3">
                  <div className="font-medium">{fitting.fitting_name || `${t('fleetConfig.newFitting')} ${index + 1}`}</div>
                  <Button type="button" variant="outline" size="sm" onClick={() => removeFitting(index)}>
                    {t('common.delete')}
                  </Button>
                </div>
                <div className="mt-3 grid gap-3 md:grid-cols-3">
                  <label className="space-y-2 md:col-span-1">
                    <span className="text-sm text-muted-foreground">{t('fleetConfig.fields.fittingName')}</span>
                    <Input
                      value={fitting.fitting_name}
                      onChange={(event) =>
                        setForm((current) => ({
                          ...current,
                          fittings: current.fittings.map((item, currentIndex) =>
                            currentIndex === index ? { ...item, fitting_name: event.target.value } : item
                          ),
                        }))
                      }
                    />
                  </label>
                  <label className="space-y-2 md:col-span-1">
                    <span className="text-sm text-muted-foreground">{t('fleetConfig.fields.srpAmount')}</span>
                    <Input
                      type="number"
                      min={0}
                      value={String(fitting.srp_amount)}
                      onChange={(event) =>
                        setForm((current) => ({
                          ...current,
                          fittings: current.fittings.map((item, currentIndex) =>
                            currentIndex === index ? { ...item, srp_amount: Number(event.target.value) } : item
                          ),
                        }))
                      }
                    />
                  </label>
                  <label className="space-y-2 md:col-span-1">
                    <span className="text-sm text-muted-foreground">{t('fleetConfig.fields.eft')}</span>
                    <textarea
                      className="min-h-28 w-full rounded-lg border border-input bg-background px-3 py-2 text-sm outline-none font-mono"
                      value={fitting.eft}
                      onChange={(event) =>
                        setForm((current) => ({
                          ...current,
                          fittings: current.fittings.map((item, currentIndex) =>
                            currentIndex === index ? { ...item, eft: event.target.value } : item
                          ),
                        }))
                      }
                    />
                  </label>
                </div>
              </div>
            ))}
          </div>
        </div>
      </ShopDialog>
    </section>
  )
}
