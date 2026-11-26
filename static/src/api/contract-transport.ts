import { AxiosHeaders, type AxiosRequestConfig } from 'axios'
import { request } from '@/utils/http'

type ContractHeaders = Parameters<typeof AxiosHeaders.concat>[number]
type ContractRequest = Omit<AxiosRequestConfig, 'headers'> & { headers?: ContractHeaders }
type ContractOptions = Pick<ContractRequest, 'timeout' | 'headers' | 'signal'> & {
  showErrorMessage?: boolean
}

export function contractRequest<T>(config: ContractRequest, options?: ContractOptions): Promise<T> {
  const headers = AxiosHeaders.concat(options?.headers, config.headers)
  if (config.data instanceof FormData) headers.delete('Content-Type')
  return request<T>({
    ...config,
    ...options,
    headers
  })
}
