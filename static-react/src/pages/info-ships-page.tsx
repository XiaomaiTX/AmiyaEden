import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Table,
  TableHeader,
  TableBody,
  TableRow,
  TableHead,
  TableCell,
} from '@/components/ui/table'
import { useEffect, useMemo, useState } from 'react'
import { fetchMyCharacters } from '@/api/auth'
import { fetchInfoShips } from '@/api/eve-info'
import { useI18n } from '@/i18n'
import type { EveCharacter } from '@/types/api/auth'
import type { ShipResponse } from '@/types/api/eve-info'

export function InfoShipsPage() {
  const { t } = useI18n()
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [characters, setCharacters] = useState<EveCharacter[]>([])
  const [selectedCharacterId, setSelectedCharacterId] = useState<number | null>(null)
  const [selectedGroup, setSelectedGroup] = useState('')
  const [ships, setShips] = useState<ShipResponse | null>(null)

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
    return (ships?.ships ?? []).filter((item) =>
      selectedGroup ? item.group_name === selectedGroup : true
    )
  }, [selectedGroup, ships?.ships])

  return (
    <section className="space-y-4">
      <h1 className="text-xl font-semibold">{t('infoShips.title')}</h1>

      <div className="flex flex-wrap items-center gap-3 rounded-lg border bg-card p-4">
        <label className="text-sm text-muted-foreground" htmlFor="ships-character">
          {t('infoShips.selectCharacter')}
        </label>
        <Select
          selectedKey={String(selectedCharacterId ?? '')}
          onSelectionChange={(key) => ((value) => setSelectedCharacterId(Number(value)))(String(key))}
        >
          <SelectTrigger id="ships-character" className="rounded border px-2 py-1 text-sm">
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

        <label className="text-sm text-muted-foreground" htmlFor="ships-group">
          {t('infoShips.group')}
        </label>
        <Select
          selectedKey={String(selectedGroup ?? '')}
          onSelectionChange={(key) => ((value) => setSelectedGroup(value))(String(key))}
        >
          <SelectTrigger id="ships-group" className="rounded border px-2 py-1 text-sm">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem id="">{t('infoShips.allGroups')}</SelectItem>
            {groupOptions.map((group) => (
              <SelectItem key={group} id={String(group ?? '')}>
                {group}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
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

          {!loading && visibleShips.length === 0 ? (
            <p className="text-sm">{t('infoShips.empty')}</p>
          ) : null}

          <div className="overflow-x-auto rounded-lg border bg-card">
            <Table className="min-w-full text-sm">
              <TableHeader>
                <TableRow className="border-b bg-muted/40 text-left">
                  <TableHead className="px-3 py-2">{t('infoShips.columns.typeName')}</TableHead>
                  <TableHead className="px-3 py-2">{t('infoShips.columns.groupName')}</TableHead>
                  <TableHead className="px-3 py-2">{t('infoShips.columns.raceName')}</TableHead>
                  <TableHead className="px-3 py-2">{t('infoShips.columns.status')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {visibleShips.map((ship) => (
                  <TableRow key={ship.type_id} className="border-b">
                    <TableCell className="px-3 py-2">{ship.type_name}</TableCell>
                    <TableCell className="px-3 py-2">{ship.group_name}</TableCell>
                    <TableCell className="px-3 py-2">{ship.race_name}</TableCell>
                    <TableCell className="px-3 py-2">
                      {ship.can_fly ? (
                        <span className="text-emerald-600">{t('infoShips.status.flyable')}</span>
                      ) : (
                        <span className="text-amber-600">{t('infoShips.status.unavailable')}</span>
                      )}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        </>
      ) : null}
    </section>
  )
}
