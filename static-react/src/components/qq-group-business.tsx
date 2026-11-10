import { useEffect, useState } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { DataTable } from '@/components/ui/data-table'
import { useI18n } from '@/i18n'
import { fetchQQGroupBusiness, saveQQGroupRule } from '@/api/qqbot'
import type { QQGroupRule, QQGroupBusinessAudit } from '@/types/api/qqbot'
const initial: QQGroupRule = {
  group_id: 0,
  version: 0,
  admission_enabled: false,
  auto_reject: false,
  corporation_ids: [],
  roles: [],
  card_enabled: false,
  card_template: '{character_name}',
  inspection_enabled: false,
  auto_kick: false,
  mismatch_confirmations: 3,
  observation_hours: 24,
  last_inspection_at: '',
  updated_at: '',
}
export function QQGroupBusiness() {
  const { t } = useI18n()
  const [audit, setAudit] = useState<QQGroupBusinessAudit | null>(null)
  const [rule, setRule] = useState(initial)
  const [corps, setCorps] = useState('')
  const [roles, setRoles] = useState('')
  const [busy, setBusy] = useState(false)
  const [message, setMessage] = useState('')
  useEffect(() => {
    let active = true
    void fetchQQGroupBusiness()
      .then((v) => {
        if (active) setAudit(v)
      })
      .catch(() => {
        if (active) setMessage(t('qqbot.error'))
      })
    return () => {
      active = false
    }
  }, [t])
  async function run(save: boolean) {
    setBusy(true)
    try {
      if (save)
        await saveQQGroupRule({
          ...rule,
          corporation_ids: corps.trim() ? corps.split(',').map(Number) : [],
          roles: roles
            .split(',')
            .map((s) => s.trim())
            .filter(Boolean),
        })
      setAudit(await fetchQQGroupBusiness())
      setMessage(t('qqbot.success'))
    } catch {
      setMessage(t('qqbot.error'))
    } finally {
      setBusy(false)
    }
  }
  function edit(v: QQGroupRule) {
    setRule(v)
    setCorps((v.corporation_ids ?? []).join(','))
    setRoles((v.roles ?? []).join(','))
  }
  return (
    <section className="space-y-3">
      <h2>{t('qqbot.groupBusiness')}</h2>
      <p>{t('qqbot.groupPolicy')}</p>
      <form
        className="grid max-w-xl gap-3"
        onSubmit={(e) => {
          e.preventDefault()
          void run(true)
        }}
      >
        <label>
          {t('qqbot.notificationGroup')}
          <Input
            type="number"
            min={1}
            value={rule.group_id}
            onChange={(e) => setRule({ ...rule, group_id: Number(e.target.value) })}
          />
        </label>
        <label>
          {t('qqbot.corporations')}
          <Input value={corps} onChange={(e) => setCorps(e.target.value)} />
        </label>
        <label>
          {t('qqbot.roles')}
          <Input value={roles} onChange={(e) => setRoles(e.target.value)} />
        </label>
        {(
          [
            'admission_enabled',
            'auto_reject',
            'card_enabled',
            'inspection_enabled',
            'auto_kick',
          ] as const
        ).map((key) => (
          <label key={key}>
            <input
              type="checkbox"
              checked={rule[key]}
              onChange={(e) => setRule({ ...rule, [key]: e.target.checked })}
            />{' '}
            {t(`qqbot.${key}`)}
          </label>
        ))}
        <label>
          {t('qqbot.cardTemplate')}
          <Input
            value={rule.card_template}
            onChange={(e) => setRule({ ...rule, card_template: e.target.value })}
          />
        </label>
        <p>{t('qqbot.cardVariables')}</p>
        <label>
          {t('qqbot.confirmations')}
          <Input
            type="number"
            min={1}
            max={100}
            value={rule.mismatch_confirmations}
            onChange={(e) => setRule({ ...rule, mismatch_confirmations: Number(e.target.value) })}
          />
        </label>
        <label>
          {t('qqbot.observation')}
          <Input
            type="number"
            min={0}
            max={8760}
            value={rule.observation_hours}
            onChange={(e) => setRule({ ...rule, observation_hours: Number(e.target.value) })}
          />
        </label>
        <Button type="submit" isDisabled={busy}>
          {t('qqbot.save')}
        </Button>
      </form>
      <Button isDisabled={busy} onClick={() => void run(false)}>
        {t('qqbot.refresh')}
      </Button>
      <p role="status">{message}</p>
      <h3>{t('qqbot.rules')}</h3>
      <DataTable
        variant="ledger"
        data={audit?.rules ?? []}
        loadingText={t('common.loading')}
        emptyText={t('common.empty')}
        columns={[
          { accessorKey: 'group_id', header: t('qqbot.notificationGroup') },
          { accessorKey: 'version', header: t('qqbot.version') },
          { accessorKey: 'auto_kick', header: t('qqbot.auto_kick') },
          {
            id: 'edit',
            header: t('qqbot.detail'),
            cell: ({ row }) => (
              <Button onClick={() => edit(row.original)}>{t('qqbot.editRule')}</Button>
            ),
          },
        ]}
      />
      <h3>{t('qqbot.applications')}</h3>
      <DataTable
        variant="ledger"
        data={audit?.applications ?? []}
        loadingText={t('common.loading')}
        emptyText={t('common.empty')}
        columns={['group_id', 'qq', 'status', 'evidence', 'operation_id'].map((key) => ({
          accessorKey: key,
          header: t(`qqbot.field_${key}`),
        }))}
      />
      <h3>{t('qqbot.members')}</h3>
      <DataTable
        variant="ledger"
        data={audit?.members ?? []}
        loadingText={t('common.loading')}
        emptyText={t('common.empty')}
        columns={[
          'group_id',
          'qq',
          'present',
          'confirmations',
          'evidence',
          'card_target',
          'card_status',
          'card_operation_id',
          'kick_status',
          'kick_operation_id',
        ].map((key) => ({ accessorKey: key, header: t(`qqbot.field_${key}`) }))}
      />
      <h3>{t('qqbot.inspections')}</h3>
      <DataTable
        variant="ledger"
        data={audit?.inspections ?? []}
        loadingText={t('common.loading')}
        emptyText={t('common.empty')}
        columns={['id', 'group_id', 'rule_version', 'status', 'operation_id'].map((key) => ({
          accessorKey: key,
          header: t(`qqbot.field_${key}`),
        }))}
      />
      <h3>{t('qqbot.judgements')}</h3>
      <DataTable
        variant="ledger"
        data={audit?.judgements ?? []}
        loadingText={t('common.loading')}
        emptyText={t('common.empty')}
        columns={['run_id', 'qq', 'done', 'decision', 'evidence', 'operation_id'].map((key) => ({
          accessorKey: key,
          header: t(`qqbot.field_${key}`),
        }))}
      />
    </section>
  )
}
