import type { BusinessError, ContactConflict } from './generated/client'
import { HttpError } from './http-error'

type ContractResponse<T> = { code: 200; msg: string; data: T } | BusinessError | ContactConflict

export async function unwrapContract<T>(
  request: Promise<ContractResponse<T>>,
  fallbackMessage = 'Request failed',
): Promise<T> {
  const response = await request
  if (response.code !== 200) {
    throw new HttpError(response.code, response.msg || fallbackMessage)
  }
  return response.data
}
