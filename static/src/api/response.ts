import type { BusinessError, ContactConflict } from './generated/client'
import { HttpError } from '../utils/http/http-error'

export async function unwrapContract<T>(
  request: Promise<{ code: 200; msg: string; data: T } | BusinessError | ContactConflict>,
  fallback?: string
): Promise<T> {
  const response = await request
  if (response.code !== 200) {
    throw new HttpError(response.msg || fallback || 'Request failed', response.code, {
      data: response.data
    })
  }
  return response.data
}
