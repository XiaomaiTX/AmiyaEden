import { useId, useState } from 'react'
import type { QQOfficialSettings, QQOfficialStatus } from '@/api/generated/client'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import { useI18n } from '@/i18n'
import { hasUsableQQCredentials, qqCredentialDraft } from '@/lib/qq-official'
import { QQConfirm } from './system-qq-official-panels'

export function QQOfficialSettingsPanel({
  status,
  busy,
  save,
}: {
  status: QQOfficialStatus
  busy: boolean
  save: (patch: QQOfficialSettings) => Promise<void>
}) {
  const { t } = useI18n()
  const hintID = useId()
  const approvalHintID = useId()
  const [appID, setAppID] = useState(status.app_id ?? '')
  const [secret, setSecret] = useState('')
  const [confirmation, setConfirmation] = useState<'credentials' | 'approval'>()
  const draft = qqCredentialDraft(status.app_id, appID, secret)
  const usable = hasUsableQQCredentials(status)
  const closeConfirmation = () => {
    if (confirmation === 'credentials') setSecret('')
    setConfirmation(undefined)
  }
  const confirm = () => {
    const patch =
      confirmation === 'credentials'
        ? draft.patch
        : confirmation === 'approval' && usable
          ? { approval_writes_enabled: true }
          : undefined
    closeConfirmation()
    if (patch) void save(patch)
  }

  return (
    <div className="space-y-3 rounded-lg border bg-card p-4">
      <h2 className="font-semibold">{t('qqOfficial.credentials')}</h2>
      <form
        className="space-y-3"
        autoComplete="off"
        onSubmit={(event) => {
          event.preventDefault()
          if (draft.patch && !busy) setConfirmation('credentials')
        }}
      >
        <div className="grid gap-3 sm:grid-cols-2">
          <label className="grid gap-1">
            {t('qqOfficial.appID')}
            <Input
              value={appID}
              inputMode="numeric"
              disabled={busy}
              onChange={(e) => setAppID(e.target.value)}
            />
          </label>
          <label className="grid gap-1">
            {t('qqOfficial.appSecret')}
            <Input
              type="password"
              autoComplete="new-password"
              aria-describedby={hintID}
              value={secret}
              disabled={busy}
              onChange={(e) => setSecret(e.target.value)}
            />
          </label>
        </div>
        <p
          id={hintID}
          aria-live="polite"
          className={`text-sm ${draft.error ? 'text-destructive' : 'text-muted-foreground'}`}
        >
          {t(draft.error ?? 'qqOfficial.appSecretHint')}
        </p>
        <p className="text-sm text-muted-foreground">{t('qqOfficial.credentialsHint')}</p>
        <Button type="submit" isDisabled={busy || !draft.patch}>
          {t('qqOfficial.saveCredentials')}
        </Button>
      </form>
      <div className="space-y-1 border-t pt-3">
        <div className="flex items-center gap-3">
          <Switch
            aria-label={t('qqOfficial.approvalWritesControl')}
            aria-describedby={approvalHintID}
            isSelected={status.approval_writes_enabled}
            isDisabled={busy || (!status.approval_writes_enabled && !usable)}
            onChange={(enabled) => {
              if (!enabled) void save({ approval_writes_enabled: false })
              else if (usable) setConfirmation('approval')
            }}
          />
          <span className="text-sm">{t('qqOfficial.approvalWritesControl')}</span>
        </div>
        <p id={approvalHintID} className="text-sm text-muted-foreground">
          {t('qqOfficial.approvalWritesHint')}
        </p>
      </div>
      <QQConfirm
        open={!!confirmation}
        title={t(
          confirmation === 'credentials'
            ? 'qqOfficial.saveCredentials'
            : 'qqOfficial.enableApprovalWrites'
        )}
        message={t(
          confirmation === 'credentials'
            ? 'qqOfficial.credentialsConfirm'
            : 'qqOfficial.approvalWritesConfirm'
        )}
        busy={busy}
        close={closeConfirmation}
        confirm={confirm}
      />
    </div>
  )
}
