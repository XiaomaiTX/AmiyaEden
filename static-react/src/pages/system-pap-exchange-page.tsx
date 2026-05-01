import { useCallback, useEffect, useMemo, useState, type ReactNode } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { notifyError, notifySuccess } from '@/feedback'
import { useI18n } from '@/i18n'
import { fetchPAPExchangeConfig, updatePAPExchangeConfig } from '@/api/pap-exchange'
import type { ConfigResponse, RateItem } from '@/types/api/pap-exchange'
import { formatDateTime } from './shop-page-utils'

const defaultConfig: ConfigResponse = {
  rates: [],
  fc_salary: 400,
  fc_salary_monthly_limit: 5,
  admin_award: 10,
  multichar_full_reward_count: 3,
  multichar_reduced_reward_count: 3,
  multichar_reduced_reward_pct: 50,
}

export function SystemPAPExchangePage() {
  const { t } = useI18n()
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [config, setConfig] = useState<ConfigResponse>(defaultConfig)

  const loadConfig = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      setConfig(await fetchPAPExchangeConfig())
    } catch {
      setError(t('papExchange.messages.loadFailed'))
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

  const rates = useMemo(() => config.rates ?? [], [config.rates])

  const saveConfig = async () => {
    setSaving(true)
    try {
      const nextConfig = await updatePAPExchangeConfig(config)
      setConfig(nextConfig)
      notifySuccess(t('papExchange.messages.saveSuccess'))
    } catch {
      notifyError(t('papExchange.messages.saveFailed'))
    } finally {
      setSaving(false)
    }
  }

  return (
    <section className="space-y-4">
      <div className="rounded-lg border bg-card p-5">
        <div className="flex flex-col gap-3 lg:flex-row lg:items-end lg:justify-between">
          <div>
            <h1 className="text-xl font-semibold">{t('papExchange.title')}</h1>
            <p className="mt-1 text-sm text-muted-foreground">{t('papExchange.subtitle')}</p>
          </div>
          <Button type="button" onClick={() => void saveConfig()} disabled={saving || loading}>
            {saving ? t('papExchange.messages.saving') : t('common.save')}
          </Button>
        </div>
      </div>

      {error ? <p className="text-sm text-destructive">{error}</p> : null}
      {loading ? <p className="text-sm text-muted-foreground">{t('papExchange.messages.loading')}</p> : null}

      <div className="grid gap-4 xl:grid-cols-3">
        <ConfigCard
          title={t('papExchange.sections.fc')}
          subtitle={t('papExchange.sections.fcSubtitle')}
        >
          <label className="space-y-2">
            <span className="text-sm text-muted-foreground">{t('papExchange.fields.fcSalary')}</span>
            <Input
              type="number"
              min={0}
              step="0.01"
              value={String(config.fc_salary)}
              onChange={(event) =>
                setConfig((current) => ({ ...current, fc_salary: Number(event.target.value) }))
              }
            />
          </label>
          <label className="space-y-2">
            <span className="text-sm text-muted-foreground">
              {t('papExchange.fields.fcSalaryMonthlyLimit')}
            </span>
            <Input
              type="number"
              min={0}
              step={1}
              value={String(config.fc_salary_monthly_limit)}
              onChange={(event) =>
                setConfig((current) => ({
                  ...current,
                  fc_salary_monthly_limit: Number(event.target.value),
                }))
              }
            />
          </label>
          <label className="space-y-2">
            <span className="text-sm text-muted-foreground">{t('papExchange.fields.adminAward')}</span>
            <Input
              type="number"
              min={0}
              step={1}
              value={String(config.admin_award)}
              onChange={(event) =>
                setConfig((current) => ({ ...current, admin_award: Number(event.target.value) }))
              }
            />
          </label>
        </ConfigCard>

        <ConfigCard
          title={t('papExchange.sections.multichar')}
          subtitle={t('papExchange.sections.multicharSubtitle')}
        >
          <label className="space-y-2">
            <span className="text-sm text-muted-foreground">
              {t('papExchange.fields.multicharFullRewardCount')}
            </span>
            <Input
              type="number"
              min={0}
              step={1}
              value={String(config.multichar_full_reward_count)}
              onChange={(event) =>
                setConfig((current) => ({
                  ...current,
                  multichar_full_reward_count: Number(event.target.value),
                }))
              }
            />
          </label>
          <label className="space-y-2">
            <span className="text-sm text-muted-foreground">
              {t('papExchange.fields.multicharReducedRewardCount')}
            </span>
            <Input
              type="number"
              min={0}
              step={1}
              value={String(config.multichar_reduced_reward_count)}
              onChange={(event) =>
                setConfig((current) => ({
                  ...current,
                  multichar_reduced_reward_count: Number(event.target.value),
                }))
              }
            />
          </label>
          <label className="space-y-2">
            <span className="text-sm text-muted-foreground">
              {t('papExchange.fields.multicharReducedRewardPct')}
            </span>
            <Input
              type="number"
              min={0}
              max={100}
              step={1}
              value={String(config.multichar_reduced_reward_pct)}
              onChange={(event) =>
                setConfig((current) => ({
                  ...current,
                  multichar_reduced_reward_pct: Number(event.target.value),
                }))
              }
            />
          </label>
        </ConfigCard>

        <ConfigCard
          title={t('papExchange.sections.tips')}
          subtitle={t('papExchange.sections.tipsSubtitle')}
        >
          <p className="text-sm text-muted-foreground">{t('papExchange.hints.rateEdit')}</p>
          <p className="text-sm text-muted-foreground">{t('papExchange.hints.fcSalary')}</p>
          <p className="text-sm text-muted-foreground">{t('papExchange.hints.multichar')}</p>
        </ConfigCard>
      </div>

      <div className="rounded-lg border bg-card">
        <div className="flex flex-col gap-3 border-b px-4 py-3 lg:flex-row lg:items-center lg:justify-between">
          <div>
            <h2 className="text-base font-semibold">{t('papExchange.sections.rates')}</h2>
            <p className="mt-1 text-sm text-muted-foreground">{t('papExchange.sections.ratesSubtitle')}</p>
          </div>
          <Button type="button" onClick={() => void saveConfig()} disabled={saving || loading}>
            {saving ? t('papExchange.messages.saving') : t('common.save')}
          </Button>
        </div>
        <div className="overflow-x-auto">
          <table className="min-w-full text-sm">
            <thead>
              <tr className="border-b bg-muted/40 text-left">
                <th className="px-3 py-2">{t('papExchange.columns.papType')}</th>
                <th className="px-3 py-2">{t('papExchange.columns.displayName')}</th>
                <th className="px-3 py-2">{t('papExchange.columns.rate')}</th>
                <th className="px-3 py-2">{t('papExchange.columns.updatedAt')}</th>
              </tr>
            </thead>
            <tbody>
              {rates.map((rate, index) => (
                <tr key={rate.pap_type} className="border-b align-top">
                  <td className="px-3 py-2 font-medium">{rate.pap_type}</td>
                  <td className="px-3 py-2">
                    <Input
                      value={rate.display_name}
                      onChange={(event) =>
                        setConfig((current) => ({
                          ...current,
                          rates: updateRate(current.rates, index, {
                            display_name: event.target.value,
                          }),
                        }))
                      }
                    />
                  </td>
                  <td className="px-3 py-2">
                    <Input
                      type="number"
                      min={0.01}
                      step="0.01"
                      value={String(rate.rate)}
                      onChange={(event) =>
                        setConfig((current) => ({
                          ...current,
                          rates: updateRate(current.rates, index, {
                            rate: Number(event.target.value),
                          }),
                        }))
                      }
                    />
                  </td>
                  <td className="px-3 py-2 text-muted-foreground">{formatDateTime(rate.updated_at)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </section>
  )
}

function updateRate(rates: RateItem[], index: number, patch: Partial<RateItem>) {
  return rates.map((rate, currentIndex) =>
    currentIndex === index ? { ...rate, ...patch } : rate
  )
}

function ConfigCard({
  title,
  subtitle,
  children,
}: {
  title: string
  subtitle: string
  children: ReactNode
}) {
  return (
    <div className="rounded-lg border bg-card p-5">
      <h2 className="text-base font-semibold">{title}</h2>
      <p className="mt-1 text-sm text-muted-foreground">{subtitle}</p>
      <div className="mt-4 space-y-4">{children}</div>
    </div>
  )
}
