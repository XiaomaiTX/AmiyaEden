import { useMemo, useState } from 'react'
import { useEffect } from 'react'
import { fetchInfoFittings } from '@/api/eve-info'
import { useI18n } from '@/i18n'

export function InfoFittingsPage() {
  const { t } = useI18n()
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [data, setData] = useState<Api.EveInfo.FittingsListResponse | null>(null)
  const [selectedRace, setSelectedRace] = useState('')
  const [selectedGroup, setSelectedGroup] = useState('')
  const [keyword, setKeyword] = useState('')
  const [collapsedGroups, setCollapsedGroups] = useState<Record<string, boolean>>({})
  const [selectedFitting, setSelectedFitting] = useState<Api.EveInfo.FittingResponse | null>(null)

  useEffect(() => {
    let cancelled = false
    const loadData = async () => {
      setLoading(true)
      setError(null)
      try {
        const response = await fetchInfoFittings({ language: 'en' })
        if (!cancelled) setData(response)
      } catch {
        if (!cancelled) {
          setError(t('infoFittings.loadFailed'))
          setData(null)
        }
      } finally {
        if (!cancelled) setLoading(false)
      }
    }
    void loadData()
    return () => {
      cancelled = true
    }
  }, [t])

  const raceOptions = useMemo(() => {
    const map = new Map<string, string>()
    for (const item of data?.fittings ?? []) {
      map.set(String(item.race_id), item.race_name || `Race ${item.race_id}`)
    }
    return Array.from(map.entries()).map(([id, name]) => ({ id, name }))
  }, [data?.fittings])

  const groupOptions = useMemo(() => {
    return Array.from(new Set((data?.fittings ?? []).map((item) => item.group_name || 'Unknown'))).sort((a, b) =>
      a.localeCompare(b)
    )
  }, [data?.fittings])

  const filtered = useMemo(() => {
    const lower = keyword.trim().toLowerCase()
    return (data?.fittings ?? [])
      .filter((item) => (selectedRace ? String(item.race_id) === selectedRace : true))
      .filter((item) => (selectedGroup ? (item.group_name || 'Unknown') === selectedGroup : true))
      .filter((item) => {
        if (!lower) return true
        return [item.name, item.ship_name, item.group_name].some((value) => (value || '').toLowerCase().includes(lower))
      })
  }, [data?.fittings, keyword, selectedGroup, selectedRace])

  const grouped = useMemo(() => {
    const map = new Map<string, Api.EveInfo.FittingResponse[]>()
    for (const item of filtered) {
      const group = item.group_name || 'Unknown'
      if (!map.has(group)) map.set(group, [])
      map.get(group)?.push(item)
    }
    return Array.from(map.entries()).sort((a, b) => a[0].localeCompare(b[0]))
  }, [filtered])

  return (
    <section className="space-y-4">
      <h1 className="text-xl font-semibold">{t('infoFittings.title')}</h1>
      <div className="flex flex-wrap items-center gap-2 rounded-lg border bg-card p-4">
        <select className="rounded border px-2 py-1 text-sm" value={selectedRace} onChange={(e) => setSelectedRace(e.target.value)}>
          <option value="">{t('infoFittings.allRaces')}</option>
          {raceOptions.map((item) => (
            <option key={item.id} value={item.id}>
              {item.name}
            </option>
          ))}
        </select>
        <select className="rounded border px-2 py-1 text-sm" value={selectedGroup} onChange={(e) => setSelectedGroup(e.target.value)}>
          <option value="">{t('infoFittings.allGroups')}</option>
          {groupOptions.map((group) => (
            <option key={group} value={group}>
              {group}
            </option>
          ))}
        </select>
        <input
          className="rounded border px-2 py-1 text-sm"
          value={keyword}
          onChange={(e) => setKeyword(e.target.value)}
          placeholder={t('infoFittings.search')}
        />
      </div>
      {loading ? <p className="text-sm">{t('infoFittings.loading')}</p> : null}
      {error ? <p className="text-sm text-destructive">{error}</p> : null}
      {!loading && grouped.length === 0 ? <p className="text-sm">{t('infoFittings.empty')}</p> : null}

      <div className="space-y-3">
        {grouped.map(([groupName, items]) => {
          const collapsed = collapsedGroups[groupName] ?? false
          return (
            <div key={groupName} className="rounded-lg border bg-card">
              <button
                type="button"
                className="flex w-full items-center justify-between px-3 py-2 text-left text-sm font-medium"
                onClick={() => setCollapsedGroups((prev) => ({ ...prev, [groupName]: !collapsed }))}
              >
                <span>{groupName}</span>
                <span>{collapsed ? '+' : '-'}</span>
              </button>
              {!collapsed ? (
                <div className="grid gap-2 px-3 pb-3 sm:grid-cols-2 lg:grid-cols-4">
                  {items.map((fit) => (
                    <button
                      type="button"
                      key={`${fit.fitting_id}-${fit.character_id}`}
                      className="rounded border p-2 text-left text-sm hover:bg-muted/40"
                      onClick={() => setSelectedFitting(fit)}
                    >
                      <div className="font-medium">{fit.name}</div>
                      <div className="text-xs text-muted-foreground">{fit.ship_name}</div>
                    </button>
                  ))}
                </div>
              ) : null}
            </div>
          )
        })}
      </div>

      {selectedFitting ? (
        <div className="rounded-lg border bg-card p-4">
          <div className="mb-2 flex items-center justify-between">
            <h2 className="text-base font-semibold">{t('infoFittings.detail')}</h2>
            <button type="button" className="rounded border px-2 py-1 text-xs" onClick={() => setSelectedFitting(null)}>
              {t('common.cancel')}
            </button>
          </div>
          <p className="text-sm font-medium">{selectedFitting.name}</p>
          <p className="text-xs text-muted-foreground">{selectedFitting.ship_name}</p>
          <div className="mt-3 space-y-2">
            {selectedFitting.slots.map((slot) => (
              <div key={slot.flag_name} className="rounded border p-2">
                <p className="text-xs font-medium">{slot.flag_text || slot.flag_name}</p>
                <ul className="mt-1 space-y-1">
                  {slot.items.map((item, idx) => (
                    <li key={`${item.type_id}-${idx}`} className="text-xs">
                      {item.type_name} {item.quantity > 1 ? `x${item.quantity}` : ''}
                    </li>
                  ))}
                </ul>
              </div>
            ))}
          </div>
        </div>
      ) : null}
    </section>
  )
}
