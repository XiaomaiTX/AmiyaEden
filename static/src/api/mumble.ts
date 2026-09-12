import request from '@/utils/http'

export function fetchMumbleCredential() {
  return request.get<Api.Mumble.CredentialStatus>({ url: '/api/v1/mumble/credential' })
}

export function createMumbleCredential() {
  return request.post<Api.Mumble.CredentialResult>({ url: '/api/v1/mumble/credential' })
}

export function rotateMumbleCredential() {
  return request.post<Api.Mumble.CredentialResult>({ url: '/api/v1/mumble/credential/rotate' })
}

export function revokeMumbleCredential() {
  return request.del({ url: '/api/v1/mumble/credential' })
}
