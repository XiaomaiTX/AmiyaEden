const HALL_OF_FAME_PREVIEW_STORAGE_PREFIX = 'hall-of-fame-preview:'

export const HALL_OF_FAME_PREVIEW_QUERY_KEY = 'preview'

type PreviewStorage = Pick<Storage, 'getItem' | 'setItem' | 'removeItem'>

export function saveHallOfFamePreviewDraft(
  storage: PreviewStorage,
  baseHref: string,
  payload: Api.HallOfFame.TempleResponse
) {
  const previewId = createPreviewId()
  storage.setItem(buildPreviewStorageKey(previewId), JSON.stringify(payload))

  const previewUrl = new URL(baseHref, 'https://hall-of-fame.preview.local')
  previewUrl.searchParams.set(HALL_OF_FAME_PREVIEW_QUERY_KEY, previewId)

  return `${previewUrl.pathname}${previewUrl.search}${previewUrl.hash}`
}

export function readHallOfFamePreviewDraft(storage: PreviewStorage, previewId: string) {
  if (!previewId) {
    return null
  }

  const key = buildPreviewStorageKey(previewId)
  const rawDraft = storage.getItem(key)
  if (!rawDraft) {
    return null
  }

  try {
    const draft = JSON.parse(rawDraft) as Api.HallOfFame.TempleResponse
    storage.removeItem(key)
    return draft
  } catch {
    storage.removeItem(key)
    return null
  }
}

function buildPreviewStorageKey(previewId: string) {
  return `${HALL_OF_FAME_PREVIEW_STORAGE_PREFIX}${previewId}`
}

function createPreviewId() {
  return `${Date.now()}-${Math.random().toString(36).slice(2, 10)}`
}
