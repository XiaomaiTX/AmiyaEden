export interface ApiResponse<T> {
  code: number
  msg: string
  data: T
}

export function assertSuccess<T>(response: ApiResponse<T>, fallbackMessage: string) {
  if (response.code !== 200 && response.code !== 0) {
    throw new Error(response.msg || fallbackMessage)
  }

  return response.data
}
