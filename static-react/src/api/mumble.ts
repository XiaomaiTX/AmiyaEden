import { requestJson } from '@/api/http-client'
import type { MumbleCredentialResult, MumbleCredentialStatus } from '@/types/api/mumble'

interface ApiResponse<T> { code: number; msg: string; data: T }

function dataOrThrow<T>(response: ApiResponse<T>): T {
  if (response.code !== 0 && response.code !== 200) throw new Error(response.msg || 'Mumble 凭据请求失败')
  return response.data
}

export async function fetchMumbleCredential() {
  return dataOrThrow(await requestJson<ApiResponse<MumbleCredentialStatus>>('/api/v1/mumble/credential'))
}

export async function createMumbleCredential() {
  return dataOrThrow(await requestJson<ApiResponse<MumbleCredentialResult>>('/api/v1/mumble/credential', { method: 'POST' }))
}

export async function rotateMumbleCredential() {
  return dataOrThrow(await requestJson<ApiResponse<MumbleCredentialResult>>('/api/v1/mumble/credential/rotate', { method: 'POST' }))
}

export async function revokeMumbleCredential() {
  dataOrThrow(await requestJson<ApiResponse<null>>('/api/v1/mumble/credential', { method: 'DELETE' }))
}
