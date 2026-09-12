import { useEffect, useState } from 'react'
import {
  createMumbleCredential,
  fetchMumbleCredential,
  revokeMumbleCredential,
  rotateMumbleCredential,
} from '@/api/mumble'
import { Button } from '@/components/ui/button'
import { useI18n } from '@/i18n'
import type { MumbleCredentialStatus } from '@/types/api/mumble'

interface MumbleCredentialCardProps {
  canonicalName: string
}

function getErrorMessage(error: unknown, fallback: string) {
  return error instanceof Error && error.message ? error.message : fallback
}

export function MumbleCredentialCard({ canonicalName }: MumbleCredentialCardProps) {
  const { t } = useI18n()
  const [status, setStatus] = useState<MumbleCredentialStatus | null>(null)
  const [password, setPassword] = useState('')
  const [busy, setBusy] = useState(false)
  const [notice, setNotice] = useState<{ kind: 'error' | 'success'; text: string } | null>(null)

  useEffect(() => {
    if (!canonicalName) return
    let active = true
    void (async () => {
      try {
        const nextStatus = await fetchMumbleCredential()
        if (active) setStatus(nextStatus)
      } catch (error) {
        if (active) {
          setNotice({
            kind: 'error',
            text: getErrorMessage(error, t('characters.mumble.loadFailed')),
          })
        }
      }
    })()
    return () => {
      active = false
    }
  }, [canonicalName, t])

  const issueCredential = async (rotate: boolean) => {
    if (rotate && !window.confirm(t('characters.mumble.rotateConfirm'))) return
    setBusy(true)
    setNotice(null)
    try {
      const result = rotate ? await rotateMumbleCredential() : await createMumbleCredential()
      setStatus(result.credential)
      setPassword(result.password)
      setNotice({ kind: 'success', text: t('characters.mumble.issued') })
    } catch (error) {
      setNotice({
        kind: 'error',
        text: getErrorMessage(error, t('characters.mumble.operationFailed')),
      })
    } finally {
      setBusy(false)
    }
  }

  const revoke = async () => {
    if (!window.confirm(t('characters.mumble.revokeConfirm'))) return
    setBusy(true)
    setNotice(null)
    try {
      await revokeMumbleCredential()
      setStatus((current) => current && { ...current, enabled: false })
      setPassword('')
      setNotice({ kind: 'success', text: t('characters.mumble.revoked') })
    } catch (error) {
      setNotice({
        kind: 'error',
        text: getErrorMessage(error, t('characters.mumble.operationFailed')),
      })
    } finally {
      setBusy(false)
    }
  }

  const copyValue = async (value: string) => {
    try {
      await navigator.clipboard.writeText(value)
      setNotice({ kind: 'success', text: t('common.copied') })
    } catch {
      setNotice({ kind: 'error', text: t('common.copyFailed') })
    }
  }

  const serverAddress = status?.server_address ?? ''
  const serverPort = status?.server_port ?? 0

  return (
    <div className="rounded-lg border bg-card p-5">
      <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
        <div className="max-w-2xl">
          <h2 className="text-lg font-semibold">{t('characters.mumble.title')}</h2>
          <p className="mt-1 text-sm text-muted-foreground">{t('characters.mumble.subtitle')}</p>
        </div>
        {status ? (
          <span
            className={`rounded-full px-3 py-1 text-xs font-medium ${status.enabled ? 'bg-emerald-500/10 text-emerald-600' : 'bg-muted text-muted-foreground'}`}
          >
            {status.enabled ? t('characters.mumble.enabled') : t('characters.mumble.disabled')}
          </span>
        ) : null}
      </div>

      {notice ? (
        <div
          className={`mt-4 rounded-lg border px-4 py-3 text-sm ${notice.kind === 'success' ? 'border-emerald-500/20 bg-emerald-500/5 text-emerald-700' : 'border-destructive/20 bg-destructive/5 text-destructive'}`}
        >
          {notice.text}
        </div>
      ) : null}

      {password ? (
        <div className="mt-4 rounded-lg border border-amber-500/30 bg-amber-500/5 p-4">
          <p className="text-sm font-medium text-amber-800">
            {t('characters.mumble.oneTimeWarning')}
          </p>
          <div className="mt-3 grid gap-2 text-sm sm:grid-cols-[auto_1fr_auto] sm:items-center">
            <span className="text-muted-foreground">{t('characters.mumble.password')}</span>
            <code className="break-all rounded bg-background px-3 py-2">{password}</code>
            <Button type="button" variant="outline" onClick={() => void copyValue(password)}>
              {t('common.copy')}
            </Button>
          </div>
          <Button type="button" variant="ghost" className="mt-2" onClick={() => setPassword('')}>
            {t('characters.mumble.dismissSecret')}
          </Button>
        </div>
      ) : null}

      <div className="mt-5 grid gap-3 text-sm sm:grid-cols-3">
        <div>
          <span className="text-muted-foreground">{t('characters.mumble.username')}</span>
          <div className="mt-1 flex items-center gap-1">
            <code className="min-w-0 flex-1 truncate rounded bg-muted px-2 py-1 font-medium">
              {canonicalName || '—'}
            </code>
            {canonicalName ? (
              <Button type="button" variant="ghost" onClick={() => void copyValue(canonicalName)}>
                {t('common.copy')}
              </Button>
            ) : null}
          </div>
        </div>
        <div>
          <span className="text-muted-foreground">{t('characters.mumble.serverAddress')}</span>
          <div className="mt-1 flex items-center gap-1">
            <code className="min-w-0 flex-1 truncate rounded bg-muted px-2 py-1 font-medium">
              {serverAddress || '—'}
            </code>
            {serverAddress ? (
              <Button type="button" variant="ghost" onClick={() => void copyValue(serverAddress)}>
                {t('common.copy')}
              </Button>
            ) : null}
          </div>
        </div>
        <div>
          <span className="text-muted-foreground">{t('characters.mumble.serverPort')}</span>
          <div className="mt-1 flex items-center gap-1">
            <code className="min-w-0 flex-1 truncate rounded bg-muted px-2 py-1 font-medium">
              {serverPort || '—'}
            </code>
            {serverPort ? (
              <Button type="button" variant="ghost" onClick={() => void copyValue(String(serverPort))}>
                {t('common.copy')}
              </Button>
            ) : null}
          </div>
        </div>
      </div>

      <div className="mt-5 flex flex-wrap justify-end gap-2">
        {!status?.enabled ? (
          <Button
            type="button"
            onClick={() => void issueCredential(false)}
            isDisabled={busy || !canonicalName}
          >
            {t('characters.mumble.create')}
          </Button>
        ) : null}
        {status?.enabled ? (
          <Button
            type="button"
            variant="outline"
            onClick={() => void issueCredential(true)}
            isDisabled={busy}
          >
            {t('characters.mumble.rotate')}
          </Button>
        ) : null}
        {status?.enabled ? (
          <Button
            type="button"
            variant="destructive"
            onClick={() => void revoke()}
            isDisabled={busy}
          >
            {t('characters.mumble.revoke')}
          </Button>
        ) : null}
      </div>
    </div>
  )
}
