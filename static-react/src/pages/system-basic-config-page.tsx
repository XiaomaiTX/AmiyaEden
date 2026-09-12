import { Textarea } from '@/components/ui/textarea'
import { useCallback, useEffect, useState } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { useI18n } from '@/i18n'
import { notifyError, notifySuccess } from '@/feedback'
import {
  fetchAllowCorporations,
  fetchBasicConfig,
  fetchMumbleConfig,
  fetchSDEConfig,
  updateAllowCorporations,
  updateMumbleConfig,
  updateSDEConfig,
} from '@/api/sys-config'
import type {
  AllowCorporationsConfig,
  BasicConfig,
  MumbleConfig,
  SDEConfig,
} from '@/types/api/sys-config'

const defaultSdeForm: SDEConfig = {
  api_key: '',
  proxy: '',
  download_url: '',
}

const defaultMumbleForm: MumbleConfig = {
  service_token: '',
  server_url: '',
  revalidate_token: '',
  revalidate_timeout_ms: 1000,
  public_address: '',
  public_port: 0,
}

function parseCorporationId(raw: string) {
  const normalized = raw.trim()
  if (!/^\d+$/.test(normalized)) {
    throw new Error('invalid')
  }

  const parsed = Number(normalized)
  if (!Number.isSafeInteger(parsed) || parsed <= 0) {
    throw new Error('invalid')
  }

  return parsed
}

export function SystemBasicConfigPage() {
  const { t } = useI18n()
  const [basicConfig, setBasicConfig] = useState<BasicConfig | null>(null)
  const [allowCorporationsInput, setAllowCorporationsInput] = useState('')
  const [allowCorporationsSaving, setAllowCorporationsSaving] = useState(false)
  const [sdeForm, setSdeForm] = useState<SDEConfig>(defaultSdeForm)
  const [sdeSaving, setSdeSaving] = useState(false)
  const [mumbleForm, setMumbleForm] = useState<MumbleConfig>(defaultMumbleForm)
  const [mumbleSaving, setMumbleSaving] = useState(false)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const loadData = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const [basic, allowCorps, sde, mumble] = await Promise.all([
        fetchBasicConfig(),
        fetchAllowCorporations(),
        fetchSDEConfig(),
        fetchMumbleConfig(),
      ])
      setBasicConfig(basic)
      setAllowCorporationsInput(renderAllowCorporations(basic.corp_id, allowCorps))
      setSdeForm(sde)
      setMumbleForm(mumble)
    } catch {
      setError(t('systemBasicConfig.messages.loadFailed'))
    } finally {
      setLoading(false)
    }
  }, [t])

  useEffect(() => {
    let active = true
    void (async () => {
      if (!active) {
        return
      }
      await loadData()
    })()

    return () => {
      active = false
    }
  }, [loadData])

  const saveAllowCorporations = async () => {
    if (!basicConfig) {
      return
    }

    setAllowCorporationsSaving(true)
    try {
      const corps = normalizeCorporations(basicConfig.corp_id, allowCorporationsInput)
      await updateAllowCorporations({ allow_corporations: corps })
      setAllowCorporationsInput(corps.join('\n'))
      notifySuccess(t('systemBasicConfig.messages.saveSuccess'))
    } catch (caughtError) {
      const message =
        caughtError instanceof Error && caughtError.message === 'invalid'
          ? t('systemBasicConfig.messages.invalidCorpId')
          : t('systemBasicConfig.messages.saveFailed')
      notifyError(message)
    } finally {
      setAllowCorporationsSaving(false)
    }
  }

  const saveSdeConfig = async () => {
    setSdeSaving(true)
    try {
      await updateSDEConfig(sdeForm)
      notifySuccess(t('systemBasicConfig.messages.saveSuccess'))
    } catch {
      notifyError(t('systemBasicConfig.messages.saveFailed'))
    } finally {
      setSdeSaving(false)
    }
  }

  const saveMumbleConfig = async () => {
    setMumbleSaving(true)
    try {
      await updateMumbleConfig(mumbleForm)
      notifySuccess(t('systemBasicConfig.messages.saveSuccess'))
    } catch {
      notifyError(t('systemBasicConfig.messages.saveFailed'))
    } finally {
      setMumbleSaving(false)
    }
  }

  return (
    <section className="space-y-4">
      <div className="rounded-lg border bg-card p-5">
        <h1 className="text-xl font-semibold">{t('systemBasicConfig.title')}</h1>
        <p className="mt-1 text-sm text-muted-foreground">{t('systemBasicConfig.subtitle')}</p>
      </div>

      {error ? <p className="text-sm text-destructive">{error}</p> : null}
      {loading ? (
        <p className="text-sm text-muted-foreground">{t('systemBasicConfig.loading')}</p>
      ) : null}

      <div className="grid gap-4 xl:grid-cols-2">
        <div className="rounded-lg border bg-card p-5">
          <h2 className="text-base font-semibold">{t('systemBasicConfig.basicInfo.title')}</h2>
          <div className="mt-4 grid gap-3 sm:grid-cols-2">
            <InfoCard
              label={t('systemBasicConfig.basicInfo.corpId')}
              value={basicConfig?.corp_id ?? '-'}
            />
            <InfoCard
              label={t('systemBasicConfig.basicInfo.siteTitle')}
              value={basicConfig?.site_title || '-'}
            />
          </div>
        </div>

        <div className="rounded-lg border bg-card p-5">
          <div className="flex items-center justify-between gap-3">
            <div>
              <h2 className="text-base font-semibold">
                {t('systemBasicConfig.allowCorporations.title')}
              </h2>
              <p className="mt-1 text-sm text-muted-foreground">
                {t('systemBasicConfig.allowCorporations.subtitle', {
                  corpId: basicConfig?.corp_id ?? 0,
                })}
              </p>
            </div>
            <Button
              type="button"
              onClick={() => void saveAllowCorporations()}
              isDisabled={allowCorporationsSaving}
            >
              {allowCorporationsSaving ? t('systemBasicConfig.messages.saving') : t('common.save')}
            </Button>
          </div>

          <div className="mt-4 space-y-2">
            <Textarea
              className="min-h-40"
              value={allowCorporationsInput}
              onChange={(event) => setAllowCorporationsInput(event.target.value)}
              placeholder={t('systemBasicConfig.allowCorporations.placeholder')}
            />
            <p className="text-xs text-muted-foreground">
              {t('systemBasicConfig.allowCorporations.hint', {
                corpId: basicConfig?.corp_id ?? 0,
              })}
            </p>
          </div>
        </div>
      </div>

      <div className="rounded-lg border bg-card p-5">
        <div className="flex items-center justify-between gap-3">
          <div>
            <h2 className="text-base font-semibold">{t('systemBasicConfig.mumble.title')}</h2>
            <p className="mt-1 text-sm text-muted-foreground">
              {t('systemBasicConfig.mumble.subtitle')}
            </p>
          </div>
          <Button type="button" onClick={() => void saveMumbleConfig()} isDisabled={mumbleSaving}>
            {mumbleSaving ? t('systemBasicConfig.messages.saving') : t('common.save')}
          </Button>
        </div>
        <div className="mt-4 grid gap-4 md:grid-cols-2">
          <label className="space-y-2 md:col-span-2">
            <span className="text-sm text-muted-foreground">
              {t('systemBasicConfig.mumble.serverUrl')}
            </span>
            <Input
              value={mumbleForm.server_url}
              onChange={(event) =>
                setMumbleForm((current) => ({ ...current, server_url: event.target.value }))
              }
              placeholder="http://go-mumble-server:64730"
            />
          </label>
          <label className="space-y-2">
            <span className="text-sm text-muted-foreground">
              {t('systemBasicConfig.mumble.publicAddress')}
            </span>
            <Input
              value={mumbleForm.public_address}
              onChange={(event) =>
                setMumbleForm((current) => ({ ...current, public_address: event.target.value }))
              }
              placeholder="mumble.example.com"
            />
          </label>
          <label className="space-y-2">
            <span className="text-sm text-muted-foreground">
              {t('systemBasicConfig.mumble.publicPort')}
            </span>
            <Input
              type="number"
              min={0}
              max={65535}
              value={mumbleForm.public_port}
              onChange={(event) =>
                setMumbleForm((current) => ({
                  ...current,
                  public_port: Number(event.target.value),
                }))
              }
            />
          </label>
          <label className="space-y-2">
            <span className="text-sm text-muted-foreground">
              {t('systemBasicConfig.mumble.serviceToken')}
            </span>
            <Input
              type="password"
              value={mumbleForm.service_token}
              onChange={(event) =>
                setMumbleForm((current) => ({ ...current, service_token: event.target.value }))
              }
            />
          </label>
          <label className="space-y-2">
            <span className="text-sm text-muted-foreground">
              {t('systemBasicConfig.mumble.revalidateToken')}
            </span>
            <Input
              type="password"
              value={mumbleForm.revalidate_token}
              onChange={(event) =>
                setMumbleForm((current) => ({ ...current, revalidate_token: event.target.value }))
              }
            />
          </label>
          <label className="space-y-2">
            <span className="text-sm text-muted-foreground">
              {t('systemBasicConfig.mumble.timeout')}
            </span>
            <Input
              type="number"
              min={100}
              max={10000}
              value={mumbleForm.revalidate_timeout_ms}
              onChange={(event) =>
                setMumbleForm((current) => ({
                  ...current,
                  revalidate_timeout_ms: Number(event.target.value),
                }))
              }
            />
          </label>
        </div>
      </div>

      <div className="rounded-lg border bg-card p-5">
        <div className="flex items-center justify-between gap-3">
          <div>
            <h2 className="text-base font-semibold">{t('systemBasicConfig.sdeConfig.title')}</h2>
            <p className="mt-1 text-sm text-muted-foreground">
              {t('systemBasicConfig.sdeConfig.subtitle')}
            </p>
          </div>
          <Button type="button" onClick={() => void saveSdeConfig()} isDisabled={sdeSaving}>
            {sdeSaving ? t('systemBasicConfig.messages.saving') : t('common.save')}
          </Button>
        </div>

        <div className="mt-4 grid gap-4 md:grid-cols-2">
          <label className="space-y-2 md:col-span-2">
            <span className="text-sm text-muted-foreground">
              {t('systemBasicConfig.sdeConfig.apiKey')}
            </span>
            <Input
              value={sdeForm.api_key}
              type="password"
              onChange={(event) =>
                setSdeForm((current) => ({ ...current, api_key: event.target.value }))
              }
              placeholder={t('systemBasicConfig.sdeConfig.apiKeyPlaceholder')}
            />
          </label>
          <label className="space-y-2">
            <span className="text-sm text-muted-foreground">
              {t('systemBasicConfig.sdeConfig.proxy')}
            </span>
            <Input
              value={sdeForm.proxy}
              onChange={(event) =>
                setSdeForm((current) => ({ ...current, proxy: event.target.value }))
              }
              placeholder={t('systemBasicConfig.sdeConfig.proxyPlaceholder')}
            />
          </label>
          <label className="space-y-2">
            <span className="text-sm text-muted-foreground">
              {t('systemBasicConfig.sdeConfig.downloadUrl')}
            </span>
            <Input
              value={sdeForm.download_url}
              onChange={(event) =>
                setSdeForm((current) => ({ ...current, download_url: event.target.value }))
              }
              placeholder={t('systemBasicConfig.sdeConfig.downloadUrlPlaceholder')}
            />
          </label>
        </div>
      </div>
    </section>
  )
}

function normalizeCorporations(corpId: number, raw: string) {
  const values = raw
    .split('\n')
    .map((line) => line.trim())
    .filter((line) => line !== '')
    .map(parseCorporationId)

  return Array.from(new Set([corpId, ...values]))
}

function renderAllowCorporations(corpId: number, allowCorps: AllowCorporationsConfig) {
  return Array.from(new Set([corpId, ...(allowCorps.allow_corporations ?? [])])).join('\n')
}

function InfoCard({ label, value }: { label: string; value: string | number }) {
  return (
    <div className="rounded-lg border bg-muted/20 px-4 py-3">
      <div className="text-xs text-muted-foreground">{label}</div>
      <div className="mt-1 text-sm font-medium">{value}</div>
    </div>
  )
}
