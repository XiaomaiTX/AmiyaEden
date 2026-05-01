import { useEffect, useMemo, useState } from 'react'
import { fetchInfoAssets } from '@/api/eve-info'
import { useI18n } from '@/i18n'

const BLUEPRINT_CATEGORY_ID = 9

function getIconUrl(item: Api.EveInfo.AssetItemNode) {
  if (item.category_id === BLUEPRINT_CATEGORY_ID) {
    const suffix = item.is_blueprint_copy ? 'bpc' : 'bp'
    return `https://images.evetech.net/types/${item.type_id}/${suffix}?size=32`
  }
  return `https://images.evetech.net/types/${item.type_id}/icon?size=32`
}

function countItems(items: Api.EveInfo.AssetItemNode[]) {
  let count = items.length
  for (const item of items) {
    if (item.children?.length) {
      count += countItems(item.children)
    }
  }
  return count
}

function matchSearch(item: Api.EveInfo.AssetItemNode, keyword: string): boolean {
  const lower = keyword.toLowerCase()
  if (item.type_name.toLowerCase().includes(lower)) return true
  if (item.group_name.toLowerCase().includes(lower)) return true
  if (item.asset_name?.toLowerCase().includes(lower)) return true
  return item.children?.some((child) => matchSearch(child, keyword)) ?? false
}

export function InfoAssetsPage() {
  const { t } = useI18n()
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [data, setData] = useState<Api.EveInfo.AssetsResponse | null>(null)
  const [keyword, setKeyword] = useState('')
  const [collapsedLocations, setCollapsedLocations] = useState<Record<number, boolean>>({})
  const [expandedItems, setExpandedItems] = useState<Record<number, boolean>>({})

  useEffect(() => {
    let cancelled = false
    const loadData = async () => {
      setLoading(true)
      setError(null)
      try {
        const response = await fetchInfoAssets({ language: 'en' })
        if (!cancelled) setData(response)
      } catch {
        if (!cancelled) {
          setError(t('infoAssets.loadFailed'))
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

  const filteredLocations = useMemo(() => {
    const locations = data?.locations ?? []
    const lower = keyword.trim().toLowerCase()
    if (!lower) return locations

    return locations
      .map((location) => {
        if (location.location_name.toLowerCase().includes(lower)) {
          return location
        }
        const items = location.items.filter((item) => matchSearch(item, lower))
        return items.length > 0 ? { ...location, items } : null
      })
      .filter((location): location is Api.EveInfo.AssetLocationNode => location !== null)
  }, [data?.locations, keyword])

  const toggleLocation = (locationId: number) => {
    setCollapsedLocations((previous) => ({ ...previous, [locationId]: !(previous[locationId] ?? false) }))
  }

  const toggleItem = (itemId: number) => {
    setExpandedItems((previous) => ({ ...previous, [itemId]: !(previous[itemId] ?? false) }))
  }

  return (
    <section className="space-y-4">
      <h1 className="text-xl font-semibold">{t('infoAssets.title')}</h1>
      <div className="flex flex-wrap items-center gap-2 rounded-lg border bg-card p-4">
        <input
          className="rounded border px-2 py-1 text-sm"
          value={keyword}
          onChange={(event) => setKeyword(event.target.value)}
          placeholder={t('infoAssets.search')}
        />
      </div>
      {loading ? <p className="text-sm">{t('infoAssets.loading')}</p> : null}
      {error ? <p className="text-sm text-destructive">{error}</p> : null}
      {!loading && filteredLocations.length === 0 ? <p className="text-sm">{t('infoAssets.empty')}</p> : null}

      <div className="space-y-3">
        {filteredLocations.map((location) => {
          const collapsed = collapsedLocations[location.location_id] ?? false
          return (
            <div key={location.location_id} className="rounded-lg border bg-card">
              <button
                type="button"
                className="flex w-full items-center justify-between px-3 py-2 text-left text-sm font-medium"
                onClick={() => toggleLocation(location.location_id)}
              >
                <span>{location.location_name}</span>
                <span>{countItems(location.items)}</span>
              </button>
              {!collapsed ? (
                <div className="space-y-1 pb-2">
                  {location.items.map((item) => {
                    const hasChildren = item.children?.length ? true : false
                    const expanded = expandedItems[item.item_id] ?? false
                    return (
                      <div key={item.item_id}>
                        <button
                          type="button"
                          className="flex w-full items-center gap-2 px-3 py-2 text-left text-sm hover:bg-muted/40"
                          onClick={() => hasChildren && toggleItem(item.item_id)}
                        >
                          <span className="w-3 text-xs">{hasChildren ? (expanded ? '▾' : '▸') : ''}</span>
                          <img src={getIconUrl(item)} alt={item.type_name} className="h-8 w-8 rounded border" />
                          <span className="min-w-0 flex-1 truncate">{item.type_name}</span>
                          <span className="w-28 shrink-0 truncate text-right text-xs text-muted-foreground">
                            {item.group_name}
                          </span>
                          <span className="w-16 shrink-0 text-right text-xs text-muted-foreground">
                            {item.quantity > 1 ? `x${item.quantity}` : ''}
                          </span>
                          <span className="w-28 shrink-0 truncate text-right text-xs text-muted-foreground">
                            {item.character_name}
                          </span>
                        </button>
                        {hasChildren && expanded ? (
                          <div className="ml-8 space-y-1 border-l pl-3">
                            {item.children!.map((child) => (
                              <div key={child.item_id} className="flex items-center gap-2 px-3 py-2 text-sm">
                                <img src={getIconUrl(child)} alt={child.type_name} className="h-8 w-8 rounded border" />
                                <span className="min-w-0 flex-1 truncate">{child.type_name}</span>
                                <span className="w-28 shrink-0 truncate text-right text-xs text-muted-foreground">
                                  {child.group_name}
                                </span>
                                <span className="w-16 shrink-0 text-right text-xs text-muted-foreground">
                                  {child.quantity > 1 ? `x${child.quantity}` : ''}
                                </span>
                                <span className="w-28 shrink-0 truncate text-right text-xs text-muted-foreground">
                                  {child.character_name}
                                </span>
                              </div>
                            ))}
                          </div>
                        ) : null}
                      </div>
                    )
                  })}
                </div>
              ) : null}
            </div>
          )
        })}
      </div>
    </section>
  )
}
