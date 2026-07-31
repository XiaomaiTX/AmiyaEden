import { useState } from 'react'
import { useParams } from 'react-router-dom'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { submitRecruitQQ } from '@/api/newbro'
import { useI18n } from '@/i18n'

const QQ_REGEX = /^\d{5,20}$/

export function RecruitLandingPage() {
  const { t } = useI18n()
  const { code } = useParams<{ code: string }>()
  const [qq, setQq] = useState('')
  const [qqError, setQqError] = useState('')
  const [loading, setLoading] = useState(false)
  const [submitted, setSubmitted] = useState(false)
  const [submitError, setSubmitError] = useState('')
  const [qqUrl, setQqUrl] = useState('')

  const validateQq = (value: string) => {
    if (!value.trim()) {
      return t('recruit.qqRequired')
    }
    if (!QQ_REGEX.test(value.trim())) {
      return t('recruit.qqInvalid')
    }
    return ''
  }

  const handleSubmit = async () => {
    const error = validateQq(qq)
    if (error) {
      setQqError(error)
      return
    }

    setQqError('')
    setLoading(true)
    setSubmitError('')

    try {
      const data = await submitRecruitQQ(code ?? '', { qq: qq.trim() })
      setQqUrl(data.qq_url)
      setSubmitted(true)
    } catch (err) {
      setSubmitError(err instanceof Error ? err.message : t('recruit.submitError'))
    } finally {
      setLoading(false)
    }
  }

  const handleRedirect = () => {
    if (!qqUrl) return
    window.open(qqUrl, '_blank', 'noopener,noreferrer')
  }

  return (
    <main className="flex min-h-screen items-center justify-center p-6">
      <div className="w-full max-w-[520px] rounded-lg border bg-card p-6 shadow-lg">
        <div className="mb-5 flex flex-col gap-2">
          <p className="text-xs font-bold uppercase tracking-widest text-primary">
            {t('newbro.recruitLink.title')}
          </p>
          <h1 className="text-3xl leading-tight">{t('recruit.title')}</h1>
          <p className="text-sm text-muted-foreground">{t('recruit.subtitle')}</p>
        </div>

        {!submitted ? (
          <>
            <div className="space-y-4">
              <div>
                <label className="text-sm font-medium">{t('recruit.qqLabel')}</label>
                <Input
                  value={qq}
                  onChange={(e) => {
                    setQq(e.target.value)
                    setQqError('')
                  }}
                  placeholder={t('recruit.qqPlaceholder')}
                  maxLength={20}
                  disabled={loading}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter') handleSubmit()
                  }}
                />
                {qqError ? <p className="mt-1 text-xs text-destructive">{qqError}</p> : null}
              </div>

              <Button className="w-full" onClick={handleSubmit} isDisabled={loading}>
                {loading ? '...' : t('recruit.submitBtn')}
              </Button>
            </div>

            {submitError ? (
              <div className="mt-4 rounded-md bg-destructive/10 p-3 text-sm text-destructive">
                {submitError}
              </div>
            ) : null}
          </>
        ) : (
          <div className="flex flex-col items-center gap-4 py-6 text-center">
            <h2 className="text-xl font-semibold">{t('recruit.successTitle')}</h2>
            <p className="text-sm text-muted-foreground">{t('recruit.successSubtitle')}</p>
            <Button onClick={handleRedirect}>{t('recruit.goToQQ')}</Button>
          </div>
        )}
      </div>
    </main>
  )
}
