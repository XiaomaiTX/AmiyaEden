import { useCallback, useEffect, useRef, useState } from 'react'
import { fetchInfoAssetLocations, fetchInfoAssetLocationItems, fetchInfoAssetChildren } from '@/api/eve-info'
import { useI18n } from '@/i18n'
import type { AssetListItemNode, AssetLocationItemsResponse, AssetLocationSummary } from '@/types/api/eve-info'

const BLUEPRINT_CATEGORY_ID = 9

function getIconUrl(item: { category_id: number; type_id: number; is_blueprint_copy?: boolean }) {
  if (item.category_id === BLUEPRINT_CATEGORY_ID) {
    const suffix = item.is_blueprint_copy ? 'bpc' : 'bp'
    return `https://images.evetech.net/types/${item.type_id}/${suffix}?size=32`
  }
  return `https://images.evetech.net/types/${item.type_id}/icon?size=32`
}

export function InfoAssetsPage() {
  const { t } = useI18n()
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [locations, setLocations] = useState<AssetLocationSummary[]>([])
  const [totalLocations, setTotalLocations] = useState(0)
  const [totalItems, setTotalItems] = useState(0)
  const [keyword, setKeyword] = useState('')
  const locationsRequestSeq = useRef(0)

  // Expanded locations
  const [expandedLocations, setExpandedLocations] = useState<Record<number, boolean>>({})
  // Location items cache
  const [locationItems, setLocationItems] = useState<Record<number, AssetListItemNode[]>>({})
  const [locationLoading, setLocationLoading] = useState<Record<number, boolean>>({})
  const [locationErrors, setLocationErrors] = useState<Record<number, string>>({})

  // Expanded children
  const [expandedItems, setExpandedItems] = useState<Record<number, boolean>>({})
  // Children cache
  const [children, setChildren] = useState<Record<number, AssetListItemNode[]>>({})
  const [childrenLoading, setChildrenLoading] = useState<Record<number, boolean>>({})
  const [childrenErrors, setChildrenErrors] = useState<Record<number, string>>({})

  const loadLocations = useCallback(async (searchKeyword?: string) => {
    locationsRequestSeq.current += 1
    const requestId = locationsRequestSeq.current
    setLoading(true)
    setError(null)
    try {
      const response = await fetchInfoAssetLocations({
        language: 'en',
        page: 1,
        page_size: 20,
        keyword: searchKeyword || undefined,
      })
      if (requestId !== locationsRequestSeq.current) return
      setLocations(response.locations)
      setTotalLocations(response.total_locations)
      setTotalItems(response.total_items)
      // Clear all caches on search refresh
      setExpandedLocations({})
      setLocationItems({})
      setLocationLoading({})
      setLocationErrors({})
      setExpandedItems({})
      setChildren({})
      setChildrenLoading({})
      setChildrenErrors({})
    } catch {
      if (requestId !== locationsRequestSeq.current) return
      setError(t('infoAssets.locationsFailed'))
      setLocations([])
    } finally {
      if (requestId === locationsRequestSeq.current) setLoading(false)
    }
  }, [t])

  useEffect(() => {
    let cancelled = false
    locationsRequestSeq.current += 1
    const requestId = locationsRequestSeq.current
    async function run() {
      setLoading(true)
      setError(null)
      try {
        const response = await fetchInfoAssetLocations({
          language: 'en',
          page: 1,
          page_size: 20,
        })
        if (cancelled || requestId !== locationsRequestSeq.current) return
        setLocations(response.locations)
        setTotalLocations(response.total_locations)
        setTotalItems(response.total_items)
        setExpandedLocations({})
        setLocationItems({})
        setLocationLoading({})
        setLocationErrors({})
        setExpandedItems({})
        setChildren({})
        setChildrenLoading({})
        setChildrenErrors({})
      } catch {
        if (cancelled || requestId !== locationsRequestSeq.current) return
        setError(t('infoAssets.locationsFailed'))
        setLocations([])
      } finally {
        if (!cancelled && requestId === locationsRequestSeq.current) setLoading(false)
      }
    }
    void run()
    return () => { cancelled = true }
  }, [t])

  const loadLocationItemsFor = useCallback(async (loc: AssetLocationSummary) => {
    setLocationLoading((prev) => ({ ...prev, [loc.location_id]: true }))
    setLocationErrors((prev) => ({ ...prev, [loc.location_id]: '' }))
    try {
      const response: AssetLocationItemsResponse = await fetchInfoAssetLocationItems({
        language: 'en',
        location_id: loc.location_id,
        page: 1,
        page_size: 50,
      })
      setLocationItems((prev) => ({ ...prev, [loc.location_id]: response.items }))
    } catch {
      setLocationErrors((prev) => ({ ...prev, [loc.location_id]: t('infoAssets.locationItemsFailed') }))
    } finally {
      setLocationLoading((prev) => ({ ...prev, [loc.location_id]: false }))
    }
  }, [t])

  const toggleLocation = (loc: AssetLocationSummary) => {
    const locId = loc.location_id
    const currently = expandedLocations[locId] ?? false
    if (currently) {
      setExpandedLocations((prev) => ({ ...prev, [locId]: false }))
      return
    }
    setExpandedLocations((prev) => ({ ...prev, [locId]: true }))
    if (!locationItems[locId]) {
      void loadLocationItemsFor(loc)
    }
  }

  const loadChildrenFor = useCallback(async (itemId: number) => {
    setChildrenLoading((prev) => ({ ...prev, [itemId]: true }))
    setChildrenErrors((prev) => ({ ...prev, [itemId]: '' }))
    try {
      const response = await fetchInfoAssetChildren({
        language: 'en',
        parent_item_id: itemId,
      })
      setChildren((prev) => ({ ...prev, [itemId]: response.items }))
    } catch {
      setChildrenErrors((prev) => ({ ...prev, [itemId]: t('infoAssets.childrenFailed') }))
    } finally {
      setChildrenLoading((prev) => ({ ...prev, [itemId]: false }))
    }
  }, [t])

  const toggleItem = (item: AssetListItemNode) => {
    const itemId = item.item_id
    const currently = expandedItems[itemId] ?? false
    if (currently) {
      setExpandedItems((prev) => ({ ...prev, [itemId]: false }))
      return
    }
    setExpandedItems((prev) => ({ ...prev, [itemId]: true }))
    if (!children[itemId]) {
      void loadChildrenFor(itemId)
    }
  }

  const handleSearch = (value: string) => {
    setKeyword(value)
    void loadLocations(value)
  }

  const renderItemRow = (item: AssetListItemNode) => {
    const hasKids = item.has_children
    const itemExpanded = expandedItems[item.item_id] ?? false
    const kids = children[item.item_id]
    const kidsLoading = childrenLoading[item.item_id] ?? false
    const kidsError = childrenErrors[item.item_id]

    return (
      <div key={item.item_id}>
        <button
          type="button"
          className="flex w-full items-center gap-2 px-3 py-2 text-left text-sm hover:bg-muted/40"
          onClick={() => hasKids && toggleItem(item)}
        >
          <span className="w-3 text-xs">{hasKids ? (itemExpanded ? '▾' : '▸') : ''}</span>
          <img src={getIconUrl(item)} alt={item.type_name} className="h-8 w-8 rounded border" />
          <span className="min-w-0 flex-1 truncate">{item.type_name}</span>
          {item.child_count > 0 ? (
            <span className="rounded bg-muted px-1 text-xs text-muted-foreground">{item.child_count}</span>
          ) : null}
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
        {hasKids && itemExpanded ? (
          <div className="ml-8 space-y-1 border-l pl-3">
            {kidsError ? <p className="px-3 py-1 text-xs text-destructive">{kidsError}</p> : null}
            {kidsLoading ? <p className="px-3 py-1 text-xs">{t('infoAssets.loading')}</p> : null}
            {kids?.map((child) => (
              <div key={child.item_id} className="flex items-center gap-2 px-3 py-2 text-sm">
                <img src={getIconUrl(child)} alt={child.type_name} className="h-8 w-8 rounded border" />
                <span className="min-w-0 flex-1 truncate">{child.type_name}</span>
                {child.child_count > 0 ? (
                  <span className="rounded bg-muted px-1 text-xs text-muted-foreground">{child.child_count}</span>
                ) : null}
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
  }

  return (
    <section className="space-y-4">
      <h1 className="text-xl font-semibold">{t('infoAssets.title')}</h1>
      <div className="flex flex-wrap items-center gap-2 rounded-lg border bg-card p-4">
        <input
          className="rounded border px-2 py-1 text-sm"
          value={keyword}
          onChange={(event) => handleSearch(event.target.value)}
          placeholder={t('infoAssets.search')}
        />
        <span className="text-xs text-muted-foreground" data-testid="asset-stats">
          {t('info.assetCount')}: {totalItems} | {t('info.locationName')}: {totalLocations}
        </span>
      </div>
      {loading ? <p className="text-sm">{t('infoAssets.loading')}</p> : null}
      {error ? <p className="text-sm text-destructive">{error}</p> : null}
      {!loading && !error && locations.length === 0 ? <p className="text-sm">{t('infoAssets.empty')}</p> : null}

      <div className="space-y-3">
        {locations.map((loc) => {
          const collapsed = !(expandedLocations[loc.location_id] ?? false)
          const items = locationItems[loc.location_id]
          const locLoading = locationLoading[loc.location_id] ?? false
          const locError = locationErrors[loc.location_id]

          return (
            <div key={loc.location_id} className="rounded-lg border bg-card">
              <button
                type="button"
                className="flex w-full items-center justify-between px-3 py-2 text-left text-sm font-medium"
                onClick={() => toggleLocation(loc)}
              >
                <span>{loc.location_name}</span>
                <span className="flex gap-3 text-xs text-muted-foreground">
                  <span>{loc.top_level_count} items</span>
                  <span>{loc.character_count} chars</span>
                </span>
              </button>
              {!collapsed ? (
                <div className="space-y-1 pb-2">
                  {locError ? <p className="px-3 py-1 text-xs text-destructive">{locError}</p> : null}
                  {locLoading ? <p className="px-3 py-1 text-xs">{t('infoAssets.loading')}</p> : null}
                  {!locLoading && !locError && items?.length === 0 ? (
                    <p className="px-3 py-1 text-xs text-muted-foreground">{t('info.assetNoItemsInLocation')}</p>
                  ) : null}
                  {items?.map((item) => renderItemRow(item))}
                </div>
              ) : null}
            </div>
          )
        })}
      </div>
    </section>
  )
}
