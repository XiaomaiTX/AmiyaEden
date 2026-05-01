import { useEffect, useMemo, useState } from 'react'
import {
  adminCreateWelfare,
  adminDeleteWelfare,
  adminListWelfares,
  adminReorderWelfares,
  adminUpdateWelfare,
  fetchWelfareAutoApproveConfig,
  updateWelfareAutoApproveConfig,
  uploadWelfareEvidence,
} from '@/api/welfare'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { useI18n } from '@/i18n'
import type { AutoApproveConfig, WelfareItem, CreateParams } from '@/types/api/welfare'

function getErrorMessage(error: unknown, fallback: string) {
  return error instanceof Error && error.message ? error.message : fallback
}

export function WelfareSettingsPage() {
  const { t } = useI18n()
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [items, setItems] = useState<WelfareItem[]>([])
  const [config, setConfig] = useState<AutoApproveConfig | null>(null)
  const [activeTab, setActiveTab] = useState<'welfares' | 'config'>('welfares')
  const [dialogVisible, setDialogVisible] = useState(false)
  const [editingId, setEditingId] = useState(0)
  const [saving, setSaving] = useState(false)
  const [configSaving, setConfigSaving] = useState(false)
  const [exampleUploading, setExampleUploading] = useState(false)
  const [form, setForm] = useState<CreateParams>({
    name: '',
    description: '',
    dist_mode: 'per_user',
    pay_by_fuxi_coin: null,
    require_skill_plan: false,
    skill_plan_ids: [],
    max_char_age_months: null,
    minimum_pap: null,
    minimum_fuxi_legion_years: null,
    require_evidence: false,
    example_evidence: '',
    status: 1,
    sort_order: 0,
  })

  const loadData = async () => {
    setLoading(true)
    setError(null)
    try {
      const [welfares, autoApproveConfig] = await Promise.all([
        adminListWelfares({ current: 1, size: 200 }),
        fetchWelfareAutoApproveConfig(),
      ])
      setItems(welfares.list ?? [])
      setConfig(autoApproveConfig)
    } catch (caughtError) {
      setError(getErrorMessage(caughtError, t('welfareSettings.loadFailed')))
      setItems([])
      setConfig(null)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void loadData()
  }, [t])

  const orderedItems = useMemo(() => [...items].sort((left, right) => left.sort_order - right.sort_order), [items])

  const openCreate = () => {
    setEditingId(0)
    setForm({
      name: '',
      description: '',
      dist_mode: 'per_user',
      pay_by_fuxi_coin: null,
      require_skill_plan: false,
      skill_plan_ids: [],
      max_char_age_months: null,
      minimum_pap: null,
      minimum_fuxi_legion_years: null,
      require_evidence: false,
      example_evidence: '',
      status: 1,
      sort_order: orderedItems.length,
    })
    setDialogVisible(true)
  }

  const openEdit = (item: WelfareItem) => {
    setEditingId(item.id)
    setForm({
      name: item.name,
      description: item.description,
      dist_mode: item.dist_mode,
      pay_by_fuxi_coin: item.pay_by_fuxi_coin,
      require_skill_plan: item.require_skill_plan,
      skill_plan_ids: item.skill_plan_ids,
      max_char_age_months: item.max_char_age_months,
      minimum_pap: item.minimum_pap,
      minimum_fuxi_legion_years: item.minimum_fuxi_legion_years,
      require_evidence: item.require_evidence,
      example_evidence: item.example_evidence,
      status: item.status,
      sort_order: item.sort_order,
    })
    setDialogVisible(true)
  }

  const save = async () => {
    if (!form.name.trim()) {
      setError(t('welfareSettings.required'))
      return
    }

    setSaving(true)
    try {
      const payload = {
        ...form,
        skill_plan_ids: form.require_skill_plan ? form.skill_plan_ids ?? [] : [],
        example_evidence: form.require_evidence ? form.example_evidence ?? '' : '',
      }
      if (editingId > 0) {
        await adminUpdateWelfare({ id: editingId, ...payload })
      } else {
        await adminCreateWelfare(payload)
      }
      setDialogVisible(false)
      await loadData()
    } catch (caughtError) {
      setError(getErrorMessage(caughtError, t('welfareSettings.saveFailed')))
    } finally {
      setSaving(false)
    }
  }

  const remove = async (id: number) => {
    if (!window.confirm(t('welfareSettings.deleteConfirm'))) return
    try {
      await adminDeleteWelfare(id)
      await loadData()
    } catch (caughtError) {
      setError(getErrorMessage(caughtError, t('welfareSettings.deleteFailed')))
    }
  }

  const moveItem = async (id: number, direction: 'up' | 'down') => {
    const currentIndex = orderedItems.findIndex((item) => item.id === id)
    if (currentIndex < 0) return

    const nextItems = [...orderedItems]
    const swapIndex = direction === 'up' ? currentIndex - 1 : currentIndex + 1
    if (swapIndex < 0 || swapIndex >= nextItems.length) return

    const [moved] = nextItems.splice(currentIndex, 1)
    nextItems.splice(swapIndex, 0, moved)
    setItems(nextItems.map((item, index) => ({ ...item, sort_order: index })))
    await adminReorderWelfares(nextItems.map((item) => item.id))
  }

  const handleExampleUpload = async (file: File) => {
    setExampleUploading(true)
    try {
      const url = await uploadWelfareEvidence(file)
      setForm((current) => ({ ...current, example_evidence: url }))
    } catch (caughtError) {
      setError(getErrorMessage(caughtError, t('welfareSettings.uploadFailed')))
    } finally {
      setExampleUploading(false)
    }
  }

  const saveConfig = async () => {
    if (!config) return
    setConfigSaving(true)
    try {
      const nextConfig = await updateWelfareAutoApproveConfig({
        auto_approve_fuxi_coin_threshold: config.auto_approve_fuxi_coin_threshold,
      })
      setConfig(nextConfig)
    } catch (caughtError) {
      setError(getErrorMessage(caughtError, t('welfareSettings.saveFailed')))
    } finally {
      setConfigSaving(false)
    }
  }

  return (
    <section className="space-y-4">
      <div className="rounded-lg border bg-card p-5">
        <div className="flex items-center justify-between gap-3">
          <div>
            <h1 className="text-xl font-semibold">{t('welfareSettings.title')}</h1>
            <p className="mt-1 text-sm text-muted-foreground">{t('welfareSettings.subtitle')}</p>
          </div>
          <Button type="button" onClick={openCreate}>
            {t('welfareSettings.create')}
          </Button>
        </div>
      </div>

      {error ? <p className="text-sm text-destructive">{error}</p> : null}
      {loading ? <p className="text-sm text-muted-foreground">{t('welfareSettings.loading')}</p> : null}

      <div className="rounded-lg border bg-card p-5">
        <div className="flex gap-2">
          <Button type="button" variant={activeTab === 'welfares' ? 'default' : 'outline'} onClick={() => setActiveTab('welfares')}>
            {t('welfareSettings.welfareListTab')}
          </Button>
          <Button type="button" variant={activeTab === 'config' ? 'default' : 'outline'} onClick={() => setActiveTab('config')}>
            {t('welfareSettings.autoApproveConfigTab')}
          </Button>
        </div>

        {activeTab === 'welfares' ? (
          <div className="mt-4 overflow-x-auto">
            <table className="min-w-full text-sm">
              <thead>
                <tr className="border-b bg-muted/40 text-left">
                  <th className="px-3 py-2">#</th>
                  <th className="px-3 py-2">{t('welfareSettings.columns.name')}</th>
                  <th className="px-3 py-2">{t('welfareSettings.columns.mode')}</th>
                  <th className="px-3 py-2">{t('welfareSettings.columns.status')}</th>
                  <th className="px-3 py-2">{t('welfareSettings.columns.sort')}</th>
                  <th className="px-3 py-2">{t('welfareSettings.columns.actions')}</th>
                </tr>
              </thead>
              <tbody>
                {orderedItems.map((item, index) => (
                  <tr key={item.id} className="border-b">
                    <td className="px-3 py-2">{index + 1}</td>
                    <td className="px-3 py-2">
                      <div className="font-medium">{item.name}</div>
                      <div className="text-xs text-muted-foreground">{item.description || '-'}</div>
                    </td>
                    <td className="px-3 py-2">{item.dist_mode}</td>
                    <td className="px-3 py-2">{item.status}</td>
                    <td className="px-3 py-2">{item.sort_order}</td>
                    <td className="px-3 py-2">
                      <div className="flex flex-wrap gap-2">
                        <Button type="button" size="sm" variant="outline" onClick={() => void moveItem(item.id, 'up')}>
                          {t('welfareSettings.moveUp')}
                        </Button>
                        <Button type="button" size="sm" variant="outline" onClick={() => void moveItem(item.id, 'down')}>
                          {t('welfareSettings.moveDown')}
                        </Button>
                        <Button type="button" size="sm" variant="outline" onClick={() => openEdit(item)}>
                          {t('common.edit')}
                        </Button>
                        <Button type="button" size="sm" variant="outline" onClick={() => void remove(item.id)}>
                          {t('common.delete')}
                        </Button>
                      </div>
                    </td>
                  </tr>
                ))}
                {!loading && orderedItems.length === 0 ? (
                  <tr>
                    <td className="px-3 py-6 text-center text-muted-foreground" colSpan={6}>
                      {t('welfareSettings.empty')}
                    </td>
                  </tr>
                ) : null}
              </tbody>
            </table>
          </div>
        ) : null}

        {activeTab === 'config' ? (
          <div className="mt-4 max-w-lg space-y-4">
            <label className="space-y-2 block">
              <span className="text-sm text-muted-foreground">{t('welfareSettings.autoApproveThreshold')}</span>
              <Input
                type="number"
                value={String(config?.auto_approve_fuxi_coin_threshold ?? 0)}
                onChange={(event) =>
                  setConfig((current) =>
                    current
                      ? { ...current, auto_approve_fuxi_coin_threshold: Number(event.target.value) }
                      : current
                  )
                }
              />
            </label>
            <Button type="button" onClick={() => void saveConfig()} disabled={configSaving || !config}>
              {configSaving ? t('welfareSettings.saving') : t('welfareSettings.saveConfig')}
            </Button>
          </div>
        ) : null}
      </div>

      {dialogVisible ? (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
          <div className="w-full max-w-2xl rounded-lg border bg-card p-5 shadow-xl">
            <h2 className="text-lg font-semibold">
              {editingId > 0 ? t('welfareSettings.edit') : t('welfareSettings.create')}
            </h2>
            <div className="mt-4 grid gap-4 md:grid-cols-2">
              <label className="space-y-2 md:col-span-2">
                <span className="text-sm text-muted-foreground">{t('welfareSettings.columns.name')}</span>
                <Input value={form.name ?? ''} onChange={(event) => setForm((current) => ({ ...current, name: event.target.value }))} />
              </label>
              <label className="space-y-2 md:col-span-2">
                <span className="text-sm text-muted-foreground">{t('welfareSettings.columns.description')}</span>
                <textarea
                  className="min-h-24 w-full rounded-md border border-input bg-background px-3 py-2 text-sm outline-none"
                  value={form.description ?? ''}
                  onChange={(event) => setForm((current) => ({ ...current, description: event.target.value }))}
                />
              </label>
              <label className="space-y-2">
                <span className="text-sm text-muted-foreground">{t('welfareSettings.columns.mode')}</span>
                <select
                  className="h-10 rounded-md border border-input bg-background px-3 text-sm"
                  value={form.dist_mode}
                  onChange={(event) => setForm((current) => ({ ...current, dist_mode: event.target.value as 'per_user' | 'per_character' }))}
                >
                  <option value="per_user">{t('welfareSettings.distModePerUser')}</option>
                  <option value="per_character">{t('welfareSettings.distModePerCharacter')}</option>
                </select>
              </label>
              <label className="space-y-2">
                <span className="text-sm text-muted-foreground">{t('welfareSettings.columns.status')}</span>
                <select
                  className="h-10 rounded-md border border-input bg-background px-3 text-sm"
                  value={String(form.status ?? 1)}
                  onChange={(event) => setForm((current) => ({ ...current, status: Number(event.target.value) }))}
                >
                  <option value="1">{t('welfareSettings.statusActive')}</option>
                  <option value="0">{t('welfareSettings.statusDisabled')}</option>
                </select>
              </label>
              <label className="space-y-2">
                <span className="text-sm text-muted-foreground">{t('welfareSettings.payByFuxiCoin')}</span>
                <Input
                  type="number"
                  value={String(form.pay_by_fuxi_coin ?? '')}
                  onChange={(event) =>
                    setForm((current) => ({
                      ...current,
                      pay_by_fuxi_coin: event.target.value ? Number(event.target.value) : null,
                    }))
                  }
                />
              </label>
              <label className="space-y-2">
                <span className="text-sm text-muted-foreground">{t('welfareSettings.sortOrder')}</span>
                <Input
                  type="number"
                  value={String(form.sort_order ?? 0)}
                  onChange={(event) => setForm((current) => ({ ...current, sort_order: Number(event.target.value) }))}
                />
              </label>
              <label className="flex items-center gap-2">
                <input
                  type="checkbox"
                  checked={Boolean(form.require_skill_plan)}
                  onChange={(event) => setForm((current) => ({ ...current, require_skill_plan: event.target.checked }))}
                />
                <span className="text-sm text-muted-foreground">{t('welfareSettings.requireSkillPlan')}</span>
              </label>
              <label className="flex items-center gap-2">
                <input
                  type="checkbox"
                  checked={Boolean(form.require_evidence)}
                  onChange={(event) => setForm((current) => ({ ...current, require_evidence: event.target.checked }))}
                />
                <span className="text-sm text-muted-foreground">{t('welfareSettings.requireEvidence')}</span>
              </label>
              <label className="space-y-2 md:col-span-2">
                <span className="text-sm text-muted-foreground">{t('welfareSettings.exampleEvidence')}</span>
                <Input
                  type="file"
                  accept="image/*"
                  disabled={exampleUploading}
                  onChange={(event) => {
                    const file = event.target.files?.[0]
                    if (file) void handleExampleUpload(file)
                  }}
                />
                {form.example_evidence ? (
                  <img src={form.example_evidence} alt="" className="max-h-36 rounded border" />
                ) : null}
              </label>
            </div>
            <div className="mt-5 flex justify-end gap-3">
              <Button type="button" variant="outline" onClick={() => setDialogVisible(false)}>
                {t('common.cancel')}
              </Button>
              <Button type="button" onClick={() => void save()} disabled={saving}>
                {saving ? t('welfareSettings.saving') : t('common.confirm')}
              </Button>
            </div>
          </div>
        </div>
      ) : null}
    </section>
  )
}
