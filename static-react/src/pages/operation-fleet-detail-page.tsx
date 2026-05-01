import { useCallback, useEffect, useMemo, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import {
  addFleetMembersByCharacterNames,
  createFleetInvite,
  deactivateFleetInvite,
  fetchFleetDetail,
  fetchFleetInvites,
  issuePap,
  pingFleet,
  refreshFleetESI,
  syncESIFleetMembers,
  fetchMembersWithPap,
} from '@/api/fleet'
import { Button } from '@/components/ui/button'
import { useI18n } from '@/i18n'
import { useSessionStore } from '@/stores'
import type { FleetInvite, FleetItem, MemberWithPap } from '@/types/api/fleet'
import { formatDateTime, getErrorMessage, ShopBadge, ShopDialog } from './shop-page-utils'

function importanceBadgeClass(value: string) {
  switch (value) {
    case 'cta':
      return 'bg-amber-100 text-amber-700 dark:bg-amber-500/10 dark:text-amber-300'
    case 'strat_op':
      return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-300'
    default:
      return 'bg-slate-100 text-slate-700 dark:bg-slate-500/10 dark:text-slate-300'
  }
}

export function OperationFleetDetailPage() {
  const { t } = useI18n()
  const navigate = useNavigate()
  const params = useParams<{ id: string }>()
  const fleetId = params.id ?? ''
  const roles = useSessionStore((state) => state.roles)
  const [fleet, setFleet] = useState<FleetItem | null>(null)
  const [loading, setLoading] = useState(true)
  const [membersLoading, setMembersLoading] = useState(true)
  const [invitesLoading, setInvitesLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [members, setMembers] = useState<MemberWithPap[]>([])
  const [memberTotal, setMemberTotal] = useState(0)
  const [memberPage, setMemberPage] = useState(1)
  const [memberPageSize, setMemberPageSize] = useState(20)
  const [invites, setInvites] = useState<FleetInvite[]>([])
  const [syncLoading, setSyncLoading] = useState(false)
  const [papLoading, setPapLoading] = useState(false)
  const [pingLoading, setPingLoading] = useState(false)
  const [inviteLoading, setInviteLoading] = useState(false)
  const [manualOpen, setManualOpen] = useState(false)
  const [manualLoading, setManualLoading] = useState(false)
  const [manualText, setManualText] = useState('')

  const canManageFleet = roles.some((role) => ['super_admin', 'admin', 'fc', 'senior_fc'].includes(role))

  const pageCount = useMemo(
    () => Math.max(1, Math.ceil(memberTotal / memberPageSize) || 1),
    [memberPageSize, memberTotal]
  )

  const loadFleet = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      try {
        const refreshed = await refreshFleetESI(fleetId)
        setFleet(refreshed)
      } catch {
        setFleet(await fetchFleetDetail(fleetId))
      }
    } catch (caughtError) {
      setError(getErrorMessage(caughtError, t('fleet.detail.loadFailed')))
      setFleet(null)
    } finally {
      setLoading(false)
    }
  }, [fleetId, t])

  const loadMembers = useCallback(async () => {
    setMembersLoading(true)
    try {
      const response = await fetchMembersWithPap(fleetId, {
        current: memberPage,
        size: memberPageSize,
      })
      setMembers(response.list ?? [])
      setMemberTotal(response.total ?? 0)
      setMemberPage(response.page ?? memberPage)
      setMemberPageSize(response.pageSize ?? memberPageSize)
    } catch (caughtError) {
      setError(getErrorMessage(caughtError, t('fleet.detail.loadFailed')))
      setMembers([])
      setMemberTotal(0)
    } finally {
      setMembersLoading(false)
    }
  }, [fleetId, memberPage, memberPageSize, t])

  const loadInvites = useCallback(async () => {
    setInvitesLoading(true)
    try {
      setInvites(await fetchFleetInvites(fleetId))
    } catch {
      setInvites([])
    } finally {
      setInvitesLoading(false)
    }
  }, [fleetId])

  useEffect(() => {
    const timer = window.setTimeout(() => {
      void loadFleet()
      void loadMembers()
      void loadInvites()
    }, 0)
    return () => window.clearTimeout(timer)
  }, [loadFleet, loadInvites, loadMembers])

  const handleSyncESI = useCallback(async () => {
    setSyncLoading(true)
    try {
      await syncESIFleetMembers(fleetId)
      setMemberPage(1)
      void loadMembers()
    } catch (caughtError) {
      setError(getErrorMessage(caughtError, t('fleet.detail.syncFailed')))
    } finally {
      setSyncLoading(false)
    }
  }, [fleetId, loadMembers, t])

  const handleIssuePap = useCallback(async () => {
    if (!window.confirm(t('fleet.detail.issuePapConfirm'))) {
      return
    }

    setPapLoading(true)
    try {
      await issuePap(fleetId)
      void loadMembers()
    } catch (caughtError) {
      setError(getErrorMessage(caughtError, t('fleet.detail.issuePapFailed')))
    } finally {
      setPapLoading(false)
    }
  }, [fleetId, loadMembers, t])

  const handlePing = useCallback(async () => {
    setPingLoading(true)
    try {
      await pingFleet(fleetId)
    } catch (caughtError) {
      setError(getErrorMessage(caughtError, t('fleet.detail.pingFailed')))
    } finally {
      setPingLoading(false)
    }
  }, [fleetId, t])

  const handleCreateInvite = useCallback(async () => {
    setInviteLoading(true)
    try {
      await createFleetInvite(fleetId)
      await loadInvites()
    } catch (caughtError) {
      setError(getErrorMessage(caughtError, t('fleet.detail.inviteCreateFailed')))
    } finally {
      setInviteLoading(false)
    }
  }, [fleetId, loadInvites, t])

  const handleDeactivateInvite = useCallback(
    async (invite: FleetInvite) => {
      if (!window.confirm(t('fleet.detail.inviteDeactivateConfirm'))) {
        return
      }

      try {
        await deactivateFleetInvite(invite.id)
        await loadInvites()
      } catch (caughtError) {
        setError(getErrorMessage(caughtError, t('fleet.detail.inviteDeactivateFailed')))
      }
    },
    [loadInvites, t]
  )

  const handleManualAddMembers = useCallback(async () => {
    const names = Array.from(
      new Set(
        manualText
          .split('\n')
          .map((name) => name.trim())
          .filter(Boolean)
      )
    )

    if (names.length === 0) {
      setError(t('fleet.detail.manualAddEmpty'))
      return
    }

    setManualLoading(true)
    try {
      const result = await addFleetMembersByCharacterNames(fleetId, { character_names: names })
      setManualOpen(false)
      setManualText('')
      if (result.added_character_names.length || result.missing_character_names.length) {
        void loadMembers()
      }
    } catch (caughtError) {
      setError(getErrorMessage(caughtError, t('fleet.detail.manualAddFailed')))
    } finally {
      setManualLoading(false)
    }
  }, [fleetId, loadMembers, manualText, t])

  const copyInviteLink = async (invite: FleetInvite) => {
    const link = `${window.location.origin}/#/operation/join?code=${invite.code}`
    try {
      await navigator.clipboard.writeText(link)
    } catch {
      setError(t('common.copyFailed'))
    }
  }

  const refreshAll = useCallback(() => {
    void loadFleet()
    void loadMembers()
    void loadInvites()
  }, [loadFleet, loadInvites, loadMembers])

  return (
    <section className="space-y-4">
      <div className="rounded-lg border bg-card p-5">
        <div className="flex flex-wrap items-center gap-3">
          <Button type="button" variant="outline" onClick={() => navigate('/operation/fleets')}>
            {t('common.back')}
          </Button>
          <div>
            <h1 className="text-xl font-semibold">{fleet?.title || t('fleet.detail.title')}</h1>
            <p className="mt-1 text-sm text-muted-foreground">{t('fleet.detail.subtitle')}</p>
          </div>
        </div>
      </div>

      {error ? <p className="text-sm text-destructive">{error}</p> : null}
      {loading ? <p className="text-sm text-muted-foreground">{t('fleet.detail.loading')}</p> : null}

      <div className="grid gap-4 xl:grid-cols-[1.2fr_0.8fr]">
        <div className="rounded-lg border bg-card p-5">
          <div className="flex flex-wrap items-start justify-between gap-3">
            <div>
              <h2 className="text-lg font-semibold">{t('fleet.detail.basicInfo')}</h2>
              <p className="mt-1 text-sm text-muted-foreground">{t('fleet.detail.basicHint')}</p>
            </div>
            <Button type="button" variant="outline" onClick={refreshAll}>
              {t('common.refresh')}
            </Button>
          </div>

          {fleet ? (
            <div className="mt-4 grid gap-3 sm:grid-cols-2">
              <div className="rounded-lg border p-3">
                <div className="text-xs text-muted-foreground">{t('fleet.fields.fc')}</div>
                <div className="mt-1 font-medium">{fleet.fc_display_name || fleet.fc_character_name}</div>
              </div>
              <div className="rounded-lg border p-3">
                <div className="text-xs text-muted-foreground">{t('fleet.fields.importance')}</div>
                <div className="mt-1">
                  <ShopBadge className={importanceBadgeClass(fleet.importance)}>{fleet.importance}</ShopBadge>
                </div>
              </div>
              <div className="rounded-lg border p-3">
                <div className="text-xs text-muted-foreground">{t('fleet.fields.papCount')}</div>
                <div className="mt-1 font-medium">{fleet.pap_count}</div>
              </div>
              <div className="rounded-lg border p-3">
                <div className="text-xs text-muted-foreground">{t('fleet.fields.esiFleetId')}</div>
                <div className="mt-1 font-medium">{fleet.esi_fleet_id ?? '-'}</div>
              </div>
              <div className="rounded-lg border p-3">
                <div className="text-xs text-muted-foreground">{t('fleet.fields.startAt')}</div>
                <div className="mt-1 font-medium">{formatDateTime(fleet.start_at)}</div>
              </div>
              <div className="rounded-lg border p-3">
                <div className="text-xs text-muted-foreground">{t('fleet.fields.endAt')}</div>
                <div className="mt-1 font-medium">{formatDateTime(fleet.end_at)}</div>
              </div>
              <div className="rounded-lg border p-3 sm:col-span-2">
                <div className="text-xs text-muted-foreground">{t('fleet.fields.description')}</div>
                <div className="mt-1 whitespace-pre-wrap">{fleet.description || '-'}</div>
              </div>
            </div>
          ) : null}
        </div>

        <div className="rounded-lg border bg-card p-5">
          <h2 className="text-lg font-semibold">{t('fleet.detail.actions')}</h2>
          <div className="mt-4 flex flex-wrap gap-2">
            <Button type="button" onClick={() => void handleSyncESI()} disabled={!canManageFleet || syncLoading}>
              {t('fleet.members.syncESI')}
            </Button>
            <Button type="button" variant="outline" onClick={() => void handleIssuePap()} disabled={!canManageFleet || papLoading}>
              {t('fleet.pap.issue')}
            </Button>
            <Button type="button" variant="outline" onClick={() => setManualOpen(true)} disabled={!canManageFleet}>
              {t('fleet.members.manualAdd')}
            </Button>
            <Button type="button" variant="outline" onClick={() => void handlePing()} disabled={!canManageFleet || pingLoading}>
              {t('fleet.ping.send')}
            </Button>
            <Button type="button" variant="outline" onClick={() => void handleCreateInvite()} disabled={!canManageFleet || inviteLoading}>
              {t('fleet.invite.create')}
            </Button>
          </div>
          <div className="mt-4 space-y-2 text-sm text-muted-foreground">
            <div>{t('fleet.detail.memberCount', { count: memberTotal })}</div>
            <div>{t('fleet.detail.inviteCount', { count: invites.length })}</div>
          </div>
        </div>
      </div>

      <div className="rounded-lg border bg-card">
        <div className="border-b px-4 py-3 text-sm font-medium">{t('fleet.detail.memberTitle')}</div>
        <div className="overflow-x-auto">
          <table className="min-w-full text-sm">
            <thead>
              <tr className="border-b bg-muted/40 text-left">
                <th className="px-3 py-2">{t('fleet.members.characterName')}</th>
                <th className="px-3 py-2">{t('fleet.members.shipType')}</th>
                <th className="px-3 py-2">{t('fleet.members.solarSystem')}</th>
                <th className="px-3 py-2">{t('fleet.members.joinedAt')}</th>
                <th className="px-3 py-2">{t('fleet.pap.count')}</th>
              </tr>
            </thead>
            <tbody>
              {members.map((member) => (
                <tr key={member.id} className="border-b">
                  <td className="px-3 py-2">{member.character_name}</td>
                  <td className="px-3 py-2">{member.ship_type_id ?? '-'}</td>
                  <td className="px-3 py-2">{member.solar_system_id ?? '-'}</td>
                  <td className="px-3 py-2">{formatDateTime(member.joined_at)}</td>
                  <td className="px-3 py-2">{member.pap_count ?? '-'}</td>
                </tr>
              ))}
              {!membersLoading && members.length === 0 ? (
                <tr>
                  <td className="px-3 py-6 text-center text-muted-foreground" colSpan={5}>
                    {t('fleet.detail.memberEmpty')}
                  </td>
                </tr>
              ) : null}
            </tbody>
          </table>
        </div>
      </div>

      <div className="flex flex-wrap items-center gap-3 text-sm">
        <span>
          {memberPage}/{pageCount}
        </span>
        <Button type="button" variant="outline" size="sm" onClick={() => setMemberPage((current) => Math.max(1, current - 1))} disabled={memberPage <= 1}>
          {t('welfareMy.pagination.prev')}
        </Button>
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={() => setMemberPage((current) => current + 1)}
          disabled={members.length < memberPageSize || memberPage * memberPageSize >= memberTotal}
        >
          {t('welfareMy.pagination.next')}
        </Button>
        <label className="flex items-center gap-2">
          <span>{t('welfareMy.pageSize')}</span>
          <select
            className="h-8 rounded-md border border-input bg-background px-2 text-sm"
            value={memberPageSize}
            onChange={(event) => {
              setMemberPageSize(Number(event.target.value))
              setMemberPage(1)
            }}
          >
            {[10, 20, 50].map((size) => (
              <option key={size} value={size}>
                {size}
              </option>
            ))}
          </select>
        </label>
      </div>

      <div className="rounded-lg border bg-card">
        <div className="border-b px-4 py-3 text-sm font-medium">{t('fleet.invite.title')}</div>
        <div className="overflow-x-auto">
          <table className="min-w-full text-sm">
            <thead>
              <tr className="border-b bg-muted/40 text-left">
                <th className="px-3 py-2">{t('fleet.invite.code')}</th>
                <th className="px-3 py-2">{t('common.status')}</th>
                <th className="px-3 py-2">{t('fleet.invite.expiresAt')}</th>
                <th className="px-3 py-2">{t('common.operation')}</th>
              </tr>
            </thead>
            <tbody>
              {invites.map((invite) => (
                <tr key={invite.id} className="border-b">
                  <td className="px-3 py-2">
                    <code className="rounded bg-muted px-2 py-0.5 text-xs">{invite.code}</code>
                  </td>
                  <td className="px-3 py-2">
                    <ShopBadge className={invite.active ? 'bg-emerald-100 text-emerald-700' : 'bg-slate-100 text-slate-700'}>
                      {invite.active ? t('fleet.invite.active') : t('fleet.invite.inactive')}
                    </ShopBadge>
                  </td>
                  <td className="px-3 py-2">{formatDateTime(invite.expires_at)}</td>
                  <td className="px-3 py-2">
                    <div className="flex flex-wrap gap-2">
                      <Button type="button" size="sm" variant="outline" onClick={() => void copyInviteLink(invite)}>
                        {t('fleet.invite.copyLink')}
                      </Button>
                      {canManageFleet && invite.active ? (
                        <Button type="button" size="sm" variant="outline" onClick={() => void handleDeactivateInvite(invite)}>
                          {t('fleet.invite.deactivate')}
                        </Button>
                      ) : null}
                    </div>
                  </td>
                </tr>
              ))}
              {!invitesLoading && invites.length === 0 ? (
                <tr>
                  <td className="px-3 py-6 text-center text-muted-foreground" colSpan={4}>
                    {t('fleet.invite.empty')}
                  </td>
                </tr>
              ) : null}
            </tbody>
          </table>
        </div>
      </div>

      <ShopDialog
        open={manualOpen}
        title={t('fleet.members.manualAdd')}
        onClose={() => setManualOpen(false)}
        closeLabel={t('common.close')}
        footer={
          <>
            <Button type="button" variant="outline" onClick={() => setManualOpen(false)} disabled={manualLoading}>
              {t('common.cancel')}
            </Button>
            <Button type="button" onClick={() => void handleManualAddMembers()} disabled={manualLoading}>
              {manualLoading ? t('fleet.members.manualAdding') : t('common.confirm')}
            </Button>
          </>
        }
      >
        <div className="space-y-2 text-sm">
          <textarea
            className="min-h-40 w-full rounded-lg border border-input bg-background px-3 py-2 text-sm outline-none"
            value={manualText}
            placeholder={t('fleet.members.manualAddPlaceholder')}
            onChange={(event) => setManualText(event.target.value)}
          />
          <p className="text-muted-foreground">{t('fleet.members.manualAddHint')}</p>
        </div>
      </ShopDialog>
    </section>
  )
}
