import { useEffect, useMemo, useState } from 'react'
import { fetchMyCharacters } from '@/api/auth'
import { fetchInfoShips } from '@/api/eve-info'
import { useI18n } from '@/i18n'

export function InfoShipsPage() {
  const { t } = useI18n()
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [characters, setCharacters] = useState<Api.Auth.EveCharacter[]>([])
  const [selectedCharacterId, setSelectedCharacterId] = useState<number | null>(null)
  const [selectedGroup, setSelectedGroup] = useState('')
  const [ships, setShips] = useState<Api.EveInfo.ShipResponse | null>(null)

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
          setError(t('infoShips.loadCharactersFailed'))
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

    const loadShips = async () => {
      setLoading(true)
      setError(null)
      try {
        const data = await fetchInfoShips({ character_id: selectedCharacterId, language: 'en' })
        if (!cancelled) setShips(data)
      } catch {
        if (!cancelled) {
          setError(t('infoShips.loadShipsFailed'))
          setShips(null)
        }
      } finally {
        if (!cancelled) setLoading(false)
      }
    }

    void loadShips()
    return () => {
      cancelled = true
    }
  }, [selectedCharacterId, t])

  const groupOptions = useMemo(() => {
    const groups = new Set((ships?.ships ?? []).map((item) => item.group_name).filter(Boolean))
    return Array.from(groups).sort((a, b) => a.localeCompare(b))
  }, [ships?.ships])

  const visibleShips = useMemo(() => {
    return (ships?.ships ?? []).filter((item) => (selectedGroup ? item.group_name === selectedGroup : true))
  }, [selectedGroup, ships?.ships])

  return (
    <section className="space-y-4">
      <h1 className="text-xl font-semibold">{t('infoShips.title')}</h1>

      <div className="flex flex-wrap items-center gap-3 rounded-lg border bg-card p-4">
        <label className="text-sm text-muted-foreground" htmlFor="ships-character">
          {t('infoShips.selectCharacter')}
        </label>
        <select
          id="ships-character"
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

        <label className="text-sm text-muted-foreground" htmlFor="ships-group">
          {t('infoShips.group')}
        </label>
        <select
          id="ships-group"
          className="rounded border px-2 py-1 text-sm"
          value={selectedGroup}
          onChange={(event) => setSelectedGroup(event.target.value)}
        >
          <option value="">{t('infoShips.allGroups')}</option>
          {groupOptions.map((group) => (
            <option key={group} value={group}>
              {group}
            </option>
          ))}
        </select>
      </div>

      {loading ? <p className="text-sm">{t('infoShips.loading')}</p> : null}
      {error ? <p className="text-sm text-destructive">{error}</p> : null}

      {ships ? (
        <>
          <div className="grid gap-3 sm:grid-cols-2">
            <div className="rounded-lg border bg-card p-4">
              <p className="text-sm text-muted-foreground">{t('infoShips.totalShips')}</p>
              <p className="mt-1 text-2xl font-semibold">{ships.total_ships}</p>
            </div>
            <div className="rounded-lg border bg-card p-4">
              <p className="text-sm text-muted-foreground">{t('infoShips.flyableShips')}</p>
              <p className="mt-1 text-2xl font-semibold">{ships.flyable_ships}</p>
            </div>
          </div>

          {!loading && visibleShips.length === 0 ? <p className="text-sm">{t('infoShips.empty')}</p> : null}

          <div className="overflow-x-auto rounded-lg border bg-card">
            <table className="min-w-full text-sm">
              <thead>
                <tr className="border-b bg-muted/40 text-left">
                  <th className="px-3 py-2">{t('infoShips.columns.typeName')}</th>
                  <th className="px-3 py-2">{t('infoShips.columns.groupName')}</th>
                  <th className="px-3 py-2">{t('infoShips.columns.raceName')}</th>
                  <th className="px-3 py-2">{t('infoShips.columns.status')}</th>
                </tr>
              </thead>
              <tbody>
                {visibleShips.map((ship) => (
                  <tr key={ship.type_id} className="border-b">
                    <td className="px-3 py-2">{ship.type_name}</td>
                    <td className="px-3 py-2">{ship.group_name}</td>
                    <td className="px-3 py-2">{ship.race_name}</td>
                    <td className="px-3 py-2">
                      {ship.can_fly ? (
                        <span className="text-emerald-600">{t('infoShips.status.flyable')}</span>
                      ) : (
                        <span className="text-amber-600">{t('infoShips.status.unavailable')}</span>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </>
      ) : null}
    </section>
  )
}
