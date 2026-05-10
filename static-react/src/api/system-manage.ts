import { requestJson } from '@/api/http-client'
import { assertSuccess, type ApiResponse } from '@/api/response'
import type {
  CorpTitleInfo,
  EsiRoleMapping,
  EsiTitleMapping,
  RoleDefinition,
  UserDetail,
  UserList,
  UserSearchParams,
} from '@/types/api/system-manage'

export async function fetchGetUserList(params?: UserSearchParams) {
  const query = params ? `?${new URLSearchParams(serializeQuery(params)).toString()}` : ''
  const response = await requestJson<ApiResponse<UserList>>(`/api/v1/system/user${query}`)
  return assertSuccess(response, 'fetch user list failed')
}

export async function fetchGetUser(id: number) {
  const response = await requestJson<ApiResponse<UserDetail>>(`/api/v1/system/user/${id}`)
  return assertSuccess(response, 'fetch user detail failed')
}

export async function fetchUpdateUser(
  id: number,
  data: { nickname?: string; qq?: string; discord_id?: string; status?: number }
) {
  const response = await requestJson<ApiResponse<null>>(`/api/v1/system/user/${id}`, {
    method: 'PUT',
    body: JSON.stringify(data),
  })
  assertSuccess(response, 'update user failed')
}

export async function fetchDeleteUser(id: number) {
  const response = await requestJson<ApiResponse<null>>(`/api/v1/system/user/${id}`, {
    method: 'DELETE',
  })
  assertSuccess(response, 'delete user failed')
}

export async function fetchImpersonateUser(id: number) {
  const response = await requestJson<ApiResponse<{ token: string; user: UserDetail }>>(
    `/api/v1/system/user/${id}/impersonate`,
    {
      method: 'POST',
    }
  )
  return assertSuccess(response, 'impersonate user failed')
}

export async function fetchGetUserRoles(userId: number) {
  const response = await requestJson<ApiResponse<RoleDefinition[]>>(
    `/api/v1/system/user/${userId}/roles`
  )
  return assertSuccess(response, 'fetch user roles failed')
}

export async function fetchSetUserRoles(userId: number, roleCodes: string[]) {
  const response = await requestJson<ApiResponse<null>>(`/api/v1/system/user/${userId}/roles`, {
    method: 'PUT',
    body: JSON.stringify({ role_codes: roleCodes }),
  })
  assertSuccess(response, 'set user roles failed')
}

export async function fetchGetRoleDefinitions() {
  const response = await requestJson<ApiResponse<RoleDefinition[]>>('/api/v1/system/role/definitions')
  return assertSuccess(response, 'fetch role definitions failed')
}

export async function fetchGetAllEsiRoles() {
  const response = await requestJson<ApiResponse<string[]>>('/api/v1/system/auto-role/esi-roles')
  return assertSuccess(response, 'fetch esi roles failed')
}

export async function fetchGetEsiRoleMappings() {
  const response = await requestJson<ApiResponse<EsiRoleMapping[]>>(
    '/api/v1/system/auto-role/esi-role-mappings'
  )
  return assertSuccess(response, 'fetch esi role mappings failed')
}

export async function fetchCreateEsiRoleMapping(data: { esi_role: string; role_code: string }) {
  const response = await requestJson<ApiResponse<EsiRoleMapping>>(
    '/api/v1/system/auto-role/esi-role-mappings',
    {
      method: 'POST',
      body: JSON.stringify(data),
    }
  )
  return assertSuccess(response, 'create esi role mapping failed')
}

export async function fetchDeleteEsiRoleMapping(id: number) {
  const response = await requestJson<ApiResponse<null>>(
    `/api/v1/system/auto-role/esi-role-mappings/${id}`,
    {
      method: 'DELETE',
    }
  )
  assertSuccess(response, 'delete esi role mapping failed')
}

export async function fetchGetEsiTitleMappings() {
  const response = await requestJson<ApiResponse<EsiTitleMapping[]>>(
    '/api/v1/system/auto-role/esi-title-mappings'
  )
  return assertSuccess(response, 'fetch esi title mappings failed')
}

export async function fetchCreateEsiTitleMapping(data: {
  corporation_id: number
  title_id: number
  title_name?: string
  role_code: string
}) {
  const response = await requestJson<ApiResponse<EsiTitleMapping>>(
    '/api/v1/system/auto-role/esi-title-mappings',
    {
      method: 'POST',
      body: JSON.stringify(data),
    }
  )
  return assertSuccess(response, 'create esi title mapping failed')
}

export async function fetchDeleteEsiTitleMapping(id: number) {
  const response = await requestJson<ApiResponse<null>>(
    `/api/v1/system/auto-role/esi-title-mappings/${id}`,
    {
      method: 'DELETE',
    }
  )
  assertSuccess(response, 'delete esi title mapping failed')
}

export async function fetchGetCorpTitles() {
  const response = await requestJson<ApiResponse<CorpTitleInfo[]>>('/api/v1/system/auto-role/corp-titles')
  return assertSuccess(response, 'fetch corp titles failed')
}

export async function fetchTriggerAutoRoleSync() {
  const response = await requestJson<ApiResponse<null>>('/api/v1/system/auto-role/sync', {
    method: 'POST',
  })
  assertSuccess(response, 'trigger auto role sync failed')
}

function serializeQuery(params: Record<string, unknown>) {
  return Object.fromEntries(
    Object.entries(params).flatMap(([key, value]) => {
      if (value === undefined || value === null || value === '') {
        return []
      }

      if (Array.isArray(value)) {
        return value.length > 0 ? [[key, value.join(',')]] : []
      }

      return [[key, String(value)]]
    })
  )
}
