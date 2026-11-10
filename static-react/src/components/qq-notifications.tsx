import { useEffect, useState } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { DataTable } from '@/components/ui/data-table'
import { useI18n } from '@/i18n'
import {
  fetchQQNotificationChannels,
  fetchQQNotificationTargets,
  saveQQNotificationChannel,
  testQQNotification,
} from '@/api/qqbot'
import type { QQNotificationTarget } from '@/types/api/qqbot'

export function QQNotifications({
  channel = 'general',
}: {
  channel?: 'general' | 'fleet' | 'structure'
}) {
  const { t } = useI18n()
  const [enabled, setEnabled] = useState(false)
  const [groups, setGroups] = useState('')
  const [targets, setTargets] = useState<QQNotificationTarget[]>([])
  const [busy, setBusy] = useState(false)
  const [message, setMessage] = useState('')
  // Keep the identity after network errors, so retrying cannot send a second test.
  const [test, setTest] = useState(() => ({
    key: crypto.randomUUID(),
    expires: new Date(Date.now() + 300000).toISOString(),
  }))
  async function refresh() {
    setTargets(await fetchQQNotificationTargets())
  }
  useEffect(() => {
    let active = true
    void Promise.all([fetchQQNotificationChannels(), fetchQQNotificationTargets()])
      .then(([channels, rows]) => {
        if (!active) return
        const general = channels.find((c) => c.name === channel)
        setEnabled(general?.enabled ?? false)
        setGroups(
          general ? ((JSON.parse(general.group_ids) as number[] | null)?.join(',') ?? '') : ''
        )
        setTargets(rows)
      })
      .catch(() => {
        if (active) setMessage(t('qqbot.error'))
      })
    return () => {
      active = false
    }
  }, [t, channel])
  async function run(fn: () => Promise<unknown>) {
    setBusy(true)
    try {
      await fn()
      await refresh()
      setMessage(t('qqbot.success'))
    } catch {
      setMessage(t('qqbot.error'))
    } finally {
      setBusy(false)
    }
  }
  return (
    <section className="space-y-3">
      <h2>
        {t(
          channel === 'general'
            ? 'qqbot.notifications'
            : channel === 'fleet'
              ? 'qqbot.fleetNotifications'
              : 'qqbot.structureNotifications'
        )}
      </h2>
      <label>
        <input type="checkbox" checked={enabled} onChange={(e) => setEnabled(e.target.checked)} />{' '}
        {t('qqbot.notificationEnabled')}
      </label>
      <label>
        {t('qqbot.notificationGroups')}
        <Input value={groups} onChange={(e) => setGroups(e.target.value)} />
      </label>
      <Button
        isDisabled={busy}
        onClick={() =>
          void run(() =>
            saveQQNotificationChannel(
              channel,
              enabled,
              groups.trim() ? groups.split(',').map(Number) : []
            )
          )
        }
      >
        {t('qqbot.save')}
      </Button>
      {channel === 'general' && (
        <>
          <Button
            isDisabled={busy}
            onClick={() => void run(() => testQQNotification(test.key, test.expires))}
          >
            {t('qqbot.notificationTest')}
          </Button>
          <Button
            isDisabled={busy}
            onClick={() =>
              setTest({
                key: crypto.randomUUID(),
                expires: new Date(Date.now() + 300000).toISOString(),
              })
            }
          >
            {t('qqbot.notificationNewTest')}
          </Button>
        </>
      )}
      <Button isDisabled={busy} onClick={() => void run(refresh)}>
        {t('qqbot.refresh')}
      </Button>
      <p role="status">{message}</p>
      <DataTable
        variant="ledger"
        data={targets.filter((row) => row.channel === channel)}
        loadingText={t('common.loading')}
        emptyText={t('common.empty')}
        columns={[
          { accessorKey: 'notification_id', header: t('qqbot.notificationID') },
          { accessorKey: 'group_id', header: t('qqbot.notificationGroup') },
          { accessorKey: 'operation_id', header: t('qqbot.id') },
          { accessorKey: 'status', header: t('qqbot.status') },
          { accessorKey: 'result', header: t('qqbot.result') },
        ]}
      />
    </section>
  )
}
