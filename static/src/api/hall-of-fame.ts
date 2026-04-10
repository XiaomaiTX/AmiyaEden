import request from '@/utils/http'
import { uploadImageAsDataUrl } from '@/api/upload'

// ─── Public ───

export function fetchTemple() {
  return request.get<Api.HallOfFame.TempleResponse>({
    url: '/api/v1/hall-of-fame/temple'
  })
}

// ─── Admin: Config ───

export function fetchHofConfig() {
  return request.get<Api.HallOfFame.Config>({
    url: '/api/v1/system/hall-of-fame/config'
  })
}

export function updateHofConfig(data: Api.HallOfFame.UpdateConfigParams) {
  return request.put<Api.HallOfFame.Config>({
    url: '/api/v1/system/hall-of-fame/config',
    data
  })
}

export function uploadHofBackground(file: File) {
  return uploadImageAsDataUrl(file, '/api/v1/system/hall-of-fame/upload-background')
}

// ─── Admin: Cards ───

export function fetchHofCards() {
  return request.get<Api.HallOfFame.Card[]>({
    url: '/api/v1/system/hall-of-fame/cards'
  })
}

export function createHofCard(data: Api.HallOfFame.CreateCardParams) {
  return request.post<Api.HallOfFame.Card>({
    url: '/api/v1/system/hall-of-fame/cards',
    data
  })
}

export function updateHofCard(id: number, data: Api.HallOfFame.UpdateCardParams) {
  return request.put<Api.HallOfFame.Card>({
    url: `/api/v1/system/hall-of-fame/cards/${id}`,
    data
  })
}

export function deleteHofCard(id: number) {
  return request.del({
    url: `/api/v1/system/hall-of-fame/cards/${id}`
  })
}

export function batchUpdateHofLayout(data: Api.HallOfFame.CardLayoutUpdate[]) {
  return request.put({
    url: '/api/v1/system/hall-of-fame/cards/batch-layout',
    data
  })
}
