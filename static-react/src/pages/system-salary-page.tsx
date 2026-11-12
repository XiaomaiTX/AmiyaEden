import { useEffect, useState } from 'react'
import { fetchSalarySettings, updateSalarySettings } from '@/api/salary'
import { Card, CardHeader, CardTitle, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { useI18n } from '@/i18n'
import { notifySuccess } from '@/feedback'
import type { SalarySettings } from '@/types/api/salary'

const operationKeys = ['recruitment', 'welfare_delivery', 'shop_delivery', 'srp_payout'] as const
export function SystemSalaryPage() {
  const { t } = useI18n()
  const [form, setForm] = useState<SalarySettings | null>(null)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const [revision, setRevision] = useState(0)
  useEffect(() => {
    let active = true
    void fetchSalarySettings()
      .then((value) => {
        if (active) setForm(value)
      })
      .catch(() => {
        if (active) setError('loadFailed')
      })
      .finally(() => {
        if (active) setLoading(false)
      })
    return () => {
      active = false
    }
  }, [revision])
  async function save(event: React.FormEvent) {
    event.preventDefault()
    if (!form || loading || saving) return
    setSaving(true)
    setError('')
    try {
      setForm(await updateSalarySettings(form))
      notifySuccess(t('salaryManagement.saved'))
    } catch {
      setError('saveFailed')
    } finally {
      setSaving(false)
    }
  }
  const field = (key: keyof SalarySettings) => (
    <label className="space-y-2" key={key}>
      <span className="block text-sm font-medium">{t(`salaryManagement.${key}`)}</span>
      <Input
        aria-label={t(`salaryManagement.${key}`)}
        type="number"
        required
        min="0"
        max={key === 'fc_salary_monthly_limit' ? undefined : '1000000000000'}
        step={key === 'fc_salary_monthly_limit' ? '1' : '0.01'}
        value={form && Number.isFinite(form[key]) ? form[key] : ''}
        onChange={(event) =>
          setForm(
            (current) =>
              current && {
                ...current,
                [key]: event.target.value === '' ? Number.NaN : Number(event.target.value),
              }
          )
        }
      />
      {key !== 'fc_salary_monthly_limit' && (
        <span className="text-xs text-muted-foreground">
          {t(key === 'srp_payout' ? 'salaryManagement.perOperationUnit' : 'salaryManagement.unit')}
        </span>
      )}
    </label>
  )
  return (
    <section className="min-w-0 space-y-4">
      <header className="flex items-center justify-between gap-4">
        <h1 className="text-xl font-semibold">{t('salaryManagement.title')}</h1>
        <Button
          variant="outline"
          isDisabled={loading || saving}
          onClick={() => {
            setLoading(true)
            setForm(null)
            setError('')
            setRevision((v) => v + 1)
          }}
        >
          {t('salaryManagement.refresh')}
        </Button>
      </header>
      <p className="text-sm text-muted-foreground">{t('salaryManagement.description')}</p>
      {loading && <p role="status">{t('salaryManagement.loading')}</p>}
      {error && (
        <p role="alert" className="text-destructive">
          {t(`salaryManagement.${error}`)}
        </p>
      )}
      {form && (
        <form onSubmit={save} className="space-y-4">
          <fieldset disabled={loading || saving} className="min-w-0 space-y-4">
            <Card>
              <CardHeader>
                <CardTitle>{t('salaryManagement.operations')}</CardTitle>
              </CardHeader>
              <CardContent className="grid gap-6 sm:grid-cols-2">
                {operationKeys.map(field)}
              </CardContent>
            </Card>
            <Card>
              <CardHeader>
                <CardTitle>{t('salaryManagement.fcSection')}</CardTitle>
              </CardHeader>
              <CardContent className="space-y-4">
                <p className="text-sm text-muted-foreground">{t('salaryManagement.fcHint')}</p>
                <div className="grid gap-6 sm:grid-cols-2">
                  {field('fc_salary')}
                  {field('fc_salary_monthly_limit')}
                </div>
              </CardContent>
            </Card>
            <Button type="submit" isDisabled={loading || saving}>
              {t(saving ? 'salaryManagement.saving' : 'salaryManagement.save')}
            </Button>
          </fieldset>
        </form>
      )}
    </section>
  )
}
