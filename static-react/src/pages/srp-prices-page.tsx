import { useEffect, useMemo, useState } from 'react'
import { deleteShipPrice, fetchSrpConfig, fetchShipPrices, updateSrpConfig, upsertShipPrice } from '@/api/srp'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { useI18n } from '@/i18n'
import type { ShipPrice, SrpConfig, UpsertShipPriceParams } from '@/types/api/srp'

function getErrorMessage(error: unknown, fallback: string) {
  return error instanceof Error && error.message ? error.message : fallback
}

export function SrpPricesPage() {
  const { t } = useI18n()
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [prices, setPrices] = useState<ShipPrice[]>([])
  const [keyword, setKeyword] = useState('')
  const [config, setConfig] = useState<SrpConfig>({ amount_limit: 0 })
  const [dialogVisible, setDialogVisible] = useState(false)
  const [editingId, setEditingId] = useState(0)
  const [saving, setSaving] = useState(false)
  const [configSaving, setConfigSaving] = useState(false)
  const [form, setForm] = useState<UpsertShipPriceParams>({
    ship_type_id: 0,
    ship_name: '',
    amount: 0,
  })

  const loadData = async () => {
    setLoading(true)
    setError(null)
    try {
      const [priceList, configData] = await Promise.all([fetchShipPrices(keyword.trim() || undefined), fetchSrpConfig()])
      setPrices(priceList ?? [])
      setConfig(configData)
    } catch (caughtError) {
      setError(getErrorMessage(caughtError, t('srpPrices.loadFailed')))
      setPrices([])
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void loadData()
  }, [keyword, t])

  const orderedPrices = useMemo(() => [...prices].sort((left, right) => left.ship_type_id - right.ship_type_id), [prices])

  const openCreate = () => {
    setEditingId(0)
    setForm({ ship_type_id: 0, ship_name: '', amount: 0 })
    setDialogVisible(true)
  }

  const openEdit = (row: ShipPrice) => {
    setEditingId(row.id)
    setForm({ id: row.id, ship_type_id: row.ship_type_id, ship_name: row.ship_name, amount: row.amount })
    setDialogVisible(true)
  }

  const save = async () => {
    if (!form.ship_type_id || !form.ship_name.trim()) {
      setError(t('srpPrices.required'))
      return
    }

    setSaving(true)
    try {
      await upsertShipPrice({
        ...form,
        ship_type_id: Number(form.ship_type_id),
        ship_name: form.ship_name.trim(),
      })
      setDialogVisible(false)
      await loadData()
    } catch (caughtError) {
      setError(getErrorMessage(caughtError, t('srpPrices.saveFailed')))
    } finally {
      setSaving(false)
    }
  }

  const remove = async (id: number) => {
    if (!window.confirm(t('srpPrices.deleteConfirm'))) return
    try {
      await deleteShipPrice(id)
      await loadData()
    } catch (caughtError) {
      setError(getErrorMessage(caughtError, t('srpPrices.deleteFailed')))
    }
  }

  const saveConfig = async () => {
    setConfigSaving(true)
    try {
      const nextConfig = await updateSrpConfig(config)
      setConfig(nextConfig)
    } catch (caughtError) {
      setError(getErrorMessage(caughtError, t('srpPrices.configSaveFailed')))
    } finally {
      setConfigSaving(false)
    }
  }

  return (
    <section className="space-y-4">
      <div className="rounded-lg border bg-card p-5">
        <div className="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
          <div>
            <h1 className="text-xl font-semibold">{t('srpPrices.title')}</h1>
            <p className="mt-1 text-sm text-muted-foreground">{t('srpPrices.subtitle')}</p>
          </div>
          <div className="flex flex-wrap items-end gap-3">
            <Input
              value={keyword}
              onChange={(event) => setKeyword(event.target.value)}
              placeholder={t('srpPrices.searchPlaceholder')}
            />
            <Button type="button" onClick={openCreate}>
              {t('srpPrices.addPrice')}
            </Button>
            <Button type="button" variant="outline" onClick={() => void loadData()}>
              {t('common.refresh')}
            </Button>
          </div>
        </div>
      </div>

      {error ? <p className="text-sm text-destructive">{error}</p> : null}
      {loading ? <p className="text-sm text-muted-foreground">{t('srpPrices.loading')}</p> : null}

      <div className="grid gap-4 xl:grid-cols-3">
        <div className="rounded-lg border bg-card p-5 xl:col-span-2">
          <h2 className="text-lg font-semibold">{t('srpPrices.priceListTitle')}</h2>
          <div className="mt-4 overflow-x-auto">
            <table className="min-w-full text-sm">
              <thead>
                <tr className="border-b bg-muted/40 text-left">
                  <th className="px-3 py-2">ID</th>
                  <th className="px-3 py-2">{t('srpPrices.columns.typeId')}</th>
                  <th className="px-3 py-2">{t('srpPrices.columns.name')}</th>
                  <th className="px-3 py-2">{t('srpPrices.columns.amount')}</th>
                  <th className="px-3 py-2">{t('srpPrices.columns.actions')}</th>
                </tr>
              </thead>
              <tbody>
                {orderedPrices.map((price) => (
                  <tr key={price.id} className="border-b">
                    <td className="px-3 py-2">{price.id}</td>
                    <td className="px-3 py-2">{price.ship_type_id}</td>
                    <td className="px-3 py-2">{price.ship_name}</td>
                    <td className="px-3 py-2">{price.amount}</td>
                    <td className="px-3 py-2">
                      <div className="flex gap-2">
                        <Button type="button" size="sm" variant="outline" onClick={() => openEdit(price)}>
                          {t('common.edit')}
                        </Button>
                        <Button type="button" size="sm" variant="outline" onClick={() => void remove(price.id)}>
                          {t('common.delete')}
                        </Button>
                      </div>
                    </td>
                  </tr>
                ))}
                {!loading && orderedPrices.length === 0 ? (
                  <tr>
                    <td className="px-3 py-6 text-center text-muted-foreground" colSpan={5}>
                      {t('srpPrices.empty')}
                    </td>
                  </tr>
                ) : null}
              </tbody>
            </table>
          </div>
        </div>

        <div className="rounded-lg border bg-card p-5">
          <h2 className="text-lg font-semibold">{t('srpPrices.configTitle')}</h2>
          <div className="mt-4 space-y-3">
            <label className="space-y-2 block">
              <span className="text-sm text-muted-foreground">{t('srpPrices.configAmountLimit')}</span>
              <Input
                type="number"
                value={String(config.amount_limit)}
                onChange={(event) => setConfig((current) => ({ ...current, amount_limit: Number(event.target.value) }))}
              />
            </label>
            <Button type="button" onClick={() => void saveConfig()} disabled={configSaving}>
              {configSaving ? t('srpPrices.saving') : t('srpPrices.saveConfig')}
            </Button>
          </div>
        </div>
      </div>

      {dialogVisible ? (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
          <div className="w-full max-w-lg rounded-lg border bg-card p-5 shadow-xl">
            <h2 className="text-lg font-semibold">
              {editingId > 0 ? t('srpPrices.editDialog') : t('srpPrices.addDialog')}
            </h2>
            <div className="mt-4 space-y-4">
              <label className="space-y-2 block">
                <span className="text-sm text-muted-foreground">{t('srpPrices.fields.typeId')}</span>
                <Input
                  type="number"
                  value={String(form.ship_type_id ?? 0)}
                  onChange={(event) => setForm((current) => ({ ...current, ship_type_id: Number(event.target.value) }))}
                />
              </label>
              <label className="space-y-2 block">
                <span className="text-sm text-muted-foreground">{t('srpPrices.fields.name')}</span>
                <Input
                  value={form.ship_name ?? ''}
                  onChange={(event) => setForm((current) => ({ ...current, ship_name: event.target.value }))}
                />
              </label>
              <label className="space-y-2 block">
                <span className="text-sm text-muted-foreground">{t('srpPrices.fields.amount')}</span>
                <Input
                  type="number"
                  value={String(form.amount ?? 0)}
                  onChange={(event) => setForm((current) => ({ ...current, amount: Number(event.target.value) }))}
                />
              </label>
            </div>
            <div className="mt-5 flex justify-end gap-3">
              <Button type="button" variant="outline" onClick={() => setDialogVisible(false)}>
                {t('common.cancel')}
              </Button>
              <Button type="button" onClick={() => void save()} disabled={saving}>
                {saving ? t('srpPrices.saving') : t('common.confirm')}
              </Button>
            </div>
          </div>
        </div>
      ) : null}
    </section>
  )
}
