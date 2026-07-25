import { useCallback, useEffect, useMemo, useState } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { notifyError, notifySuccess } from '@/feedback'
import { useI18n } from '@/i18n'
import { fetchWebhookConfig, setWebhookConfig, testWebhook } from '@/api/webhook'
import type { WebhookConfig, WebhookTestParams } from '@/types/api/webhook'

const defaultWebhookConfig: WebhookConfig = {
  url: '',
  enabled: false,
  type: 'discord',
  fleet_template: '',
  ob_target_type: 'group',
  ob_target_id: 0,
  ob_token: '',
  qq_governance_group_ids: [],
}

const defaultWebhookTest: WebhookTestParams = {
  url: '',
  type: 'discord',
  content: '',
  ob_target_type: 'group',
  ob_target_id: 0,
  ob_token: '',
  qq_governance_group_ids: [],
}

function parseQQGroupIDs(text: string): number[] {
  const out: number[] = []
  for (const part of text.split(/[\s,，;；\n]+/)) {
    const trimmed = part.trim()
    if (!trimmed) continue
    const id = Number(trimmed)
    if (!Number.isFinite(id) || id <= 0 || !Number.isInteger(id)) {
      throw new Error('invalid')
    }
    out.push(id)
  }
  return out
}

function formatQQGroupIDs(ids: number[] | undefined): string {
  if (!ids || ids.length === 0) return ''
  return ids.join('\n')
}

export function SystemWebhookPage() {
  const { t } = useI18n()
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [testing, setTesting] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [config, setConfig] = useState<WebhookConfig>(defaultWebhookConfig)
  const [testForm, setTestForm] = useState<WebhookTestParams>(defaultWebhookTest)
  const [qqGroupIDsText, setQQGroupIDsText] = useState('')
  const [testQQGroupIDsText, setTestQQGroupIDsText] = useState('')

  const loadConfig = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const nextConfig = await fetchWebhookConfig()
      setConfig(nextConfig)
      setQQGroupIDsText(formatQQGroupIDs(nextConfig.qq_governance_group_ids))
      setTestForm({
        url: nextConfig.url,
        type: nextConfig.type,
        content: '',
        ob_target_type: nextConfig.ob_target_type,
        ob_target_id: nextConfig.ob_target_id,
        ob_token: nextConfig.ob_token,
        qq_governance_group_ids: nextConfig.qq_governance_group_ids ?? [],
      })
      setTestQQGroupIDsText(formatQQGroupIDs(nextConfig.qq_governance_group_ids))
    } catch {
      setError(t('webhook.messages.loadFailed'))
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
      await loadConfig()
    })()

    return () => {
      active = false
    }
  }, [loadConfig])

  const isQQGovernanceConfig = config.type === 'qq_governance_onebot'
  const isQQGovernanceTest = testForm.type === 'qq_governance_onebot'

  const canSubmitTest = useMemo(() => {
    if (isQQGovernanceTest) {
      return (testForm.qq_governance_group_ids?.length ?? 0) > 0
    }
    return !!testForm.url?.trim()
  }, [isQQGovernanceTest, testForm.qq_governance_group_ids, testForm.url])

  const saveConfig = async () => {
    let nextConfig: WebhookConfig
    if (isQQGovernanceConfig) {
      let ids: number[]
      try {
        ids = parseQQGroupIDs(qqGroupIDsText)
      } catch {
        notifyError(t('webhook.messages.invalidQQGroupId'))
        return
      }
      if (ids.length === 0) {
        notifyError(t('webhook.messages.qqGroupIdsRequired'))
        return
      }
      nextConfig = { ...config, qq_governance_group_ids: ids }
      setConfig(nextConfig)
    } else {
      nextConfig = { ...config, qq_governance_group_ids: [] }
      setConfig(nextConfig)
    }
    setSaving(true)
    try {
      await setWebhookConfig(nextConfig)
      notifySuccess(t('webhook.messages.saveSuccess'))
    } catch {
      notifyError(t('webhook.messages.saveFailed'))
    } finally {
      setSaving(false)
    }
  }

  const sendTest = async () => {
    let nextTest = testForm
    if (isQQGovernanceTest) {
      let ids: number[]
      try {
        ids = parseQQGroupIDs(testQQGroupIDsText)
      } catch {
        notifyError(t('webhook.messages.invalidQQGroupId'))
        return
      }
      if (ids.length === 0) {
        notifyError(t('webhook.messages.qqGroupIdsRequired'))
        return
      }
      nextTest = { ...testForm, qq_governance_group_ids: ids, url: undefined }
      setTestForm(nextTest)
    } else if (!nextTest.url?.trim()) {
      notifyError(t('webhook.messages.urlRequired'))
      return
    }
    setTesting(true)
    try {
      await testWebhook(nextTest)
      notifySuccess(t('webhook.test.success'))
    } catch {
      notifyError(t('webhook.test.failed'))
    } finally {
      setTesting(false)
    }
  }

  return (
    <section className="space-y-4">
      <div className="rounded-lg border bg-card p-5">
        <h1 className="text-xl font-semibold">{t('webhook.title')}</h1>
        <p className="mt-1 text-sm text-muted-foreground">{t('webhook.subtitle')}</p>
      </div>

      {error ? <p className="text-sm text-destructive">{error}</p> : null}
      {loading ? <p className="text-sm text-muted-foreground">{t('webhook.messages.loading')}</p> : null}

      <div className="rounded-lg border bg-card p-5">
        <div className="flex flex-col gap-3 border-b pb-4 lg:flex-row lg:items-start lg:justify-between">
          <div>
            <h2 className="text-base font-semibold">{t('webhook.config.title')}</h2>
            <p className="mt-1 text-sm text-muted-foreground">{t('webhook.config.subtitle')}</p>
          </div>
          <Button type="button" onClick={() => void saveConfig()} disabled={saving || loading}>
            {saving ? t('webhook.messages.saving') : t('common.save')}
          </Button>
        </div>

        <div className="mt-4 grid gap-4 md:grid-cols-2">
          <label className="flex items-center gap-3 md:col-span-2">
            <input
              checked={config.enabled}
              type="checkbox"
              onChange={(event) =>
                setConfig((current) => ({ ...current, enabled: event.target.checked }))
              }
            />
            <span className="text-sm">{t('webhook.fields.enabled')}</span>
          </label>
          <label className="space-y-2">
            <span className="text-sm text-muted-foreground">{t('webhook.fields.type')}</span>
            <select
              className="h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
              value={config.type}
              onChange={(event) =>
                setConfig((current) => ({ ...current, type: event.target.value }))
              }
            >
              <option value="discord">{t('webhook.types.discord')}</option>
              <option value="feishu">{t('webhook.types.feishu')}</option>
              <option value="dingtalk">{t('webhook.types.dingtalk')}</option>
              <option value="onebot">{t('webhook.types.onebot')}</option>
              <option value="qq_governance_onebot">{t('webhook.types.qqGovernanceOnebot')}</option>
            </select>
          </label>
          {!isQQGovernanceConfig ? (
            <label className="space-y-2">
              <span className="text-sm text-muted-foreground">{t('webhook.fields.url')}</span>
              <Input
                value={config.url}
                onChange={(event) => setConfig((current) => ({ ...current, url: event.target.value }))}
                placeholder={t('webhook.fields.urlPlaceholder')}
              />
            </label>
          ) : (
            <label className="space-y-2 md:col-span-2">
              <span className="text-sm text-muted-foreground">{t('webhook.fields.qqGroupIds')}</span>
              <textarea
                className="min-h-32 w-full rounded-lg border border-input bg-background px-3 py-2 text-sm outline-none"
                value={qqGroupIDsText}
                onChange={(event) => setQQGroupIDsText(event.target.value)}
                placeholder={t('webhook.fields.qqGroupIdsPlaceholder')}
              />
              <p className="text-xs text-muted-foreground">{t('webhook.fields.qqGroupIdsHint')}</p>
            </label>
          )}
          <label className="space-y-2 md:col-span-2">
            <span className="text-sm text-muted-foreground">{t('webhook.fields.template')}</span>
            <textarea
              className="min-h-40 w-full rounded-lg border border-input bg-background px-3 py-2 text-sm outline-none"
              value={config.fleet_template}
              onChange={(event) =>
                setConfig((current) => ({ ...current, fleet_template: event.target.value }))
              }
              placeholder={t('webhook.fields.templatePlaceholder')}
            />
            <p className="text-xs text-muted-foreground">{t('webhook.fields.templateHint')}</p>
          </label>
          {config.type === 'onebot' ? (
            <>
              <label className="space-y-2">
                <span className="text-sm text-muted-foreground">{t('webhook.fields.obTargetType')}</span>
                <select
                  className="h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
                  value={config.ob_target_type}
                  onChange={(event) =>
                    setConfig((current) => ({
                      ...current,
                      ob_target_type: event.target.value as 'group' | 'private',
                    }))
                  }
                >
                  <option value="group">{t('webhook.fields.obGroup')}</option>
                  <option value="private">{t('webhook.fields.obPrivate')}</option>
                </select>
              </label>
              <label className="space-y-2">
                <span className="text-sm text-muted-foreground">{t('webhook.fields.obTargetId')}</span>
                <Input
                  type="number"
                  min={0}
                  value={String(config.ob_target_id)}
                  onChange={(event) =>
                    setConfig((current) => ({
                      ...current,
                      ob_target_id: Number(event.target.value),
                    }))
                  }
                  placeholder={t('webhook.fields.obTargetIdPlaceholder')}
                />
              </label>
              <label className="space-y-2 md:col-span-2">
                <span className="text-sm text-muted-foreground">{t('webhook.fields.obToken')}</span>
                <Input
                  value={config.ob_token}
                  onChange={(event) =>
                    setConfig((current) => ({ ...current, ob_token: event.target.value }))
                  }
                  placeholder={t('webhook.fields.obTokenPlaceholder')}
                />
              </label>
            </>
          ) : null}
        </div>
      </div>

      <div className="rounded-lg border bg-card p-5">
        <div className="flex flex-col gap-3 border-b pb-4 lg:flex-row lg:items-start lg:justify-between">
          <div>
            <h2 className="text-base font-semibold">{t('webhook.test.title')}</h2>
            <p className="mt-1 text-sm text-muted-foreground">{t('webhook.test.subtitle')}</p>
          </div>
          <Button type="button" variant="outline" onClick={() => void sendTest()} disabled={testing || !canSubmitTest}>
            {testing ? t('webhook.messages.testing') : t('webhook.test.sendBtn')}
          </Button>
        </div>

        <div className="mt-4 grid gap-4 md:grid-cols-2">
          <label className="space-y-2">
            <span className="text-sm text-muted-foreground">{t('webhook.test.type')}</span>
            <select
              className="h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
              value={testForm.type}
              onChange={(event) =>
                setTestForm((current) => ({ ...current, type: event.target.value }))
              }
            >
              <option value="discord">{t('webhook.types.discord')}</option>
              <option value="feishu">{t('webhook.types.feishu')}</option>
              <option value="dingtalk">{t('webhook.types.dingtalk')}</option>
              <option value="onebot">{t('webhook.types.onebot')}</option>
              <option value="qq_governance_onebot">{t('webhook.types.qqGovernanceOnebot')}</option>
            </select>
          </label>
          {!isQQGovernanceTest ? (
            <label className="space-y-2">
              <span className="text-sm text-muted-foreground">{t('webhook.test.url')}</span>
              <Input
                value={testForm.url ?? ''}
                onChange={(event) => setTestForm((current) => ({ ...current, url: event.target.value }))}
                placeholder={t('webhook.fields.urlPlaceholder')}
              />
            </label>
          ) : (
            <label className="space-y-2 md:col-span-2">
              <span className="text-sm text-muted-foreground">{t('webhook.fields.qqGroupIds')}</span>
              <textarea
                className="min-h-32 w-full rounded-lg border border-input bg-background px-3 py-2 text-sm outline-none"
                value={testQQGroupIDsText}
                onChange={(event) => setTestQQGroupIDsText(event.target.value)}
                placeholder={t('webhook.fields.qqGroupIdsPlaceholder')}
              />
              <p className="text-xs text-muted-foreground">{t('webhook.test.qqGroupIdsHint')}</p>
            </label>
          )}
          <label className="space-y-2 md:col-span-2">
            <span className="text-sm text-muted-foreground">{t('webhook.test.content')}</span>
            <Input
              value={testForm.content ?? ''}
              onChange={(event) =>
                setTestForm((current) => ({ ...current, content: event.target.value }))
              }
              placeholder={t('webhook.test.contentPlaceholder')}
            />
          </label>
          {testForm.type === 'onebot' ? (
            <>
              <label className="space-y-2">
                <span className="text-sm text-muted-foreground">{t('webhook.fields.obTargetType')}</span>
                <select
                  className="h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
                  value={testForm.ob_target_type ?? 'group'}
                  onChange={(event) =>
                    setTestForm((current) => ({
                      ...current,
                      ob_target_type: event.target.value,
                    }))
                  }
                >
                  <option value="group">{t('webhook.fields.obGroup')}</option>
                  <option value="private">{t('webhook.fields.obPrivate')}</option>
                </select>
              </label>
              <label className="space-y-2">
                <span className="text-sm text-muted-foreground">{t('webhook.fields.obTargetId')}</span>
                <Input
                  type="number"
                  min={0}
                  value={String(testForm.ob_target_id ?? 0)}
                  onChange={(event) =>
                    setTestForm((current) => ({
                      ...current,
                      ob_target_id: Number(event.target.value),
                    }))
                  }
                />
              </label>
              <label className="space-y-2 md:col-span-2">
                <span className="text-sm text-muted-foreground">{t('webhook.fields.obToken')}</span>
                <Input
                  value={testForm.ob_token ?? ''}
                  onChange={(event) =>
                    setTestForm((current) => ({ ...current, ob_token: event.target.value }))
                  }
                  placeholder={t('webhook.fields.obTokenPlaceholder')}
                />
              </label>
            </>
          ) : null}
        </div>
      </div>
    </section>
  )
}
