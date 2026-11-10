import { dispatchUnauthorized } from '@/auth'
import { useSessionStore } from '@/stores'

export class HttpError extends Error {
  status: number

  constructor(status: number, message: string) {
    super(message)
    this.name = 'HttpError'
    this.status = status
  }
}

export async function requestJson<T>(input: RequestInfo | URL, init?: RequestInit): Promise<T> {
  const accessToken = useSessionStore.getState().accessToken
  const headers = new Headers(init?.headers)

  headers.set('Content-Type', 'application/json')
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
    throw new HttpError(response.status, `Request failed: ${response.status}`)
  }

  if (response.status === 204) {
    return undefined as T
  }

  const contentType = response.headers.get('content-type') ?? ''
  if (contentType.includes('application/json')) {
    const data = await response.json()
    if (
      data?.code === 409 &&
      data.msg === 'contact_already_bound' &&
      data.data?.primary_character_name
    ) {
      window.dispatchEvent(new CustomEvent('auth:contact-conflict', { detail: data.data }))
      throw new HttpError(409, data.msg)
    }
    return data as T
  }

  return (await response.text()) as T
}
