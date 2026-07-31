import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { useEffect, useMemo, useState } from 'react'
import { fetchMyCharacters } from '@/api/auth'
import { fetchInfoSkills, runMyCharacterESIRefresh } from '@/api/eve-info'
import { useI18n } from '@/i18n'
import type { EveCharacter } from '@/types/api/auth'
import type { SkillResponse } from '@/types/api/eve-info'

export function InfoSkillPage() {
  const { t } = useI18n()
  const [loading, setLoading] = useState(true)
  const [esiRefreshing, setEsiRefreshing] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [characters, setCharacters] = useState<EveCharacter[]>([])
  const [selectedCharacterId, setSelectedCharacterId] = useState<number | null>(null)
  const [reloadVersion, setReloadVersion] = useState(0)
  const [keyword, setKeyword] = useState('')
  const [selectedGroup, setSelectedGroup] = useState('')
  const [skillData, setSkillData] = useState<SkillResponse | null>(null)

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
        const data = await fetchInfoSkills({ character_id: selectedCharacterId, language: 'en' })
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
  }, [reloadVersion, selectedCharacterId, t])

  const groups = useMemo(() => {
    const map = new Map<string, number>()
    for (const skill of skillData?.skills ?? []) {
      const key = skill.group_name || 'Unknown'
      map.set(key, (map.get(key) ?? 0) + 1)
    }
    return Array.from(map.entries())
      .map(([name, count]) => ({ name, count }))
      .sort((a, b) => a.name.localeCompare(b.name))
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

  const queue = useMemo(
    () => [...(skillData?.skill_queue ?? [])].sort((a, b) => a.queue_position - b.queue_position),
    [skillData?.skill_queue]
  )

  const refreshESI = async () => {
    if (!selectedCharacterId) return
    setEsiRefreshing(true)
    try {
      await runMyCharacterESIRefresh({
        task_name: 'character_skill',
        character_id: selectedCharacterId,
      })
      setError(null)
    } catch {
      setError(t('infoSkill.esiRefreshFailed'))
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
          onSelectionChange={(key) => ((value) => setSelectedCharacterId(Number(value)))(String(key))}
        >
          <SelectTrigger id="skill-character" className="h-8">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {characters.map((character) => (
              <SelectItem key={character.character_id} id={String(character.character_id ?? '')}>
                {character.character_name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>

        <Button
          className="rounded border px-2 py-1 text-sm"
          onClick={() => setReloadVersion((v) => v + 1)}
        >
          {t('common.refresh')}
        </Button>
        <Button
          className="rounded border px-2 py-1 text-sm"
          onClick={() => void refreshESI()}
          isDisabled={esiRefreshing}
        >
          {esiRefreshing ? t('infoSkill.esiRefreshing') : t('infoSkill.esiRefresh')}
        </Button>
      </div>

      <div className="grid gap-4 lg:grid-cols-[minmax(0,1fr)_360px]">
        <div className="space-y-3 rounded-lg border bg-card p-4">
          <div className="flex flex-wrap items-center gap-2">
            <Select
              selectedKey={String(selectedGroup ?? '')}
              onSelectionChange={(key) => ((value) => setSelectedGroup(value))(String(key))}
            >
              <SelectTrigger className="h-8">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem id="">{t('infoSkill.allGroups')}</SelectItem>
                {groups.map((group) => (
                  <SelectItem key={group.name} id={String(group.name ?? '')}>
                    {group.name} ({group.count})
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <Input
              className="rounded border px-2 py-1 text-sm"
              value={keyword}
              onChange={(event) => setKeyword(event.target.value)}
              placeholder={t('infoSkill.searchPlaceholder')}
            />
          </div>
          {loading ? <p className="text-sm">{t('infoSkill.loading')}</p> : null}
          {error ? <p className="text-sm text-destructive">{error}</p> : null}
          {!loading && filteredSkills.length === 0 ? (
            <p className="text-sm">{t('infoSkill.empty')}</p>
          ) : null}
          <ul className="space-y-2">
            {filteredSkills.map((skill) => (
              <li
                key={skill.skill_id}
                className="flex items-center justify-between rounded border px-3 py-2 text-sm"
              >
                <span>
                  {skill.skill_name} ({skill.group_name})
                </span>
                <span className="text-muted-foreground">
                  {t('info.skillLevel', { level: `${skill.active_level}/${skill.trained_level}` })}
                </span>
              </li>
            ))}
          </ul>
        </div>

        <div className="space-y-3 rounded-lg border bg-card p-4">
          <h2 className="text-base font-semibold">{t('infoSkill.queueTitle')}</h2>
          <p className="text-sm text-muted-foreground">
            {t('infoSkill.queueCount')}: {queue.length}
          </p>
          {queue.length === 0 ? <p className="text-sm">{t('infoSkill.queueEmpty')}</p> : null}
          <ul className="space-y-2">
            {queue.map((item) => (
              <li key={item.queue_position} className="rounded border px-3 py-2 text-sm">
                {item.skill_name} {t('info.skillLevel', { level: item.finished_level })}
              </li>
            ))}
          </ul>
        </div>
      </div>
    </section>
  )
}
