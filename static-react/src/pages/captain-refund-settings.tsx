import { useEffect, useState } from 'react'
import { getRefundSettings, saveRefundSettings } from '@/api/captain-refund'
import type { RefundSettings } from '@/types/api/captain-refund'
import { useI18n } from '@/i18n'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { Button } from '@/components/ui/button'

const settingKeys = {
  max_total_sp: 'refund.max_total_sp',
  max_age_days: 'refund.max_age_days',
  tax_rate: 'refund.tax_rate',
} as const

export function CaptainRefundSettingsPanel() {
  const { t } = useI18n()
  const [cfg, setCfg] = useState<RefundSettings | null>(null)
  const [systems, setSystems] = useState('')
  const [loading, setLoading] = useState(true)
  const [busy, setBusy] = useState(false)
  const [message, setMessage] = useState('')
  const [messageType, setMessageType] = useState<'success' | 'error'>('error')

  useEffect(() => {
    let active = true
    void getRefundSettings()
      .then((settings) => {
        if (active) {
          setCfg(settings)
          setSystems(settings.allowed_system_ids.join('\n'))
        }
      })
      .catch(() => {
        if (active) {
          setMessageType('error')
          setMessage(t('refund.failed'))
        }
      })
      .finally(() => {
        if (active) setLoading(false)
      })
    return () => {
      active = false
    }
  }, [t])

  const save = async () => {
    if (!cfg || loading || busy) return
    setMessage('')
    const entries = systems.split(/[\s,;]+/).filter(Boolean)
    const ids = entries.map(Number)
    if (
      !Number.isInteger(cfg.max_total_sp) ||
      cfg.max_total_sp < 0 ||
      !Number.isInteger(cfg.max_age_days) ||
      cfg.max_age_days < 0 ||
      cfg.max_age_days > 36500 ||
      !Number.isFinite(cfg.tax_rate) ||
      cfg.tax_rate < 0 ||
      cfg.tax_rate > 100 ||
      entries.some((entry) => !/^\d+$/.test(entry)) ||
      ids.some((id) => !Number.isSafeInteger(id) || id <= 0)
    ) {
      setMessageType('error')
      setMessage(t('refund.settingsInvalid'))
      return
    }

    setBusy(true)
    try {
      const updated = await saveRefundSettings({
        ...cfg,
        allowed_system_ids: [...new Set(ids)],
      })
      setCfg(updated)
      setSystems(updated.allowed_system_ids.join('\n'))
      setMessageType('success')
      setMessage(t('refund.saved'))
    } catch {
      setMessageType('error')
      setMessage(t('refund.failed'))
    } finally {
      setBusy(false)
    }
  }

  return (
    <section className="space-y-4 rounded-lg border bg-card p-5">
      <h2 className="text-lg font-semibold">{t('refund.settingsTitle')}</h2>
      <p className="text-sm text-muted-foreground">{t('refund.settingsHint')}</p>
      {message ? (
        <p
          role={messageType === 'error' ? 'alert' : 'status'}
          className={messageType === 'error' ? 'text-destructive' : 'text-success-foreground'}
        >
          {message}
        </p>
      ) : null}
      {loading ? <p role="status">{t('common.loading')}</p> : null}
      {cfg ? (
        <>
          {(['max_total_sp', 'max_age_days', 'tax_rate'] as const).map((key) => (
            <label className="block space-y-1" key={key}>
              <span>{t(settingKeys[key])}</span>
              <Input
                type="number"
                min="0"
                max={key === 'tax_rate' ? 100 : key === 'max_age_days' ? 36500 : undefined}
                step={key === 'tax_rate' ? '0.01' : '1'}
                value={cfg[key]}
                onChange={(event) => setCfg({ ...cfg, [key]: Number(event.target.value) })}
              />
            </label>
          ))}
          <label className="block space-y-1">
            <span>{t('refund.allowedSystems')}</span>
            <Textarea value={systems} onChange={(event) => setSystems(event.target.value)} />
          </label>
          <Button isDisabled={loading || busy} onClick={() => void save()}>
            {t('common.save')}
          </Button>
        </>
      ) : null}
    </section>
  )
}
