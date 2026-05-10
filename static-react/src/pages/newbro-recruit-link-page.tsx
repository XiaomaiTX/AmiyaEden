import { useCallback, useEffect, useMemo, useState } from 'react'
import {
  fetchAdminNewbroRecruitSettings,
  fetchAdminRecruitLinks,
  fetchMyRecruitLinks,
  generateRecruitLink,
  updateAdminNewbroRecruitSettings,
} from '@/api/newbro'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { useI18n } from '@/i18n'
import { formatDateTime, formatNumber, getErrorMessage } from '@/pages/newbro-page-utils'
import { useSessionStore } from '@/stores'
import type { AdminRecruitLink, RecruitLink, RecruitSettings } from '@/types/api/newbro'

export function NewbroRecruitLinkPage() {
  const { t } = useI18n()
  const roles = useSessionStore((state) => state.roles)
  const isAdmin = useMemo(
    () => roles.includes('super_admin') || roles.includes('admin'),
    [roles]
  )

  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [myLinks, setMyLinks] = useState<RecruitLink[]>([])
  const [adminLinks, setAdminLinks] = useState<AdminRecruitLink[]>([])
  const [adminTotal, setAdminTotal] = useState(0)
  const [generating, setGenerating] = useState(false)
  const [adminLoading, setAdminLoading] = useState(false)
  const [settingsLoading, setSettingsLoading] = useState(false)
  const [settingsSaving, setSettingsSaving] = useState(false)
  const [settings, setSettings] = useState<RecruitSettings | null>(null)
  const [refreshSeed, setRefreshSeed] = useState(0)

  const loadMyLinks = useCallback(async () => {
    const links = await fetchMyRecruitLinks()
    setMyLinks(links)
  }, [])

  const loadAdminLinks = useCallback(async () => {
    if (!isAdmin) {
      setAdminLinks([])
      setAdminTotal(0)
      return
    }

    setAdminLoading(true)
    try {
      const response = await fetchAdminRecruitLinks({ current: 1, size: 20 })
      setAdminLinks(response.list ?? [])
      setAdminTotal(response.total ?? 0)
    } finally {
      setAdminLoading(false)
    }
  }, [isAdmin])

  const loadSettings = useCallback(async () => {
    if (!isAdmin) {
      setSettings(null)
      return
    }

    setSettingsLoading(true)
    try {
      setSettings(await fetchAdminNewbroRecruitSettings())
    } finally {
      setSettingsLoading(false)
    }
  }, [isAdmin])

  const loadAll = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      await Promise.all([loadMyLinks(), loadAdminLinks(), loadSettings()])
    } catch (caughtError) {
      setError(getErrorMessage(caughtError, t('newbroRecruitLink.messages.loadFailed')))
    } finally {
      setLoading(false)
    }
  }, [loadAdminLinks, loadMyLinks, loadSettings, t])

  useEffect(() => {
    void loadAll()
  }, [loadAll, refreshSeed])

  const handleGenerate = async () => {
    setGenerating(true)
    try {
      await generateRecruitLink()
      await loadMyLinks()
    } catch (caughtError) {
      setError(getErrorMessage(caughtError, t('newbroRecruitLink.messages.generateFailed')))
    } finally {
      setGenerating(false)
    }
  }

  const handleSaveSettings = async () => {
    if (!settings) {
      return
    }

    setSettingsSaving(true)
    try {
      const updated = await updateAdminNewbroRecruitSettings(settings)
      setSettings(updated)
    } catch (caughtError) {
      setError(getErrorMessage(caughtError, t('newbroRecruitLink.messages.saveFailed')))
    } finally {
      setSettingsSaving(false)
    }
  }

  return (
    <section className="space-y-4">
      <div className="rounded-lg border bg-card p-5">
        <div className="flex flex-col gap-3 lg:flex-row lg:items-end lg:justify-between">
          <div>
            <h1 className="text-xl font-semibold">{t('newbroRecruitLink.title')}</h1>
            <p className="mt-1 text-sm text-muted-foreground">{t('newbroRecruitLink.subtitle')}</p>
          </div>
          <Button type="button" variant="outline" onClick={() => setRefreshSeed((v) => v + 1)}>
            {t('common.refresh')}
          </Button>
        </div>
      </div>

      {error ? <p className="text-sm text-destructive">{error}</p> : null}
      {loading ? <p className="text-sm text-muted-foreground">{t('newbroRecruitLink.loading')}</p> : null}

      <div className="grid gap-4 xl:grid-cols-[1fr_0.9fr]">
        <div className="space-y-4">
          <div className="rounded-lg border bg-card p-5">
            <div className="flex items-center justify-between gap-3">
              <div>
                <h2 className="text-base font-semibold">{t('newbroRecruitLink.myLinksTitle')}</h2>
                <p className="mt-1 text-sm text-muted-foreground">{t('newbroRecruitLink.myLinksHint')}</p>
              </div>
              <Button type="button" onClick={() => void handleGenerate()} disabled={generating}>
                {generating ? t('newbroRecruitLink.generating') : t('newbroRecruitLink.generate')}
              </Button>
            </div>

            <div className="mt-4 space-y-3">
              {myLinks.map((link) => (
                <div key={link.id} className="rounded-md border p-4">
                  <div className="flex flex-col gap-2 md:flex-row md:items-center md:justify-between">
                    <div>
                      <div className="font-medium">{t('newbroRecruitLink.linkCode')}</div>
                      <div className="text-sm text-muted-foreground">{link.code}</div>
                      <div className="text-sm text-muted-foreground">
                        {t('newbroRecruitLink.generatedAt')}: {formatDateTime(link.generated_at)}
                      </div>
                    </div>
                    <div className="text-sm text-muted-foreground">
                      {t('newbroRecruitLink.entryCount')}: {formatNumber(link.entries.length)}
                    </div>
                  </div>
                </div>
              ))}

              {!loading && myLinks.length === 0 ? (
                <p className="rounded-md border border-dashed p-6 text-center text-sm text-muted-foreground">
                  {t('newbroRecruitLink.noLinks')}
                </p>
              ) : null}
            </div>
          </div>

          {isAdmin ? (
            <div className="rounded-lg border bg-card p-5">
              <div className="flex items-center justify-between">
                <h2 className="text-base font-semibold">{t('newbroRecruitLink.adminLinksTitle')}</h2>
                <span className="text-sm text-muted-foreground">{adminTotal}</span>
              </div>

              <div className="mt-4 space-y-3">
                {adminLinks.map((link) => (
                  <div key={link.id} className="rounded-md border p-4">
                    <div className="flex items-center justify-between gap-3">
                      <div>
                        <div className="font-medium">{link.code}</div>
                        <div className="text-sm text-muted-foreground">{link.user_id}</div>
                      </div>
                      <div className="text-sm text-muted-foreground">{formatDateTime(link.generated_at)}</div>
                    </div>
                  </div>
                ))}

                {!adminLoading && adminLinks.length === 0 ? (
                  <p className="rounded-md border border-dashed p-6 text-center text-sm text-muted-foreground">
                    {t('newbroRecruitLink.noAdminLinks')}
                  </p>
                ) : null}
              </div>
            </div>
          ) : null}
        </div>

        {isAdmin ? (
          <div className="rounded-lg border bg-card p-5">
            <h2 className="text-base font-semibold">{t('newbroRecruitLink.settingsTitle')}</h2>
            <p className="mt-1 text-sm text-muted-foreground">{t('newbroRecruitLink.settingsHint')}</p>

            {settings ? (
              <div className="mt-4 space-y-4">
                <label className="block space-y-1">
                  <span className="text-sm text-muted-foreground">{t('newbroRecruitLink.fields.recruitQqUrl')}</span>
                  <Input
                    value={settings.recruit_qq_url}
                    onChange={(event) => setSettings({ ...settings, recruit_qq_url: event.target.value })}
                  />
                </label>
                <label className="block space-y-1">
                  <span className="text-sm text-muted-foreground">{t('newbroRecruitLink.fields.rewardAmount')}</span>
                  <Input
                    type="number"
                    value={settings.recruit_reward_amount}
                    onChange={(event) =>
                      setSettings({ ...settings, recruit_reward_amount: Number(event.target.value) })
                    }
                  />
                </label>
                <label className="block space-y-1">
                  <span className="text-sm text-muted-foreground">{t('newbroRecruitLink.fields.cooldownDays')}</span>
                  <Input
                    type="number"
                    value={settings.recruit_cooldown_days}
                    onChange={(event) =>
                      setSettings({ ...settings, recruit_cooldown_days: Number(event.target.value) })
                    }
                  />
                </label>
                <Button type="button" onClick={() => void handleSaveSettings()} disabled={settingsSaving}>
                  {settingsSaving ? t('newbroRecruitLink.saving') : t('newbroRecruitLink.save')}
                </Button>
              </div>
            ) : (
              <p className="mt-4 text-sm text-muted-foreground">
                {settingsLoading ? t('newbroRecruitLink.loadingSettings') : t('newbroRecruitLink.noSettings')}
              </p>
            )}
          </div>
        ) : null}
      </div>
    </section>
  )
}
