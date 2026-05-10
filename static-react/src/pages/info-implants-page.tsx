import { useEffect, useState } from 'react'
import { fetchMyCharacters } from '@/api/auth'
import { fetchInfoImplants } from '@/api/eve-info'
import { useI18n } from '@/i18n'
import type { EveCharacter } from '@/types/api/auth'
import type { ImplantsResponse } from '@/types/api/eve-info'

function formatDateTime(value: string | null) {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString()
}

export function InfoImplantsPage() {
  const { t } = useI18n()
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [characters, setCharacters] = useState<EveCharacter[]>([])
  const [selectedCharacterId, setSelectedCharacterId] = useState<number | null>(null)
  const [implants, setImplants] = useState<ImplantsResponse | null>(null)
  const [isFatigueExpired, setIsFatigueExpired] = useState(true)

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
          setError(t('infoImplants.loadCharactersFailed'))
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
    const loadImplants = async () => {
      setLoading(true)
      setError(null)
      try {
        const data = await fetchInfoImplants({ character_id: selectedCharacterId, language: 'en' })
        if (!cancelled) {
          setImplants(data)
          const expired = !data.jump_fatigue_expire || new Date(data.jump_fatigue_expire).getTime() <= Date.now()
          setIsFatigueExpired(expired)
        }
      } catch {
        if (!cancelled) {
          setError(t('infoImplants.loadImplantsFailed'))
          setImplants(null)
          setIsFatigueExpired(true)
        }
      } finally {
        if (!cancelled) setLoading(false)
      }
    }
    void loadImplants()
    return () => {
      cancelled = true
    }
  }, [selectedCharacterId, t])

  return (
    <section className="space-y-4">
      <h1 className="text-xl font-semibold">{t('infoImplants.title')}</h1>

      <div className="flex flex-wrap items-center gap-3 rounded-lg border bg-card p-4">
        <label className="text-sm text-muted-foreground" htmlFor="implants-character">
          {t('infoImplants.selectCharacter')}
        </label>
        <select
          id="implants-character"
          className="rounded border px-2 py-1 text-sm"
          value={selectedCharacterId ?? ''}
          onChange={(event) => setSelectedCharacterId(Number(event.target.value))}
        >
          {characters.map((character) => (
            <option key={character.character_id} value={character.character_id}>
              {character.character_name}
            </option>
          ))}
        </select>
      </div>

      {loading ? <p className="text-sm">{t('infoImplants.loading')}</p> : null}
      {error ? <p className="text-sm text-destructive">{error}</p> : null}

      {implants ? (
        <>
          <div className="grid gap-3 rounded-lg border bg-card p-4 sm:grid-cols-3">
            <div>
              <p className="text-xs text-muted-foreground">{t('infoImplants.jumpFatigue')}</p>
              <p className="mt-1 text-sm font-medium">
                {isFatigueExpired ? t('infoImplants.fatigueReady') : formatDateTime(implants.jump_fatigue_expire)}
              </p>
            </div>
            <div>
              <p className="text-xs text-muted-foreground">{t('infoImplants.lastJumpDate')}</p>
              <p className="mt-1 text-sm font-medium">{formatDateTime(implants.last_jump_date)}</p>
            </div>
            <div>
              <p className="text-xs text-muted-foreground">{t('infoImplants.lastCloneJump')}</p>
              <p className="mt-1 text-sm font-medium">{formatDateTime(implants.last_clone_jump_date)}</p>
            </div>
          </div>

          <div className="rounded-lg border bg-card p-4">
            <h2 className="text-base font-semibold">{t('infoImplants.activeImplants')}</h2>
            {implants.active_implants.length === 0 ? (
              <p className="mt-2 text-sm text-muted-foreground">{t('infoImplants.noImplants')}</p>
            ) : (
              <ul className="mt-2 grid gap-2 sm:grid-cols-2">
                {implants.active_implants.map((item) => (
                  <li key={item.implant_id} className="rounded border px-3 py-2 text-sm">
                    {item.implant_name || `Type ${item.implant_id}`}
                  </li>
                ))}
              </ul>
            )}
          </div>

          <div className="rounded-lg border bg-card p-4">
            <h2 className="text-base font-semibold">
              {t('infoImplants.jumpClones')} ({implants.jump_clones.length})
            </h2>
            {implants.jump_clones.length === 0 ? (
              <p className="mt-2 text-sm text-muted-foreground">{t('infoImplants.noJumpClones')}</p>
            ) : (
              <div className="mt-3 space-y-3">
                {implants.jump_clones.map((clone) => (
                  <article key={clone.jump_clone_id} className="rounded border p-3">
                    <p className="text-sm font-medium">
                      {clone.location.location_name ||
                        `${clone.location.location_type}-${clone.location.location_id}`}
                    </p>
                    <p className="mt-1 text-xs text-muted-foreground">#{clone.jump_clone_id}</p>
                    {clone.implants.length === 0 ? (
                      <p className="mt-2 text-sm text-muted-foreground">{t('infoImplants.noImplants')}</p>
                    ) : (
                      <ul className="mt-2 grid gap-2 sm:grid-cols-2">
                        {clone.implants.map((item) => (
                          <li key={item.implant_id} className="rounded border px-3 py-2 text-sm">
                            {item.implant_name || `Type ${item.implant_id}`}
                          </li>
                        ))}
                      </ul>
                    )}
                  </article>
                ))}
              </div>
            )}
          </div>
        </>
      ) : null}
    </section>
  )
}
