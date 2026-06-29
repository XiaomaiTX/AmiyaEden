import { useEffect, useState } from 'react'
import { fetchCorpNpcKills } from '@/api/npc-kill'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { formatIskPlain } from '@/lib/isk'
import { useI18n } from '@/i18n'
import type { NpcKillCorpRequest, NpcKillCorpResponse } from '@/types/api/npc-kill'

type DateRangeState = {
  startDate: string
  endDate: string
}

type FilterState = {
  refTypes: string[]
  solarSystemIds: string
  userIds: string
  characterIds: string
  minAmount: string
  maxAmount: string
}

function parseIdList(value: string) {
  return value
    .split(',')
    .map((item) => Number(item.trim()))
    .filter((item) => Number.isInteger(item) && item > 0)
}

function buildNpcKillFilterPayload(filters: FilterState) {
  const solarSystemIds = parseIdList(filters.solarSystemIds)
  const userIds = parseIdList(filters.userIds)
  const characterIds = parseIdList(filters.characterIds)
  const minAmount = filters.minAmount === '' ? undefined : Number(filters.minAmount)
  const maxAmount = filters.maxAmount === '' ? undefined : Number(filters.maxAmount)

  return {
    ref_types: filters.refTypes.length > 0 ? filters.refTypes : undefined,
    solar_system_ids: solarSystemIds.length > 0 ? solarSystemIds : undefined,
    user_ids: userIds.length > 0 ? userIds : undefined,
    character_ids: characterIds.length > 0 ? characterIds : undefined,
    min_amount: Number.isFinite(minAmount) ? minAmount : undefined,
    max_amount: Number.isFinite(maxAmount) ? maxAmount : undefined,
  }
}

function getErrorMessage(error: unknown, fallback: string) {
  return error instanceof Error && error.message ? error.message : fallback
}

function SummaryCard({ label, value, tone }: { label: string; value: string; tone: string }) {
  return (
    <article className="rounded-lg border bg-card p-4 text-center">
      <p className="text-sm text-muted-foreground">{label}</p>
      <p className={`mt-1 text-xl font-semibold ${tone}`}>{value}</p>
    </article>
  )
}

export function DashboardNpcKillsPage() {
  const { t } = useI18n()
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [reportData, setReportData] = useState<NpcKillCorpResponse | null>(null)
  const [draftDateRange, setDraftDateRange] = useState<DateRangeState>({
    startDate: '',
    endDate: '',
  })
  const [appliedDateRange, setAppliedDateRange] = useState<DateRangeState>({
    startDate: '',
    endDate: '',
  })
  const [draftFilters, setDraftFilters] = useState<FilterState>({
    refTypes: [],
    solarSystemIds: '',
    userIds: '',
    characterIds: '',
    minAmount: '',
    maxAmount: '',
  })
  const [appliedFilters, setAppliedFilters] = useState<FilterState>({
    refTypes: [],
    solarSystemIds: '',
    userIds: '',
    characterIds: '',
    minAmount: '',
    maxAmount: '',
  })
  const refTypeOptions = [
    'bounty_prizes',
    'ess_escrow_transfer',
    'corporate_reward_payout',
    'agent_mission_reward',
  ]
  const formatRefTypeLabel = (refType: string) =>
    ({
      bounty_prizes: t('npcKill.refTypes.bounty_prizes'),
      ess_escrow_transfer: t('npcKill.refTypes.ess_escrow_transfer'),
      corporate_reward_payout: t('npcKill.refTypes.corporate_reward_payout'),
      agent_mission_reward: t('npcKill.refTypes.agent_mission_reward'),
    })[refType] ?? refType

  useEffect(() => {
    let cancelled = false

    const loadData = async () => {
      setLoading(true)
      setError(null)

      try {
        const payload: NpcKillCorpRequest = {}
        if (appliedDateRange.startDate) {
          payload.start_date = appliedDateRange.startDate
        }
        if (appliedDateRange.endDate) {
          payload.end_date = appliedDateRange.endDate
        }
        Object.assign(payload, buildNpcKillFilterPayload(appliedFilters))

        const data = await fetchCorpNpcKills(payload)
        if (!cancelled) {
          setReportData(data)
        }
      } catch (caughtError) {
        if (!cancelled) {
          setError(getErrorMessage(caughtError, t('npcKill.loadReportFailed')))
          setReportData(null)
        }
      } finally {
        if (!cancelled) {
          setLoading(false)
        }
      }
    }

    void loadData()
    return () => {
      cancelled = true
    }
  }, [appliedDateRange, appliedFilters, t])

  const handleSearch = () => {
    setAppliedDateRange({ ...draftDateRange })
    setAppliedFilters({ ...draftFilters })
  }

  const handleReset = () => {
    const emptyRange = { startDate: '', endDate: '' }
    const emptyFilters = {
      refTypes: [],
      solarSystemIds: '',
      userIds: '',
      characterIds: '',
      minAmount: '',
      maxAmount: '',
    }
    setDraftDateRange(emptyRange)
    setAppliedDateRange(emptyRange)
    setDraftFilters(emptyFilters)
    setAppliedFilters(emptyFilters)
  }

  return (
    <section className="space-y-4">
      <div className="rounded-lg border bg-card p-5">
        <div className="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
          <div>
            <h1 className="text-xl font-semibold">{t('nav.dashboard.npcKills')}</h1>
            <p className="mt-1 text-sm text-muted-foreground">{t('nav.group.dashboard')}</p>
          </div>

          <div className="flex flex-wrap items-end gap-3">
            <label className="space-y-1">
              <span className="text-sm text-muted-foreground">{t('npcKill.startDate')}</span>
              <Input
                type="date"
                value={draftDateRange.startDate}
                onChange={(event) =>
                  setDraftDateRange((current) => ({ ...current, startDate: event.target.value }))
                }
              />
            </label>

            <label className="space-y-1">
              <span className="text-sm text-muted-foreground">{t('npcKill.endDate')}</span>
              <Input
                type="date"
                value={draftDateRange.endDate}
                onChange={(event) =>
                  setDraftDateRange((current) => ({ ...current, endDate: event.target.value }))
                }
              />
            </label>

            <Button type="button" onClick={handleSearch}>
              {t('npcKill.search')}
            </Button>
            <Button type="button" variant="outline" onClick={handleReset}>
              {t('npcKill.reset')}
            </Button>
          </div>

          <div className="flex flex-wrap items-end gap-3">
            <label className="space-y-1">
              <span className="text-sm text-muted-foreground">{t('npcKill.filters.refTypes')}</span>
              <select
                multiple
                className="min-h-10 rounded-md border border-input bg-background px-3 py-2 text-sm"
                value={draftFilters.refTypes}
                onChange={(event) =>
                  setDraftFilters((current) => ({
                    ...current,
                    refTypes: Array.from(event.target.selectedOptions, (option) => option.value),
                  }))
                }
              >
                {refTypeOptions.map((refType) => (
                  <option key={refType} value={refType}>
                    {formatRefTypeLabel(refType)}
                  </option>
                ))}
              </select>
            </label>
            <label className="space-y-1">
              <span className="text-sm text-muted-foreground">
                {t('npcKill.filters.solarSystemIds')}
              </span>
              <Input
                value={draftFilters.solarSystemIds}
                onChange={(event) =>
                  setDraftFilters((current) => ({ ...current, solarSystemIds: event.target.value }))
                }
              />
            </label>
            <label className="space-y-1">
              <span className="text-sm text-muted-foreground">{t('npcKill.filters.userIds')}</span>
              <Input
                value={draftFilters.userIds}
                onChange={(event) =>
                  setDraftFilters((current) => ({ ...current, userIds: event.target.value }))
                }
              />
            </label>
            <label className="space-y-1">
              <span className="text-sm text-muted-foreground">
                {t('npcKill.filters.characterIds')}
              </span>
              <Input
                value={draftFilters.characterIds}
                onChange={(event) =>
                  setDraftFilters((current) => ({ ...current, characterIds: event.target.value }))
                }
              />
            </label>
            <label className="space-y-1">
              <span className="text-sm text-muted-foreground">
                {t('npcKill.filters.minAmount')}
              </span>
              <Input
                type="number"
                min="0"
                value={draftFilters.minAmount}
                onChange={(event) =>
                  setDraftFilters((current) => ({ ...current, minAmount: event.target.value }))
                }
              />
            </label>
            <label className="space-y-1">
              <span className="text-sm text-muted-foreground">
                {t('npcKill.filters.maxAmount')}
              </span>
              <Input
                type="number"
                min="0"
                value={draftFilters.maxAmount}
                onChange={(event) =>
                  setDraftFilters((current) => ({ ...current, maxAmount: event.target.value }))
                }
              />
            </label>
          </div>
        </div>
      </div>

      {loading ? <p className="text-sm text-muted-foreground">{t('npcKill.loading')}</p> : null}
      {error ? <p className="text-sm text-destructive">{error}</p> : null}
      {!loading && !error && !reportData ? (
        <p className="text-sm text-muted-foreground">{t('npcKill.noData')}</p>
      ) : null}

      {reportData ? (
        <>
          <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
            <SummaryCard
              label={t('npcKill.totalBounty')}
              value={formatIskPlain(reportData.summary.total_bounty)}
              tone="text-emerald-600"
            />
            <SummaryCard
              label={t('npcKill.totalTax')}
              value={formatIskPlain(reportData.summary.total_tax)}
              tone="text-destructive"
            />
            <SummaryCard
              label={t('npcKill.actualIncome')}
              value={formatIskPlain(reportData.summary.actual_income)}
              tone="text-emerald-600"
            />
            <SummaryCard
              label={t('npcKill.totalRecords')}
              value={String(reportData.summary.total_records)}
              tone="text-foreground"
            />
          </div>

          <div className="grid gap-4 lg:grid-cols-2">
            <section className="overflow-hidden rounded-lg border bg-card">
              <div className="border-b px-4 py-3 text-sm font-medium">{t('npcKill.members')}</div>
              <div className="overflow-x-auto">
                <table className="min-w-full text-sm">
                  <thead>
                    <tr className="border-b bg-muted/40 text-left">
                      <th className="px-3 py-2">#</th>
                      <th className="px-3 py-2">{t('npcKill.userDisplayName')}</th>
                      <th className="px-3 py-2 text-right">{t('npcKill.characterCount')}</th>
                      <th className="px-3 py-2 text-right">{t('npcKill.totalBounty')}</th>
                      <th className="px-3 py-2 text-right">{t('npcKill.totalTax')}</th>
                      <th className="px-3 py-2 text-right">{t('npcKill.actualIncome')}</th>
                      <th className="px-3 py-2 text-right">{t('npcKill.recordCount')}</th>
                    </tr>
                  </thead>
                  <tbody>
                    {reportData.members.map((member, index) => (
                      <tr key={member.user_id} className="border-b">
                        <td className="px-3 py-2 text-muted-foreground">{index + 1}</td>
                        <td className="px-3 py-2">{member.display_name}</td>
                        <td className="px-3 py-2 text-right">{member.character_count}</td>
                        <td className="px-3 py-2 text-right text-emerald-600">
                          {formatIskPlain(member.total_bounty)}
                        </td>
                        <td className="px-3 py-2 text-right text-destructive">
                          {formatIskPlain(member.total_tax)}
                        </td>
                        <td className="px-3 py-2 text-right text-emerald-600">
                          {formatIskPlain(member.actual_income)}
                        </td>
                        <td className="px-3 py-2 text-right">{member.record_count}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </section>

            <section className="overflow-hidden rounded-lg border bg-card">
              <div className="border-b px-4 py-3 text-sm font-medium">{t('npcKill.bySystem')}</div>
              <div className="overflow-x-auto">
                <table className="min-w-full text-sm">
                  <thead>
                    <tr className="border-b bg-muted/40 text-left">
                      <th className="px-3 py-2">#</th>
                      <th className="px-3 py-2">{t('npcKill.solarSystem')}</th>
                      <th className="px-3 py-2 text-right">{t('npcKill.systemCount')}</th>
                      <th className="px-3 py-2 text-right">{t('npcKill.systemAmount')}</th>
                    </tr>
                  </thead>
                  <tbody>
                    {reportData.by_system.map((system, index) => (
                      <tr key={system.solar_system_id} className="border-b">
                        <td className="px-3 py-2 text-muted-foreground">{index + 1}</td>
                        <td className="px-3 py-2">{system.solar_system_name}</td>
                        <td className="px-3 py-2 text-right">{system.count}</td>
                        <td className="px-3 py-2 text-right text-emerald-600">
                          {formatIskPlain(system.amount)}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </section>
          </div>

          <section className="overflow-hidden rounded-lg border bg-card">
            <div className="border-b px-4 py-3 text-sm font-medium">{t('npcKill.trend')}</div>
            <div className="overflow-x-auto">
              <table className="min-w-full text-sm">
                <thead>
                  <tr className="border-b bg-muted/40 text-left">
                    <th className="px-3 py-2">{t('npcKill.trendDate')}</th>
                    <th className="px-3 py-2 text-right">{t('npcKill.trendAmount')}</th>
                    <th className="px-3 py-2 text-right">{t('npcKill.trendCount')}</th>
                  </tr>
                </thead>
                <tbody>
                  {reportData.trend.map((item) => (
                    <tr key={item.date} className="border-b">
                      <td className="px-3 py-2">{item.date}</td>
                      <td className="px-3 py-2 text-right text-emerald-600">
                        {formatIskPlain(item.amount)}
                      </td>
                      <td className="px-3 py-2 text-right">{item.count}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </section>
        </>
      ) : null}
    </section>
  )
}
