import { BookOpen, Check, ChevronsRight } from 'lucide-react'
import { useEffect, useMemo, useState } from 'react'
import { fetchMyCharacters } from '@/api/auth'
import { fetchInfoSkills, runMyCharacterESIRefresh } from '@/api/eve-info'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { confirmAction, notifyError, notifySuccess } from '@/feedback'
import { useI18n } from '@/i18n'
import { buildEveCharacterPortraitUrl } from '@/lib/eve-image'
import type { EveCharacter } from '@/types/api/auth'
import type { SkillItem, SkillQueueItem, SkillResponse } from '@/types/api/eve-info'

const numberFormatter = new Intl.NumberFormat('en-US')

function formatNumber(value: number) {
  return numberFormatter.format(value)
}

function romanLevel(level: number) {
  const numerals = ['', 'I', 'II', 'III', 'IV', 'V']
  return numerals[level] || String(level)
}

function calcTimeProgress(item: SkillQueueItem, nowSeconds: number) {
  if (!item.start_date || !item.finish_date) return 0
  const total = item.finish_date - item.start_date
  if (total <= 0) return 100
  const elapsed = nowSeconds - item.start_date
  return Math.min(100, Math.max(0, Math.round((elapsed / total) * 100)))
}

function getErrorMessage(error: unknown, fallback: string) {
  return error instanceof Error && error.message ? error.message : fallback
}

interface SkillGroup {
  groupName: string
  count: number
  skills: SkillItem[]
  progress: number
}

function LevelBars({
  size,
  mode,
  activeLevel,
  trainedLevel,
  finishedLevel,
}: {
  size: 'md' | 'sm'
} & (
  | { mode: 'trained'; activeLevel: number; trainedLevel: number; finishedLevel?: never }
  | { mode: 'queue'; finishedLevel: number; activeLevel?: never; trainedLevel?: never }
)) {
  const pipClass = size === 'md' ? 'size-2.5' : 'size-2'
  return (
    <div className="flex shrink-0 gap-0.5">
      {[1, 2, 3, 4, 5].map((i) => {
        let fillClass = 'bg-muted'
        if (mode === 'trained') {
          if (i <= activeLevel) {
            fillClass = 'bg-primary'
          } else if (i === activeLevel + 1 && trainedLevel > activeLevel) {
            fillClass = 'bg-primary/50'
          }
        } else if (i < finishedLevel) {
          fillClass = 'bg-primary'
        }
        return <span key={i} className={`inline-block ${pipClass} ${fillClass}`} />
      })}
    </div>
  )
}

export function InfoSkillPage() {
  const { t, locale } = useI18n()
  const language = locale.startsWith('zh') ? 'zh' : 'en'
  const [loading, setLoading] = useState(true)
  const [esiRefreshing, setEsiRefreshing] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [characters, setCharacters] = useState<EveCharacter[]>([])
  const [selectedCharacterId, setSelectedCharacterId] = useState<number | null>(null)
  const [reloadVersion, setReloadVersion] = useState(0)
  const [keyword, setKeyword] = useState('')
  const [selectedGroup, setSelectedGroup] = useState('')
  const [skillData, setSkillData] = useState<SkillResponse | null>(null)

  const [nowSeconds] = useState(() => Math.floor(Date.now() / 1000))

  useEffect(() => {
    let cancelled = false
    const loadCharacters = async () => {
      setLoading(true)
      setError(null)
      try {
        const list = await fetchMyCharacters()
        if (cancelled) return
        setCharacters(list)
        if (list.length > 0) {
          setSelectedCharacterId(list[0].character_id)
        } else {
          setLoading(false)
        }
      } catch {
        if (!cancelled) {
          setError(t('infoSkill.loadCharactersFailed'))
          setLoading(false)
        }
      }
    }
    void loadCharacters()
    return () => {
      cancelled = true
    }
  }, [t])

  useEffect(() => {
    if (!selectedCharacterId) return
    let cancelled = false
    const loadSkills = async () => {
      setLoading(true)
      setError(null)
      try {
        const data = await fetchInfoSkills({ character_id: selectedCharacterId, language })
        if (!cancelled) setSkillData(data)
      } catch {
        if (!cancelled) {
          setError(t('infoSkill.loadSkillsFailed'))
          setSkillData(null)
        }
      } finally {
        if (!cancelled) setLoading(false)
      }
    }
    void loadSkills()
    return () => {
      cancelled = true
    }
  }, [reloadVersion, selectedCharacterId, language, t])

  const formatRemainingTime = (finishDate: number) => {
    if (!finishDate) return ''
    let remaining = finishDate - nowSeconds
    if (remaining <= 0) return t('infoSkill.trainingComplete')
    const days = Math.floor(remaining / 86400)
    remaining %= 86400
    const hours = Math.floor(remaining / 3600)
    remaining %= 3600
    const minutes = Math.floor(remaining / 60)
    const seconds = remaining % 60
    let result = ''
    if (days > 0) result += `${t('infoSkill.durationDays', { count: days })} `
    if (hours > 0 || days > 0) result += t('infoSkill.durationHours', { count: hours })
    if (days === 0) {
      if (minutes > 0) result += ` ${t('infoSkill.durationMinutes', { count: minutes })}`
      if (hours === 0 && minutes < 10) {
        result += ` ${t('infoSkill.durationSeconds', { count: seconds })}`
      }
    }
    return result.trim()
  }

  const skillGroups = useMemo<SkillGroup[]>(() => {
    const map = new Map<string, SkillGroup>()
    for (const skill of skillData?.skills ?? []) {
      const key = skill.group_name || 'Unknown'
      if (!map.has(key)) {
        map.set(key, { groupName: key, count: 0, skills: [], progress: 0 })
      }
      const group = map.get(key)!
      group.count++
      group.skills.push(skill)
    }
    for (const group of map.values()) {
      const totalLevels = group.count * 5
      const trainedLevels = group.skills.reduce((sum, skill) => sum + (skill.active_level ?? 0), 0)
      group.progress = totalLevels > 0 ? Math.round((trainedLevels / totalLevels) * 100) : 0
    }
    return Array.from(map.values()).sort((a, b) => a.groupName.localeCompare(b.groupName))
  }, [skillData?.skills])

  const filteredSkills = useMemo(() => {
    const normalized = keyword.trim().toLowerCase()
    return (skillData?.skills ?? [])
      .filter((skill) => !selectedGroup || skill.group_name === selectedGroup)
      .filter((skill) => {
        if (!normalized) return true
        return (
          skill.skill_name.toLowerCase().includes(normalized) ||
          skill.group_name.toLowerCase().includes(normalized)
        )
      })
      .sort(
        (a, b) =>
          a.group_name.localeCompare(b.group_name) || a.skill_name.localeCompare(b.skill_name)
      )
  }, [keyword, selectedGroup, skillData?.skills])

  const queueSkillMap = useMemo(() => {
    const map = new Map<number, SkillQueueItem>()
    for (const item of skillData?.skill_queue ?? []) {
      if (!map.has(item.skill_id)) map.set(item.skill_id, item)
    }
    return map
  }, [skillData?.skill_queue])

  const currentTraining = useMemo(() => {
    return (skillData?.skill_queue ?? []).find((item) => item.finish_date > nowSeconds) ?? null
  }, [skillData?.skill_queue, nowSeconds])

  const queueWithoutFirst = useMemo(() => {
    if (!currentTraining) return []
    return (skillData?.skill_queue ?? []).filter(
      (item) => item.queue_position > currentTraining.queue_position
    )
  }, [currentTraining, skillData?.skill_queue])

  const queue = skillData?.skill_queue ?? []
  const lastQueueItem = queue.length > 0 ? queue[queue.length - 1] : null
  const totalQueueTime = lastQueueItem?.finish_date ? formatRemainingTime(lastQueueItem.finish_date) : '-'

  const totalQueueSP = useMemo(() => {
    return (skillData?.skill_queue ?? []).reduce(
      (sum, item) => sum + Math.max(0, item.level_end_sp - item.training_start_sp),
      0
    )
  }, [skillData?.skill_queue])

  const isInQueue = (skillId: number) => queueSkillMap.has(skillId)
  const getQueueRemainingTime = (skillId: number) => {
    const item = queueSkillMap.get(skillId)
    return item ? formatRemainingTime(item.finish_date) : ''
  }

  const toggleGroup = (name: string) => {
    setSelectedGroup((current) => (current === name ? '' : name))
  }

  const selectedCharacter = characters.find(
    (character) => character.character_id === selectedCharacterId
  )

  const refreshESI = async () => {
    if (!selectedCharacterId || !selectedCharacter) return
    const confirmed = await confirmAction({
      title: t('infoSkill.esiRefreshTitle'),
      message: t('infoSkill.skillESIRefreshConfirm', { name: selectedCharacter.character_name }),
      confirmText: t('infoSkill.esiRefreshConfirmButton'),
      cancelText: t('common.cancel'),
    })
    if (!confirmed) return

    setEsiRefreshing(true)
    try {
      await runMyCharacterESIRefresh({
        task_name: 'character_skill',
        character_id: selectedCharacterId,
      })
      notifySuccess(t('infoSkill.skillESIRefreshSubmitted'))
    } catch (caughtError) {
      const message = getErrorMessage(caughtError, t('infoSkill.esiRefreshSubmitFailed'))
      if (message.includes('403') || message.includes('无权')) {
        notifyError(t('infoSkill.esiRefreshUnauthorized'))
      } else if (message.includes('角色不存在') || message.toLowerCase().includes('not found')) {
        notifyError(t('infoSkill.esiRefreshCharacterNotFound'))
      } else {
        notifyError(message)
      }
    } finally {
      setEsiRefreshing(false)
    }
  }

  return (
    <section className="space-y-4">
      <h1 className="text-xl font-semibold">{t('infoSkill.title')}</h1>

      <div className="flex flex-wrap items-center gap-3 rounded-lg border bg-card p-4">
        <label className="text-sm text-muted-foreground" htmlFor="skill-character">
          {t('infoSkill.selectCharacter')}
        </label>
        <Select
          selectedKey={String(selectedCharacterId ?? '')}
          onSelectionChange={(key) => {
            setSelectedCharacterId(Number(key))
            setSelectedGroup('')
            setKeyword('')
          }}
        >
          <SelectTrigger id="skill-character" className="w-56">
            <SelectValue>{t('infoSkill.selectCharacterPlaceholder')}</SelectValue>
          </SelectTrigger>
          <SelectContent>
            {characters.map((character) => (
              <SelectItem key={character.character_id} id={String(character.character_id ?? '')}>
                <span className="flex items-center gap-2">
                  <img
                    src={buildEveCharacterPortraitUrl(character.character_id, 32)}
                    alt=""
                    className="size-6 rounded-full object-cover"
                  />
                  <span>{character.character_name}</span>
                </span>
              </SelectItem>
            ))}
          </SelectContent>
        </Select>

        <Button variant="outline" onClick={() => setReloadVersion((v) => v + 1)}>
          {t('common.refresh')}
        </Button>
        <Button onClick={() => void refreshESI()} isDisabled={esiRefreshing}>
          {esiRefreshing ? t('infoSkill.esiRefreshing') : t('infoSkill.esiRefresh')}
        </Button>
      </div>

      {error ? <p className="text-sm text-destructive">{error}</p> : null}

      <div className="grid gap-3 lg:grid-cols-[minmax(0,1fr)_380px]">
        <div className="flex flex-col gap-3 overflow-hidden rounded-lg border bg-card p-4">
          <div className="flex items-baseline justify-between border-b pb-2">
            <h2 className="text-base font-semibold">{t('infoSkill.skillList')}</h2>
            <span className="text-sm text-muted-foreground">
              {formatNumber(skillData?.total_sp ?? 0)} {t('infoSkill.totalSPLabel')}
            </span>
          </div>

          <div className="flex flex-wrap items-center gap-2">
            <Select
              selectedKey={String(selectedGroup ?? '')}
              onSelectionChange={(key) => setSelectedGroup(String(key))}
            >
              <SelectTrigger className="w-48">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem id="">{t('infoSkill.allSkills')}</SelectItem>
                {skillGroups.map((group) => (
                  <SelectItem key={group.groupName} id={String(group.groupName ?? '')}>
                    {group.groupName}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <Input
              className="w-48"
              value={keyword}
              onChange={(event) => setKeyword(event.target.value)}
              placeholder={t('infoSkill.searchPlaceholder')}
            />
          </div>

          <div className="grid grid-cols-3 gap-1">
            {skillGroups.map((group) => (
              <button
                key={group.groupName}
                type="button"
                onClick={() => toggleGroup(group.groupName)}
                className={`relative flex items-center justify-between overflow-hidden rounded px-2.5 py-1.5 text-xs transition-colors ${
                  selectedGroup === group.groupName
                    ? 'bg-primary/10 font-medium text-primary'
                    : 'bg-muted text-foreground hover:bg-muted/70'
                }`}
              >
                <span
                  className="absolute inset-0 bg-primary/10"
                  style={{ width: `${group.progress}%` }}
                />
                <span className="relative truncate">{group.groupName}</span>
                <span className="relative shrink-0 pl-1 text-muted-foreground">{group.count}</span>
              </button>
            ))}
          </div>

          {loading ? <p className="text-sm">{t('infoSkill.loading')}</p> : null}
          {!loading && filteredSkills.length === 0 ? (
            <p className="text-sm">{t('infoSkill.empty')}</p>
          ) : null}

          <div className="grid grid-cols-[repeat(auto-fill,minmax(300px,1fr))] content-start gap-0.5 overflow-y-auto">
            {filteredSkills.map((skill) => {
              const queuedRemaining = getQueueRemainingTime(skill.skill_id)
              return (
                <div
                  key={skill.skill_id}
                  className={`flex min-w-0 items-center gap-2 rounded-sm px-2 py-1 text-sm hover:bg-muted/60 ${
                    isInQueue(skill.skill_id) ? 'bg-primary/5' : ''
                  } ${!skill.learned ? 'opacity-50 hover:opacity-80' : ''}`}
                >
                  {!skill.learned ? (
                    <div className="flex w-16 shrink-0 items-center" title={t('infoSkill.skillNotLearned')}>
                      <BookOpen className="size-3.5 text-muted-foreground" />
                    </div>
                  ) : (
                    <LevelBars
                      size="md"
                      mode="trained"
                      activeLevel={skill.active_level}
                      trainedLevel={skill.trained_level}
                    />
                  )}
                  <span className="flex-1 truncate">{skill.skill_name}</span>
                  <span className="shrink-0 text-xs text-muted-foreground">
                    {queuedRemaining ? (
                      <span className="text-primary">{queuedRemaining}</span>
                    ) : skill.active_level >= 5 ? (
                      <Check className="size-3.5 text-primary" />
                    ) : null}
                  </span>
                </div>
              )
            })}
          </div>
        </div>

        <div className="flex flex-col gap-3 overflow-hidden rounded-lg border bg-card p-4">
          <div className="flex items-baseline justify-between border-b pb-2">
            <h2 className="text-base font-semibold">{t('infoSkill.queueTitle')}</h2>
            <span className="text-lg font-semibold">
              {skillData?.skill_queue?.length ?? 0}
              <span className="text-sm font-normal text-muted-foreground">
                {t('infoSkill.queueCapacitySuffix')}
              </span>
            </span>
          </div>

          {currentTraining ? (
            <div className="space-y-1 rounded-md bg-muted p-2.5">
              <div className="flex items-center gap-1.5">
                <ChevronsRight className="size-4 text-primary" />
                <span className="text-sm font-medium">
                  {currentTraining.skill_name} {romanLevel(currentTraining.finished_level)}
                </span>
              </div>
              <span className="text-sm text-muted-foreground">
                {formatRemainingTime(currentTraining.finish_date)}
              </span>
              <div className="h-1.5 w-full overflow-hidden rounded-full bg-background">
                <div
                  className="h-full rounded-full bg-primary"
                  style={{ width: `${calcTimeProgress(currentTraining, nowSeconds)}%` }}
                />
              </div>
            </div>
          ) : null}

          {!loading && queueWithoutFirst.length === 0 && !currentTraining ? (
            <p className="text-sm">{t('infoSkill.queueEmpty')}</p>
          ) : null}

          <div className="flex-1 overflow-y-auto">
            {queueWithoutFirst.map((item) => (
              <div
                key={item.queue_position}
                className="flex items-center gap-2 rounded-sm px-1.5 py-1 text-sm hover:bg-muted/60"
              >
                <LevelBars size="sm" mode="queue" finishedLevel={item.finished_level} />
                <span className="flex-1 truncate">
                  {item.skill_name} {romanLevel(item.finished_level)}
                </span>
                <span className="min-w-[80px] shrink-0 text-right text-xs text-muted-foreground">
                  {formatRemainingTime(item.finish_date)}
                </span>
              </div>
            ))}
          </div>

          {skillData ? (
            <div className="mt-auto space-y-1 border-t pt-3 text-sm">
              <div className="text-right text-muted-foreground">
                <span className="font-medium text-foreground">
                  {formatNumber(skillData.unallocated_sp)}
                </span>
                {t('infoSkill.unallocatedSPSuffix')}
              </div>
              <div className="flex items-baseline justify-between">
                <span className="text-muted-foreground">{t('infoSkill.totalTrainingTime')}</span>
                <span className="text-lg font-semibold">{totalQueueTime}</span>
              </div>
              <div className="text-right text-xs text-muted-foreground">
                {formatNumber(totalQueueSP)}
                {t('infoSkill.queuedSPSuffix')}
              </div>
            </div>
          ) : null}
        </div>
      </div>
    </section>
  )
}
