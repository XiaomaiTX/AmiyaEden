import request from '@/utils/http'

// ─── Logged-in directory view ───

export function fetchFuxiAdminDirectory() {
  return request.get<Api.FuxiAdmin.DirectoryResponse>({
    url: '/api/v1/fuxi-admins',
    showErrorMessage: false
  })
}

export function fetchFuxiAdminManageDirectory() {
  return request.get<Api.FuxiAdmin.ManageDirectoryResponse>({
    url: '/api/v1/system/fuxi-admins/manage-directory',
    showErrorMessage: false
  })
}

// ─── Admin: Config ───

export function fetchFuxiAdminConfig() {
  return request.get<Api.FuxiAdmin.Config>({
    url: '/api/v1/system/fuxi-admins/config'
  })
}

export function updateFuxiAdminConfig(data: Api.FuxiAdmin.UpdateConfigParams) {
  return request.put<Api.FuxiAdmin.Config>({
    url: '/api/v1/system/fuxi-admins/config',
    data
  })
}

// ─── Admin: Tiers ───

export function fetchFuxiAdminTiers() {
  return request.get<Api.FuxiAdmin.Tier[]>({
    url: '/api/v1/system/fuxi-admins/tiers'
  })
}

export function createFuxiAdminTier(data: Api.FuxiAdmin.CreateTierParams) {
  return request.post<Api.FuxiAdmin.Tier>({
    url: '/api/v1/system/fuxi-admins/tiers',
    data
  })
}

export function updateFuxiAdminTier(id: number, data: Api.FuxiAdmin.UpdateTierParams) {
  return request.put<Api.FuxiAdmin.Tier>({
    url: `/api/v1/system/fuxi-admins/tiers/${id}`,
    data
  })
}

export function deleteFuxiAdminTier(id: number) {
  return request.del({
    url: `/api/v1/system/fuxi-admins/tiers/${id}`
  })
}

// ─── Admin: Admins ───

export function createFuxiAdmin(data: Api.FuxiAdmin.CreateAdminParams) {
  return request.post<Api.FuxiAdmin.ManageAdmin>({
    url: '/api/v1/system/fuxi-admins',
    data
  })
}

export function updateFuxiAdmin(id: number, data: Api.FuxiAdmin.UpdateAdminParams) {
  return request.put<Api.FuxiAdmin.ManageAdmin>({
    url: `/api/v1/system/fuxi-admins/${id}`,
    data
  })
}

export function deleteFuxiAdmin(id: number) {
  return request.del({
    url: `/api/v1/system/fuxi-admins/${id}`
  })
}
