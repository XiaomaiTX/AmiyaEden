import { requestJson } from './http-client'

interface ContractRequest {
  url?: string
  method?: string
  data?: unknown
  params?: Record<string, unknown>
  headers?: HeadersInit
  signal?: AbortSignal
  responseType?: 'blob'
}

type ContractOptions = Pick<RequestInit, 'signal' | 'headers'>

export function contractRequest<T>(
  { url, method, data, params, headers, signal, responseType }: ContractRequest,
  options?: ContractOptions,
): Promise<T> {
  if (!url) throw new Error('Missing API request URL')
  const query = new URLSearchParams()
  for (const [name, value] of Object.entries(params ?? {})) {
    if (value === undefined || value === null) continue
    for (const item of Array.isArray(value) ? value : [value]) query.append(name, String(item))
  }
  const suffix = query.toString()
  const input = suffix ? `${url}${url.includes('?') ? '&' : '?'}${suffix}` : url
  const requestHeaders = new Headers(options?.headers)
  new Headers(headers).forEach((value, name) => requestHeaders.set(name, value))
  if (data instanceof FormData) requestHeaders.delete('Content-Type')
  const init: RequestInit = {
    method,
    headers: requestHeaders,
    body: data instanceof FormData || data === undefined ? data : JSON.stringify(data),
    signal: options?.signal ?? signal,
  }
  return responseType ? requestJson<T>(input, init, { responseType }) : requestJson<T>(input, init)
}
