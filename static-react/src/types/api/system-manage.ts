import type { PaginatedResponse, CommonSearchParams } from '@/types/api/common'

export interface UserListCharacter {
  character_id: number
  character_name: string
  total_sp: number
  token_invalid: boolean
}

export interface UserListItem {
  id: number
  nickname: string
  qq: string
  discord_id: string
  primary_character_id: number
  status: number
  roles: string[]
  characters: UserListCharacter[]
  last_login_at: string | null
  last_login_ip: string
  created_at: string
  updated_at: string
}

export type UserList = PaginatedResponse<UserListItem>

export interface UserDetail {
  id: number
  nickname: string
  qq: string
  discord_id: string
  status: number
  role: string
  primary_character_id: number
  last_login_at: string | null
  last_login_ip: string
  created_at: string
  updated_at: string
}

export type UserSearchParams = Partial<{
  keyword: string
  status: number
  role: string
}> &
  Partial<CommonSearchParams>

export interface RoleDefinition {
  code: string
  name: string
  description: string
  sort: number
}

export interface EsiRoleMapping {
  id: number
  esi_role: string
  role_code: string
  role_name: string
  created_at: string
}

export interface EsiTitleMapping {
  id: number
  corporation_id: number
  title_id: number
  title_name: string
  role_code: string
  role_name: string
  created_at: string
}

export interface CorpTitleInfo {
  corporation_id: number
  corporation_name: string
  title_id: number
  title_name: string
}
