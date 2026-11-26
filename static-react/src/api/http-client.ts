import { dispatchUnauthorized } from '@/auth'
import { useSessionStore } from '@/stores'
import { HttpError } from './http-error'

export { HttpError } from './http-error'

interface ResponseOptions {
  responseType?: 'blob'
}

export async function requestJson<T>(
  input: RequestInfo | URL,
  init?: RequestInit,
  options?: ResponseOptions
): Promise<T> {
  const accessToken = useSessionStore.getState().accessToken
  const headers = new Headers(init?.headers)

  if (!(init?.body instanceof FormData) && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json')
  }
  if (accessToken) {
    headers.set('Authorization', `Bearer ${accessToken}`)
  }

  const response = await fetch(input, {
    ...init,
    headers,
  })

  if (response.status === 401) {
    dispatchUnauthorized({ reason: 'http_401' })
    throw new HttpError(401, 'Unauthorized')
  }

  if (!response.ok) {
    const data = (await response.json().catch(() => null)) as { msg?: string } | null
    throw new HttpError(response.status, data?.msg || `Request failed: ${response.status}`)
  }

  if (response.status === 204) {
    return undefined as T
  }

  const mediaType = response.headers.get('content-type')?.split(';', 1)[0].trim().toLowerCase()
  if (mediaType === 'application/json' || mediaType?.endsWith('+json')) {
    const data = await response.json()
    if (data?.code === 401) {
      dispatchUnauthorized({ reason: 'http_401' })
      throw new HttpError(401, data.msg || 'Unauthorized')
    }
    if (
      data?.code === 409 &&
      data.msg === 'contact_already_bound' &&
      data.data?.primary_character_name
    ) {
      window.dispatchEvent(new CustomEvent('auth:contact-conflict', { detail: data.data }))
    }
    return data as T
  }

  if (options?.responseType === 'blob') return (await response.blob()) as T
  return (await response.text()) as T
}
