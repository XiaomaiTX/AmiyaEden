import { useCallback, useEffect, useMemo, useState } from 'react'
import { Button } from '@/components/ui/button'
import {
  fetchCreateEsiRoleMapping,
  fetchCreateEsiTitleMapping,
  fetchDeleteEsiRoleMapping,
  fetchDeleteEsiTitleMapping,
  fetchGetAllEsiRoles,
  fetchGetCorpTitles,
  fetchGetEsiRoleMappings,
  fetchGetEsiTitleMappings,
  fetchGetRoleDefinitions,
  fetchTriggerAutoRoleSync,
} from '@/api/system-manage'
import { notifyError, notifySuccess } from '@/feedback'
import { useI18n } from '@/i18n'
import type { CorpTitleInfo, EsiRoleMapping, EsiTitleMapping, RoleDefinition } from '@/types/api/system-manage'
import { ShopDialog, formatDateTime } from './shop-page-utils'

type AutoRoleTab = 'esi-role' | 'title'

type EsiRoleFormState = {
  esi_role: string
  role_code: string
}

type TitleFormState = {
  title_key: string
  corporation_id: number
  title_id: number
  title_name: string
  role_code: string
}

const defaultEsiRoleForm: EsiRoleFormState = {
  esi_role: '',
  role_code: '',
}

const defaultTitleForm: TitleFormState = {
  title_key: '',
  corporation_id: 0,
  title_id: 0,
  title_name: '',
  role_code: '',
}

export function SystemAutoRolePage() {
  const { t } = useI18n()
  const [activeTab, setActiveTab] = useState<AutoRoleTab>('esi-role')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [esiRoles, setEsiRoles] = useState<string[]>([])
  const [roleDefinitions, setRoleDefinitions] = useState<RoleDefinition[]>([])
  const [corpTitles, setCorpTitles] = useState<CorpTitleInfo[]>([])
  const [esiRoleMappings, setEsiRoleMappings] = useState<EsiRoleMapping[]>([])
  const [titleMappings, setTitleMappings] = useState<EsiTitleMapping[]>([])
  const [syncing, setSyncing] = useState(false)
  const [esiRoleDialogOpen, setEsiRoleDialogOpen] = useState(false)
  const [titleDialogOpen, setTitleDialogOpen] = useState(false)
  const [esiRoleSaving, setEsiRoleSaving] = useState(false)
  const [titleSaving, setTitleSaving] = useState(false)
  const [esiRoleForm, setEsiRoleForm] = useState<EsiRoleFormState>(defaultEsiRoleForm)
  const [titleForm, setTitleForm] = useState<TitleFormState>(defaultTitleForm)

  const systemRoles = useMemo(
    () => roleDefinitions.filter((role) => role.code !== 'super_admin').sort((a, b) => a.sort - b.sort),
    [roleDefinitions]
  )
  const corpTitleNameMap = useMemo(
    () => new Map(corpTitles.map((item) => [item.corporation_id, item.corporation_name])),
    [corpTitles]
  )

  const loadCatalogs = useCallback(async () => {
    try {
      const [roles, definitions, titles] = await Promise.all([
        fetchGetAllEsiRoles(),
        fetchGetRoleDefinitions(),
        fetchGetCorpTitles(),
      ])
      setEsiRoles(roles)
      setRoleDefinitions(definitions)
      setCorpTitles(titles)
    } catch {
      setError(t('autoRolePage.messages.loadBaseFailed'))
      setEsiRoles([])
      setRoleDefinitions([])
      setCorpTitles([])
    }
  }, [t])

  const loadMappings = useCallback(async () => {
    try {
      const [esi, title] = await Promise.all([fetchGetEsiRoleMappings(), fetchGetEsiTitleMappings()])
      setEsiRoleMappings(esi)
      setTitleMappings(title)
    } catch {
      setError(t('autoRolePage.messages.loadFailed'))
      setEsiRoleMappings([])
      setTitleMappings([])
    }
  }, [t])

  const loadAll = useCallback(async () => {
    setLoading(true)
    setError(null)
    await Promise.all([loadCatalogs(), loadMappings()])
    setLoading(false)
  }, [loadCatalogs, loadMappings])

  useEffect(() => {
    let active = true
    void (async () => {
      if (!active) {
        return
      }
      await loadAll()
    })()

    return () => {
      active = false
    }
  }, [loadAll])

  const reloadMappings = async () => {
    await loadMappings()
  }

  const createEsiRoleMapping = async () => {
    if (!esiRoleForm.esi_role || !esiRoleForm.role_code) {
      notifyError(t('autoRolePage.messages.validationFailed'))
      return
    }

    setEsiRoleSaving(true)
    try {
      await fetchCreateEsiRoleMapping(esiRoleForm)
      notifySuccess(t('autoRolePage.messages.created'))
      setEsiRoleDialogOpen(false)
      setEsiRoleForm(defaultEsiRoleForm)
      await reloadMappings()
    } catch {
      notifyError(t('autoRolePage.messages.createFailed'))
    } finally {
      setEsiRoleSaving(false)
    }
  }

  const createTitleMapping = async () => {
    if (!titleForm.corporation_id || !titleForm.title_id || !titleForm.role_code) {
      notifyError(t('autoRolePage.messages.validationFailed'))
      return
    }

    setTitleSaving(true)
    try {
      await fetchCreateEsiTitleMapping({
        corporation_id: titleForm.corporation_id,
        title_id: titleForm.title_id,
        title_name: titleForm.title_name || undefined,
        role_code: titleForm.role_code,
      })
      notifySuccess(t('autoRolePage.messages.created'))
      setTitleDialogOpen(false)
      setTitleForm(defaultTitleForm)
      await reloadMappings()
    } catch {
      notifyError(t('autoRolePage.messages.createFailed'))
    } finally {
      setTitleSaving(false)
    }
  }

  const deleteEsiRoleMapping = async (id: number) => {
    if (!window.confirm(t('autoRolePage.deleteConfirm'))) {
      return
    }

    try {
      await fetchDeleteEsiRoleMapping(id)
      notifySuccess(t('autoRolePage.messages.deleted'))
      await reloadMappings()
    } catch {
      notifyError(t('autoRolePage.messages.deleteFailed'))
    }
  }

  const deleteTitleMapping = async (id: number) => {
    if (!window.confirm(t('autoRolePage.deleteConfirm'))) {
      return
    }

    try {
      await fetchDeleteEsiTitleMapping(id)
      notifySuccess(t('autoRolePage.messages.deleted'))
      await reloadMappings()
    } catch {
      notifyError(t('autoRolePage.messages.deleteFailed'))
    }
  }

  const triggerSync = async () => {
    setSyncing(true)
    try {
      await fetchTriggerAutoRoleSync()
      notifySuccess(t('autoRolePage.messages.syncTriggered'))
    } catch {
      notifyError(t('autoRolePage.messages.syncFailed'))
    } finally {
      setSyncing(false)
    }
  }

  return (
    <section className="space-y-4">
      <div className="rounded-lg border bg-card p-5">
        <div className="flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
          <div>
            <h1 className="text-xl font-semibold">{t('autoRolePage.title')}</h1>
            <p className="mt-1 text-sm text-muted-foreground">{t('autoRolePage.description')}</p>
          </div>
          <Button type="button" onClick={() => void triggerSync()} disabled={syncing}>
            {syncing ? t('autoRolePage.messages.syncing') : t('autoRolePage.triggerSync')}
          </Button>
        </div>
      </div>

      {error ? <p className="text-sm text-destructive">{error}</p> : null}
      {loading ? <p className="text-sm text-muted-foreground">{t('autoRolePage.messages.loading')}</p> : null}

      <div className="flex flex-wrap gap-2 rounded-lg border bg-card p-2">
        {([
          ['esi-role', t('autoRolePage.tabs.esiRole')],
          ['title', t('autoRolePage.tabs.title')],
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

      {activeTab === 'esi-role' ? (
        <div className="space-y-4">
          <div className="flex items-center justify-between gap-3">
            <p className="text-sm text-muted-foreground">{t('autoRolePage.descriptions.esiRole')}</p>
            <Button type="button" onClick={() => setEsiRoleDialogOpen(true)}>
              {t('autoRolePage.addMapping')}
            </Button>
          </div>

          <div className="overflow-hidden rounded-lg border bg-card">
            <div className="overflow-x-auto">
              <table className="min-w-full text-sm">
                <thead>
                  <tr className="border-b bg-muted/40 text-left">
                    <th className="px-3 py-2">#</th>
                    <th className="px-3 py-2">{t('autoRolePage.columns.esiRole')}</th>
                    <th className="px-3 py-2">{t('autoRolePage.columns.mappedRole')}</th>
                    <th className="px-3 py-2">{t('common.createdAt')}</th>
                    <th className="px-3 py-2">{t('common.operation')}</th>
                  </tr>
                </thead>
                <tbody>
                  {esiRoleMappings.map((row, index) => (
                    <tr key={row.id} className="border-b align-top">
                      <td className="px-3 py-2">{index + 1}</td>
                      <td className="px-3 py-2">
                        <span className="rounded-full bg-amber-100 px-2 py-0.5 text-xs font-medium text-amber-700 dark:bg-amber-500/10 dark:text-amber-300">
                          {row.esi_role}
                        </span>
                      </td>
                      <td className="px-3 py-2">
                        <span className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${getRoleTone(row.role_code)}`}>
                          {row.role_name || row.role_code}
                        </span>
                        <span className="ml-2 text-xs text-muted-foreground">{row.role_code}</span>
                      </td>
                      <td className="px-3 py-2">{formatDateTime(row.created_at)}</td>
                      <td className="px-3 py-2">
                        <Button type="button" size="sm" variant="outline" onClick={() => void deleteEsiRoleMapping(row.id)}>
                          {t('common.delete')}
                        </Button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        </div>
      ) : null}

      {activeTab === 'title' ? (
        <div className="space-y-4">
          <div className="flex items-center justify-between gap-3">
            <p className="text-sm text-muted-foreground">{t('autoRolePage.descriptions.title')}</p>
            <Button type="button" onClick={() => setTitleDialogOpen(true)}>
              {t('autoRolePage.addMapping')}
            </Button>
          </div>

          <div className="overflow-hidden rounded-lg border bg-card">
            <div className="overflow-x-auto">
              <table className="min-w-full text-sm">
                <thead>
                  <tr className="border-b bg-muted/40 text-left">
                    <th className="px-3 py-2">#</th>
                    <th className="px-3 py-2">{t('autoRolePage.columns.corporationId')}</th>
                    <th className="px-3 py-2">{t('autoRolePage.columns.titleId')}</th>
                    <th className="px-3 py-2">{t('autoRolePage.columns.titleName')}</th>
                    <th className="px-3 py-2">{t('autoRolePage.columns.mappedRole')}</th>
                    <th className="px-3 py-2">{t('common.createdAt')}</th>
                    <th className="px-3 py-2">{t('common.operation')}</th>
                  </tr>
                </thead>
                <tbody>
                  {titleMappings.map((row, index) => (
                    <tr key={row.id} className="border-b align-top">
                      <td className="px-3 py-2">{index + 1}</td>
                      <td className="px-3 py-2">{corpTitleNameMap.get(row.corporation_id) || row.corporation_id}</td>
                      <td className="px-3 py-2">{row.title_id}</td>
                      <td className="px-3 py-2">{row.title_name || t('autoRolePage.titleFallback', { id: row.title_id })}</td>
                      <td className="px-3 py-2">
                        <span className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${getRoleTone(row.role_code)}`}>
                          {row.role_name || row.role_code}
                        </span>
                        <span className="ml-2 text-xs text-muted-foreground">{row.role_code}</span>
                      </td>
                      <td className="px-3 py-2">{formatDateTime(row.created_at)}</td>
                      <td className="px-3 py-2">
                        <Button type="button" size="sm" variant="outline" onClick={() => void deleteTitleMapping(row.id)}>
                          {t('common.delete')}
                        </Button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        </div>
      ) : null}

      <ShopDialog
        open={esiRoleDialogOpen}
        title={t('autoRolePage.createEsiRoleTitle')}
        onClose={() => setEsiRoleDialogOpen(false)}
        closeLabel={t('common.close')}
        widthClass="max-w-lg"
        footer={
          <>
            <Button type="button" variant="outline" onClick={() => setEsiRoleDialogOpen(false)} disabled={esiRoleSaving}>
              {t('common.cancel')}
            </Button>
            <Button type="button" onClick={() => void createEsiRoleMapping()} disabled={esiRoleSaving}>
              {esiRoleSaving ? t('autoRolePage.messages.saving') : t('common.confirm')}
            </Button>
          </>
        }
      >
        <div className="space-y-4">
          <label className="space-y-2">
            <span className="text-sm text-muted-foreground">{t('autoRolePage.fields.esiRole')}</span>
            <select
              className="h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
              value={esiRoleForm.esi_role}
              onChange={(event) =>
                setEsiRoleForm((current) => ({ ...current, esi_role: event.target.value }))
              }
            >
              <option value="">{t('autoRolePage.placeholders.esiRole')}</option>
              {esiRoles.map((role) => (
                <option key={role} value={role}>
                  {role}
                </option>
              ))}
            </select>
          </label>
          <label className="space-y-2">
            <span className="text-sm text-muted-foreground">{t('autoRolePage.fields.systemRole')}</span>
            <select
              className="h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
              value={esiRoleForm.role_code}
              onChange={(event) =>
                setEsiRoleForm((current) => ({ ...current, role_code: event.target.value }))
              }
            >
              <option value="">{t('autoRolePage.placeholders.systemRole')}</option>
              {systemRoles.map((role) => (
                <option key={role.code} value={role.code}>
                  {role.name} ({role.code})
                </option>
              ))}
            </select>
          </label>
        </div>
      </ShopDialog>

      <ShopDialog
        open={titleDialogOpen}
        title={t('autoRolePage.createTitleTitle')}
        onClose={() => setTitleDialogOpen(false)}
        closeLabel={t('common.close')}
        widthClass="max-w-xl"
        footer={
          <>
            <Button type="button" variant="outline" onClick={() => setTitleDialogOpen(false)} disabled={titleSaving}>
              {t('common.cancel')}
            </Button>
            <Button type="button" onClick={() => void createTitleMapping()} disabled={titleSaving}>
              {titleSaving ? t('autoRolePage.messages.saving') : t('common.confirm')}
            </Button>
          </>
        }
      >
        <div className="space-y-4">
          <label className="space-y-2">
            <span className="text-sm text-muted-foreground">{t('autoRolePage.fields.title')}</span>
            <select
              className="h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
              value={titleForm.title_key}
              onChange={(event) => {
                const selected = corpTitles.find(
                  (item) => `${item.corporation_id}_${item.title_id}` === event.target.value
                )
                if (!selected) {
                  setTitleForm(defaultTitleForm)
                  return
                }

                setTitleForm({
                  title_key: event.target.value,
                  corporation_id: selected.corporation_id,
                  title_id: selected.title_id,
                  title_name: selected.title_name,
                  role_code: titleForm.role_code,
                })
              }}
            >
              <option value="">{t('autoRolePage.placeholders.title')}</option>
              {corpTitles.map((item) => (
                <option key={`${item.corporation_id}_${item.title_id}`} value={`${item.corporation_id}_${item.title_id}`}>
                  {item.title_name || t('autoRolePage.titleFallback', { id: item.title_id })} - {item.corporation_name || t('autoRolePage.corpFallback', { id: item.corporation_id })}
                </option>
              ))}
            </select>
          </label>
          <label className="space-y-2">
            <span className="text-sm text-muted-foreground">{t('autoRolePage.fields.systemRole')}</span>
            <select
              className="h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
              value={titleForm.role_code}
              onChange={(event) =>
                setTitleForm((current) => ({ ...current, role_code: event.target.value }))
              }
            >
              <option value="">{t('autoRolePage.placeholders.systemRole')}</option>
              {systemRoles.map((role) => (
                <option key={role.code} value={role.code}>
                  {role.name} ({role.code})
                </option>
              ))}
            </select>
          </label>
        </div>
      </ShopDialog>
    </section>
  )
}

function getRoleTone(roleCode: string) {
  switch (roleCode) {
    case 'super_admin':
      return 'bg-red-100 text-red-700 dark:bg-red-500/10 dark:text-red-300'
    case 'admin':
      return 'bg-amber-100 text-amber-700 dark:bg-amber-500/10 dark:text-amber-300'
    case 'senior_fc':
    case 'fc':
      return 'bg-sky-100 text-sky-700 dark:bg-sky-500/10 dark:text-sky-300'
    case 'captain':
      return 'bg-indigo-100 text-indigo-700 dark:bg-indigo-500/10 dark:text-indigo-300'
    case 'mentor':
    case 'welfare':
      return 'bg-teal-100 text-teal-700 dark:bg-teal-500/10 dark:text-teal-300'
    case 'srp':
    case 'shop_order_manage':
      return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-300'
    default:
      return 'bg-muted text-muted-foreground'
  }
}
