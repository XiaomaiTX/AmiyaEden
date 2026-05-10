import { useEffect, useState } from 'react'
import { fetchMyCharacters } from '@/api/auth'
import { fetchNpcKills, fetchNpcKillsAll } from '@/api/npc-kill'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { formatIskPlain } from '@/lib/isk'
import { useI18n } from '@/i18n'

type DateRangeState = {
  startDate: string
  endDate: string
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

export function InfoNpcKillsPage() {
  const { t } = useI18n()
  const [characterLoading, setCharacterLoading] = useState(true)
  const [reportLoading, setReportLoading] = useState(true)
  const [characterError, setCharacterError] = useState<string | null>(null)
  const [reportError, setReportError] = useState<string | null>(null)
  const [characters, setCharacters] = useState<Api.Auth.EveCharacter[]>([])
  const [selectedCharacterId, setSelectedCharacterId] = useState(0)
  const [draftDateRange, setDraftDateRange] = useState<DateRangeState>({ startDate: '', endDate: '' })
  const [appliedDateRange, setAppliedDateRange] = useState<DateRangeState>({ startDate: '', endDate: '' })
  const [reportData, setReportData] = useState<Api.NpcKill.NpcKillResponse | null>(null)
  const formatRefTypeLabel = (refType: string) =>
    ({
      bounty_prizes: t('npcKill.refTypes.bounty_prizes'),
      ess_escrow_transfer: t('npcKill.refTypes.ess_escrow_transfer'),
    }[refType] ?? refType)

  useEffect(() => {
    let cancelled = false

    const loadCharacters = async () => {
      setCharacterLoading(true)
      setCharacterError(null)
      try {
        const list = await fetchMyCharacters()
        if (!cancelled) {
          setCharacters(list)
        }
      } catch (caughtError) {
        if (!cancelled) {
          setCharacterError(getErrorMessage(caughtError, t('npcKill.loadCharactersFailed')))
          setCharacters([])
        }
      } finally {
        if (!cancelled) {
          setCharacterLoading(false)
        }
      }
    }

    void loadCharacters()
    return () => {
      cancelled = true
    }
  }, [t])

  useEffect(() => {
    let cancelled = false

    const loadReport = async () => {
      setReportLoading(true)
      setReportError(null)

      try {
        const payload = {
          start_date: appliedDateRange.startDate || undefined,
          end_date: appliedDateRange.endDate || undefined,
        }

        const data =
          selectedCharacterId === 0
            ? await fetchNpcKillsAll(payload)
            : await fetchNpcKills({
                character_id: selectedCharacterId,
                ...payload,
              })

        if (!cancelled) {
          setReportData(data)
        }
      } catch (caughtError) {
        if (!cancelled) {
          setReportError(getErrorMessage(caughtError, t('npcKill.loadReportFailed')))
          setReportData(null)
        }
      } finally {
        if (!cancelled) {
          setReportLoading(false)
        }
      }
    }

    void loadReport()
    return () => {
      cancelled = true
    }
  }, [appliedDateRange, selectedCharacterId, t])

  const loading = characterLoading || reportLoading

  const handleSearch = () => {
    setAppliedDateRange({ ...draftDateRange })
  }

  const handleReset = () => {
    const emptyRange = { startDate: '', endDate: '' }
    setDraftDateRange(emptyRange)
    setAppliedDateRange(emptyRange)
  }

  return (
    <section className="space-y-4">
      <div className="rounded-lg border bg-card p-5">
        <div className="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
          <div>
            <h1 className="text-xl font-semibold">{t('nav.info.npcKills')}</h1>
            <p className="mt-1 text-sm text-muted-foreground">{t('nav.group.info')}</p>
          </div>

          <div className="flex flex-wrap items-end gap-3">
            <label className="space-y-1">
              <span className="text-sm text-muted-foreground">{t('npcKill.selectCharacter')}</span>
              <select
                className="h-10 rounded-md border border-input bg-background px-3 text-sm"
                value={selectedCharacterId}
                onChange={(event) => setSelectedCharacterId(Number(event.target.value))}
              >
                <option value={0}>{t('npcKill.allCharacters')}</option>
                {characters.map((character) => (
                  <option key={character.character_id} value={character.character_id}>
                    {character.character_name}
                  </option>
                ))}
              </select>
            </label>

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
        </div>
      </div>

      {loading ? <p className="text-sm text-muted-foreground">{t('npcKill.loading')}</p> : null}
      {characterError ? <p className="text-sm text-destructive">{characterError}</p> : null}
      {reportError ? <p className="text-sm text-destructive">{reportError}</p> : null}
      {!loading && !reportError && !reportData ? <p className="text-sm text-muted-foreground">{t('npcKill.noData')}</p> : null}

      {reportData ? (
        <>
          <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-5">
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
            <SummaryCard
              label={t('npcKill.estimatedHours')}
              value={String(reportData.summary.estimated_hours)}
              tone="text-foreground"
            />
          </div>

          <section className="overflow-hidden rounded-lg border bg-card">
            <div className="border-b px-4 py-3 text-sm font-medium">{t('npcKill.byNpc')}</div>
            <div className="overflow-x-auto">
              <table className="min-w-full text-sm">
                <thead>
                  <tr className="border-b bg-muted/40 text-left">
                    <th className="px-3 py-2">#</th>
                    <th className="px-3 py-2">{t('npcKill.npcName')}</th>
                    <th className="px-3 py-2 text-right">{t('npcKill.npcCount')}</th>
                  </tr>
                </thead>
                <tbody>
                  {reportData.by_npc.map((item, index) => (
                    <tr key={item.npc_id} className="border-b">
                      <td className="px-3 py-2 text-muted-foreground">{index + 1}</td>
                      <td className="px-3 py-2">{item.npc_name}</td>
                      <td className="px-3 py-2 text-right">{item.count}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </section>

          <div className="grid gap-4 lg:grid-cols-2">
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
                        <td className="px-3 py-2 text-right text-emerald-600">{formatIskPlain(system.amount)}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </section>

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
                        <td className="px-3 py-2 text-right text-emerald-600">{formatIskPlain(item.amount)}</td>
                        <td className="px-3 py-2 text-right">{item.count}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </section>
          </div>

          <section className="overflow-hidden rounded-lg border bg-card">
            <div className="border-b px-4 py-3 text-sm font-medium">{t('npcKill.journals')}</div>
            <div className="overflow-x-auto">
              <table className="min-w-full text-sm">
                <thead>
                  <tr className="border-b bg-muted/40 text-left">
                    <th className="px-3 py-2">{t('npcKill.journalDate')}</th>
                    <th className="px-3 py-2">{t('npcKill.journalRefType')}</th>
                    <th className="px-3 py-2 text-right">{t('npcKill.journalAmount')}</th>
                    <th className="px-3 py-2 text-right">{t('npcKill.journalTax')}</th>
                    <th className="px-3 py-2">{t('npcKill.journalSystem')}</th>
                    <th className="px-3 py-2">{t('npcKill.characterName')}</th>
                    <th className="px-3 py-2">{t('npcKill.journalReason')}</th>
                  </tr>
                </thead>
                <tbody>
                  {reportData.journals.map((journal) => (
                    <tr key={journal.id} className="border-b">
                      <td className="px-3 py-2">{journal.date}</td>
                      <td className="px-3 py-2">
                        <span className="rounded-full border px-2 py-0.5 text-xs">
                          {formatRefTypeLabel(journal.ref_type)}
                        </span>
                      </td>
                      <td className="px-3 py-2 text-right">
                        <span className={journal.amount >= 0 ? 'text-emerald-600' : 'text-destructive'}>
                          {journal.amount >= 0 ? '+' : ''}
                          {formatIskPlain(journal.amount)}
                        </span>
                      </td>
                      <td className="px-3 py-2 text-right text-destructive">
                        {journal.tax !== 0 ? formatIskPlain(journal.tax) : '-'}
                      </td>
                      <td className="px-3 py-2">{journal.solar_system_name}</td>
                      <td className="px-3 py-2">{journal.character_name}</td>
                      <td className="px-3 py-2">{journal.reason}</td>
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
