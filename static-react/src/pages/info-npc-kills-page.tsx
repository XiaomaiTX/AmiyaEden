import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { MultiSelect } from '@/components/ui/multi-select'
import {
  Table,
  TableHeader,
  TableBody,
  TableRow,
  TableHead,
  TableCell,
} from '@/components/ui/table'
import { useEffect, useState } from 'react'
import { fetchMyCharacters } from '@/api/auth'
import { fetchNpcKills, fetchNpcKillsAll } from '@/api/npc-kill'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { formatIskPlain } from '@/lib/isk'
import { useI18n } from '@/i18n'
import type { EveCharacter } from '@/types/api/auth'
import type { NpcKillResponse } from '@/types/api/npc-kill'

type DateRangeState = {
  startDate: string
  endDate: string
}

type FilterState = {
  refTypes: string[]
  solarSystemIds: string
  characterIds: string
  userIds: string
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
  const characterIds = parseIdList(filters.characterIds)
  const userIds = parseIdList(filters.userIds)
  const minAmount = filters.minAmount === '' ? undefined : Number(filters.minAmount)
  const maxAmount = filters.maxAmount === '' ? undefined : Number(filters.maxAmount)

  return {
    ref_types: filters.refTypes.length > 0 ? filters.refTypes : undefined,
    solar_system_ids: solarSystemIds.length > 0 ? solarSystemIds : undefined,
    character_ids: characterIds.length > 0 ? characterIds : undefined,
    user_ids: userIds.length > 0 ? userIds : undefined,
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

export function InfoNpcKillsPage() {
  const { t } = useI18n()
  const [characterLoading, setCharacterLoading] = useState(true)
  const [reportLoading, setReportLoading] = useState(true)
  const [characterError, setCharacterError] = useState<string | null>(null)
  const [reportError, setReportError] = useState<string | null>(null)
  const [characters, setCharacters] = useState<EveCharacter[]>([])
  const [selectedCharacterId, setSelectedCharacterId] = useState(0)
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
    characterIds: '',
    userIds: '',
    minAmount: '',
    maxAmount: '',
  })
  const [appliedFilters, setAppliedFilters] = useState<FilterState>({
    refTypes: [],
    solarSystemIds: '',
    characterIds: '',
    userIds: '',
    minAmount: '',
    maxAmount: '',
  })
  const [reportData, setReportData] = useState<NpcKillResponse | null>(null)
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
          ...buildNpcKillFilterPayload(appliedFilters),
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
  }, [appliedDateRange, appliedFilters, selectedCharacterId, t])

  const loading = characterLoading || reportLoading

  const handleSearch = () => {
    setAppliedDateRange({ ...draftDateRange })
    setAppliedFilters({ ...draftFilters })
  }

  const handleReset = () => {
    const emptyRange = { startDate: '', endDate: '' }
    const emptyFilters = {
      refTypes: [],
      solarSystemIds: '',
      characterIds: '',
      userIds: '',
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
            <h1 className="text-xl font-semibold">{t('nav.info.npcKills')}</h1>
            <p className="mt-1 text-sm text-muted-foreground">{t('nav.group.info')}</p>
          </div>

          <div className="flex flex-wrap items-end gap-3">
            <label className="space-y-1">
              <span className="text-sm text-muted-foreground">{t('npcKill.selectCharacter')}</span>
              <Select
                selectedKey={String(selectedCharacterId ?? '')}
                onSelectionChange={(key) => ((value) => setSelectedCharacterId(Number(value)))(String(key))}
              >
                <SelectTrigger className="h-10 rounded-md border border-input bg-background px-3 text-sm">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem id={String(0)}>{t('npcKill.allCharacters')}</SelectItem>
                  {characters.map((character) => (
                    <SelectItem
                      key={character.character_id}
                      id={String(character.character_id ?? '')}
                    >
                      {character.character_name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
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

          <div className="flex flex-wrap items-end gap-3">
            <label className="space-y-1">
              <span className="text-sm text-muted-foreground">{t('npcKill.filters.refTypes')}</span>
              <MultiSelect
                className="min-w-48"
                placeholder={t('npcKill.filters.refTypes')}
                value={draftFilters.refTypes}
                onValueChange={(value) =>
                  setDraftFilters((current) => ({ ...current, refTypes: value }))
                }
                options={refTypeOptions.map((refType) => ({
                  value: refType,
                  label: formatRefTypeLabel(refType),
                }))}
              />
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
      {characterError ? <p className="text-sm text-destructive">{characterError}</p> : null}
      {reportError ? <p className="text-sm text-destructive">{reportError}</p> : null}
      {!loading && !reportError && !reportData ? (
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

          <section className="overflow-hidden rounded-lg border bg-card">
            <div className="border-b px-4 py-3 text-sm font-medium">{t('npcKill.byNpc')}</div>
            <div className="overflow-x-auto">
              <Table className="min-w-full text-sm">
                <TableHeader>
                  <TableRow className="border-b bg-muted/40 text-left">
                    <TableHead className="px-3 py-2">#</TableHead>
                    <TableHead className="px-3 py-2">{t('npcKill.npcName')}</TableHead>
                    <TableHead className="px-3 py-2 text-right">{t('npcKill.npcCount')}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {reportData.by_npc.map((item, index) => (
                    <TableRow key={item.npc_id} className="border-b">
                      <TableCell className="px-3 py-2 text-muted-foreground">{index + 1}</TableCell>
                      <TableCell className="px-3 py-2">{item.npc_name}</TableCell>
                      <TableCell className="px-3 py-2 text-right">{item.count}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          </section>

          <div className="grid gap-4 lg:grid-cols-2">
            <section className="overflow-hidden rounded-lg border bg-card">
              <div className="border-b px-4 py-3 text-sm font-medium">{t('npcKill.bySystem')}</div>
              <div className="overflow-x-auto">
                <Table className="min-w-full text-sm">
                  <TableHeader>
                    <TableRow className="border-b bg-muted/40 text-left">
                      <TableHead className="px-3 py-2">#</TableHead>
                      <TableHead className="px-3 py-2">{t('npcKill.solarSystem')}</TableHead>
                      <TableHead className="px-3 py-2 text-right">
                        {t('npcKill.systemCount')}
                      </TableHead>
                      <TableHead className="px-3 py-2 text-right">
                        {t('npcKill.systemAmount')}
                      </TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {reportData.by_system.map((system, index) => (
                      <TableRow key={system.solar_system_id} className="border-b">
                        <TableCell className="px-3 py-2 text-muted-foreground">
                          {index + 1}
                        </TableCell>
                        <TableCell className="px-3 py-2">{system.solar_system_name}</TableCell>
                        <TableCell className="px-3 py-2 text-right">{system.count}</TableCell>
                        <TableCell className="px-3 py-2 text-right text-emerald-600">
                          {formatIskPlain(system.amount)}
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            </section>

            <section className="overflow-hidden rounded-lg border bg-card">
              <div className="border-b px-4 py-3 text-sm font-medium">{t('npcKill.trend')}</div>
              <div className="overflow-x-auto">
                <Table className="min-w-full text-sm">
                  <TableHeader>
                    <TableRow className="border-b bg-muted/40 text-left">
                      <TableHead className="px-3 py-2">{t('npcKill.trendDate')}</TableHead>
                      <TableHead className="px-3 py-2 text-right">
                        {t('npcKill.trendAmount')}
                      </TableHead>
                      <TableHead className="px-3 py-2 text-right">
                        {t('npcKill.trendCount')}
                      </TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {reportData.trend.map((item) => (
                      <TableRow key={item.date} className="border-b">
                        <TableCell className="px-3 py-2">{item.date}</TableCell>
                        <TableCell className="px-3 py-2 text-right text-emerald-600">
                          {formatIskPlain(item.amount)}
                        </TableCell>
                        <TableCell className="px-3 py-2 text-right">{item.count}</TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            </section>
          </div>

          <section className="overflow-hidden rounded-lg border bg-card">
            <div className="border-b px-4 py-3 text-sm font-medium">{t('npcKill.journals')}</div>
            <div className="overflow-x-auto">
              <Table className="min-w-full text-sm">
                <TableHeader>
                  <TableRow className="border-b bg-muted/40 text-left">
                    <TableHead className="px-3 py-2">{t('npcKill.journalDate')}</TableHead>
                    <TableHead className="px-3 py-2">{t('npcKill.journalRefType')}</TableHead>
                    <TableHead className="px-3 py-2 text-right">
                      {t('npcKill.journalAmount')}
                    </TableHead>
                    <TableHead className="px-3 py-2 text-right">
                      {t('npcKill.journalTax')}
                    </TableHead>
                    <TableHead className="px-3 py-2">{t('npcKill.journalSystem')}</TableHead>
                    <TableHead className="px-3 py-2">{t('npcKill.characterName')}</TableHead>
                    <TableHead className="px-3 py-2">{t('npcKill.journalReason')}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {reportData.journals.map((journal) => (
                    <TableRow key={journal.id} className="border-b">
                      <TableCell className="px-3 py-2">{journal.date}</TableCell>
                      <TableCell className="px-3 py-2">
                        <span className="rounded-full border px-2 py-0.5 text-xs">
                          {formatRefTypeLabel(journal.ref_type)}
                        </span>
                      </TableCell>
                      <TableCell className="px-3 py-2 text-right">
                        <span
                          className={journal.amount >= 0 ? 'text-emerald-600' : 'text-destructive'}
                        >
                          {journal.amount >= 0 ? '+' : ''}
                          {formatIskPlain(journal.amount)}
                        </span>
                      </TableCell>
                      <TableCell className="px-3 py-2 text-right text-destructive">
                        {journal.tax !== 0 ? formatIskPlain(journal.tax) : '-'}
                      </TableCell>
                      <TableCell className="px-3 py-2">{journal.solar_system_name}</TableCell>
                      <TableCell className="px-3 py-2">{journal.character_name}</TableCell>
                      <TableCell className="px-3 py-2">{journal.reason}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          </section>
        </>
      ) : null}
    </section>
  )
}
