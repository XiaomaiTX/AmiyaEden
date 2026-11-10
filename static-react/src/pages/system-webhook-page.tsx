import { Textarea } from '@/components/ui/textarea'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { useCallback, useEffect, useState } from 'react'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
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
}

const defaultWebhookTest: WebhookTestParams = {
  url: '',
  type: 'discord',
  content: '',
}

export function SystemWebhookPage() {
  const { t } = useI18n()
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [testing, setTesting] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [config, setConfig] = useState<WebhookConfig>(defaultWebhookConfig)
  const [testForm, setTestForm] = useState<WebhookTestParams>(defaultWebhookTest)

  const loadConfig = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const nextConfig = await fetchWebhookConfig()
      setConfig(nextConfig)
      setTestForm({
        url: nextConfig.url,
        type: nextConfig.type,
        content: '',
      })
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

  const canSubmitTest = !!testForm.url?.trim()

  const saveConfig = async () => {
    setSaving(true)
    try {
      await setWebhookConfig(config)
      notifySuccess(t('webhook.messages.saveSuccess'))
    } catch {
      notifyError(t('webhook.messages.saveFailed'))
    } finally {
      setSaving(false)
    }
  }

  const sendTest = async () => {
    setTesting(true)
    try {
      await testWebhook(testForm)
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
      {loading ? (
        <p className="text-sm text-muted-foreground">{t('webhook.messages.loading')}</p>
      ) : null}

      <div className="rounded-lg border bg-card p-5">
        <div className="flex flex-col gap-3 border-b pb-4 lg:flex-row lg:items-start lg:justify-between">
          <div>
            <h2 className="text-base font-semibold">{t('webhook.config.title')}</h2>
            <p className="mt-1 text-sm text-muted-foreground">{t('webhook.config.subtitle')}</p>
          </div>
          <Button type="button" onClick={() => void saveConfig()} isDisabled={saving || loading}>
            {saving ? t('webhook.messages.saving') : t('common.save')}
          </Button>
        </div>

        <div className="mt-4 grid gap-4 md:grid-cols-2">
          <label className="flex items-center gap-3 md:col-span-2">
            <Checkbox
              isSelected={config.enabled}
              onChange={(selected) =>
                setConfig((current) => ({ ...current, enabled: selected === true }))
              }
            />
            <span className="text-sm">{t('webhook.fields.enabled')}</span>
          </label>
          <label className="space-y-2">
            <span className="text-sm text-muted-foreground">{t('webhook.fields.type')}</span>
            <Select
              selectedKey={String(config.type ?? '')}
              onSelectionChange={(key) => ((value) => setConfig((current) => ({ ...current, type: value })))(String(key))}
            >
              <SelectTrigger className="h-10 w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem id="discord">{t('webhook.types.discord')}</SelectItem>
                <SelectItem id="feishu">{t('webhook.types.feishu')}</SelectItem>
                <SelectItem id="dingtalk">{t('webhook.types.dingtalk')}</SelectItem>
              </SelectContent>
            </Select>
          </label>

            <label className="space-y-2">
              <span className="text-sm text-muted-foreground">{t('webhook.fields.url')}</span>
              <Input
                value={config.url}
                onChange={(event) =>
                  setConfig((current) => ({ ...current, url: event.target.value }))
                }
                placeholder={t('webhook.fields.urlPlaceholder')}
              />
            </label>

          <label className="space-y-2 md:col-span-2">
            <span className="text-sm text-muted-foreground">{t('webhook.fields.template')}</span>
            <Textarea
              className="min-h-40"
              value={config.fleet_template}
              onChange={(event) =>
                setConfig((current) => ({ ...current, fleet_template: event.target.value }))
              }
              placeholder={t('webhook.fields.templatePlaceholder')}
            />
            <p className="text-xs text-muted-foreground">{t('webhook.fields.templateHint')}</p>
          </label>
        </div>
      </div>

      <div className="rounded-lg border bg-card p-5">
        <div className="flex flex-col gap-3 border-b pb-4 lg:flex-row lg:items-start lg:justify-between">
          <div>
            <h2 className="text-base font-semibold">{t('webhook.test.title')}</h2>
            <p className="mt-1 text-sm text-muted-foreground">{t('webhook.test.subtitle')}</p>
          </div>
          <Button
            type="button"
            variant="outline"
            onClick={() => void sendTest()}
            isDisabled={testing || !canSubmitTest}
          >
            {testing ? t('webhook.messages.testing') : t('webhook.test.sendBtn')}
          </Button>
        </div>

        <div className="mt-4 grid gap-4 md:grid-cols-2">
          <label className="space-y-2">
            <span className="text-sm text-muted-foreground">{t('webhook.test.type')}</span>
            <Select
              selectedKey={String(testForm.type ?? '')}
              onSelectionChange={(key) => ((value) => setTestForm((current) => ({ ...current, type: value })))(String(key))}
            >
              <SelectTrigger className="h-10 w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem id="discord">{t('webhook.types.discord')}</SelectItem>
                <SelectItem id="feishu">{t('webhook.types.feishu')}</SelectItem>
                <SelectItem id="dingtalk">{t('webhook.types.dingtalk')}</SelectItem>
              </SelectContent>
            </Select>
          </label>

            <label className="space-y-2">
              <span className="text-sm text-muted-foreground">{t('webhook.test.url')}</span>
              <Input
                value={testForm.url ?? ''}
                onChange={(event) =>
                  setTestForm((current) => ({ ...current, url: event.target.value }))
                }
                placeholder={t('webhook.fields.urlPlaceholder')}
              />
            </label>

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
        </div>
      </div>
    </section>
  )
}
