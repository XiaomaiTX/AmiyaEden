import { useCallback, useEffect, useState } from 'react'
import { Button } from '@/components/ui/button'
import { DataTable } from '@/components/ui/data-table'
import { Input } from '@/components/ui/input'
import { useI18n } from '@/i18n'
import {
  fetchQQBotConfig,
  fetchQQBotSnapshot,
  fetchQQBotOperation,
  saveQQBotConfig,
  cancelQQBotOperation,
  retryQQBotOperation,
  retryQQBotDelivery,
  pauseQQBotCircuit,
} from '@/api/qqbot'
import type { QQBotConfig, QQBotSnapshot, QQBotOperation } from '@/types/api/qqbot'

export function SystemQQBotPage() {
  const { t } = useI18n()
  const [config, setConfig] = useState<QQBotConfig>({
    enabled: false,
    bot_id: 0,
    cidrs: '127.0.0.1/32',
    interval_ms: 1000,
  })
  const [snapshot, setSnapshot] = useState<QQBotSnapshot | null>(null)
  const [query, setQuery] = useState('')
  const [selected, setSelected] = useState<QQBotOperation | null>(null)
  const [token, setToken] = useState('')
  const [message, setMessage] = useState('')
  const [busy, setBusy] = useState(false)
  const load = useCallback(async () => {
    const [c, s] = await Promise.all([fetchQQBotConfig(), fetchQQBotSnapshot()])
    setConfig(c)
    setSnapshot(s)
  }, [])
  useEffect(() => {
    let active = true
    void (async () => {
      try {
        const [c, snapshot] = await Promise.all([fetchQQBotConfig(), fetchQQBotSnapshot()])
        if (active) {
          setConfig(c)
          setSnapshot(snapshot)
        }
      } catch {
        if (active) setMessage(t('qqbot.error'))
      }
    })()
    return () => {
      active = false
    }
  }, [t])

  async function run(fn: () => Promise<unknown>) {
    setBusy(true)
    try {
      await fn()
      await load()
      if (selected) setSelected(await fetchQQBotOperation(selected.id))
      setMessage(t('qqbot.success'))
    } catch {
      setMessage(t('qqbot.error'))
    } finally {
      setBusy(false)
    }
  }
  return (
    <section className="space-y-6 p-6">
      <h1>{t('qqbot.title')}</h1>
      <p>{t('qqbot.unknown')}</p>
      <Button isDisabled={busy} onClick={() => void run(load)}>
        {t('qqbot.refresh')}
      </Button>
      {message && <p role="status">{message}</p>}
      <form
        className="grid max-w-xl gap-3"
        onSubmit={(e) => {
          e.preventDefault()
          void run(async () => {
            await saveQQBotConfig({ ...config, ...(token ? { token } : {}) })
            setToken('')
          })
        }}
      >
        <label>
          <input
            type="checkbox"
            checked={config.enabled}
            onChange={(e) => setConfig({ ...config, enabled: e.target.checked })}
          />{' '}
          {t('qqbot.enabled')}
        </label>
        <label>
          {t('qqbot.bot')}
          <Input
            type="number"
            value={config.bot_id}
            onChange={(e) => setConfig({ ...config, bot_id: Number(e.target.value) })}
          />
        </label>
        <label>
          {t('qqbot.cidrs')}
          <Input
            value={config.cidrs}
            onChange={(e) => setConfig({ ...config, cidrs: e.target.value })}
          />
        </label>
        <label>
          {t('qqbot.interval')}
          <Input
            type="number"
            min={100}
            max={60000}
            value={config.interval_ms}
            onChange={(e) => setConfig({ ...config, interval_ms: Number(e.target.value) })}
          />
        </label>
        <label>
          {t('qqbot.token')}
          <Input
            type="password"
            autoComplete="new-password"
            value={token}
            onChange={(e) => setToken(e.target.value)}
          />
        </label>
        <Button isDisabled={busy} type="submit">
          {t('qqbot.save')}
        </Button>
      </form>
      <p>
        {t('qqbot.connected')}: {t(snapshot?.connected ? 'qqbot.online' : 'qqbot.offline')}{' '}
        {snapshot?.last_frame}
      </p>
      {snapshot?.ingress_error && <p role="alert">{snapshot.ingress_error}</p>}
      <p>
        {t('qqbot.rate')}: {t(snapshot?.redis_available ? 'qqbot.online' : 'qqbot.offline')} /{' '}
        {snapshot?.rate_wait_ms}
      </p>
      <h2>{t('qqbot.risk')}</h2>
      {snapshot?.circuits.map((c) => (
        <div key={c.class}>
          {c.class}: {c.state}, {c.successes} / {c.errors}{' '}
          <Button
            isDisabled={busy}
            onClick={() => void run(() => pauseQQBotCircuit(c.class, !c.paused))}
          >
            {t(c.paused ? 'qqbot.resume' : 'qqbot.pause')}
          </Button>
        </div>
      ))}
      <h2>{t('qqbot.operations')}</h2>
      <label>
        {t('qqbot.id')}
        <Input value={query} onChange={(e) => setQuery(e.target.value)} />
      </label>
      <Button
        isDisabled={busy}
        onClick={() =>
          void run(async () => setSelected(query ? await fetchQQBotOperation(query) : null))
        }
      >
        {t('common.search')}
      </Button>
      <DataTable
        variant="ledger"
        data={selected ? [selected] : (snapshot?.operations ?? [])}
        loadingText={t('common.loading')}
        emptyText={t('common.empty')}
        columns={[
          { accessorKey: 'business_key', header: t('qqbot.key') },
          { accessorKey: 'kind', header: t('qqbot.class') },
          { accessorKey: 'status', header: t('qqbot.status') },
          { accessorKey: 'wait_reason', header: t('qqbot.waiting') },
          { accessorKey: 'result', header: t('qqbot.result') },
          {
            id: 'actions',
            header: t('qqbot.detail'),
            cell: ({ row: { original: o } }) => (
              <>
                {['queued', 'leased', 'executing'].includes(o.status) && (
                  <Button
                    isDisabled={busy || o.cancel_requested}
                    onClick={() => void run(() => cancelQQBotOperation(o.id))}
                  >
                    {t('qqbot.cancel')}
                  </Button>
                )}
                {o.status === 'failed' &&
                  o.failures >= 3 &&
                  !o.business_key.startsWith('retry:') && (
                    <Button
                      isDisabled={busy}
                      onClick={() => void run(() => retryQQBotOperation(o.id))}
                    >
                      {t('qqbot.retry')}
                    </Button>
                  )}
              </>
            ),
          },
        ]}
      />
      <h2>{t('qqbot.attempts')}</h2>
      <DataTable
        variant="ledger"
        data={snapshot?.attempts ?? []}
        loadingText={t('common.loading')}
        emptyText={t('common.empty')}
        columns={[
          { accessorKey: 'operation_id', header: t('qqbot.id') },
          { accessorKey: 'classification', header: t('qqbot.status') },
          { accessorKey: 'started_at', header: t('qqbot.detail') },
        ]}
      />
      <h2>{t('qqbot.deliveries')}</h2>
      <DataTable
        variant="ledger"
        data={snapshot?.deliveries ?? []}
        loadingText={t('common.loading')}
        emptyText={t('common.empty')}
        columns={[
          { accessorKey: 'consumer', header: t('qqbot.key') },
          { accessorKey: 'status', header: t('qqbot.status') },
          { accessorKey: 'attempts', header: t('qqbot.failures') },
          { accessorKey: 'last_error', header: t('qqbot.detail') },
          {
            id: 'actions',
            header: t('qqbot.retry'),
            cell: ({ row: { original: d } }) =>
              d.status === 'dead' &&
              d.attempts < 10 && (
                <Button isDisabled={busy} onClick={() => void run(() => retryQQBotDelivery(d.id))}>
                  {t('qqbot.retry')}
                </Button>
              ),
          },
        ]}
      />
      <h2>{t('qqbot.events')}</h2>
      {snapshot?.events.map((e) => (
        <details key={e.id}>
          <summary>
            {e.created_at} {e.id}
          </summary>
          <pre className="overflow-auto">{e.payload}</pre>
        </details>
      ))}
    </section>
  )
}
