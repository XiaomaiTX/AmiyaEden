import { dispatchUnauthorized } from '@/auth'
import { useSessionStore } from '@/stores'

export async function uploadImageAsDataUrl(file: File, url = '/api/v1/upload/image') {
  const formData = new FormData()
  formData.append('file', file)

  const headers = new Headers()
  const accessToken = useSessionStore.getState().accessToken
  if (accessToken) {
    headers.set('Authorization', `Bearer ${accessToken}`)
  }

  const response = await fetch(url, {
    method: 'POST',
    body: formData,
    headers,
  })

  if (response.status === 401) {
    dispatchUnauthorized({ reason: 'http_401' })
    throw new Error('Unauthorized')
  }

  if (!response.ok) {
    throw new Error(`Request failed: ${response.status}`)
  }

  const data = (await response.json()) as { code?: number; msg?: string; data?: { url?: string } }
  if ((data.code ?? 0) !== 0 && (data.code ?? 0) !== 200) {
    throw new Error(data.msg || 'upload image failed')
  }

  return data.data?.url ?? ''
}
