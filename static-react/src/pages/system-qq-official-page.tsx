import { useCallback, useState } from 'react'
import { getQQOfficialStatus, updateQQOfficialSettings } from '@/api/generated/client'
import type { QQOfficialGroup, QQOfficialSettings, QQOfficialStatus } from '@/api/generated/client'
import { unwrapContract } from '@/api/response'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { useQQOfficialActions, useQQOfficialRead } from '@/hooks/use-qq-official-console'
import { useI18n } from '@/i18n'
import { formatTime } from '@/lib/format'
import { hasUsableQQCredentials, qqStateLabel } from '@/lib/qq-official'
import { notifySuccess } from '@/feedback'
import {
  QQConfirm,
  QQGroupPanel,
  QQRequestsPanel,
  QQTaskDetail,
  QQTasksPanel,
} from './system-qq-official-panels'
import { QQOfficialSettingsPanel } from './system-qq-official-settings'

const readStatus = (signal: AbortSignal) => unwrapContract(getQQOfficialStatus({ signal }))
const settingsFlags = [
  'configured',
  'enabled',
  'paused',
  'certified',
  'approval_writes_enabled',
] as const

export function SystemQQOfficialPage() {
  const { t } = useI18n()
  const actions = useQQOfficialActions()
  const { data: status, error, loading, reload, replace } = useQQOfficialRead(readStatus, 15_000)
  const [credentialRevision, setCredentialRevision] = useState(0)
  const [confirmResume, setConfirmResume] = useState(false)
  const scopeKey = `${status?.app_id ?? ''}:${credentialRevision}`
  const saveSettings = async (patch: QQOfficialSettings) => {
    const credentialsChanged = patch.app_secret !== undefined
    const saved = await actions.run(() => unwrapContract(updateQQOfficialSettings(patch)))
    if (saved) {
      if (credentialsChanged) setCredentialRevision((revision) => revision + 1)
      replace(saved)
      setConfirmResume(false)
      notifySuccess(t('qqOfficial.saved'))
    } else reload()
  }

  return (
    <section className="space-y-4">
      <header>
        <h1 className="text-xl font-semibold">{t('qqOfficial.title')}</h1>
        <p className="text-sm text-muted-foreground">{t('qqOfficial.subtitle')}</p>
      </header>
      <section className="space-y-3 rounded-lg border bg-card p-4" aria-labelledby="qq-runtime">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <h2 id="qq-runtime" className="font-semibold">
            {t('qqOfficial.runtime')}
          </h2>
          <div className="flex flex-wrap gap-2">
            <Button variant="outline" onPress={reload}>
              {t('qqOfficial.localReload')}
            </Button>
            {actions.canManage &&
              status?.has_app_secret &&
              (status.paused ? (
                <Button
                  variant="outline"
                  isDisabled={actions.busy}
                  onPress={() => setConfirmResume(true)}
                >
                  {t('qqOfficial.resume')}
                </Button>
              ) : (
                <Button
                  variant="destructive"
                  isDisabled={actions.busy}
                  onPress={() => void saveSettings({ paused: true })}
                >
                  {t('qqOfficial.pause')}
                </Button>
              ))}
          </div>
        </div>
        {error && (
          <p role="alert" className="text-destructive">
            {t('qqOfficial.loadFailed')}
          </p>
        )}
        {loading && <p>{t('common.loading')}</p>}
        {status && (
          <>
            <dl className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
              {settingsFlags.map((flag) => (
                <div key={flag}>
                  <dt className="text-sm text-muted-foreground">
                    {t(
                      `qqOfficial.${flag === 'approval_writes_enabled' ? 'approvalWrites' : flag}`
                    )}
                  </dt>
                  <dd>{t(status[flag] ? 'qqOfficial.yes' : 'qqOfficial.no')}</dd>
                </div>
              ))}
              {[
                ['appID', status.app_id ?? '—'],
                ['appSecretStored', t(status.has_app_secret ? 'qqOfficial.yes' : 'qqOfficial.no')],
                ['token', qqStateLabel(t, status.token_state)],
                ['gateway', t(`qqOfficial.gatewayState.${status.gateway_state}`)],
                ['lastEvent', formatTime(status.last_event_at)],
                ['queueDepth', status.queue_depth],
                ['attemptsMinute', status.attempts_last_minute],
                ['messagesMinute', status.messages_last_minute],
                ['messageBudget', status.message_budget],
                ['nextRequest', formatTime(status.next_request_at)],
                ['oldestQueued', formatTime(status.oldest_queued_at)],
              ].map(([label, value]) => (
                <div key={label}>
                  <dt className="text-sm text-muted-foreground">{t(`qqOfficial.${label}`)}</dt>
                  <dd>{value}</dd>
                </div>
              ))}
            </dl>
            <div className="text-sm">
              <h3 className="text-muted-foreground">{t('qqOfficial.cooldowns')}</h3>
              {status.cooldowns.length ? (
                <ul>
                  {status.cooldowns.map((cooldown, index) => (
                    <li key={`${cooldown.scope}-${index}`}>
                      {qqStateLabel(
                        t,
                        cooldown.scope.startsWith('group:') ? 'group' : cooldown.scope
                      )}{' '}
                      · {formatTime(cooldown.until)}
                    </li>
                  ))}
                </ul>
              ) : (
                '—'
              )}
            </div>
            {actions.canManage && (
              <div className="flex flex-wrap items-center gap-4 border-t pt-3">
                <Checkbox
                  isSelected={status.enabled}
                  isDisabled={actions.busy || (!status.enabled && !hasUsableQQCredentials(status))}
                  onChange={(enabled) => void saveSettings({ enabled })}
                >
                  {t('qqOfficial.enabled')}
                </Checkbox>
                <Checkbox
                  isSelected={status.certified}
                  isDisabled={actions.busy || !status.has_app_secret}
                  onChange={(certified) => void saveSettings({ certified })}
                >
                  {t('qqOfficial.certified')}
                </Checkbox>
                <p className="text-sm text-muted-foreground">{t('qqOfficial.certifyHint')}</p>
                {!hasUsableQQCredentials(status) && (
                  <p className="w-full text-sm text-muted-foreground">
                    {t(
                      status.has_app_secret
                        ? 'qqOfficial.credentialsUnreadable'
                        : 'qqOfficial.enableNeedsCredentials'
                    )}
                  </p>
                )}
              </div>
            )}
          </>
        )}
      </section>
      {status && actions.canManage && (
        <QQOfficialSettingsPanel
          key={`settings:${scopeKey}`}
          status={status}
          busy={actions.busy}
          save={saveSettings}
        />
      )}
      {status && <QQOfficialScope key={scopeKey} status={status} actions={actions} />}
      <QQConfirm
        open={confirmResume}
        title={t('qqOfficial.resume')}
        message={t('qqOfficial.resumeConfirm')}
        busy={actions.busy}
        close={() => setConfirmResume(false)}
        confirm={() => void saveSettings({ paused: false })}
      />
    </section>
  )
}

function QQOfficialScope({
  status,
  actions,
}: {
  status: QQOfficialStatus
  actions: ReturnType<typeof useQQOfficialActions>
}) {
  const { t } = useI18n()
  const [tab, setTab] = useState('groups')
  const [taskID, setTaskID] = useState('')
  const [requestGroup, setRequestGroup] = useState<QQOfficialGroup>()
  const approvalScope = String(status.approval_writes_enabled)
  const openRequests = useCallback((group: QQOfficialGroup) => {
    setRequestGroup(group)
    setTab('requests')
  }, [])
  return (
    <>
      <Tabs selectedKey={tab} onSelectionChange={(key) => setTab(String(key))}>
        <TabsList aria-label={t('qqOfficial.title')}>
          {(['groups', 'tasks', 'requests'] as const).map((name) => (
            <TabsTrigger key={name} id={name}>
              {t(`qqOfficial.${name}`)}
            </TabsTrigger>
          ))}
        </TabsList>
        <TabsContent id="groups">
          <QQGroupPanel
            status={status}
            actions={actions}
            openTask={setTaskID}
            openRequests={openRequests}
          />
        </TabsContent>
        <TabsContent id="tasks">
          <QQTasksPanel key={approvalScope} openTask={setTaskID} />
        </TabsContent>
        <TabsContent id="requests">
          {requestGroup ? (
            <QQRequestsPanel
              key={`${requestGroup.id}:${approvalScope}`}
              group={requestGroup}
              status={status}
              actions={actions}
              openTask={setTaskID}
            />
          ) : (
            <p>{t('qqOfficial.requestGroupMissing')}</p>
          )}
        </TabsContent>
      </Tabs>
      {taskID && (
        <QQTaskDetail
          key={`${taskID}:${approvalScope}`}
          id={taskID}
          actions={actions}
          close={() => setTaskID('')}
        />
      )}
    </>
  )
}
