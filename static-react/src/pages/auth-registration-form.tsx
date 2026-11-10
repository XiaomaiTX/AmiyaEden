import { useEffect, useState } from 'react'
import { completeRegistration, getRegistration } from '@/api/auth'
import { HttpError } from '@/api/http-client'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { useI18n } from '@/i18n'
import { buildEveCharacterPortraitUrl } from '@/lib/eve-image'
import { usePreferenceStore } from '@/stores'
import './auth-registration-form.css'

export function AuthRegistrationForm({
  ticket,
  onComplete,
  onLogin,
}: {
  ticket: string
  onComplete: (token: string) => void
  onLogin: () => void
}) {
  const { t, locale } = useI18n()
  const setLocale = usePreferenceStore((state) => state.setLocale)
  const [form, setForm] = useState({ nickname: '', qq: '', discord_id: '' })
  const [character, setCharacter] = useState('')
  const [characterId, setCharacterId] = useState(0)
  const [portraitFailed, setPortraitFailed] = useState(false)
  const [contactMethod, setContactMethod] = useState<'qq' | 'discord_id'>('qq')
  const [ready, setReady] = useState(false)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  useEffect(() => {
    let cancelled = false
    getRegistration(ticket)
      .then((info) => {
        if (cancelled) return
        setCharacter(info.character_name)
        setCharacterId(info.character_id)
        setReady(true)
      })
      .catch(() => {
        if (!cancelled) setError('registration.expired')
      })
    return () => {
      cancelled = true
    }
  }, [ticket])

  return (
    <form
      className="contact-registration"
      aria-labelledby="registration-title"
      aria-busy={!ready || saving}
      onSubmit={async (event) => {
        event.preventDefault()
        if (saving || !ready) return
        if (Array.from(form.nickname.trim()).length < 2) {
          setError('registration.nicknameLength')
          return
        }
        if (!form[contactMethod].trim()) {
          setError('registration.contactRequired')
          return
        }
        setSaving(true)
        setError('')
        try {
          const result = await completeRegistration(ticket, {
            nickname: form.nickname,
            qq: contactMethod === 'qq' ? form.qq : '',
            discord_id: contactMethod === 'discord_id' ? form.discord_id : '',
          })
          onComplete(result.token)
        } catch (err) {
          if (!(err instanceof HttpError && err.status === 409))
            setError(err instanceof Error ? err.message : 'registration.failed')
        } finally {
          setSaving(false)
        }
      }}
    >
      <div className="registration-language" role="group" aria-label={t('registration.language')}>
        <button
          type="button"
          lang="zh-CN"
          aria-pressed={locale === 'zh-CN'}
          onClick={() => setLocale('zh-CN')}
        >
          {t('registration.chinese')}
        </button>
        <button
          type="button"
          lang="en"
          aria-pressed={locale === 'en-US'}
          onClick={() => setLocale('en-US')}
        >
          {t('registration.english')}
        </button>
      </div>
      <header className="registration-heading">
        <h1 id="registration-title">{t('registration.title')}</h1>
        <p>{t('registration.intro')}</p>
      </header>
      <div className="registration-identity">
        <div className="registration-portrait" aria-hidden="true">
          <span>{character.slice(0, 1)}</span>
          {characterId && !portraitFailed ? (
            <img
              src={buildEveCharacterPortraitUrl(characterId)}
              alt=""
              onError={() => setPortraitFailed(true)}
            />
          ) : null}
        </div>
        <div className="registration-character">
          <span>{t('registration.character')}</span>
          <strong>{character || t('common.loading')}</strong>
        </div>
      </div>
      <div className="registration-field">
        <label htmlFor="registration-nickname">
          {t('registration.nickname')}
          <span className="registration-required" aria-hidden="true">
            *
          </span>
        </label>
        <Input
          id="registration-nickname"
          className="registration-input"
          required
          minLength={2}
          maxLength={20}
          placeholder={t('registration.nicknamePlaceholder')}
          aria-describedby="registration-nickname-hint"
          value={form.nickname}
          onChange={(event) => setForm({ ...form, nickname: event.target.value })}
        />
        <small id="registration-nickname-hint" className="registration-hint">
          {t('registration.nicknameHint')}
        </small>
      </div>
      <fieldset className="registration-contact-method">
        <legend>
          {t('registration.contactMethod')}
          <span className="registration-required" aria-hidden="true">
            *
          </span>
        </legend>
        <div className="registration-options">
          {(['qq', 'discord_id'] as const).map((method) => (
            <label
              key={method}
              className="registration-option"
              data-selected={contactMethod === method}
            >
              <input
                type="radio"
                name="contact-method"
                value={method}
                checked={contactMethod === method}
                onChange={() => {
                  setContactMethod(method)
                  setError('')
                }}
              />
              <span>{t(method === 'qq' ? 'common.qq' : 'common.discord')}</span>
            </label>
          ))}
        </div>
      </fieldset>
      <label className="registration-field">
        <span>
          {t(contactMethod === 'qq' ? 'characters.profile.qq' : 'characters.profile.discordId')}
          <span className="registration-required" aria-hidden="true">
            *
          </span>
        </span>
        <Input
          key={contactMethod}
          className="registration-input"
          inputMode={contactMethod === 'qq' ? 'numeric' : 'text'}
          pattern={contactMethod === 'qq' ? '[0-9]*' : undefined}
          aria-required="true"
          maxLength={20}
          placeholder={t(
            contactMethod === 'qq'
              ? 'registration.qqPlaceholder'
              : 'registration.discordPlaceholder'
          )}
          value={form[contactMethod]}
          onChange={(event) => setForm({ ...form, [contactMethod]: event.target.value })}
        />
      </label>
      {error ? (
        <p role="alert" className="registration-error">
          {error.startsWith('registration.') ? t(error) : error}
        </p>
      ) : null}
      <Button className="registration-submit" type="submit" isDisabled={!ready || saving}>
        {t(saving ? 'registration.saving' : 'registration.submit')}
      </Button>
      <Button className="registration-login" type="button" variant="ghost" onClick={onLogin}>
        {t('auth.backToLogin')}
      </Button>
    </form>
  )
}
