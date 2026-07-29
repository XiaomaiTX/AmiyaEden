import { requestJson } from '@/api/http-client'
import type {
  FuxiHallCard,
  FuxiHallCardCreate,
  FuxiHallCardReorder,
  FuxiHallCardUpdate,
  FuxiHallPageConfig,
  FuxiHallPageConfigUpdate,
  FuxiHallPageKey,
  FuxiHallPublicPageResponse,
} from '@/types/api/fuxi-hall'

interface ApiResponse<T> { code: number; msg: string; data: T }

function unwrap<T>(response: ApiResponse<T>, fallback: string) {
  if (response.code !== 0 && response.code !== 200) throw new Error(response.msg || fallback)
  return response.data
}

async function get<T>(path: string, fallback: string) {
  return unwrap(await requestJson<ApiResponse<T>>(path), fallback)
}

async function send<T>(path: string, method: 'POST' | 'PUT' | 'DELETE', body: unknown, fallback: string) {
  return unwrap(await requestJson<ApiResponse<T>>(path, { method, body: body === undefined ? undefined : JSON.stringify(body) }), fallback)
}

export const fetchFuxiHallLeadership = () => get<FuxiHallPublicPageResponse>('/api/v1/fuxi-hall/leadership', 'Failed to load leadership')
export const fetchFuxiHallContributors = () => get<FuxiHallPublicPageResponse>('/api/v1/fuxi-hall/contributors', 'Failed to load contributors')
export const fetchFuxiHallPage = (pageKey: FuxiHallPageKey) => get<FuxiHallPageConfig>(`/api/v1/system/fuxi-hall/pages/${pageKey}`, 'Failed to load page configuration')
export const fetchFuxiHallCards = (pageKey: FuxiHallPageKey) => get<FuxiHallCard[]>(`/api/v1/system/fuxi-hall/cards/${pageKey}`, 'Failed to load cards')
export const updateFuxiHallPage = (pageKey: FuxiHallPageKey, data: FuxiHallPageConfigUpdate) => send<FuxiHallPageConfig>(`/api/v1/system/fuxi-hall/pages/${pageKey}`, 'PUT', data, 'Failed to save page configuration')
export const createFuxiHallCard = (data: FuxiHallCardCreate) => send<FuxiHallCard>('/api/v1/system/fuxi-hall/cards', 'POST', data, 'Failed to create card')
export const updateFuxiHallCard = (id: number, data: FuxiHallCardUpdate) => send<FuxiHallCard>(`/api/v1/system/fuxi-hall/cards/${id}`, 'PUT', data, 'Failed to save card')
export const reorderFuxiHallCards = (data: FuxiHallCardReorder) => send<null>('/api/v1/system/fuxi-hall/cards/reorder', 'PUT', data, 'Failed to reorder cards')
export const deleteFuxiHallCard = (id: number) => send<null>(`/api/v1/system/fuxi-hall/cards/${id}`, 'DELETE', undefined, 'Failed to delete card')
